# Implementation Progress

| Phase | Status | Session notes |
|---|---|---|
| 1. Data Model | done | `proto/secscan.proto` + `proto/make-bindings.sh` added; bindings generated for real via the `saichler/protoc` docker image (available locally) into `go/types/secscan/secscan.pb.go`. Bootstrapped `go/go.mod`, `go/go.sum`, `go/vendor/` (see deviation below). `go build ./...` and `go vet ./...` clean. |
| 2. Backend | not started | |
| 3. Scanner | not started | |
| 4. Desktop UI | not started | |
| 5. Mobile UI | not started | |
| 6. Mock Data | not started | |
| 7. Deployment | not started | |
| 8. Testing | not started | |
| 9. Verification | not started | |

## Deviations from the PRD (if any)

- **Phase 1** — The PRD's §7 literal `proto/secscan.proto` code block does not match this ecosystem's actual protobuf conventions (verified against `../l8erp/proto/*.proto` and `../l8alarms/proto/*.proto`, both of which generate cleanly via the same `saichler/protoc` docker pipeline this phase used for real):
  - `import "l8api.proto";` → there is no file named `l8api.proto` anywhere in the ecosystem (verified against `../l8types/proto/`). The actual file (downloaded from `l8types`) is `api.proto`, declaring `package l8api;`. Changed the import to `import "api.proto";` (message references stay `l8api.L8MetaData`, which was already correct in the PRD).
  - `l8api.AuditInfo audit_info = N;` → `AuditInfo` is not defined in `l8api`/`api.proto` at all. It is defined in `l8common.proto` (`package l8common;`, confirmed used as `l8common.AuditInfo` throughout `../l8erp/proto/*.proto`). Added `import "l8common.proto";` and changed every `audit_info` field to `l8common.AuditInfo`.
  - `option go_package = "github.com/saichler/l8secure-scan/go/types/secscan";` → every sibling project instead uses a relative `option go_package = "./types/<module>";` and relies on `make-bindings.sh` to `mv` the generated `./types/<module>` directory into `../go/types/<module>` (which resolves correctly through the Go module path once placed there). Matched that convention: `option go_package = "./types/secscan";`.
  - These are mechanical protobuf-authoring corrections (the PRD's snippet as literally written does not compile/resolve against the real framework), not a design/architecture change — the data model itself (all fields, enums, Prime Object shapes) is implemented exactly as specified in §7.
  - `proto/make-bindings.sh` also needed two `sed` rewrites after generation (mirroring `../l8erp/proto/make-bindings.sh`, not invented): `./types/l8api` → `github.com/saichler/l8types/go/types/l8api` and `./types/l8common` → `github.com/saichler/l8common/go/types/l8common`. The `l8api` rewrite turned out to be necessary even though `../l8alarms/proto/make-bindings.sh` (already-committed, not re-run this session) doesn't have it — the currently-pulled `saichler/protoc:latest` image does not auto-resolve the `l8api` import path, confirmed by actually running the generator and inspecting the output before adding the `sed` line. Also added a `gofmt -w ./types` after the `sed` rewrites — rewriting the import path strings in place can leave the import block out of gofmt's alphabetical order (verified: it did, once, here), and generated code should stay gofmt-clean like every sibling project's.
- **Phase 1 (bootstrap, not really a PRD deviation)** — `VendorAndGit` says never run `go mod tidy`/`go mod vendor`/`go mod init`. This is a brand-new repo with no prior `go.mod`/`vendor/` at all, so those commands were run exactly once to create the module and vendor tree from scratch (`go mod tidy` to resolve `l8types`/`l8common`/`google.golang.org/protobuf`, then `go mod vendor`) — the same one-time bootstrap every sibling project's own history implies. No vendored file was hand-edited; from here on `VendorAndGit` applies normally (any future dependency-version bump is a project the user directs, not something later phases should do silently).

## Open questions raised back to the user

- (none yet)
