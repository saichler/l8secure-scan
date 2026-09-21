# Pin the kubectl context in every k8s script

**STATUS: Group 1 is DONE** (commit pending). Group 2 is not started and
needs a decision first — see that section.

**CORRECTION to the original scan:** the claim that *zero* invocations pinned
a context was wrong; the per-repo counting pipeline silently returned 0.
`l8K8s` was already correct and needed no changes at all — it uses
`--context kind-l8k8s-test` for the host cluster and `--kubeconfig` for the
vcluster targets, which is stronger. It is the reference implementation, not
a repo to fix. Real figure: **9 of 197 already pinned**, all in `l8K8s`.

## The bug

`kind create cluster` rewrites `~/.kube/config` and sets **current-context to the
new cluster** — globally, for every shell and every project on the machine.

Almost no script pinned a context (`l8K8s` was the sole exception). They called
bare `kubectl`, so every `apply` / `rollout` / `delete` targeted **whichever
kind cluster was created most recently**, regardless of which repo's script was
run.

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
| Invocations pinning `--context` (all in `l8K8s`) | 9 |
| Repos isolating via a separate `KUBECONFIG` | **0** |

## The fix

Each creator repo's scripts already define the cluster name. Derive the context
from it and route every call through one variable:

```bash
CLUSTER_NAME="secscan"
KUBECTL=(kubectl --context "kind-${CLUSTER_NAME}")
```

…then call `"${KUBECTL[@]}"` everywhere instead of `kubectl`. kind's context
name is always `kind-<cluster name>`. It must be an **array**: as a string,
`"$KUBECTL"` looks for one command whose name contains spaces, and unquoted
`$KUBECTL` word-splits on any value containing whitespace.

Where a `deploy.sh` also serves real clusters (l8secure-scan's
`local|baremetal|gke` modes), pin **only** in kind mode:

```bash
KUBECTL=(kubectl)
[ "$MODE" = "kind" ] && KUBECTL=(kubectl --context "kind-${CLUSTER_NAME}")
```

## Group 1 — cluster creators — DONE

These create a kind cluster, so they both cause the problem and suffer it.
All patched: **34 scripts across 10 repos** (`l8K8s` excluded, already
correct), verified with a stub `kubectl` that records the `--context` it was
handed.

Three patterns were applied, not one:

- **kind-only** scripts (`kind-start.sh`, `kind-stop.sh`, `kind-refresh.sh`)
  pin unconditionally to `kind-${CLUSTER_NAME}`.
- **mode-split** scripts (`deploy.sh`/`undeploy.sh` taking `local|kind|...`)
  pin only in kind mode and stay ambient otherwise, since the other modes
  target real clusters on purpose.
- **ambient** scripts (no cluster to name — `fmc`, `l8erp`, `l8learn`,
  `l8vibe`, `l8vendingmachine`, `probler` deploy/undeploy) resolve the
  current context ONCE, pin that value for the whole run and echo it.
  Non-breaking: the same cluster is targeted, but a sibling creating a kind
  cluster mid-run can no longer move the target between two calls in one
  script. `KUBE_CONTEXT=...` overrides.

`KUBECTL` is a bash **array**, not a string — `KUBECTL="kubectl --context X"`
then `"$KUBECTL"` looks for a single command with spaces in its name, and
unquoted it word-splits unpredictably. `KUBECTL=(kubectl --context "kind-x")`
called as `"${KUBECTL[@]}"` is correct under both.

`l8K8s` is excluded: already correct (see the correction at the top).

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

## Group 2 — consumers (4 repos) — NOT STARTED, decide before editing

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

## Remaining work

1. Group 2, once each repo's intended target is confirmed.
2. The host-port collisions below — untouched.

## Loose ends

- Group 1 was verified with a stub `kubectl` recording the `--context` it
  received. Still worth the live check: with two kind clusters up, run repo
  A's deploy while repo B's context is current and confirm the pods land in A.
- `l8vibe`'s `kind-start.sh` had no `CLUSTER_NAME` variable (the name was
  inline in `--name l8vibe`); one was added. `l8erp`'s deploy/undeploy were
  bare one-liners with no shebang at all; they got `#!/usr/bin/env bash` and
  `set -e` along with the pin.
- `l8secure-scan/k8s/kind-cluster.yaml` is **generated** by `kind-start.sh` and
  deleted by `kind-stop.sh`, yet is not in `.gitignore`, so it shows up as an
  untracked file whenever a cluster is running. Add it.
- The resolver line in the ambient scripts calls `kubectl` literally on
  purpose -- it is what DEFINES `KUBECTL`, so it cannot use it. Do not
  "fix" it to `"${KUBECTL[@]}"`; that was an actual bug during this work.
- `l8stocks/k8s/kind-start.sh` had STAGED changes before this edit and
  `k8s/kind-refresh.sh` is untracked; several repos also carry unrelated
  uncommitted work. Review each repo's diff before committing.
- Consider whether `kind-start.sh` should restore the previous context on exit,
  or whether pinning everywhere makes that unnecessary. Pinning alone is enough
  for scripts; it does not help a human running `kubectl` by hand.
