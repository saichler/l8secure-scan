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

echo "Removing secscan (${MODE})..."
kubectl delete -f "$FILE" --ignore-not-found
echo "secscan removed (${MODE})."
