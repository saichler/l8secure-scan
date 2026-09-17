# l8SecureScan

A multi-tenant dashboard for managing container image vulnerabilities, built on the Layer 8 ecosystem framework. It ingests image references from customer build/CI pipelines, groups them by logical image name (regardless of registry/repo/tag), scans them on demand with [Trivy](https://github.com/aquasecurity/trivy), and gives each customer a single-tenant view of their vulnerability posture, remediation trends, and CVE detail — plus a CSV report for sharing outside the tool.

Full product design: [`plans/image-security-scan-dashboard-prd.md`](plans/image-security-scan-dashboard-prd.md). Implementation history: [`plans/PROGRESS.md`](plans/PROGRESS.md).

## Core concepts

- **Image reference (`ImageRef`)** — one concrete, scannable artifact a build produces: repo + tag/digest + build date.
- **Image group (`ImageGroup`)** — all `ImageRef`s that share the same `(customerId, imageName)`, where `imageName` is the last path segment of the repo (`registry.example.com/org/backend` → `backend`). This is what lets the same logical image built across different repos/tags roll up into one row.
- **Category** — a user-managed tag applied to an image group, for organizing/filtering.
- **Scan job** — a batch Trivy run against one or more selected image references in a group.
- Each group caches the **newest** and **oldest** scanned ref's per-severity vulnerability counts (Critical/High/Medium/Low), so the dashboard, CSV report, and Service Cards can show remediation trend ("Reduction %") without re-scanning.

## Features

- **Image Groups dashboard** — KPI strip (Images, Pending Scans, CVEs by severity, Groups Not Yet Scanned), sortable/filterable table with a chart view (severity per image), bulk "Add Images" (free-text paste) and "Scan Images" actions.
- **Group Detail** — image refs sorted newest-first, multi-select to trigger a scan, per-ref delete, CVE list per ref sorted Critical→Low.
- **Categories** — full CRUD, assignable per image group.
- **Scan History** — every scan job with status/progress, sorted newest-first.
- **CSV export** — one row per image group: name, category, newest/oldest severity counts, reduction %, and ref count.
- **Multi-customer** — every session shows exactly one customer's data (row-level security scoping); an `opsadmin` role can view/manage across all customers.
- **Desktop and mobile web UI**, both built on the shared `l8ui` component framework (submodule at `go/secscan/ui/web/l8ui`).

## Architecture

Six binaries, deployed as separate Kubernetes workloads (see `k8s/`):

| Binary | Image | Role |
|---|---|---|
| `secscan` | `saichler/secscan` | Backend: ORM-backed services (`Customer`, `ImageCategory`, `ImageGroup`, `ImageRef`, `Cve`, `ImageRefCve`, `ScanJob`) + action services (`ImgRefAdd`, `ImgRefDel`, `VulnRep`). Postgres-backed, in `saichler/secscan-postgres`. |
| `secscan-web` | `saichler/secscan-web` | Serves the desktop/mobile UI and routes HTTP requests to the backend over the Layer 8 vnet. |
| `secscan-scanner` | `saichler/secscan-scanner` | Bundles the Trivy CLI. Runs two poll-claim-dispatch loops: a metadata **resolver** (fetches build dates for newly-added refs via the registry) and the **scan loop** (claims queued `ScanJob`s, shells out to Trivy, writes CVE findings). |
| `secscan-vnet` | `saichler/secscan-vnet` | Layer 8 virtual network hub the other services connect through. |
| `secscan-log-vnet` / `secscan-log-agent` | `saichler/secscan-log-vnet` / `-log-agent` | Centralized logging plane. |

Proto-defined data model: [`proto/secscan.proto`](proto/secscan.proto) (regenerate bindings with `proto/make-bindings.sh`).

## Running locally

Requires Docker and [KIND](https://kind.sigs.k8s.io/).

```bash
go/build-all-images.sh          # build all 6 images
k8s/kind-start.sh                # create a local KIND cluster and deploy
kubectl get pods -n secscan      # check status
```

Seed sample data (3 customers — `local`, `probler`, `l8erp` — each with a category set and its own real container images registered as image refs):

```bash
cd go && go run ./tests/cmd -address https://localhost:2790 -user opsadmin -password opsadmin
```

Tear down: `k8s/kind-stop.sh`. Other deployment modes (bare-metal, GKE) are under `k8s/secscan-{baremetal,gke}.yaml`, applied via `k8s/deploy.sh <mode>`.

## Testing

- **Go unit/integration tests**: `cd go && go test ./tests/...` — in-process backend + DB + security topology, no live cluster needed.
- **End-to-end (Playwright)**: `cd e2e && npm test` (or `npm run test:desktop` / `npm run test:mobile`). Runs against a live deployment (see `e2e/README.md`).

## Repository layout

```
proto/            Protobuf data model + binding generator
go/secscan/       Backend services, scanner, UI server, deployment binaries
go/tests/         Go test suite + mock-data seeder
go/secscan/ui/web/  Desktop + mobile UI (l8ui submodule + project-specific modules)
k8s/              Kubernetes manifests and cluster scripts (local/baremetal/gke/kind)
e2e/              Playwright end-to-end tests
plans/            Product requirements and implementation history
```
