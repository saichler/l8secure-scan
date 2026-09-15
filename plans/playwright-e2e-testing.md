# Plan: Playwright End-to-End Testing (all UI functionality, desktop + mobile)

## 0. Why this plan exists / scope

Every existing test in this project (`go/tests/*_test.go`) exercises the backend through `IVNic`/HTTP only, per `TestLocationAndApproach` — none of them open a real browser, click a real button, or watch a real page update. The generic-websocket-change-notifications work (this session) was verified with hand-written `curl`/Node `WebSocket` scripts, not a repeatable test suite.

This plan adds a **Playwright** test suite that drives the real deployed app (`app.html` desktop + `m/app.html` mobile) in a real browser against the live KIND cluster, covering every functional area in `plans/image-security-scan-dashboard-prd.md` §11 plus the live-progress-bar feature from `plans/scanjob-live-progress.md`. Backend-only edge cases already covered by `go/tests/` (§19 of the PRD) are **not** re-implemented here — this suite tests what only a browser can verify: navigation, rendering, forms, table interactions, popups, and the websocket-driven live UI update.

### Duplication audit (`PlanRequirements`)

No existing UI test code exists to duplicate. To avoid the suite itself becoming 2×+ duplicated behavioral code across ~25 spec files, Phase 1 builds a shared fixture/page-object layer (login, navigation, table helpers, popup helpers, API-seeding helpers) that every later phase's specs reuse — this is the extraction-first step applied before, not after, duplication would occur.

## 1. New directory: `e2e/` (project root, sibling to `go/`, `k8s/`, `plans/`)

```
e2e/
├── playwright.config.ts
├── package.json
├── .gitignore                    # node_modules/, playwright-report/, test-results/
├── fixtures/
│   ├── auth.ts                   # login as opsadmin / customer role, storageState reuse
│   ├── api.ts                    # direct HTTP helpers to seed/clean up data via the real API
│   │                              #   (never DB access directly -- same boundary the UI itself uses)
│   └── selectors.ts               # centralized data-* selectors / container IDs
├── pages/                         # page-object layer, one per l8ui surface reused across specs
│   ├── login.page.ts
│   ├── nav.page.ts                # sidebar + module/submodule/service navigation (desktop)
│   ├── table.page.ts              # Layer8DTable wrapper: row click, sort, filter, paginate, select
│   ├── popup.page.ts              # Layer8DPopup wrapper: open/close, form fields, save, tabs
│   ├── mobile-nav.page.ts
│   └── mobile-table.page.ts
├── tests/
│   ├── desktop/
│   │   ├── login.spec.ts
│   │   ├── dashboard.spec.ts
│   │   ├── categories.spec.ts
│   │   ├── add-images.spec.ts
│   │   ├── group-detail.spec.ts
│   │   ├── vulnerability-detail.spec.ts
│   │   ├── scan-live-progress.spec.ts
│   │   ├── csv-export.spec.ts
│   │   ├── customers-admin.spec.ts
│   │   └── multi-tenancy.spec.ts
│   └── mobile/
│       ├── login.spec.ts
│       ├── vulnmgmt.spec.ts        # groups/categories/scanjobs list+CRUD, one file (mirrors nav-config shape)
│       ├── group-detail.spec.ts
│       ├── scan-live-progress.spec.ts
│       └── customers-admin.spec.ts
└── README.md                      # how to run, env vars, credential expectations
```

Rationale for `e2e/` at project root rather than under `go/`: `TestLocationAndApproach`'s "all tests in `go/tests/`" rule is scoped to Go `_test.go` files exercising the Go module; this is a separate-language, separate-runtime suite (Node/TypeScript), same reasoning `k8s/` already lives outside `go/`.

## 2. Environment & credentials (no hardcoded secrets)

`playwright.config.ts` reads from env vars, defaulted for local KIND use:

| Var | Default | Purpose |
|---|---|---|
| `BASE_URL` | `https://172.18.0.6:2790` | node IP of the KIND cluster (`hostNetwork:true`) |
| `OPSADMIN_USER` / `OPSADMIN_PASS` | `opsadmin` / `opsadmin` | already-known bootstrap identity |
| `CUSTOMER_USER` / `CUSTOMER_PASS` | `local-user` / `Vx9!TangoQm` | the already-provisioned `customer`-role user for tenant `local` (confirmed via `/scan/73/users`, `userId:"local-user"`, `customer:"local"`; password from `go/tests/mocks/seed.go`'s `localUserPassword` constant, verified with a live login returning `customer:"local"`) |

`ignoreHTTPSErrors: true` (self-signed cert, same as the hand-written `ws_test*.mjs` scripts used earlier this session). No `.env` file with real secrets is committed — `e2e/README.md` documents the vars, CI/local runs export them.

## 3. Data lifecycle strategy

Tests run against the **real** live cluster/Postgres, not a mock. To avoid polluting real data or flaking on leftover state from prior runs:

- Every entity a test creates (Category, Customer, ImageGroup/ImageRef via Add Images, ScanJob) is named/tagged with a per-run-unique prefix (`e2e-<timestamp>-...`).
- `fixtures/api.ts` provides `cleanupByPrefix()` helpers that DELETE anything matching that prefix via the real REST endpoints (same ones the UI uses) in an `afterAll`/`afterEach` — never direct DB access.
- Read-only assertions against pre-existing data (e.g. the `local-user`/`opsadmin` seeded users, already-seeded `ScanJob`/`ImageRef` rows from this session's manual testing) are allowed but must not assume exact counts — assert *presence of the row this test created*, never *total row count*, per the same reasoning as `Layer8DTablePaginationMetadata`.

## 4. Traceability Matrix

| # | Area | Gap / Action Item | Platform | Phase |
|---|---|---|---|---|
| 1 | Infra | Playwright project scaffold, config, fixtures, page objects | Both | 1 |
| 2 | Login | Valid login (opsadmin), invalid login error, logout | Desktop | 2 |
| 3 | Login | Valid login (customer role), session persists `userCustomer` | Desktop | 2 |
| 4 | Login | Mobile login + logout | Mobile | 5 |
| 5 | Dashboard | KPI strip renders real numbers, `ImageGroup` table loads, sort/filter, severity bar chart renders | Desktop | 3 |
| 6 | Categories | CRUD: create, edit (name/color), delete; reference picker sees new category on `ImageGroup` edit | Desktop | 3 |
| 7 | Add Images | Bulk paste with valid/duplicate/unparseable lines → correct created/skipped/error summary shown, no silent drops | Desktop | 3 |
| 8 | Group Detail | Trend panel newest/oldest counts + reduction %, `ImageRef` table sorted by build date desc, checkbox multi-select | Desktop | 3 |
| 9 | Group Detail → Scan | "Scan Selected" → real `POST /60/ScanJob`, success notification, row status flips off `PENDING` | Desktop | 3 |
| 10 | Vulnerability Detail | Row click opens popup, summary strip counts, findings table sorted severity desc | Desktop | 3 |
| 11 | Live Progress | Real scan run: progress bar attaches, updates via websocket (no polling) as `ScanJob`/`ImageRef` status changes, reaches a terminal state | Desktop | 4 |
| 12 | Live Progress | Same, mobile (`Layer8DProgressBar` reused directly per this session's Phase 5-m) | Mobile | 5 |
| 13 | CSV Export | "Export CSV Report" downloads a file with the 15-column set for a real group | Desktop | 3 |
| 14 | Customers (admin) | opsadmin: CRUD on `Customer`; nav link visible but customer-role write rejected server-side | Desktop | 4 |
| 15 | Multi-tenancy | `customer` role sees only its own `ImageGroup`/`ImageRef`/`ScanJob`/`ImageCategory` rows; `opsadmin` sees all tenants + `Customer` catalog | Desktop | 4 |
| 16 | Mobile parity | Groups (customInit list), Categories/ScanJobs/Customers CRUD via generic pipeline, Group/Vuln Detail popups, checkbox multi-select | Mobile | 5 |
| 17 | Full suite | All specs green in one run against a freshly-deployed cluster; flake triage | Both | 6 |

## 5. Phases

### Phase 1 — Scaffold
`e2e/` skeleton, `playwright.config.ts` (projects: `chromium-desktop` at desktop viewport, `chromium-mobile` at a phone viewport per this project's own `AskUserQuestion`-free convention of testing the real `m/` bundle, not a responsive resize of the desktop bundle — `m/app.html` is a separate URL), `fixtures/auth.ts` using Playwright's `storageState` so login runs once per role and is reused, `fixtures/api.ts` seeding/cleanup helpers, base page objects (`login`, `nav`, `table`, `popup`). One smoke spec: log in as opsadmin, land on dashboard, log out.

### Phase 2 — Auth
`tests/desktop/login.spec.ts`: valid opsadmin login, valid customer-role login (needs `CUSTOMER_PASS`), invalid credentials shows error, logout clears session and redirects to login.

### Phase 3 — Vulnerability Management (desktop, opsadmin or seeded customer context)
One spec file per PRD §11 subsection: dashboard/groups, categories, add-images, group-detail (trend + scan-selected), vulnerability-detail, csv-export. Each seeds its own data via `fixtures/api.ts`, drives the UI, asserts on rendered DOM state (not just the API response).

### Phase 4 — Admin, multi-tenancy, live progress (desktop)
`customers-admin.spec.ts` (opsadmin CRUD + nav-visible-but-denied assertion for a customer-role token calling the API directly), `multi-tenancy.spec.ts` (log in as `local-user`, assert dashboard/groups/scanjobs/categories show only tenant `local` rows; log in as opsadmin, assert full visibility), `scan-live-progress.spec.ts` (trigger a real `Scan Selected`, watch the progress bar reach a terminal state driven by the websocket push this session just fixed — asserts on DOM text/attribute changes over time via `page.waitForFunction`, not on a fixed sleep).

### Phase 5 — Mobile parity
Mirrors Phase 2–4 against `m/app.html`'s actual nav/list/popup mechanics (`Layer8MNav`, `Layer8MEditTable`, `Layer8MPopup`, the `customInit` groups list) — per `PlatformConversionDataFlow`, each mobile spec is written from tracing the real mobile API (already documented in PRD §11.7), not assumed to mirror desktop 1:1.

### Phase 6 — Final verification
Full suite run (`npx playwright test`) against a freshly-restarted deployment (clean pod state), all specs green, HTML report reviewed, flaky tests (if any) triaged and fixed or explicitly quarantined with a documented reason — never silently skipped.

## 6. Out of scope (explicit)

- Backend edge cases already covered by `go/tests/` (§19 of the PRD) — not re-verified at the UI layer redundantly.
- `System` section (Health/Security/Logs) — generic `l8ui` built-ins, not this project's own functionality; not part of "all this project's functionality."
- Load/performance testing, visual regression screenshots, cross-browser matrix (Firefox/WebKit) — single Chromium project only unless you ask otherwise.
- TFA flows — not configured for either seeded test user.
