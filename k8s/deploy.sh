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

# vnet/log-vnet/web/log-agent are DaemonSets in local/gke mode, StatefulSets
# in baremetal/kind mode (K8sRules).
WORKLOAD_KIND="daemonset"
if [ "$MODE" = "baremetal" ] || [ "$MODE" = "kind" ]; then
  WORKLOAD_KIND="statefulset"
fi

echo "Applying secscan (${MODE})..."
kubectl apply -f "$FILE"

# Dependency order (PRD §16): vnet -> backend -> scanner -> web -> log-vnet -> log-agent
echo "Waiting for secscan-vnet..."
kubectl -n secscan rollout status "${WORKLOAD_KIND}/secscan-vnet" --timeout=180s

echo "Waiting for secscan (backend)..."
kubectl -n secscan rollout status statefulset/secscan --timeout=180s

echo "Waiting for secscan-scanner..."
kubectl -n secscan rollout status deployment/secscan-scanner --timeout=120s

echo "Waiting for secscan-web..."
kubectl -n secscan rollout status "${WORKLOAD_KIND}/secscan-web" --timeout=120s

echo "Waiting for secscan-log-vnet..."
kubectl -n secscan rollout status "${WORKLOAD_KIND}/secscan-log-vnet" --timeout=120s

echo "Waiting for secscan-log-agent..."
kubectl -n secscan rollout status "${WORKLOAD_KIND}/secscan-log-agent" --timeout=120s

echo "secscan deployed (${MODE})."
