#!/usr/bin/env bash
set -e

MODE="${1:-local}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FILE="${SCRIPT_DIR}/secscan-${MODE}.yaml"

if [ ! -f "$FILE" ]; then
  echo "Usage: $0 [local|baremetal|gke|kind]"
  echo "  (no secscan-${MODE}.yaml found in ${SCRIPT_DIR})"
  exit 1
fi

# Pin the target context instead of inheriting the ambient one. `kind create
# cluster` rewrites the global current-context, so whichever kind cluster was
# created last would otherwise own every kubectl call here -- which is how
# sibling projects' pods ended up inside this cluster (plans/kubectl-context-pinning.md).
# Only for kind: the other modes deploy to real clusters, where inheriting the
# caller's context is the intent.
CLUSTER_NAME="secscan"
KUBECTL=(kubectl)
if [ "$MODE" = "kind" ]; then
  KUBECTL=(kubectl --context "kind-${CLUSTER_NAME}")
fi

echo "Removing secscan (${MODE})..."
"${KUBECTL[@]}" delete -f "$FILE" --ignore-not-found
echo "secscan removed (${MODE})."
