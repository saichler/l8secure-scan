#!/usr/bin/env bash
# Decrypts the GCR/GAR registry credentials stored next to this script and
# wires them into the running secscan-scanner pod, so Trivy (scanloop) and
# go-containerregistry (resolver) can pull private images.
#
# Asks for the encryption key, and does nothing at all unless the key is
# right. The plaintext only ever exists in a 0700 tmpdir that is wiped on
# exit, including on error or Ctrl-C.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

NAMESPACE="secscan"
DEPLOYMENT="secscan-scanner"
CONTAINER="secscan-scanner"

# Where the two credentials land inside the scanner container.
# DOCKER_CONFIG is honoured by both Trivy and go-containerregistry's
# authn.DefaultKeychain (keychain.go: falls back to $DOCKER_CONFIG/config.json
# when $HOME/.docker/config.json is absent, which it is in this image).
DOCKER_CONFIG_DIR="/etc/secscan/docker"
SA_KEY_DIR="/etc/secscan/gcloud"

AUTH_ENC="${SCRIPT_DIR}/gcr-gar-auth-secret.yaml.enc"
SA_ENC="${SCRIPT_DIR}/gcr-gar-sa-key-secret.yaml.enc"

for f in "$AUTH_ENC" "$SA_ENC"; do
  if [ ! -f "$f" ]; then
    echo "error: $f not found -- run ./encrypt.sh first" >&2
    exit 1
  fi
done

WORK="$(mktemp -d)"
chmod 700 "$WORK"
cleanup() { rm -rf "$WORK"; }
trap cleanup EXIT INT TERM

read -r -s -p "Decryption key: " KEY
echo
if [ -z "$KEY" ]; then
  echo "error: empty key" >&2
  exit 1
fi

decrypt() {
  local enc="$1" out="$2"
  if ! openssl enc -d -aes-256-cbc -md sha256 -pbkdf2 -iter 600000 -a \
        -pass fd:3 -in "$enc" -out "$out" 3<<<"$KEY" 2>/dev/null; then
    return 1
  fi
  # openssl's padding check catches most wrong keys, but not all -- a
  # decrypt that produced something other than the Secret manifest we put
  # in is a wrong key too.
  grep -q '^kind: Secret' "$out"
}

for pair in "${AUTH_ENC}:${WORK}/auth.yaml" "${SA_ENC}:${WORK}/sa.yaml"; do
  if ! decrypt "${pair%%:*}" "${pair##*:}"; then
    echo "Wrong key -- nothing was changed." >&2
    exit 1
  fi
done
unset KEY
echo "Key accepted, credentials decrypted."

# --- read the identity of each Secret out of the decrypted manifests -----
# Not hardcoded: whatever ../../gcr ships is what gets applied.
secret_name() {
  awk '
    $1 == "metadata:" { in_md = 1; next }
    /^[^[:space:]]/   { in_md = 0 }
    in_md && $1 == "name:" { print $2; exit }
  ' "$1"
}

# The single key under `data:` -- the filename the Secret projects.
secret_data_key() {
  awk '
    $1 == "data:" { in_data = 1; next }
    /^[^[:space:]]/ { in_data = 0 }
    in_data && $1 ~ /:$/ { k = $1; sub(/:$/, "", k); print k; exit }
  ' "$1"
}

AUTH_NAME="$(secret_name "${WORK}/auth.yaml")"
AUTH_KEY="$(secret_data_key "${WORK}/auth.yaml")"
SA_NAME="$(secret_name "${WORK}/sa.yaml")"
SA_KEY_FILE="$(secret_data_key "${WORK}/sa.yaml")"

for v in AUTH_NAME AUTH_KEY SA_NAME SA_KEY_FILE; do
  if [ -z "${!v}" ]; then
    echo "error: could not read $v out of the decrypted manifests" >&2
    exit 1
  fi
done

echo "  pull secret : ${AUTH_NAME} (${AUTH_KEY})"
echo "  SA key      : ${SA_NAME} (${SA_KEY_FILE})"

# --- apply the Secrets into the secscan namespace ------------------------
# The manifests in ../../gcr target their own namespace; the scanner lives
# in ${NAMESPACE}, so the namespace is rewritten on the way in.
for f in "${WORK}/auth.yaml" "${WORK}/sa.yaml"; do
  sed "s|^\([[:space:]]*\)namespace:.*|\1namespace: ${NAMESPACE}|" "$f" \
    | kubectl apply -f -
done

# --- point the scanner at them -------------------------------------------
# Strategic merge patch: volumes/volumeMounts/env/imagePullSecrets all merge
# on `name`, so re-running this is idempotent.
kubectl -n "$NAMESPACE" patch deployment "$DEPLOYMENT" --type=strategic -p "$(cat <<PATCH
{
  "spec": {
    "template": {
      "spec": {
        "imagePullSecrets": [
          {"name": "${AUTH_NAME}"}
        ],
        "volumes": [
          {
            "name": "registry-docker-config",
            "secret": {
              "secretName": "${AUTH_NAME}",
              "items": [{"key": "${AUTH_KEY}", "path": "config.json"}]
            }
          },
          {
            "name": "registry-sa-key",
            "secret": {
              "secretName": "${SA_NAME}",
              "items": [{"key": "${SA_KEY_FILE}", "path": "${SA_KEY_FILE}"}]
            }
          }
        ],
        "containers": [
          {
            "name": "${CONTAINER}",
            "env": [
              {"name": "DOCKER_CONFIG", "value": "${DOCKER_CONFIG_DIR}"},
              {"name": "GOOGLE_APPLICATION_CREDENTIALS", "value": "${SA_KEY_DIR}/${SA_KEY_FILE}"}
            ],
            "volumeMounts": [
              {"name": "registry-docker-config", "mountPath": "${DOCKER_CONFIG_DIR}", "readOnly": true},
              {"name": "registry-sa-key", "mountPath": "${SA_KEY_DIR}", "readOnly": true}
            ]
          }
        ]
      }
    }
  }
}
PATCH
)"

echo "Waiting for ${DEPLOYMENT} to roll out..."
kubectl -n "$NAMESPACE" rollout status "deployment/${DEPLOYMENT}" --timeout=180s

echo "secscan-scanner is configured with the GCR/GAR credentials."
