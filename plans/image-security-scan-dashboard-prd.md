# PRD: Layer 8 Image Security Scan Dashboard (l8secure-scan)

Status: Draft for review — do not implement until explicitly approved (`PlanRequirements` / `NeverActOnQuestions`).

Source ask: `ask.txt` (repo root).
Guide lines applied: `guide-lines/layer-8-ecosystem.md`, `guide-lines/layer-8-guide-lines.md` (compaction of `../l8book/rules/*`).
Canonical reference project: `../l8erp` (ERP-style, CRUD + persistence — this is not an observation/collection project, so `probler`/`l8pollaris` patterns do not apply, per `CanonicalProjectSelection`).

---

## 1. Purpose

Layer 8 Secure Scan is a multi-customer SaaS dashboard for managing container image vulnerabilities. It ingests image references pushed by customers' build/CI pipelines, groups them by logical image name, scans them on demand with **Trivy**, and gives each customer a single-tenant view of their vulnerability posture, trends, and remediation categorization — plus a CSV report suitable for sharing outside the tool.

## 2. Goals

- G1: Group image references by **image name only** (ignore registry/repo path and tag) so the same logical image built across repos/tags rolls up into one group.
- G2: Show image references within a group **sorted descending by build date** (latest first).
- G3: Let a user **select one or many** image references inside a group and trigger a **Trivy** scan.
- G4: After scanning, show per-image **total (sum-up)** and **distinct** vulnerability counts by severity (Critical/High/Medium/Low), plus a **detailed CVE list sorted Critical→Low**.
- G5: Let users **categorize** image groups with a tag/category, and fully manage (CRUD) those categories from the app.
- G6: Produce a **CSV report**, one row per image group, with the columns specified in §10.
- G7: Serve many customers from one deployment, but every UI session shows **exactly one customer's** data (multi-tenant row-level isolation).
- G8: Let a user **bulk-register** new image references by pasting a free-text list (one or many refs at once) rather than adding them one at a time.

## 3. Non-Goals (out of scope for this PRD)

- Automatic, continuous discovery/crawling of customer registries (see §6.1/§18 — ingestion is bulk free-text paste through the UI, or the same `ImgRefAdd` endpoint called directly by a CI pipeline; no scheduled crawl/scan of a registry's tag list).
- Actual remediation automation (e.g., auto-opening a PR to bump a base image). "Remediate" here means: visibility + categorization + scan-driven trend data to drive a human remediation workflow.
- Registry credential vaulting UI (the scanner pod is provisioned with registry access out-of-band — see §18).
- Scanners other than Trivy.

## 4. Users & Multi-Tenancy Model

Two roles, provisioned only through the Security API / security config JSON (`SecurityRules`, `SecurityConfigStructure` — never a project-owned users table, never `import l8secure`):

| Role | Scope | Notes |
|---|---|---|
| `customer` | One customer's data only | `L8User.associate_ids = [customerId]` (single-element list). Reuses the `AssociateIdsScopeView` pattern exactly as documented — no new placeholder is invented. |
| `opsadmin` | All customers | Internal Layer 8 support/ops role, allow-all rule, used to manage the `Customer` catalog itself and for support triage. Committed v1 scope (confirmed); UI-gated to the `System` area, not exposed to customer users. |

Row-level scoping (deny rule, applied to every customer-scoped Prime Object — `ImageCategory`, `ImageGroup`, `ImageRef`, `ScanJob`):

```json
{
  "ruleId": "customer-scope-imagegroup",
  "elemType": "ImageGroup",
  "allowed": false,
  "actions": {},
  "attributes": {
    "ImageGroup": "select * from ImageGroup where customerId not in ${associateIds}"
  }
}
```

One rule per elemType (`ImageCategory`, `ImageGroup`, `ImageRef`, `ScanJob`), same shape, per `AssociateIdsScopeView` / `SecurityConfigStructure`. No custom filtering is added in any ServiceCallback (`SecurityConfigStructure`: "Do NOT implement custom data filtering in ServiceCallbacks").

Because the UI only ever renders one customer's rows, no "customer switcher" UI is needed — the deny rule guarantees a `customer` user's queries already return only their tenant's data. `opsadmin` sees all customers; that role is not covered by the deny rule (allow-all `Customer` management rule).

**Read-side scoping is enforced by the framework; write-side `customer_id` is a trusted-client value, by design.** Per `SecurityConfigStructure`'s own pipeline note, the deny rule's `ScopeView()` filters **GET results** — confirmed against `l8secure/go/secure/provider/ScopeView.go`: it only ever operates on the response, never the incoming request. Verified against the actual source (`l8types/go/ifs/ServiceLevelAgreement.go`, `l8services/go/services/base/BaseService*.go`), `IServiceCallback.Before()`/`.After()` never receive the caller's identity at all — no `AAAId`, no session, nothing; the only place it exists is one layer up in `ServiceManager.Handle()`, used solely for the coarse type+action `CanDoAction` check and for `ScopeView` on the response. So there is **no extension point available to a project** that could validate or override a write's `customer_id` against the caller's identity — this is a framework gap, not something closable with more careful `ServiceCallback` code (adding an `AAAId` parameter to `IServiceCallback` would be a fundamental interface change, per `FrameworkInterfaceBoundaries` something to flag to the framework owner, not add locally).

**Accepted for v1:** every write in this app carries a client-supplied `customer_id`, sourced by the UI from the logged-in user's own session/customer context (the UI never lets a user type or pick a different `customerId` — it's implicit, not a form field). The threat model this PRD covers is the UI itself and the standard credential/session boundary (a `customer`-role bearer token only exists for, and is only ever used by, that customer's own session); a maliciously crafted direct API call using a valid token to name a *different* tenant's `customerId` in a write is **out of scope** — accepted, not mitigated, per explicit product decision. `CanDoAction`'s coarse allow/deny check and `ScopeView`'s read-side filtering remain the real security boundary; nothing here weakens those.

## 5. Terminology & Grouping Algorithm

- **Image reference**: a fully-qualified string a build pushes, e.g. `registry.example.com/myorg/backend:v1.4.2@sha256:...`.
- **repoName**: everything before the tag (`registry.example.com/myorg/backend`).
- **tag**: the tag component (`v1.4.2`), may be empty if only a digest is supplied.
- **digest**: the `sha256:...` component if present.
- **imageName (grouping key)**: the **last path segment** of `repoName`, lower-cased (`backend`). This is what "group by image name, regardless of repo and tag" means: `docker.io/myorg/backend` and `ghcr.io/otherorg/backend` both group under `backend` for a given customer. Grouping key is scoped **per customer** (`customerId + imageName`), never across customers.
- **ImageGroup**: the durable entity keyed by `(customerId, imageName)` that holds the category assignment and rollup fields.
- **ImageRef**: one concrete, scannable artifact (a specific repo+tag+digest+buildDate) belonging to exactly one `ImageGroup`.

## 6. Functional Requirements → Design Mapping

| # | Ask.txt requirement | Design |
|---|---|---|
| 1 | Multi-customer, single-customer UI | §4 — `associate_ids` row-level scoping |
| 2 | Manage vulnerabilities/remediate images per customer | §9 (services), §11 (UI) |
| 3 | Use Trivy | §13 (Trivy Scanning Pipeline) |
| 4 | Group by image name regardless of repo/tag | §5 (grouping algorithm), §6.1 (ImageRef ingestion) |
| 5 | Group detail: image refs sorted descending by build date | §11.3 |
| 6 | Each image selectable, multi-select, scan via Trivy | §11.3, §13 |
| 7 | Per-image sum-up + distinct counts by severity | §7 data model (`VulnerabilityCounts`), §13 |
| 8 | Detailed vulnerability/CVE list sorted Critical→Low | §11.4 |
| 9 | Category per image group, managed from the app | §9 (`ImageCategory` service), §11.2 |
| 10 | CSV report with specified columns | §10 |
| 11 | Bulk "Add Images" action, free-text paste of a list of image refs | §6.1, §11.5 |

### 6.1 Image Reference Ingestion

Ingestion is UI-driven and bulk-first (confirmed): an **"Add Images"** action opens a form with a single free-text textarea where the user pastes a list of image references, **one per line**. There is no manual build-date field — build dates are never typed by a human; they are resolved automatically from the registry (see the two-phase flow below), since the scanner pod already carries full registry credentials (confirmed — able to pull/inspect every referenced image).

**New action-service `ImgRefAdd`** (`ServiceArea 60`, POST-only, not a plain-CRUD entity service — same "custom action beyond CRUD" precedent as `L8ImportTemplate`'s `/ImprtExec` in `DataImportSystem`):

```
POST /60/ImgRefAdd
{ "customerId": "...", "imageRefStrings": ["repoA/backend:v1.2.3", "repoB/worker:v0.9", ...] }

-> { "created": ["<imageRefId>", ...],
     "skipped":  [{ "ref": "...", "reason": "duplicate of existing image ref" }],
     "errors":   [{ "ref": "...", "reason": "could not parse image reference" }] }
```

`customerId` is supplied by the client — the UI sets it from the logged-in user's own session context, never from user input. The server does not independently validate it against the caller's identity; see §4 for why (confirmed framework limitation) and why that's an accepted v1 trust boundary, not a gap this endpoint tries to close.

**Phase A — synchronous (in the `ImgRefAdd` handler, shared with the plain `ImageRef` POST path via one common Go helper — `Duplication Prevention` "Second Instance Rule", extract-on-second-use):**

1. Trim/split the pasted text into non-blank lines.
2. Per line: parse into `repoName` / `tag` / `digest` (per §5); derive `imageName` (grouping key).
3. **Dedupe**: skip (report in `skipped`) if an `ImageRef` already exists for `(customerId, repoName, tag, digest)` — re-pasting the same build is a no-op, not a new row. A tag re-pointed to a new digest (e.g. `latest`) is a genuinely new build and is inserted.
4. Look up `ImageGroup` for `(customerId, imageName)`; create if absent (`categoryId` empty = "Uncategorized").
5. Insert the `ImageRef` (`common.GenerateID`), `imageGroupId` set, `scanStatus = SCAN_STATUS_PENDING`, `buildDate = 0` (sentinel for "not yet resolved" — **not** the date-picker's `0 = Current` convention; this field is never bound to `Layer8DDatePicker`, so there is no collision, but the UI must render `0` here as "Resolving…", not as a date, per §11.3).

This keeps every hook fast and non-blocking (`MainPackageMinimal` — no network I/O inside a synchronous CRUD path); rows appear in the group detail list immediately with a "Resolving…" build date.

**Phase B — asynchronous (`secscan-scanner`'s "resolver" loop — one configuration of the shared poll-claim-dispatch harness described in §13, alongside the Trivy scan loop):**

1. The harness polls `select * from ImageRef where buildDate=0` via `vnic`; for each match, the `work` function queries the registry (credentials already present on the pod) for the image's manifest/config `Created` timestamp — recommend `go-containerregistry` (`crane`)'s pure-Go client so the scanner pod needs no Docker daemon — then `vnic.Put`s the resolved `buildDate` back to the `secscan` backend.
2. The `ImageRefServiceCallback.After()` hook (PUT, `buildDate` transitions from `0` to a real value) recomputes the parent `ImageGroup.latestBuildDate`/`imageRefCount` cache — kept centralized in the callback rather than duplicated in the scanner, per `Duplication Prevention`.

If the registry lookup fails (bad ref, no access, image deleted), the resolver sets a `scanError` and leaves `buildDate=0`; the UI surfaces this the same way a failed scan is surfaced (§11.3), so a user isn't left wondering why a row is stuck "Resolving…".

## 7. Data Model (Protobuf)

Module: `secscan`. File: `proto/secscan.proto`. Follows `ProtobufRules` (enum zero = `..._UNSPECIFIED`, list wrapper `repeated X list = 1` + `l8api.L8MetaData metadata = 2`, no direct struct references between Prime Objects — only string ID fields).

```protobuf
syntax = "proto3";
package secscan;
option go_package = "github.com/saichler/l8secure-scan/go/types/secscan";

import "l8api.proto";

enum ScanStatus {
  SCAN_STATUS_UNSPECIFIED = 0;
  SCAN_STATUS_PENDING     = 1;
  SCAN_STATUS_SCANNING    = 2;
  SCAN_STATUS_COMPLETED   = 3;
  SCAN_STATUS_FAILED      = 4;
}

enum Severity {
  SEVERITY_UNSPECIFIED = 0;
  SEVERITY_LOW         = 1;
  SEVERITY_MEDIUM      = 2;
  SEVERITY_HIGH        = 3;
  SEVERITY_CRITICAL    = 4;
}

enum JobStatus {
  JOB_STATUS_UNSPECIFIED = 0;
  JOB_STATUS_QUEUED      = 1;
  JOB_STATUS_RUNNING     = 2;
  JOB_STATUS_COMPLETED   = 3;
  JOB_STATUS_FAILED      = 4;
  JOB_STATUS_PARTIAL     = 5;   // some selected images failed to scan
}

message VulnerabilityCounts {
  int32 critical = 1;
  int32 high     = 2;
  int32 medium   = 3;
  int32 low      = 4;
}

message Customer {
  string customer_id     = 1;
  string name             = 2;
  bool   is_active        = 3;
  l8api.AuditInfo audit_info = 4;
}
message CustomerList {
  repeated Customer list  = 1;
  l8api.L8MetaData metadata = 2;
}

message ImageCategory {
  string category_id     = 1;
  string customer_id     = 2;
  string name             = 3;
  string color_code       = 4;
  l8api.AuditInfo audit_info = 5;
}
message ImageCategoryList {
  repeated ImageCategory list = 1;
  l8api.L8MetaData metadata  = 2;
}

message ImageGroup {
  string image_group_id  = 1;
  string customer_id     = 2;
  string image_name      = 3;   // grouping key
  string category_id     = 4;   // ref ImageCategory by ID, empty = Uncategorized
  int32  image_ref_count = 5;   // cached
  int64  latest_build_date = 6; // cached, for default sort
  VulnerabilityCounts newest_counts = 7; // cached: total_counts of the newest scanned ImageRef in this group
  VulnerabilityCounts oldest_counts = 8; // cached: total_counts of the oldest scanned ImageRef in this group
  l8api.AuditInfo audit_info = 9;
}
message ImageGroupList {
  repeated ImageGroup list = 1;
  l8api.L8MetaData metadata = 2;
}

message ImageRef {
  string image_ref_id       = 1;
  string customer_id        = 2;
  string image_group_id     = 3;  // ref ImageGroup by ID
  string repo_name          = 4;
  string tag                = 5;
  string digest              = 6;
  int64  build_date          = 7;
  ScanStatus scan_status      = 8;
  int64  last_scanned_at      = 9;
  VulnerabilityCounts total_counts    = 10; // sum-up
  VulnerabilityCounts distinct_counts = 11; // distinct CVEs
  string scan_error           = 12;
  l8api.AuditInfo audit_info  = 13;
}
message ImageRefList {
  repeated ImageRef list    = 1;
  l8api.L8MetaData metadata = 2;
}

// Cve is a global catalog entity — not customer-scoped, no deny rule.
// Normalizes each CVE's canonical data once instead of repeating it on
// every finding; also the Prime Object a future cross-image "which
// images have CVE-X" query would target.
message Cve {
  string cve_id           = 1;  // natural key, e.g. "CVE-2023-1234" — no generated ID needed
  Severity severity        = 2;
  string title             = 3;
  l8api.AuditInfo audit_info = 4;
}
message CveList {
  repeated Cve list         = 1;
  l8api.L8MetaData metadata = 2;
}

// ImageRefCve is one Trivy finding: "package P in ImageRef R was found
// vulnerable to Cve C." A Prime Object (§8) — despite its two parent
// FKs — specifically so a single image's finding list is a normal,
// server-side-filtered/sorted/paginated root-type query, never an
// embedded repeated field.
message ImageRefCve {
  string image_ref_cve_id   = 1;
  string customer_id        = 2;  // denormalized from the parent ImageRef, for row scoping
  string image_ref_id       = 3;  // ref ImageRef by ID
  string cve_id             = 4;  // ref Cve by ID
  Severity severity         = 5;  // denormalized from Cve at write time — this ORM has no join, so sort/filter needs it local
  string package_name       = 6;
  string installed_version  = 7;
  string fixed_version      = 8;
  string title              = 9;  // denormalized from Cve — the finding list needs no lookup to render
  l8api.AuditInfo audit_info = 10;
}
message ImageRefCveList {
  repeated ImageRefCve list = 1;
  l8api.L8MetaData metadata = 2;
}

message ScanJob {
  string scan_job_id       = 1;
  string customer_id       = 2;
  repeated string image_ref_ids = 3;
  JobStatus status         = 4;
  int32  total_images      = 5;
  int32  completed_images  = 6;
  int32  failed_images     = 7;
  int64  requested_at      = 8;
  int64  completed_at      = 9;
  string requested_by      = 10;
  l8api.AuditInfo audit_info = 11;
}
message ScanJobList {
  repeated ScanJob list      = 1;
  l8api.L8MetaData metadata = 2;
}
```

Generation: `cd proto && ./make-bindings.sh` (never hand-edit `.pb.go`), per `ProtobufRules`.

`ImageGroup.newest_counts`/`oldest_counts` extend the same denormalized-cache pattern already used for `image_ref_count`/`latest_build_date` — they exist specifically to give every consumer that needs "the newest/oldest scanned image ref's counts" (the CSV report, the dashboard table, the Group Detail Trend panel — §10/§11.1/§11.3) **one** already-computed, already-fetched source to read, instead of each of those three independently re-deriving "find the newest/oldest scanned `ImageRef` in this group" from scratch. See §9 for the single hook that maintains them.

## 8. Prime Object Classification

| Type | Classification | Reasoning |
|---|---|---|
| `Customer` | Prime Object | Independent identity, own lifecycle, queried directly (opsadmin catalog). |
| `ImageCategory` | Prime Object | Independent CRUD lifecycle, managed directly from the app (ask requirement 9), referenced by ID from `ImageGroup`. |
| `ImageGroup` | Prime Object | Independent identity (`customerId`+`imageName`), own lifecycle (category reassignment, rollup cache updates) independent of any single `ImageRef`, queried/listed directly as the dashboard's primary table. |
| `ImageRef` | **Prime Object** (flagged deviation — see below) | Referenced by ID from `ImageGroup` via `image_group_id`. |
| `Cve` | Prime Object | Global catalog entity — independent of any image, own lifecycle (catalog entries can be enriched independent of any scan), directly queryable, no parent at all. Not customer-scoped. |
| `ImageRefCve` | **Prime Object** (flagged deviation, same reasoning as `ImageRef` — see below) | References both `ImageRef` and `Cve` by ID; verified — not just argued — necessary (see below). |
| `ScanJob` | Prime Object | Independent identity/lifecycle (`QUEUED→RUNNING→COMPLETED/FAILED`), directly queryable (scan history), audit trail of who requested what. |

**Why `ImageRef` is a Prime Object despite carrying a required `image_group_id`.** The guide's heuristic for "not a Prime Object" ("has a required `parent_id` field", e.g. order lines) assumes the child's only mutation path is through editing the parent record. `ImageRef` fails that assumption on all four independence tests:

1. **Independence** — an `ImageRef` is individually addressable by registry digest; it has meaning without the UI ever opening its group.
2. **Own lifecycle** — its `scanStatus`/`totalCounts`/`distinctCounts` are mutated **asynchronously by the scanner backend**, completely outside of any edit to `ImageGroup`. A plain embedded child only changes when its parent form is saved; this one changes on its own.
3. **Direct query need** — the UI needs server-side sortable/paginated/filterable queries ("sorted descending by build date", "select multiple for scan", eventually "all pending scans for this customer") — exactly what `Layer8DTable` + `baseWhereClause=imageGroupId=X` gives for free, and what an embedded `repeated` field (fetched only as part of the whole parent payload, unbounded, unpaginated) does not.
4. **No parent-derived identity** — its identity is the repo/tag/digest tuple, not a position within the parent's list.

Given the async, backend-driven lifecycle and the mandatory paginated/sortable/multi-select table UI, treating `ImageRef` as an embedded `repeated ImageGroup.image_refs` field would force fetching an unbounded list on every group view and would not support `Layer8DTablePaginationMetadata`/sorting. It is therefore modeled as a Prime Object with `image_group_id` as an ID-only reference (never a struct ref, per `PrimeObjectReferences` Rule 2), gets its own service/columns/forms/reference-registry entry, and its own mock data ID slice.

**Why `ImageRefCve` is a Prime Object too — this one verified against the ORM's actual read path, not just argued by analogy.** An earlier draft of this PRD modeled each Trivy finding as an embedded `repeated Vulnerability` field of `ImageRef`. Reading `l8orm`'s actual query-to-SQL translation (`l8orm/go/orm/stmt/QueryToSql.go`) showed that a nested/child table only gets the query's `WHERE`/`ORDER BY`/`LIMIT` applied **when that table is the query's root type** (`Query2Sql`: `if typeName == query.RootType().TypeName`) — a child table is otherwise read via a bare, unfiltered `SELECT` of the *entire* table, with parent-key filtering happening after the fact in Go. So `select * from ImageRef where imageRefId=X` would have pulled **every finding for every image, for every customer** into memory on every single request just to keep the ~50–500 rows for one image. Making `ImageRefCve` a Prime Object turns "this image's findings, sorted Critical→Low" into `select * from ImageRefCve where imageRefId=X sort-by severity desc` — `ImageRefCve` **is** the query's root type there, so the filter and sort are applied in the actual SQL, not after an unbounded read. `Cve` and `ImageRefCve` are referenced only by ID (`cve_id`/`image_ref_id`), never by struct, per `PrimeObjectReferences` Rule 2.

**Project-wide principle this establishes:** any child collection whose volume can grow large (parents × average children) must be modeled as a Prime Object with an ID-only parent reference — never as an embedded `repeated`-struct field, regardless of how naturally it reads as "belongs to one parent." A *singular* (non-repeated) embedded struct like `VulnerabilityCounts` (used 4× in this schema) doesn't need this treatment — same underlying child-table mechanism, but bounded to ~1-2 rows per parent, not hundreds.

## 9. Service Architecture

Module `secscan`, single `ServiceArea = 60` for all `secscan`-owned services (`Maintainability`: "ServiceArea same for all services in a module").

| Prime Object | ServiceName (≤10 chars) | ServiceArea | Owner process |
|---|---|---|---|
| `Customer` | `Customer` | 60 | `secscan` backend |
| `ImageCategory` | `ImgCat` | 60 | `secscan` backend |
| `ImageGroup` | `ImgGroup` | 60 | `secscan` backend |
| `ImageRef` | `ImageRef` | 60 | `secscan` backend |
| `Cve` | `Cve` | 60 | `secscan` backend (global catalog, not customer-scoped) |
| `ImageRefCve` | `ImgRefCve` | 60 | `secscan` backend |
| `ScanJob` | `ScanJob` | 60 | `secscan` backend |
| CSV report generator | `VulnRep` | 60 | `secscan` backend (POST-only action service, see §10) |
| Bulk image ingestion | `ImgRefAdd` | 60 | `secscan` backend (POST-only action service, see §6.1) |

Per `SingleOwnerDatabaseTable`, the ORM for the seven Prime-Object-backed services (`Customer`, `ImageCategory`, `ImageGroup`, `ImageRef`, `Cve`, `ImageRefCve`, `ScanJob`) is activated in exactly one process (`secscan` backend/`main`). `VulnRep` and `ImgRefAdd` are stateless action handlers hosted in that same process — they have no table/ORM of their own, they read and write the already-owned tables directly, so no second-owner question arises for them. The `secscan-scanner` worker never activates a local ORM for any of these — it reaches them exclusively through `vnic` RPC (`vnic.Get/Post/Put`).

**`customer_id` is a trusted, client-supplied value everywhere (§4) — not derived or validated server-side.** This was verified against the actual framework source, not assumed: `IServiceCallback.Before()`/`.After()` (`l8types/go/ifs/ServiceLevelAgreement.go`) have no access to the caller's identity, so no `ServiceCallback` in this list can independently confirm "does this `customer_id` belong to whoever is calling." Every `Before()` hook that sets `customer_id` on POST simply takes it from the request body (as the UI populated it from the logged-in user's session, §11) or copies it from an already-trusted parent record — it does not attempt to re-derive or cross-check it against caller identity, because no hook here has the information to do that.

`ServiceCallback` responsibilities (`Before`/`After` hooks only, per `MainPackageMinimal` / `FrameworkInterfaceBoundaries`):

- `CustomerServiceCallback`: `common.GenerateID` on POST. (`Customer` rows are created only by `opsadmin`, which is not customer-scoped.)
- `ImageCategoryServiceCallback`: `Before()` on POST — takes `customer_id` from the request body, `common.GenerateID`.
- `ImageGroupServiceCallback`: `common.GenerateID` on POST (used internally by the `ImageRef` ingestion flow, not directly by end users — there is no user-facing "Add Group" form; groups only ever emerge from ingestion, category reassignment happens via `PUT`). `customer_id` is copied from the triggering `ImageRef`.
- `ImageRefServiceCallback`: `Before()` on POST calls the shared ingestion helper (takes `customer_id` from the request, parse ref, dedupe, find-or-create group, `common.GenerateID`, set `PENDING`, §6.1 Phase A) — used both by a direct single-entity POST and, internally, by the `ImgRefAdd` bulk handler. `After()` on PUT recomputes the parent `ImageGroup` rollup cache (`imageRefCount`, `latestBuildDate`, and — on scan completion — `newestCounts`/`oldestCounts`, §7) whenever `buildDate` resolves from `0` to a real value (§6.1 Phase B) or a scan completes (§13).
- `CveServiceCallback`: no `common.GenerateID` needed — `cve_id` (e.g. `"CVE-2023-1234"`) is a natural key supplied by the scanner's find-or-create logic (§13.1), not generated. Not customer-scoped (§4's row-scoping deny rules don't apply to this service — see §14).
- `ImageRefCveServiceCallback`: `Before()` on POST — `common.GenerateID`; `customer_id` is copied from the `ImageRef` being processed (not a client-trusted value in the ordinary sense — this row is written only by `secscan-scanner`'s own system identity, §14, never by an end-user session).
- `ScanJobServiceCallback`: `Before()` on POST — takes `customer_id` from the request body, validates all `image_ref_ids` actually belong to that `customer_id` (a data-integrity sanity check against accidental cross-group mixing in the request, not a security boundary — it only checks internal consistency of the request's own stated `customer_id`, which itself isn't independently verified, §4), `common.GenerateID`, set `status = QUEUED`, `totalImages = len(imageRefIds)`, `requestedAt = now`. No scanning happens inline in the callback (`MainPackageMinimal` — business/long-running work does not belong in a synchronous CRUD hook); the `secscan-scanner` worker polls for queued jobs (§13).

Types registered in `go/secscan/ui/main.go` via `introspect.AddPrimaryKeyDecorator` + `registry.Register`, per `Maintainability`.

`EventsServiceRequired` and `LogServicesRequired` apply: `l8events` is activated in `secscan` backend `main.go` (`evtservices.ActivateEvents(dbcred, dbname, nic)`); `go/secscan/ui/main.go` registers `l8events.EventRecord`/`EventRecordList` (`l8c.RegisterType(resources, &l8events.EventRecord{}, &l8events.EventRecordList{}, "EventId")`) so the built-in Events view renders. `log-vnet`/`log-agent` are separate binaries copied/adapted from `probler`.

## 10. CSV Report

Ask requirement 10 is a **cross-group summary report** (one row per `ImageGroup`), not a per-entity dump. The generic, built-in `Layer8CsvExport` (`Layer8CsvExport`) exports one row per raw entity with its own columns and cannot compute cross-row aggregates (reduction %) or a formatted nested list column — so it is **not reused** for this report; a dedicated action-service is added instead, following the same "custom action beyond plain CRUD" precedent as `L8ImportTemplate`'s `/ImprtExec`/`/ImprtXfer` endpoints (`DataImportSystem`).

- Endpoint: `POST /scan/60/VulnRep` — no request body needed beyond auth (report is generated for the caller's scoped customer via the standard row-level deny rule); returns `text/csv`.
- Server-side generation (Go), one row per `ImageGroup` visible to the caller:

| Column | Source |
|---|---|
| `Name` | `ImageGroup.image_name` |
| `Category` | `ImageCategory.name` for `ImageGroup.category_id` (blank → `Uncategorized`) |
| `Newest Critical` / `Newest High` / `Newest Medium` / `Newest Low` | `ImageGroup.newest_counts` (cached, §7) — 4 columns |
| `Oldest Critical` / `Oldest High` / `Oldest Medium` / `Oldest Low` | `ImageGroup.oldest_counts` (cached, §7) — 4 columns |
| `Reduction % Critical` / `Reduction % High` / `Reduction % Medium` / `Reduction % Low` | Per severity, from the two cached count structs above: `(oldest.sev − newest.sev) / oldest.sev × 100` — 4 columns |
| `Image Refs` | All image refs in the group, **descending by build date**, one cell, semicolon-separated: `repoName:tag (buildDate ISO-8601)` |

15 columns total. Note `VulnRep` never re-derives "which `ImageRef` is newest/oldest" itself — that lookup happens exactly once, in the `ImageRefServiceCallback.After()` hook that maintains `ImageGroup.newest_counts`/`oldest_counts` (§9); the report just reads those two cached structs per group.

A cell is blank (not `0`) when the newest/oldest image ref hasn't completed a scan yet — counts and reduction % are only meaningful once `scanStatus = COMPLETED`. **Canonical `N/A` rule set for `Reduction %`** (the one spec every consumer of this figure — this report, the dashboard's Trend indicator §11.1, the Group Detail Trend panel §11.3 — must apply identically, since each renders it independently from the same two cached structs): `N/A` when the group has fewer than 2 scanned image refs, when the newest and oldest resolve to the same image ref (single data point), or when `oldest.sev = 0` for that severity (division by zero — a 0→N vulnerability increase isn't expressible as a "reduction" percentage; it is reported as `N/A`, not a negative/undefined number).

UI entry point: an **"Export CSV Report"** button on the main Image Groups view, calling `VulnRep` directly. Note this is *in addition to*, not instead of, the generic per-row export button `Layer8CsvExport` auto-attaches to the Image Groups table's pagination bar (raw `ImageGroup` rows) — the two are visually distinct ("Export CSV Report" vs. the default "Export") so a user can't confuse the aggregated report with a plain table dump (§12.2).

## 11. UI / UX

Namespace `SecScan`, per `ArchitectureOverview` / `AddingModule`. Two modules: `vulnmgmt` (every role) and `admin` (opsadmin only, §11.6):

```js
Layer8ModuleConfigFactory.create({
    namespace: 'SecScan',
    modules: {
        'vulnmgmt': mod('Vulnerability Management', 'icon-shield', [
            svc('groups', 'Image Groups', 'icon-image', '/60/ImgGroup', 'ImageGroup'),
            svc('categories', 'Categories', 'icon-tag', '/60/ImgCat', 'ImageCategory'),
            svc('scanjobs', 'Scan History', 'icon-history', '/60/ScanJob', 'ScanJob')
        ]),
        'admin': mod('Administration', 'icon-settings', [
            svc('customers', 'Customers', 'icon-building', '/60/Customer', 'Customer')
        ])
    },
    submodules: ['SecScanVuln', 'SecScanAdmin']
});
```

### 11.1 Dashboard / Image Groups (default view)

- KPI strip (`Layer8DWidget.renderEnhancedStatsGrid`): total image groups, total pending scans, total critical CVEs (customer-wide), groups with no scan yet.
- `Layer8DTable` over `ImageGroup`, columns: Name, Category (enum-style tag via `col.custom`), Image Ref Count, Newest Build Date, Newest Critical/High/Medium/Low (`col.custom` reading `ImageGroup.newest_counts` directly off the already-fetched row — no extra query, no client-side re-derivation). **Design call:** the main list table shows only the *newest* image's counts, not the full newest+oldest+per-severity-reduction breakdown from the CSV (§10) — 15 columns in a row-scanning list table would hurt usability. The full breakdown is one click away (Group Detail's Trend panel, §11.3) and in the CSV export; the list table adds a compact "Trend" indicator column whose per-severity percentages are computed client-side from the row's own `newest_counts`/`oldest_counts` fields, applying the canonical `N/A` rule set from §10 (e.g. ▼12% / ▲/flat, tooltip shows the four per-severity percentages). Sortable/filterable, server-side.
- Toolbar: **Add Images** button (bulk paste, §11.5) and **Export CSV Report** button (posts to `VulnRep`, downloads response).
- Row click → opens **Image Group Detail** (large `Layer8DPopup`).
- A small bar chart (`Layer8DChart`, `viewConfig.chartType:'bar'`, `categoryField:'imageName'`, using latest counts) gives an at-a-glance severity comparison across groups — the only `Layer8DViewFactory` chart type used in this PRD; other view types (kanban/calendar/gantt/tree/wizard) are not applicable (see §12).

### 11.2 Categories

- Plain CRUD `Layer8DTable` + form over `ImageCategory`: Name, Color. `f.text('name','Name',true)` + a color field.
- Reference registry entry so `ImageGroup`'s edit form can pick a category via `Layer8DReferencePicker`.

### 11.3 Image Group Detail

Opened via `Layer8DPopup.show({size:'xlarge', ...})`:

- Header: group name, editable Category reference field (saves via `PUT /60/ImgGroup`), and a **Trend panel**: Newest vs. Oldest scanned image's Critical/High/Medium/Low counts side by side, plus the per-severity reduction %. This panel reads `newest_counts`/`oldest_counts` straight off the same `ImageGroup` record already fetched to render this popup's header — no separate query, no separate computation of "which image ref is newest/oldest" (that's centralized once, §7/§9); only the trivial reduction-% arithmetic is (re-)done here, applying the same canonical `N/A` rule set as §10.
- `Layer8DTable` over `ImageRef`, `baseWhereClause: "imageGroupId='<id>'"`, default `sort-by buildDate desc` (requirement 5). Columns: checkbox (multi-select — table's native row-selection, not a `showActions` CRUD column), Repo, Tag, Build Date (renders "Resolving…" while `buildDate=0` and no `scanError`, or an error indicator if resolution failed — §6.1 Phase B), Scan Status (`createStatusRenderer`), Total C/H/M/L, Distinct C/H/M/L.
- Toolbar button **"Scan Selected"**: enabled when ≥1 row selected; `POST /60/ScanJob` with `{customerId, imageRefIds: [...selected]}` (`customerId` set by the UI from the logged-in session, never user-entered, §4). On success, `Layer8DNotification.success('Scan job queued')`; selected rows' status flips to `SCANNING` once the scanner claims the job (poll/refresh, or WebSocket notification push per `l8web`'s notification channel — reuse, don't build a new transport).
- Row click on an `ImageRef` → **Vulnerability Detail** popup (§11.4).

### 11.4 Vulnerability Detail (per Image Ref)

- Summary strip: `ImageRef.total_counts`/`distinct_counts` per severity (still embedded scalars on `ImageRef` — small, bounded, no §8 concern).
- Findings list: a standard `Layer8DTable` over `ImageRefCve`, `baseWhereClause: "imageRefId='<id>'"`, default `sort-by severity desc` — enum values already rank `CRITICAL=4 → LOW=1`, so a numeric-desc sort produces "Critical→Low" for free (requirement 8), and because `ImageRefCve` is the query's own root type, this sort/filter is applied in the actual SQL (§8), not after an unbounded read. Columns: CVE ID, Package, Installed Version, Fixed Version, Severity tag, Title — all denormalized directly onto `ImageRefCve` (§7), so no lookup against the `Cve` catalog is needed to render this view.
- This is now a normal, paginated table fetch like every other list in this app (not a bespoke local-list render) — one less one-off UI pattern to maintain.

### 11.5 Add Images (bulk ingestion)

- Toolbar action on the Image Groups view, opens a `Layer8DPopup` with a single `f.textarea('imageRefStrings', 'Image References (one per line)', true)` field — no build-date input (resolved automatically, §6.1 Phase B).
- On save: client splits the textarea into lines, `POST /60/ImgRefAdd` with `{customerId, imageRefStrings}` (`customerId` set by the UI from the logged-in session, §4/§9).
- Response summary rendered in a follow-up notification/popup: N created, N skipped (duplicates, with the offending ref shown), N errors (unparseable ref, with the offending line shown) — per `ReportInfraBugs` "No Silent Fallbacks", every skipped/errored line is shown to the user, never silently dropped.
- Newly created rows appear immediately in the relevant group(s)' detail view with `scanStatus=PENDING` and Build Date "Resolving…".

### 11.6 Customer Management (opsadmin only)

Committed v1 scope (§4). The `admin` module (config in §11 above) provides plain CRUD over `Customer` (Name, Active). This is the only place `Customer` rows are created; `customer`-role users never see or edit `Customer` records, only their own scoped data.

**Nav visibility is project-specific logic, not `Layer8DModuleFilter`.** `Layer8DModuleFilter` (`await Layer8DModuleFilter.load(bearerToken)` / `.applyToSidebar()`) is the **global, ModConfig-backed** module enable/disable mechanism (fetches `/0/ModConfig`) — this project has no `ModConfig` service, so per `ModconfigFailureNoLogout` that block is deliberately **not** wired into `app.js` (its 404 would otherwise trigger an internal `logout()` and an infinite redirect loop). Hiding the `admin` module from `customer`-role users is therefore ordinary project-specific nav logic in `secscan`'s own `sections.js`/init code (per `L8UINoProjectSpecificCode` — project-specific conditionals belong in the project, not in `l8ui`): after login, the app reads the current user's role from the auth response and conditionally omits the `admin` entry from the rendered sidebar. This is a client-side UX convenience only — the actual security boundary is the server-side deny-before-allow permission check (§14): a `customer`-role user's request against `Customer` is rejected by `ISecurityProvider` regardless of what the nav shows, since the `customer` role's rules grant no `Customer` permissions at all.

### 11.7 Mobile

Full parity per `MobileRules`: `Layer8MModuleRegistry.create('MobileSecScan', {...})`, `Layer8MEditTable` for Image Groups/Categories/Scan History/Customers (opsadmin), `Layer8MTable` (read-only, `getItemId`) for the Image Ref list inside a group drill-down (`Layer8MNav.navigateToService`), `Layer8MPopup` for the Vulnerability Detail view and the Add Images bulk-paste form, `Layer8MChart` for the severity bar chart. CSV export reuses the same `VulnRep` endpoint via `Layer8MAuth.get`.

## 12. L8UI Includes Audit

Per `PrdL8uiIncludesAudit`, every file in `../l8erp/rules/desktop-script-loading-order.md` and `.../mobile-script-loading-order.md` is accounted for below. (This PRD is single-module/no multi-module ERP concerns — `l8erp`-specific reference registries like `reference-registry-fin.js` are replaced by `secscan`'s own.)

### 12.1 Desktop CSS

| File | Included | Reason |
|---|---|---|
| `layer8d-theme.css` | Yes | Required base theme |
| `layer8d-animations.css` | Yes | Base |
| `layer8d-scrollbar.css` | Yes | Base |
| `layer8-section-layout.css` | Yes | Base |
| `layer8-section-responsive.css` | Yes | Base |
| `layer8d-table.css` | Yes | Image Groups / Categories / Scan History / Image Ref tables |
| `layer8d-toggle-tree.css` | Yes | Required by module filter infra |
| `layer8d-popup.css` | Yes | Group/Vulnerability detail popups |
| `layer8d-popup-forms.css` | Yes | Category/Image forms |
| `layer8d-popup-content.css` | Yes | Detail popups |
| `layer8d-datepicker.css` | N/A | No user-editable date field exists in this app: `buildDate` is auto-resolved from the registry and always read-only (§6.1), never bound to `Layer8DDatePicker`; audit-info dates render via plain `formatDate`, not the interactive picker |
| `layer8d-reference-picker.css` | Yes | Category picker on Image Group |
| `layer8d-input-formatter.css` | Yes | Color field, generic formatters |
| `layer8d-notification.css` | Yes | Scan queued / errors |
| `layer8-view-switcher.css` | Yes | Table/Chart switch on dashboard |
| `layer8d-chart.css` | Yes | Severity bar chart (§11.1) |
| `layer8d-kanban.css` | N/A | No kanban/board view in this domain |
| `layer8d-timeline.css` | N/A | No timeline view |
| `layer8d-calendar.css` | N/A | No calendar view |
| `layer8d-gantt.css` | N/A | No scheduling/gantt view |
| `layer8d-tree-grid.css` | N/A | No hierarchical tree data |
| `layer8d-wizard.css` | N/A | No multi-step wizard flow |
| `layer8d-widget.css` | Yes | KPI strip |
| `layer8-markdown.css` | N/A | No markdown content anywhere in this app |
| `layer8-file-upload.css` | N/A | No file uploads (image refs are strings, not uploaded files, per `FileUploadPattern`) |
| `l8sys.css` | Yes | Built-in System module (always included) |
| `l8health.css` | Yes | Built-in |
| `l8sys-modules.css` | Yes | Built-in |
| `l8logs.css` | Yes | Built-in (`L8Logs`) |
| `l8dataimport.css` | N/A | No CSV/JSON/XML import requirement (only export) |
| `l8agent-chat.css` | N/A | No AI chat requested in this PRD |

### 12.2 Desktop JS

| Category | Included | Reason |
|---|---|---|
| App shell (`sections.js`, `app.js`) | Yes | Required |
| Core (`layer8d-config/utils/renderers/reference-registry`, `websocket`, `theme-switcher`) | Yes | Required |
| `portal-switcher.js` | N/A | Single portal for this app (`PortalsSameWebServer` — no admin/member/ess split requested); present in `app.html` inert-safe but not wired to multiple portals |
| Factories (`enum/ref/column/form/form-presets/svg/module-config`) | Yes | Required for all module data files |
| `layer8-section-generator.js` | Yes | Required |
| Project reference registries (`reference-registry-secscan.js`) | Yes | Registers `ImageCategory` — the only Prime Object picked via a `Layer8DReferencePicker` in this app (`ImageGroup`'s category field, §11.2). `ImageGroup`/`ImageRef` are never `lookupModel` targets, so they need no registry entry (`ReferenceRegistryCompleteness`) |
| Notification | Yes | Required |
| Input formatters (6 files) | Yes | `colorCode` field type on `ImageCategory`, `layer8d-input-formatter.js` attach |
| `layer8-markdown.js` | N/A | See CSS row |
| `layer8-file-upload.js` | N/A | See CSS row |
| `layer8-csv-export.js` | Yes | Per `Layer8CsvExport`, its export button **auto-attaches** to the pagination bar of every `Layer8DTable` with `endpoint`+`modelName` set — so it appears by default on Image Groups, Categories, Scan History, and the Image Ref list (generic, per-row raw-entity export). The bespoke aggregated report (§10) is a **separate, distinctly-labelled** "Export CSV Report" toolbar button calling `VulnRep` directly (not `Layer8CsvExport.export()`), so the two do not collide in the UI |
| Forms (fields/fields-ext/data/pickers/modal/facade) | Yes | Category, Customer, and Add-Images forms |
| Popup | Yes | Detail popups |
| Date picker (4 files) | N/A | See CSS row — no editable date field in this app |
| Reference picker (6 files) | Yes | Category picker |
| Table (6 files) | Yes | All list views |
| View system: view-factory, view-switcher, data-source | Yes | Dashboard table + chart |
| Chart (3 files) | Yes | Severity bar chart |
| Kanban (2), Timeline, Calendar (2), Gantt (2), Tree-grid, Wizard (2) | N/A | No matching view types in this domain |
| Widget | Yes | KPI strip |
| Module abstractions (service-registry, module-crud, module-navigation, toggle-tree, module-filter, module-factory-core, module-factory) | Yes (script loaded per canonical order) | Required, **except** `Layer8DModuleFilter.load()`/`.applyToSidebar()` is deliberately never *called* from `app.js` — this project has no `ModConfig` service, and invoking it would hit `ModconfigFailureNoLogout`'s 404→logout loop. The `admin`-module visibility gating is separate, project-specific role logic (§11.6), not this component |
| Module data (per submodule: enums/columns/forms/init) | Yes | `secscan` module files |
| SYS module (config, health, security ×3, modules ×3, logs, init) | Yes | Built-in, always included |
| SYS dataimport (4 files) | N/A | No import requirement |
| AI agent (4 files) | N/A | Not requested |

### 12.3 Mobile CSS/JS

Same rationale as desktop, mirrored 1:1: core mobile CSS/JS, popup/confirm/table/edit-table/forms/reference-registry all **Yes**; `layer8m-datepicker.js`/CSS **N/A** (same reasoning as desktop — no editable date field); `layer8m-chart.js` **Yes** (severity chart parity, `MobileRules` desktop/mobile parity); `layer8m-kanban/calendar/timeline/gantt/tree-grid/wizard.js` **N/A** for the same domain reasons as desktop; AI agent mobile files **N/A**. Project-specific: `layer8m-reference-registry-secscan.js` (registers `ImageCategory` only, mirroring desktop), `layer8m-nav-config-secscan.js` (nav core loads before this, per `ScriptLoadingOrder`).

First implementation phase creates `app.html` and `m/app.html` with every "Yes" row above, in the exact order given in `desktop-script-loading-order.md` / `mobile-script-loading-order.md` (`VerifyAppHtmlScriptsAgainstLoadingOrder`).

## 13. Trivy Scanning Pipeline

New binary: `secscan-scanner` (not a UI/backend-DB process — a stateless worker pool, per `l8utils` bounded worker pool with fan-out/fan-in). **Runs as a single replica for v1** — see the concurrency note below for why.

**Shared poll-claim-dispatch harness (one implementation, two configurations — `Duplication Prevention` "Second Instance Rule").** Both loops `secscan-scanner` runs — the metadata **resolver** loop (§6.1 Phase B, fills in `buildDate` for newly-added refs) and the **scan** loop below (Trivy) — are structurally identical: poll a table on an interval for rows matching a status predicate, mark a matched row claimed, dispatch claimed rows to a bounded worker pool, and write the result back over `vnic`. Rather than writing that harness twice, it is one generic internal helper, e.g. `pollworker.Run(query L8Query, claim ClaimFunc, work WorkFunc)`, and each loop is just one configuration of it:

- **Resolver loop**: `query = "select * from ImageRef where buildDate=0"`; `claim` = no-op (a metadata lookup is idempotent, safe to retry, no exclusive claim needed); `work` = the registry lookup in §6.1 Phase B.
- **Scan loop** (§13.1 below): `query = "select * from ScanJob where status='JOB_STATUS_QUEUED'"`; `claim` = the `QUEUED → RUNNING` update; `work` = the Trivy invocation.

They also share one registry client (credentials, §18).

**Concurrency note — verified, corrected from an earlier draft.** `l8services`' "2-phase commit" transaction support (`l8services/go/services/transaction/states/*.go`) is a leader-coordinated **write-replication** protocol for consistency across a service's own replica nodes — it is not a distributed job-claim/lock primitive, and there is no compare-and-swap on `PUT`/`PATCH` (a `PUT` just overwrites). So two `secscan-scanner` replicas both polling the same `QUEUED` job could both claim and both run Trivy on it — a real double-processing bug if this ran as more than one replica. Rather than invent an unverified locking scheme, `secscan-scanner` runs as **exactly one replica** for v1 (§16); its internal worker pool still gives real per-pod concurrency across images within a job. Scaling to multiple replicas would need the framework's actual leader-election primitives (`IServices.GetLeader`/`IsLeader`/`TriggerElections` — real, confirmed to exist) wired up for a non-ORM-owning process, which is unverified and left as explicit future work (§18), not designed here.

### 13.1 Scan loop

Configures the shared harness above with the `ScanJob`/`QUEUED` query and claim transition; per claimed job:

1. For each `imageRefId` in the job (worker pool, bounded concurrency, this is the harness's `work` function): fetch the `ImageRef` (`vnic.Get`), set its `scanStatus = SCANNING` (`vnic.Put`), shell out to `trivy image --format json <repoName>:<tag or digest>`.
2. Parse Trivy's JSON output:
   - `total_counts`: count every finding entry per `Severity` (raw finding count, duplicates across packages included) — computed in Go from the parsed JSON, not via an ORM aggregate query.
   - `distinct_counts`: count unique `VulnerabilityID` (CVE) per `Severity`, same source.
   - Per finding: find-or-create the `Cve` catalog entry (`vnic.Get` by `cveId`; if absent, `vnic.Post` a new `Cve` with `severity`/`title`) — the CVE-catalog equivalent of the ImageGroup find-or-create in §6.1, same "look up by natural key, create if absent" shape, different concrete type (no Go generics, per `NoGoGenerics` — two small concrete functions, not one parameterized one). Then `vnic.Post` an `ImageRefCve` row (`customerId` from this `ImageRef`, `imageRefId`, `cveId`, `severity`/`packageName`/`installedVersion`/`fixedVersion`/`title` denormalized straight from the same finding — no second lookup against the just-written `Cve`).
3. `vnic.Put` the completed `ImageRef` (`scanStatus=COMPLETED`, `total_counts`/`distinct_counts`, `lastScannedAt=now`) — or `scanStatus=FAILED` + `scanError` on Trivy failure.
4. Updates `ScanJob.completedImages`/`failedImages`; when all images are done, sets `status = COMPLETED` (or `PARTIAL` if any failed, or `FAILED` if all failed) and `completedAt`.
5. The `ImageRefServiceCallback.After()` hook (§9) recomputes the parent `ImageGroup`'s cached `latestBuildDate`, and — since this `ImageRef` just transitioned to `COMPLETED` — its `newestCounts`/`oldestCounts` (§7) by comparing this ref's `buildDate` against the group's current newest/oldest scanned refs. This is the **one** place that comparison happens; the dashboard, CSV report, and Trend panel all just read the resulting cached fields (§10, §11.1, §11.3) rather than each re-deriving it.

Per `SingleOwnerDatabaseTable`, `secscan-scanner` never activates a local ORM for `ImageRef`/`ScanJob` — every read/write above is a remote `vnic` call to the `secscan` backend, which is the sole ORM owner.

### 13.2 Trivy vulnerability database cache

Trivy needs its own vulnerability database (`trivy-db`, several hundred MB) refreshed periodically; re-downloading it on every single scan would make each `ScanJob` unnecessarily slow. `secscan-scanner`'s pod mounts a local `emptyDir` (per-replica, not shared, not customer data) as Trivy's DB cache dir, refreshed on pod start and periodically thereafter (`trivy image --download-db-only` on an interval, independent of any single scan). This is purely an operational cache — it carries no durable/customer state, so it is **not** part of the K8s storage-mode matrix in §16 (no `hostPath`/`PVC`/`StorageClass` variance needed across local/bare-metal/GKE/KIND; the same `emptyDir` works identically in all four).

## 14. Security Config

`go/secure/plugin/secscan/secscan.json`, per `SecurityConfigStructure` (canonical reference: `l8secure/go/secure/plugin/*/*.json`):

```json
{
  "credentials": { "postgres": { "aside": "secscan", "yside": "5432", "zside": "<pwd>" } },
  "key": "<AES key>",
  "secret": "<shared secret>",
  "roles": {
    "customer": {
      "rules": {
        "allow-imagegroup": { "elemType": "ImageGroup", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-imageref": { "elemType": "ImageRef", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-category": { "elemType": "ImageCategory", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-scanjob": { "elemType": "ScanJob", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-imgrefcve": { "elemType": "ImageRefCve", "allowed": true, "actions": {"5": true}, "attributes": {"*": "*"} },
        "allow-cve": { "elemType": "Cve", "allowed": true, "actions": {"5": true}, "attributes": {"*": "*"} },
        "allow-imgrefadd": { "elemType": "ImgRefAdd", "allowed": true, "actions": {"1": true}, "attributes": {"*": "*"} },
        "allow-vulnrep": { "elemType": "VulnRep", "allowed": true, "actions": {"1": true}, "attributes": {"*": "*"} },
        "scope-imagegroup": { "elemType": "ImageGroup", "allowed": false, "actions": {}, "attributes": {"ImageGroup": "select * from ImageGroup where customerId not in ${associateIds}"} },
        "scope-imageref": { "elemType": "ImageRef", "allowed": false, "actions": {}, "attributes": {"ImageRef": "select * from ImageRef where customerId not in ${associateIds}"} },
        "scope-category": { "elemType": "ImageCategory", "allowed": false, "actions": {}, "attributes": {"ImageCategory": "select * from ImageCategory where customerId not in ${associateIds}"} },
        "scope-scanjob": { "elemType": "ScanJob", "allowed": false, "actions": {}, "attributes": {"ScanJob": "select * from ScanJob where customerId not in ${associateIds}"} },
        "scope-imgrefcve": { "elemType": "ImageRefCve", "allowed": false, "actions": {}, "attributes": {"ImageRefCve": "select * from ImageRefCve where customerId not in ${associateIds}"} }
      }
    },
    "opsadmin": {
      "rules": {
        "allow-customer": { "elemType": "Customer", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} }
      }
    },
    "scanner": {
      "rules": {
        "allow-imageref-scan": { "elemType": "ImageRef", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-scanjob-scan": { "elemType": "ScanJob", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-cve-scan": { "elemType": "Cve", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} },
        "allow-imgrefcve-scan": { "elemType": "ImageRefCve", "allowed": true, "actions": {"-999": true}, "attributes": {"*": "*"} }
      }
    }
  },
  "users": {
    "acme-user": { "userName": "acme-user", "password": "<hash>", "associateIds": ["CUST-ACME"], "roles": {"customer": true} },
    "scanner-svc": { "userName": "scanner-svc", "password": "<hash>", "roles": {"scanner": true} }
  },
  "sysconfig": { "dataStoreType": 1, "dataStoreName": "secscan", "webPort": 2790 }
}
```

`scanner-svc` is `secscan-scanner`'s own system identity (§9, §13) — not customer-scoped (no `associateIds`), since it must read/write `ImageRef`/`ScanJob`/`Cve`/`ImageRefCve` across every tenant. `ImageRefCve` rows it writes carry a `customer_id` copied from the `ImageRef` being processed, not from this identity's own scope (§9).

`login.json` (`LoginJsonAdaptation`): `apiPrefix` = `/scan` (not `/erp`); `appTitle` = "Layer 8 Secure Scan".

## 15. Mock Data (`go/tests/mocks/`)

Phased per `MockDataRules`:

1. **Customers** — 3 customers.
2. **Categories** — 4–6 per customer (e.g. Production, Staging, Base Images, Deprecated) via Security-API-adjacent `ImgCat` POSTs.
3. **Image Groups** — 15–20 per customer (varied `imageName`s), with a mix of categorized/uncategorized.
4. **Image Refs** — 3–8 per group, varied `repoName`/`tag`/`buildDate` (descending order verifiable), most `PENDING`.
5. **Cve catalog** — a bounded pool of ~40-60 realistic-looking CVE IDs (mixed severities) generated once, shared across all simulated scan results (mirrors how the real scanner's find-or-create would naturally reuse common CVEs across images).
6. **Scan results** — simulate completed scans for ~half of image refs per group: for each, pick a random subset of the `Cve` pool, create matching `ImageRefCve` rows (`customerId` from the `ImageRef`, `packageName`/`installedVersion`/`fixedVersion`/`title` fabricated, `severity` copied from the chosen `Cve`), derive `total_counts`/`distinct_counts` from that subset, set `scanStatus=COMPLETED`. Mock data generation bypasses `secscan-scanner`, so it must also set the parent `ImageGroup`'s `newest_counts`/`oldest_counts` (§7) directly for every group that gets at least one simulated completed scan — otherwise the dashboard/CSV/Trend panel (§10, §11.1, §11.3) would show blank figures for seeded data even though real usage would have them populated by the `After()` hook (§9).
7. **Scan Jobs** — a handful of historical `ScanJob` records (`COMPLETED`/`FAILED`/`PARTIAL`) referencing the scanned image refs, for Scan History view content.
8. **Security users** — one `customer`-role user per customer (`associateIds=[customerId]`), one `opsadmin` user, one `scanner-svc` user (§14), provisioned via Security API (`/73/users`), never a project-owned endpoint (`SecurityRules`).

## 16. Deployment Artifacts

Per `DeploymentArtifacts`, `L8PollarisBinaryDeployment` (pattern, not literal applicability — this isn't a pollaris pipeline, but the "one binary/image/K8s-resource per service, never collapsed" principle is followed the same way), and `LogServicesRequired`.

| Binary | Directory | Image | Base | K8s workload |
|---|---|---|---|---|
| `secscan` (backend, owns ORM) | `go/secscan/main/` | `saichler/secscan` | `secscan-postgres` | StatefulSet |
| `secscan-web` (UI server) | `go/secscan/ui/` | `saichler/secscan-web` | `secscan-security` | DaemonSet (hostNetwork) |
| `secscan-vnet` | `go/secscan/vnet/` | `saichler/secscan-vnet` | `secscan-security` | DaemonSet (hostNetwork) |
| `secscan-scanner` | `go/secscan/scanner/` | `saichler/secscan-scanner` | `secscan-security` + Trivy binary layer | Deployment, `replicas: 1` for v1 (§13), no hostNetwork/durable state |
| `secscan-log-vnet` | `go/secscan/log-vnet/` | `saichler/secscan-log-vnet` | `secscan-security` | DaemonSet (hostNetwork) |
| `secscan-log-agent` | `go/secscan/log-agent/` | `saichler/secscan-log-agent` | `secscan-security` | DaemonSet |

Own base images required (`DeploymentArtifacts` "Base image rule"): `secscan-security`, `secscan-postgres` — never reuse `erp-security`/`erp-postgres` (protobuf namespace conflict).

`secscan-scanner`'s Dockerfile is the one deliberate customization: multi-stage build adds the Trivy CLI (`aquasec/trivy` release binary) on top of `secscan-security`, since this binary is the only one that shells out to an external scanner.

Required per binary: `build.sh`, `Dockerfile`; project-wide: `build-all-images.sh` (calls each), `k8s/deploy.sh`/`undeploy.sh` (dependency order: vnet → backend → scanner → web → log-vnet → log-agent), and four K8s YAMLs each (`K8sRules`):

- `secscan-{local,baremetal,gke,kind}.yaml` for every workload above (local: `hostPath`; bare-metal: `rancher.io/local-path` + anti-affinity StatefulSets; GKE: `kubernetes.io/gce-pd` + shared `secscan-data` PVC; KIND: bare-metal shape, `storageClassName: standard`).
- `kind-start.sh` / `kind-stop.sh`.

`secscan-scanner` needs no *durable* volume (it owns no customer data) — its K8s manifests carry no `PersistentVolumeClaim`/`StorageClass` section in any of the four modes, only the `emptyDir` Trivy-DB cache from §13.2, identical across all four.

## 17. Local Development Setup

**Before any UI implementation** (`L8UICopyToNewProject`): add `l8ui` as a git submodule under `go/secscan/ui/web/` — copy `setup-l8ui-submodule.sh` from an existing project into that directory and run it; never `cp -r` or symlink `l8ui` from a sibling project. Project-specific files (nav configs, reference registries, section HTML) are then created inside `go/secscan/ui/web/` itself, referencing `../l8ui/`.

`go/run-local.sh`, copied and adapted from `l8erp/go/run-local.sh` (`RunLocalScript`): binary names/paths per §16, web asset path `go/secscan/ui/web/`, mock data path `go/tests/mocks/`, DB name `secscan`, port `2790`. Starts `secscan-vnet` first, then `secscan` (backend), `secscan-scanner`, `secscan-web`, uploads mock data, generates `kill_demo.sh`.

`go/secscan/ui/web/index.html` redirects to `login.html` (`IndexHtmlRedirect`). `app.html` body copied verbatim from `l8erp`'s, only title/sidebar/CSS-module-name adapted (`AppHtmlBodyFromL8erp`), including `css/base-core.css`/`css/responsive.css` and the built-in SYS section HTML (`sections/system.html`) with the exact container IDs from `AppHtmlBodyFromL8erp` (health/modules/logs/security), since mismatched IDs silently produce empty tabs.

`go/demo/` is generated fresh by `run-local.sh` on every run — per `DemoDirectorySync`, it is never edited or hand-synced directly; all source changes go into `go/secscan/ui/web/`.

## 18. Decisions Log (previously Open Questions — now resolved)

1. **Ingestion mechanism** — resolved: bulk, UI-driven, free-text paste (§6.1, §11.5). No registry crawl/discovery in this PRD.
2. **Registry access for scanning** — resolved: `secscan-scanner`'s pod carries full registry credentials, used both for Trivy scanning (§13.1) and build-date metadata resolution (the resolver loop, §6.1 Phase B / §13 intro). No customer-facing registry-credential UI needed.
3. **CSV "Reduction %"** — resolved: per-severity (4 columns: Critical/High/Medium/Low), not one overall figure (§10).
4. **Group-level vulnerability counts** — resolved: both **newest and oldest** scanned image ref's totals are reported (8 columns in the CSV), not just the latest (§10). The main list table still shows only the newest, for table-width reasons (§11.1); full newest/oldest/reduction detail lives in the Group Detail Trend panel (§11.3) and the CSV.
5. **opsadmin / Customer catalog** — resolved: committed v1 scope, not optional (§4).
6. **Write-side `customer_id` trust boundary** — verified against actual `l8types`/`l8services`/`l8secure` source (not assumed): `IServiceCallback` has no access to the caller's identity, so a project cannot independently validate a write's `customer_id` against who's calling — this is a framework limitation, not a project bug to fix. Explicitly accepted for v1: `customer_id` is a client-supplied value the UI sets from the logged-in session (never a user-editable field); a malicious direct API call bypassing the UI is out of scope by product decision (§4, §9). If a stricter guarantee is ever needed, it requires a framework enhancement (e.g., threading `AAAId` into `IServiceCallback`) — flagged for the framework owner, not solved here.
7. **Scanner job-processing concurrency** — verified `l8services`' "2-phase commit" transaction support is a leader-coordinated write-replication protocol (for replica consistency), not a distributed job-claim/lock primitive; there is no compare-and-swap on `PUT`/`PATCH`. `secscan-scanner` therefore runs as a **single replica** for v1 (§13, §16) rather than a horizontally-scaled pool with an invented claim mechanism. Multi-replica coordination via the framework's real leader-election primitives (`IServices.GetLeader`/`IsLeader`/`TriggerElections`) is possible but unverified for a non-ORM-owning worker process — left as explicit future work, not designed here.
8. **CVE findings storage** — verified against `l8orm`'s actual query-to-SQL code (`l8orm/go/orm/stmt/QueryToSql.go`): a nested/child table (what an embedded `repeated Vulnerability` field of `ImageRef` would have become) only gets the query's `WHERE`/`ORDER BY` applied when it *is* the query's root type — otherwise it's read via an unfiltered full-table `SELECT`, with parent-key matching done afterward in application memory. At the volume a per-package-per-image CVE finding table could reach, that would mean either an unbounded read on every "view this image's vulnerabilities" click (no cache) or an ever-growing, all-tenants, all-time in-memory dataset (cache enabled). Resolved: `Vulnerability` is replaced by two Prime Objects — `Cve` (global catalog, §7) and `ImageRefCve` (per-image finding, customer-scoped, denormalized display fields, §7/§8) — so the per-image lookup is a normal, root-type, SQL-filtered/sorted query (§8, §11.4). This also generalizes: any child collection whose volume can grow large must be a Prime Object, never an embedded `repeated`-struct field.

### Residual open items (none blocking, flagged for awareness)

- **Registry client library** — recommended `go-containerregistry` (`crane`) for the resolver loop, since it's pure Go and needs no Docker daemon in the scanner pod; final choice is an implementation detail, not a PRD blocker.
- **Bulk-paste line tolerance** — spec assumes one image ref per line, blank lines ignored. Whitespace/comma-separated variants are not accepted in v1; if pasted CI output contains extra formatting, that's a client-side pre-processing concern, not a server one.

## 19. Testing Strategy

Per `TestLocationAndApproach`: all tests in `go/tests/`, exercised through `IVNic`/HTTP end-to-end — no `_test.go` beside source, no internal-function calls.

- Bulk ingestion (`ImgRefAdd`): paste a mixed batch (valid new refs, an exact duplicate, an unparseable line) and assert the `created`/`skipped`/`errors` split is correct; assert correct `imageName` derivation and group reuse across differing repo hosts/tags.
- Metadata resolution: seed `ImageRef`s with `buildDate=0`, run the resolver loop against a fixture/mock registry client, assert `buildDate` populates and the parent `ImageGroup` rollup cache updates; assert a registry lookup failure sets `scanError` and leaves the row visibly "Resolving…/Failed", never silently stuck with no explanation.
- Scan pipeline: seed a `ScanJob`, run `secscan-scanner` against a fixture/mock Trivy JSON payload (or real Trivy against a known small test image), assert `total_counts`/`distinct_counts` and `ImageRef` status transitions; assert `ImageGroup.newest_counts`/`oldest_counts` update correctly, including the case where a newly-completed scan's `buildDate` is *older* than the group's current cached oldest (cache must move, not just append); assert `Cve` find-or-create doesn't duplicate an already-catalogued CVE across two different images' findings, and that `ImageRefCve` rows carry the scanned `ImageRef`'s `customer_id`, not the scanner's own identity.
- `ImageRefCve` query shape: seed two customers' images with overlapping/duplicate `cveId`s, assert `select * from ImageRefCve where imageRefId=X sort-by severity desc` returns only that image's findings correctly sorted — this is the specific query pattern §8's fix depends on; a regression back to an embedded field here would silently reintroduce the full-table-scan problem.
- Cache/reduction consistency: seed a group with ≥3 scanned image refs at different build dates, assert `VulnRep`'s CSV, the dashboard's Trend indicator, and the Group Detail Trend panel all report identical newest/oldest counts and reduction percentages for that group (single source of truth, §7/§9/§10) — including each of the three `N/A` edge cases applied the same way in all three places.
- Row-level scoping: query as a `customer` user, assert zero cross-tenant leakage; query as `opsadmin`, assert full visibility including `Customer` management.
- `ScanJob` data-integrity check: a `ScanJob` request whose `imageRefIds` don't all belong to the request's own `customerId` is rejected by `ScanJobServiceCallback` (§9) — this is a sanity check on the request's internal consistency, not a cross-tenant security test (see §4 for why write-side identity validation isn't attempted).
- CSV report: assert the full 15-column set, sort order of the `Image Refs` cell, and per-severity reduction-% math (including the "<2 scanned refs", "single data point", and "oldest severity count = 0" → `N/A` edge cases).

## 20. Compliance Checklist (`PrdCompliance`)

- [x] Project structure follows `l8erp` layout (`go/secscan/{common,ui,main,vnet,scanner,log-vnet,log-agent}`, `types/secscan/`, `tests/mocks/`, `proto/`).
- [x] Protobuf: enum zero = `UNSPECIFIED`, list types use `repeated X list = 1` + `metadata`, no direct Prime-Object struct refs (ID-only).
- [x] Service: all ServiceNames ≤ 10 chars, one `ServiceArea` (60) for the module, every `ServiceCallback` (including `Customer`) auto-generates PK on POST, types registered in UI `main.go`, `l8events.EventRecord` registered per `EventsServiceRequired`.
- [x] UI: all `AddingModule` integration steps enumerated (§11), desktop/mobile parity (§11.7), no immutable-entity gaps (all entities here are user-editable), `ImageRefCve` (never user-created/edited — written only by `secscan-scanner`) rendered via a read-only `Layer8DTable` with no add/edit/delete actions, not an `f.inlineTable` edit surface, components follow `l8ui` API (§11–§12).
- [x] Mock Data: generators phased and dependency-ordered (§15), Security-API user provisioning (not project-owned).
- [x] Deployment: `build.sh` + `Dockerfile` per binary, K8s YAMLs for all 4 modes + KIND scripts, `run-local.sh` (§16–§17).
- [x] Configuration: `login.json` adapted (`apiPrefix=/scan`), no `ModConfig` dependency — the l8erp `Layer8DModuleFilter.load()` block is deliberately not called from copied `app.js` (§11.6, §12.2), per `ModconfigFailureNoLogout`; the `admin` module's visibility uses project-specific role logic instead, never that mechanism.
- [x] `l8ui` added as a git submodule before any UI work (§17, `L8UICopyToNewProject`); `go/demo/` never hand-edited (§17, `DemoDirectorySync`).
- [x] `PrdL8uiIncludesAudit` section present (§12).
- [x] No `l8secure` import anywhere; all AAA via `ISecurityProvider`/Security API (§4, §14).
- [x] `SingleOwnerDatabaseTable` respected — only `secscan` backend owns the ORM for its seven Prime-Object-backed services; `secscan-scanner` is vnic-only (§9, §13).
- [x] `PlanRequirements` duplication audit performed — two behavioral patterns that would otherwise be reimplemented 2-3× each are named as single shared abstractions instead: the `pollworker` poll-claim-dispatch harness (§13, used by both the resolver and scan loops), and the `ImageGroup.newest_counts`/`oldest_counts` cache maintained in exactly one hook (§7/§9, read — never re-derived — by the CSV report, dashboard, and Trend panel).
- [x] Multi-tenancy: read-side scoping (`ScopeView`/deny rules) is enforced by the framework and verified against actual source (§4). Write-side `customer_id` validation against caller identity is a **confirmed framework gap** (`IServiceCallback` has no access to caller identity) — not something this PRD works around; it is an explicit, accepted v1 trust boundary (the UI is the only client, and it always supplies its own session's `customerId`), documented in §4/§9 rather than silently assumed.

## 21. Traceability Matrix

| # | Gap / Action Item | Phase |
|---|---|---|
| 1 | Proto definitions + `make-bindings.sh` | Phase 1 — Data Model |
| 2 | Backend services + `ServiceCallback`s (Customer, ImgCat, ImgGroup, ImageRef, Cve, ImgRefCve, ScanJob) | Phase 2 — Backend |
| 3 | `VulnRep` CSV report action-service (15-column, per-severity reduction) | Phase 2 — Backend |
| 3a | `ImgRefAdd` bulk ingestion action-service (parse, dedupe, find-or-create group) | Phase 2 — Backend |
| 4 | Security config JSON + row-level scoping rules (incl. `opsadmin`/`Customer`) | Phase 2 — Backend |
| 5 | `secscan-scanner` binary: Trivy scan loop (§13.1) + metadata resolver loop (§6.1 Phase B) | Phase 3 — Scanner |
| 6 | Desktop UI (module config, enums/columns/forms, dashboard, group detail, vulnerability popup, categories) | Phase 4 — Desktop UI |
| 7 | Mobile UI parity | Phase 5 — Mobile UI |
| 8 | Mock data generators + phased seeding | Phase 6 — Mock Data |
| 9 | Deployment artifacts (Dockerfiles, K8s ×4 modes, KIND, deploy/undeploy, `run-local.sh`) | Phase 7 — Deployment |
| 10 | End-to-end tests (`go/tests/`) | Phase 8 — Testing |
| 11 | Final verification pass (`VerifyPrdCompletenessBeforeDone`, `VerifyAppHtmlScriptsAgainstLoadingOrder`) | Phase 9 — Verification |

No orphaned gaps — every ask.txt requirement (§6 table) resolves to at least one row above.

## 22. Out of Scope / Future Work

- Registry crawling/auto-discovery of new image builds (ingestion is bulk-paste only, §6.1).
- Automated remediation (PRs, base-image bump suggestions).
- Scheduled/periodic re-scans (v1 is on-demand, user-triggered only).
- Multi-scanner support (Grype, Snyk, etc.) — Trivy only, per ask.
