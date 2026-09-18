#!/usr/bin/env bash
set -e

CLUSTER_NAME="secscan"
KIND_CONFIG="kind-cluster.yaml"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Install KIND if not present
if ! command -v kind &>/dev/null; then
  echo "kind not found — installing..."
  if [[ "$(uname)" == "Darwin" ]]; then
    brew install kind
  else
    curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-amd64
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/kind
  fi
  echo "kind installed: $(kind version)"
fi

# Check if cluster already exists
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "KIND cluster '${CLUSTER_NAME}' already exists."
  echo "Run ./kind-stop.sh first if you want to recreate it."
  exit 1
fi

# Single node, control-plane only. No taints override needed -- KIND
# itself automatically removes the control-plane NoSchedule taint on
# cluster creation whenever the cluster has only one node (confirmed
# live: an explicit `taints: []` override actually broke cluster
# creation, since KIND's own untaint step then failed trying to remove a
# taint that was never applied). secscan-kind.yaml's StatefulSet replica
# counts already assume exactly one schedulable node, so no change
# needed there.
cat > "${SCRIPT_DIR}/${KIND_CONFIG}" <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 2790
        hostPort: 2790
        protocol: TCP
EOF

echo "Creating KIND cluster '${CLUSTER_NAME}' (1 node)..."
kind create cluster --name "${CLUSTER_NAME}" --config "${SCRIPT_DIR}/${KIND_CONFIG}"

echo "Waiting for nodes to be Ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s

echo "Loading Docker images into KIND cluster..."
IMAGES=(
  saichler/secscan-vnet:latest
  saichler/secscan:latest
  saichler/secscan-scanner:latest
  saichler/secscan-web:latest
  saichler/secscan-log-vnet:latest
  saichler/secscan-log-agent:latest
)

for img in "${IMAGES[@]}"; do
  if docker image inspect "$img" &>/dev/null; then
    echo "  Loading $img..."
    kind load docker-image "$img" --name "${CLUSTER_NAME}"
  else
    echo "  SKIP $img (not found locally — will pull from registry)"
  fi
done

"${SCRIPT_DIR}/deploy.sh" kind

echo ""
echo "KIND cluster '${CLUSTER_NAME}' is up and secscan is deployed."
echo "Run 'kubectl get pods -n secscan' to check status."
echo "Run './kind-stop.sh' to tear down."
