# Pin the kubectl context in every k8s script

Handoff plan. Nothing here has been started.

## The bug

`kind create cluster` rewrites `~/.kube/config` and sets **current-context to the
new cluster** — globally, for every shell and every project on the machine.

No script in any repo pins a context. They all call bare `kubectl`, so every
`apply` / `rollout` / `delete` targets **whichever kind cluster was created most
recently**, regardless of which repo's script was run.

Observed symptom: with the `secscan` cluster created last, running `l8nasfile`'s
or `l8stocks`' deploy created *their* pods inside the **secscan** cluster. The
mirror happens just as easily — secscan pods landing in their clusters — it's
just less likely to be noticed.

Made worse by teardown: `kind delete cluster` removes that context and does
**not** restore a previous one, leaving no current context at all. The next
unpinned `kubectl` then either fails against `localhost:8080` or, once another
cluster is created, silently targets the wrong one.

## Evidence

| | |
|---|---|
| Repos calling `kubectl` in scripts | 15 |
| Total `kubectl` invocations | 197 |
| Invocations pinning `--context` / `use-context` | **0** |
| Repos isolating via a separate `KUBECONFIG` | **0** |

## The fix

Each creator repo's scripts already define the cluster name. Derive the context
from it and route every call through one variable:

```bash
CLUSTER_NAME="secscan"
KUBECTL="kubectl --context kind-${CLUSTER_NAME}"
```

…then use `$KUBECTL` everywhere instead of `kubectl`. kind's context name is
always `kind-<cluster name>`.

Where a `deploy.sh` also serves real clusters (l8secure-scan's
`local|baremetal|gke` modes), pin **only** in kind mode:

```bash
KUBECTL="kubectl"
[ "$MODE" = "kind" ] && KUBECTL="kubectl --context kind-${CLUSTER_NAME}"
```

## Group 1 — cluster creators (11 repos)

These create a kind cluster, so they both cause the problem and suffer it. The
cluster name is known for each, so the fix is mechanical.

| Repo | Cluster | Scripts (kubectl calls) |
|---|---|---|
| `probler` | `probler` | `k8s/deploy.sh` (12), `k8s/un-deploy.sh` (12), `k8s/kind-start.sh` (8), `k8s/deploy-adm.sh` (5), `k8s/label-nodes.sh` (4), `k8s/un-deploy-adm.sh` (2) |
| `l8vendingmachine` | `vend` | `go/k8s/deploy.sh` (8), `go/k8s/undeploy.sh` (8), `go/k8s/kind-start.sh` (10) |
| `l8stocks` | `l8stocks` | `k8s/kind-start.sh` (7), `k8s/kind-stop.sh` (5), `k8s/kind-refresh.sh` (3), `k8s/deploy.sh` (1), `k8s/un-deploy.sh` (1) |
| `l8secure-scan` | `secscan` | `k8s/deploy.sh` (7), `k8s/kind-start.sh` (2), `k8s/undeploy.sh` (1), `encripted/apply-registry-credentials.sh` (5) |
| `l8K8s` | `l8k8s-test` | `kind-test/kind-up.sh` (9), `kind-test/drivers/drive-cross.sh` (2), `kind-test/drivers/drive-intra.sh` (2) |
| `fmc` | `fmc` | `k8s/kind-start.sh` (6), `k8s/deploy.sh` (1), `k8s/undeploy.sh` (1) |
| `l8erp` | `l8erp` | `k8s/kind-start.sh` (6), `k8s/deploy.sh` (1), `k8s/undeploy.sh` (1) |
| `l8learn` | `l8learn` | `k8s/kind-start.sh` (6), `k8s/deploy.sh` (1), `k8s/undeploy.sh` (1) |
| `l8vibe` | `l8vibe` | `k8s/kind-start.sh` (5), `k8s/deploy.sh` (1), `k8s/undeploy.sh` (1) |
| `l8alarms` | `l8alarms` | `k8s/kind-start.sh` (6) |
| `l8nasfile` | `nasfile` | `k8s/deploy.sh` (3), `k8s/kind-start.sh` (2), `k8s/un-deploy.sh` (1) |

Note `l8vendingmachine` and `l8rubi` keep their scripts under `go/k8s/`, not
`k8s/` — a glob over `*/k8s/*.sh` misses them.

## Group 2 — consumers (4 repos) — decide before editing

These call `kubectl` but never create a cluster, so there is no cluster name to
derive a context from. **Do not blindly pin these to a kind context.**

| Repo | Calls | Mentions kind? |
|---|---|---|
| `shakudo` | 19 | no |
| `l8rubi` | 10 | no |
| `l8spring` | 6 | no |
| `l8secure` | 5 | yes (`go/tests/kind/e2e.sh`) |

`shakudo`, `l8rubi` and `l8spring` may be deploying to a **real** cluster on
purpose, in which case inheriting the ambient context is intended and pinning
would break them. For those, the safer fix is to require the context to be
stated explicitly — e.g. read `${KUBE_CONTEXT:?set KUBE_CONTEXT}` — so an
unnoticed kind cluster can never capture a production deploy. Confirm each
repo's intent before changing it.

`l8secure`'s e2e script does reference kind and can likely be pinned like
Group 1, once its cluster name is established.

## Secondary issue — host-port collisions

Separate from the context bug and a different failure mode. Two clusters
claiming the same host port cannot be **up at the same time**; the second
`kind create cluster` fails rather than producing wrong-cluster pods.

| Port | Claimed by |
|---|---|
| `2443` | `fmc`, `l8learn`, `probler` |
| `4443` | `fmc`, `l8erp`, `l8nasfile`, `probler` |

Unique and therefore safe: `l8secure-scan` (2790), `l8alarms` (2780),
`l8stocks` (4141), `l8nasfile` (3443, 10005), `l8erp` (2773), `fmc` (6767),
`l8learn` (6969).

Worth assigning each project a distinct host port in the same pass, since the
files are already open.

## Suggested order

1. `l8secure-scan` (15 calls) as the pilot — smallest creator with a real
   `deploy.sh` mode split, so it settles both patterns at once.
2. The rest of Group 1, ascending: `l8alarms` (6), `l8nasfile` (6), `fmc` (8),
   `l8erp` (8), `l8learn` (8), `l8vibe` (7), `l8K8s` (13), `l8stocks` (17),
   `l8vendingmachine` (26), `probler` (43).
3. Group 2 only after confirming each repo's intended target.

## Loose ends

- Verify the fix the honest way: with two kind clusters up, run repo A's deploy
  while repo B's context is current, and confirm the pods land in A.
- `l8K8s`'s cluster name comes from a bare `--name l8k8s-test` rather than a
  `CLUSTER_NAME` variable; add the variable while pinning it.
- `l8secure-scan/k8s/kind-cluster.yaml` is **generated** by `kind-start.sh` and
  deleted by `kind-stop.sh`, yet is not in `.gitignore`, so it shows up as an
  untracked file whenever a cluster is running. Add it.
- Consider whether `kind-start.sh` should restore the previous context on exit,
  or whether pinning everywhere makes that unnecessary. Pinning alone is enough
  for scripts; it does not help a human running `kubectl` by hand.
