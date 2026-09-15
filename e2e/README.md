# secscan e2e (Playwright)

Browser-driven end-to-end tests against the real deployed app (desktop `app.html` + mobile `m/app.html`), running against a live cluster (KIND by default). See `plans/playwright-e2e-testing.md` for the full plan and traceability matrix.

## Setup

```bash
cd e2e
npm install
npx playwright install chromium
```

## Environment variables

All have defaults matching this project's own KIND cluster / seeded mock data (`go/tests/mocks/seed.go`, `go/run-local.sh`). Override only if pointing at a different deployment.

| Var | Default |
|---|---|
| `BASE_URL` | `https://172.18.0.6:2790` |
| `OPSADMIN_USER` / `OPSADMIN_PASS` | `opsadmin` / `opsadmin` |
| `CUSTOMER_USER` / `CUSTOMER_PASS` | `local-user` / `Vx9!TangoQm` |

## Run

```bash
npm test                # both projects
npm run test:desktop    # chromium-desktop only
npm run test:mobile     # chromium-mobile only
npm run report          # open the last HTML report
```

Logged-in sessions are cached under `.auth/` (gitignored) after first login, per role, and reused across spec files/runs — delete that directory to force a fresh login.
