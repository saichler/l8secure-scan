#!/usr/bin/env bash
set -e

CLUSTER_NAME="secscan"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if ! command -v kind &>/dev/null; then
  echo "kind not found — nothing to stop."
  exit 0
fi

if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "KIND cluster '${CLUSTER_NAME}' does not exist."
  exit 0
fi

echo "Deleting KIND cluster '${CLUSTER_NAME}'..."
kind delete cluster --name "${CLUSTER_NAME}"

rm -f "${SCRIPT_DIR}/kind-cluster.yaml"

echo "KIND cluster '${CLUSTER_NAME}' deleted."
