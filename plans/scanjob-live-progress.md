# ScanJob: Direct-Invoke Scanner + Live Progress Bar

## Background

`plans/PROGRESS.md`'s original 9-phase build gave `ScanJob` a poll-claim-dispatch
design (`secscan-scanner` polls `select * from ScanJob where status=1` every 2s,
claims, scans, PUTs once at the end). A later live-KIND session found a real bug
chasing this design: `ImageRef.scanStatus` gets stuck at `SCANNING` and
`ScanJob.status` stuck at `RUNNING` even though other fields from the same final
`PutEntity` call persist correctly (commit `1d9c9c2`, debug logging still present in
`scanner/scanloop/image.go:50,56` and `scanloop.go:110`, not yet removed).

While investigating, a `l8utils` framework file was edited directly without
approval and reverted (`l8utils/go/utils/cache/internalCache.go`) — not repeated
here; this plan touches no framework repo's source, only consumes the separate
`l8utils/plans/generic-websocket-change-notifications.md` plan, implemented as an
explicit phase of its own below (Phase 4), not a footnote.

Your redesign, given directly, replaces poll-claim entirely:

- **`ScanJobs`** (renamed from today's `ScanJob` service registration): stays
  ORM/Postgres-backed, does nothing but persist — no `ServiceCallback`, nothing
  polls it.
- **`ScanJob`** (new): stateless, no ORM, no cache, hosted **inside the
  `secscan-scanner` process**. The Dashboard's "Scan Selected" POSTs to it
  directly. It scans image-by-image in the background and, after every single
  image finishes, sends the updated job state to `ScanJobs`.
- Scanner: no more polling/queue — direct invoke from the POST.
- A new generic l8ui progress-bar component, updated live via websocket, replacing
  today's 1.5s-interval polling.
- No manual notification/callback code for the live-update part — that's the
  generic mechanism `l8utils/plans/generic-websocket-change-notifications.md`
  adds to `Cache` itself; this project writes none of it.

## Decisions

1. **No proto changes needed.** The `secscan.ScanJob` protobuf message stays
   exactly as-is and is reused by both services — an L8Query's `from` clause uses
   the protobuf type name (`ScanJob`), not the `ServiceName`, so `ScanJobs` (the
   renamed persistence service) is queried as `select * from ScanJob where ...`
   even though its `ServiceName` is `"ScanJobs"`. Only the two `ServiceName`
   constants change (see Phase 1).
2. **No `QUEUED` status anymore.** With no poll/claim step, a job goes straight to
   `RUNNING` the moment the new `ScanJob.Post()` handler builds it — there's nothing
   left to be queued behind. `JobStatus_JOB_STATUS_QUEUED` (proto value `1`) stays
   defined (enum zero must stay invalid, unrelated) but is never set by this code
   path again.
3. **Per-image `ScanJobs` PUTs must be serialized**, even though image scanning
   itself stays concurrently pooled (`scanner/scanloop/scanloop.go`'s existing
   `workers.NewWorkers(imagePoolSize)`). PUT is a full-record replace — two
   goroutines finishing images at the same moment and PUTting the shared job struct
   concurrently could clobber each other's `completedImages`/`failedImages`. A
   single mutex around "increment counters, then PUT" inside the job's own work
   function is sufficient; no new framework-level compare-and-swap needed (matches
   the already-accepted "no CAS, single replica" note from `plans/PROGRESS.md`).
4. **No `secscan.json` change needed — verified, not assumed.** The security config
   (`l8secure/go/secure/plugin/secscan/secscan.json`) has three rules keyed to
   `elemType: "ScanJob"` (`allow-scanjob`, `scope-scanjob` — the tenant row-level
   deny `select * from ScanJob where customerId not in ${associateIds}` —, and
   `allow-scanjob-scan`). Read the actual matching code, not just the config's own
   doc comment: both `CanDoAction` (`l8secure/go/secure/provider/SecurityProvider.go:220-240`)
   and `ScopeView` (`l8secure/go/secure/provider/ScopeView.go:37`,
   `typeName := reflect.ValueOf(elem.Element()).Elem().Type().Name()`) derive
   `elemType`/`typeName` purely from the Go/protobuf type of the actual element
   being read or written — never from `ServiceName`. Since Decision 1 keeps the
   protobuf message `secscan.ScanJob` unchanged for both the renamed `ScanJobs`
   persistence service and the new stateless `ScanJob` action service, all three
   existing rules continue to apply identically and correctly to both, with zero
   config changes.

## Traceability Matrix

| # | Gap / Action | Platform | Phase |
|---|---|---|---|
| 1 | Two service names collide (`ScanJob` used today for persistence; the new stateless service also needs to be called `ScanJob`) | Both | Phase 1 |
| 2 | `ScanJobServiceCallback`'s validation (`scanjob/ScanJobServiceCallback.go`) has no home once `ScanJobs` becomes callback-free persistence | n/a (backend) | Phase 2 |
| 3 | `secscan-scanner` has never hosted a web-facing `IServiceHandler` before (`ImgRefAdd`/`VulnRep` are both backend-hosted) | n/a (backend) | Phase 2 |
| 4 | `scanner/scanloop`'s poll-claim harness (`pollworker.Run`, `claim`, 2s tick) still exists, contradicts direct-invoke | n/a (backend) | Phase 3 |
| 5 | Per-image progress is only persisted once, at job finalize, not after each image | n/a (backend) | Phase 3 |
| 6 | Debug logging from the stuck-status investigation (`image.go:50,56`, `scanloop.go:~110`) still present | n/a (backend) | Phase 3 |
| 7 | `l8utils` generic websocket mechanism not yet implemented/vendored — required before the progress bar phases have anything live to show | Both | Phase 4 |
| 8 | Dashboard's progress bar polls `GetEntitiesByQuery` every 1.5s (`dashboard-page.js:90-162`) | Desktop | Phase 5 |
| 9 | No generic, project-agnostic live progress-bar l8ui component exists | Shared (l8ui) | Phase 5 |
| 10 | Mobile's "Scan Selected" (`group-detail-m.js:197-214`) fires-and-forgets with no progress feedback at all — not a polling-to-live change like desktop, a from-nothing addition | Mobile | Phase 5-m |
| 11 | Security config (`secscan.json`) elemType-vs-ServiceName risk | Both | Resolved — Decision 4, no phase needed |

## Phase 1: Service naming

In `go/secscan/common/defaults.go:11-21`:
- Rename `ScanJobServiceName = "ScanJob"` → `ScanJobsServiceName = "ScanJobs"` (the
  persistence service).
- Add `ScanJobServiceName = "ScanJob"` back as a *new* constant — now naming the new
  stateless action service, area `60` (same `ServiceArea` as everything else in this
  module, per Maintainability's "ServiceArea same for all services in a module").

## Phase 2: Split the service

- **Rename** package `go/secscan/scanjob/` → `go/secscan/scanjobs/`. Strip
  `ScanJobServiceCallback.go` entirely — `Activate()` in the renamed
  `ScanJobsService.go` calls `l8common.ActivateService` with `Callback: nil`. No
  `ServiceCallback`, no ID generation, no defaulting — pure CRUD persistence, as
  specified.
- **New package** `go/secscan/scanjob/` (the name just freed up), modeled directly
  on `go/secscan/imgrefadd/` (`ImgRefAdd.go`/`ImgRefAddPost.go`/`ImgRefAddStubs.go`
  — the real, already-used pattern for a stateless `IServiceHandler`, not
  `l8common.ActivateService`):
  - `Activate(vnic ifs.IVNic)`: `handler := &ScanJob{}`,
    `sla := ifs.NewServiceLevelAgreement(handler, ScanJobServiceName, ServiceArea, false, nil)`,
    `ws := web.New(ScanJobServiceName, ServiceArea, 0)`,
    `ws.AddEndpoint(&secscan.ScanJob{}, ifs.POST, &secscan.ScanJob{})` (reuses the
    existing `secscan.ScanJob` message as both request and response shape — no new
    proto message needed, per Decision 1), `sla.SetWebService(ws)`,
    `vnic.Resources().Services().Activate(sla, vnic)`.
  - `Post()` absorbs today's `ScanJobServiceCallback.beforePostScanJob` validation
    verbatim (customer_id required, imageRefIds required, each ref's `customerId`
    must match, via the same `l8common.GetEntity` loop), then `l8common.GenerateID`,
    `Status = JOB_STATUS_RUNNING` (not `QUEUED`, per Decision 2),
    `TotalImages = len(imageRefIds)`, `RequestedAt = now`, `PostEntity` to
    `ScanJobsServiceName` — then launches the scan as a background goroutine
    (`go func() { scanjob.Run(job, vnic) }()`) and returns the created job
    immediately (HTTP response returns fast; scanning continues after, matching
    today's `MainPackageMinimal` — no long-running work synchronously in the
    handler).
- `go/secscan/services/services.go`: `ActivateSecscanServices` calls
  `scanjobs.Activate(...)` instead of `scanjob.Activate(...)` (persistence, stays
  backend-hosted). The new `scanjob.Activate(vnic)` is **not** called here — it's
  called from `scanner/main.go` (Phase 3), the first service in this project ever
  hosted somewhere other than the backend `main` process. Verified for real in
  Phase 6 (End-to-End Verification) — real precedent exists (`ImgRefAdd`/`VulnRep`
  prove the mechanism), but this project has never exercised "web-facing service
  hosted in a non-`main` binary" before.

## Phase 3: Scanner — remove polling, add incremental progress

- `scanner/scanloop/`: keep `scanOneImage`/`RunTrivy`/`replaceFindings`/
  `countSeverities` (`image.go`, `trivy.go`, `cve.go`) exactly as-is — reused, not
  rewritten. Remove `pollworker.Run`/`claim`/the `Run(vnic, stop)` poll-tick entry
  point from `scanloop.go`; replace with a plain `Run(job *secscan.ScanJob, vnic ifs.IVNic)`
  function called directly from the new `scanjob` package's background goroutine
  (Phase 2).
- Add incremental persistence (Traceability #5): after **every** `scanOneImage`
  call returns (success or fail), under the mutex from Decision 3, update
  `job.CompletedImages`/`job.FailedImages` and `PutEntity` to `ScanJobsServiceName`
  immediately — not just once at the very end. This is the literal "whenever
  `ScanJob` finishes an image, it sends the `ScanJob` to the `ScanJobs` service"
  requirement, and it's what the progress bar (Phase 5) actually has something to
  react to besides the start/end states.
- Remove the `"DEBUG ..."` logger lines added in commit `1d9c9c2`
  (`image.go:50,56`, `scanloop.go`'s finalize-PUT debug block) — the redesign
  replaces the poll-claim path those were diagnosing, not patches it in place.
- `scanner/main.go`: replace `go scanloop.Run(nic, stop)` with
  `scanjob.Activate(nic)` (Phase 2's new service registration) — no background
  poll goroutine for scan jobs anymore. `resolver.Run(nic, stop)` (metadata
  resolution — a separate, unrelated always-on loop) is untouched.

## Phase 4: Implement the `l8utils` generic websocket plan (external repos)

This phase is real, ordered work — not a footnote — even though it happens in
other repos. Nothing in Phase 5/5-m has anything live to show without it.

1. Implement `l8utils/plans/generic-websocket-change-notifications.md` Phases 1-4
   (give `Cache` a `vnic`; generic add/delete broadcast; generic update broadcast
   matched by registered query, carrying the full record; `l8web` forwarding the
   record) in the `l8utils`, `l8orm`, and `l8web` repos. That plan's own Phase 5
   (removing the now-redundant `base`/`l8inventory` call sites) is explicitly
   deferred there — not needed for this project, since `l8secure-scan` uses neither
   `base.BaseService` nor `l8inventory`.
2. Bump this project's `go.mod`/`go.sum`/`vendor/` to the updated `l8utils`,
   `l8orm`, `l8web` — the user's own step, per `VendorAndGit` (this project never
   runs `go mod tidy`/`vendor` itself). Phase 6's verification confirms this landed
   before running any KIND checks that depend on it.

## Phase 5: Live progress bar — Desktop (depends on Phase 4)

- New file `go/secscan/ui/web/l8ui/shared/layer8d-progress-bar.js` — generic,
  project-agnostic (`l8ui-no-project-specific-code`), modeled on the same
  subscribe-by-modelType + filter-by-primaryKey shape `Layer8DTable`'s `realtime`
  option and probler's `LivePopup` already use (`Layer8DWebSocket.subscribe`,
  already present in this project's `l8ui/shared/layer8d-websocket.js`) — not a new
  invented mechanism.
  - `Layer8DProgressBar.attach(container, {modelType, primaryKey, fetchCurrent, getProgress(record) => {percent, label}})`.
  - One initial `fetchCurrent()` call on attach to render the starting state (same
    as `LivePopup`'s real pattern), then live thereafter.
  - Once Phase 4 lands, an update notification for a matching `primaryKey` carries
    the full record directly (`msg.record`) — render straight from it via
    `getProgress()`, no re-fetch needed per tick.
- `dashboard-page.js`: replace `fetchScanJob`/`renderProgress`/`statusLabel`/
  `stopPolling`/`startPolling`/`POLL_MS` (lines `90-162`) with
  `Layer8DProgressBar.attach(...)` targeting `modelType:'ScanJob', primaryKey: job.scanJobId`
  (protobuf type name, per Decision 1) right after `scanSelected()`
  (`dashboard-page.js:164-193`) receives the created job's `scanJobId` from the POST
  response — the POST URL itself (`/60/ScanJob`, `dashboard-page.js:172`) is
  unchanged, since `ServiceArea`/`ServiceName` routing is location-agnostic
  regardless of which process (backend vs. scanner) actually hosts the service.

## Phase 5-m: Live progress bar — Mobile (depends on Phase 4)

Today, mobile's "Scan Selected" (`group-detail-m.js:197-214`) POSTs to
`/60/ScanJob` and just shows a static "Scan job queued" toast (`Layer8MUtils.showSuccess`,
`group-detail-m.js:209`) — no progress feedback at all, not even polling. This is
not "swap polling for websocket" like desktop; it's the first progress feedback of
any kind on mobile.

- New file `go/secscan/ui/web/m/js/shared/layer8m-progress-bar.js` (or, if a
  closer generic-mobile-component location already exists under `l8ui/m/` for this
  kind of thing, use that instead — check before creating a new one, per
  `l8ui-no-project-specific-code`/`maintainability`'s "check for existing shared
  utilities before creating new ones"). Mirrors Phase 5's desktop
  `Layer8DProgressBar` shape but built on `Layer8MAuth`/`Layer8MConfig`'s
  mobile-flavored APIs and `Layer8MWebSocket`-equivalent subscribe mechanism (if
  one exists in this project's `l8ui/m/` already — Phase 5-m's first task is
  confirming whether mobile even has a websocket-subscribe primitive today at all,
  since every real-time example found so far, desktop `Layer8DTable.realtime` and
  probler's `LivePopup`, is desktop-only; if mobile has no equivalent primitive
  yet, that primitive itself becomes an earlier task within this phase, not an
  assumption).
- `group-detail-m.js`'s scan-selected handler (`attachScanSelected`,
  `group-detail-m.js:197-214`): once the POST resolves with the created job's
  `scanJobId`, attach the new mobile progress-bar component the same way desktop's
  Phase 5 does, instead of just showing the one-shot toast.

## Phase 6: End-to-End Verification (this project's own KIND cluster)

1. `go build ./...` / `go vet ./...` / `gofmt` clean.
2. Confirm `go.mod` has picked up the `l8utils`/`l8orm`/`l8web` versions with
   Phase 4 landed.
3. Deploy to `k8s/kind-start.sh` (already exists in this repo).
4. - [ ] Desktop: Click "Scan Selected" against real seeded images; confirm the
     HTTP response returns immediately with a `scanJobId`, not after the whole
     scan completes.
   - [ ] Desktop: Confirm the progress bar updates live, incrementally, after each
     image finishes — not just at start/end — via the real websocket connection,
     no manual page refresh.
   - [ ] Desktop: Confirm final status (`COMPLETED`/`FAILED`/`PARTIAL`) reaches the
     UI live.
   - [ ] Mobile: Click "Scan Selected" in Group Detail; confirm the new progress
     component appears and updates live (Phase 5-m) instead of just the old
     one-shot "queued" toast.
   - [ ] Mobile: Confirm final status reaches the UI live, same as desktop.
   - [ ] Confirm `ImageRef.scanStatus` no longer gets stuck at `SCANNING` — the
     actual regression test for the bug that started this investigation
     (commit `1d9c9c2`).
   - [ ] Two browser tabs (or one desktop + one mobile) scanning two different
     jobs simultaneously — confirm each reflects only its own job (per-AAAId
     targeting from the `l8utils` plan working correctly end-to-end, no
     cross-talk).
   - [ ] Confirm `secscan-scanner`'s new web-facing `ScanJob` endpoint actually
     receives and routes the POST correctly when deployed as a separate pod from
     `secscan-web`/backend (Traceability #3's first-of-its-kind concern).
   - [ ] Log in as two different customer-scoped users; confirm each only ever
     sees/receives progress notifications for their own `ScanJobs` rows — the live
     confirmation of Decision 4's code-level finding (`scope-scanjob`'s
     `customerId not in ${associateIds}` deny rule still applying correctly to the
     split services).
