# Migrate service activation to the SLA (l8common 32b8273)

Handoff plan. Nothing here has been started — all 9 repos below are still on
the old API.

## The fix being propagated

`l8common` commit **`32b8273`** (2026-09-19), *"Remove ServiceConfig, configure
services through the SLA"*.

`ServiceConfig` mirrored a subset of the SLA's setters, so any setter the SLA
gained was silently unavailable to services activated through the wrapper (e.g.
`IntegCfg` could not declare `Name` unique even though `SetUniqueKeys` existed).
Now `NewOrmSLA(...)` returns the SLA with the house defaults applied —
transactional, replication 3, service group `L8SG`, not a voter — and the caller
configures it directly. `ActivateService` keeps only what the SLA cannot
express: credentials, the Postgres plugin, the REST surface, activation.

Target version: `github.com/saichler/l8common v0.0.0-20260920032814-32b827380ab2`

## The transform

```go
// before
l8c.ActivateService(l8c.ServiceConfig{
    ServiceName: ServiceName, ServiceArea: ServiceArea,
    PrimaryKey: "ReportId", Callback: newPrtReportServiceCallback(),
}, &types.PortfolioReport{}, &types.PortfolioReportList{}, creds, dbname, vnic)

// after
sla := l8c.NewOrmSLA(ServiceName, ServiceArea, "ReportId", newPrtReportServiceCallback(),
    &types.PortfolioReport{}, &types.PortfolioReportList{})
l8c.ActivateService(sla, creds, dbname, vnic)
```

**Every call site in all 9 repos uses only the defaults.** Verified: no
`Voter:`, `NonUniqueKeys:`, `Replication:`, `ReplicationCount:` or
`ServiceGroup:` field is set anywhere outside vendor. So this is one uniform
mechanical rewrite with no `sla.SetX(...)` lines to carry over. If a call site
turns up that does set one, map it to the matching setter on `sla` before
calling `ActivateService`.

Watch the local import alias — it is `l8c` in some repos and `l8common` in
others.

## Reference implementation

`l8secure-scan` is already migrated (7 files). Smallest example:
`go/secscan/imagegroup/ImageGroupService.go`. `l8events` and `l8notify` are
migrated too.

## Per-repo procedure

1. Refresh the l8common dependency. Use whichever script the repo has — it
   deletes `go.mod`/`go.sum`/`vendor` and re-resolves against
   `GOPROXY=direct`, so it picks up the new l8common with no manual `go.mod`
   edit.
2. Rewrite every `ServiceConfig{...}` call site as above.
3. `go build ./...` from the repo's `go/` directory.
4. `go vet ./...` where the repo has tests that compile.
5. Commit per repo. Do not push without asking.

## Repos to migrate

| Repo | Call-site files | Dep refresh | Vendors? |
|---|---|---|---|
| `l8stocks` | 3 | `go/vendor.sh` | yes |
| `l8spring` | 5 | *none — check `go/`* | yes |
| `l8alarms` | 6 | `go/vendor.sh` | yes |
| `l8bugs` | 6 | `go/test.sh` | no |
| `l8rubi` | 11 | `go/gomod.sh` | no |
| `fmc` | 12 | `go/vendor.sh` | yes |
| `l8vendingmachine` | 38 | `go/run-local.sh` | no |
| `l8learn` | 43 | *none — check `go/`* | yes |
| `l8erp` | **258** | `go/vendor.sh` | yes |

382 files total. Suggested order is the table order (ascending size), so the
pattern is settled on `l8stocks` before hitting `l8erp`'s 258.

**Not affected:** `l8secure` uses neither API. `l8common`, `l8events`,
`l8notify`, `l8secure-scan` are already on the new one.

## Loose ends to resolve in the new session

- `l8bugs` has Go sources under `go/` but **no `go.mod` and no `vendor/` on
  disk** — they are generated. Confirm how it builds before editing.
- `l8spring` and `l8learn` vendor but have no `vendor.sh`; find their
  equivalent before re-vendoring.
- `l8rubi` and `l8vendingmachine` do not vendor at all — a plain `go mod tidy`
  against `GOPROXY=direct` may be all they need.
- Decide whether each repo also needs its images rebuilt/redeployed, or whether
  the source change alone is the deliverable.
