#!/usr/bin/env bash
set -e

# One-time (rerun-safe) host-side step: `docker save`s every image this
# project's seed data references that already exists locally (this
# project's own 6, plus its sibling ../probler (16) and ../l8erp (6)
# projects' own images -- confirmed via `docker images` to already be
# built locally on this host, never needing a registry pull at all) into
# tarballs, then `docker cp`s them straight into the KIND worker node's
# filesystem (KIND nodes are themselves plain docker containers, so this
# needs no cluster config/restart). secscan-kind.yaml hostPath-mounts that
# same node path into secscan-scanner, and trivy.go's runTrivyCLI scans a
# tarball via `--input` when one exists there for the target image,
# instead of Trivy's own containerd/remote image-src chain.
#
# Real fix for two separate, confirmed-live problems (not merely a nice-
# to-have): (1) Docker Hub's anonymous pull rate limit exhausting on
# repeated scans of images already available locally, and (2) a genuine
# Trivy v0.74.0 + containerd v2.2.1 incompatibility (containerd image
# source only reliably works on the FIRST containerd-sourced scan in a
# scanner pod's lifetime, confirmed even for images actively running as
# real pods, unaffected by clearing containerd's leases/snapshots).
# Scanning a local tarball via --input bypasses both the registry and
# containerd entirely.

CLUSTER_NAME="secscan"
WORKER_NODE="${CLUSTER_NAME}-worker"
NODE_DIR="/trivy-images"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

IMAGES=(
  # This project's own (go/build-all-images.sh's tags)
  saichler/secscan:latest
  saichler/secscan-web:latest
  saichler/secscan-vnet:latest
  saichler/secscan-scanner:latest
  saichler/secscan-log-vnet:latest
  saichler/secscan-log-agent:latest
  # ../probler's own (go/tests/mocks/seed.go's problerImageRefs)
  saichler/probler-admission:latest
  saichler/probler-alarms:latest
  saichler/probler-collector:latest
  saichler/probler-inv-box:latest
  saichler/probler-inv-gpu:latest
  saichler/probler-inv-k8s:latest
  saichler/probler-logagent:latest
  saichler/logs-vnet:latest
  saichler/probler-maint:latest
  saichler/probler-webui2:latest
  saichler/probler-orm:latest
  saichler/probler-parser:latest
  saichler/probler-ptctl:latest
  saichler/probler-topo:latest
  saichler/probler-vnet:latest
  saichler/layer8-webui:latest
  # ../l8erp's own (go/tests/mocks/seed.go's l8erpImageRefs)
  saichler/erp-vnet:latest
  saichler/erp-logs-vnet:latest
  saichler/erp-web:latest
  saichler/erp-maint:latest
  saichler/erp:latest
  saichler/erp-log-agent:latest
)

docker exec "${WORKER_NODE}" mkdir -p "${NODE_DIR}"

for img in "${IMAGES[@]}"; do
  if ! docker image inspect "${img}" &>/dev/null; then
    echo "  SKIP ${img} (not present locally)"
    continue
  fi
  # Must match trivy.go's localImageTarPath sanitization exactly: "/", ":", "@" -> "_".
  safe_name="$(echo "${img}" | tr '/:@' '_')"
  tar_path="${TMP_DIR}/${safe_name}.tar"
  echo "  Saving ${img} -> ${safe_name}.tar..."
  docker save "${img}" -o "${tar_path}"
  docker cp "${tar_path}" "${WORKER_NODE}:${NODE_DIR}/${safe_name}.tar"
  rm -f "${tar_path}"
done

echo "Preloaded images available to secscan-scanner at ${NODE_DIR} on ${WORKER_NODE}."
