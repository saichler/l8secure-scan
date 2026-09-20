#!/usr/bin/env bash
# Decrypts the GCR/GAR service-account key stored next to this script and
# wires it into the running secscan-scanner pod, so Trivy (scanloop) and
# go-containerregistry (resolver) can both pull private images.
#
# Asks for the encryption key, and does nothing at all unless the key is
# right. The plaintext only ever exists in a 0700 tmpdir that is wiped on
# exit, including on error or Ctrl-C.
#
# The docker config is GENERATED here from the service-account key rather
# than shipped alongside it. ../../gcr's own dockerconfigjson authenticates
# as "oauth2accesstoken" -- a gcloud OAuth access token with a ~1h TTL, so
# storing it encrypted in git means storing an already-expired credential.
# Trivy papered over that by reading the service-account key directly via
# GOOGLE_APPLICATION_CREDENTIALS, but go-containerregistry's
# authn.DefaultKeychain reads ONLY the docker config and has no GCP support
# (that lives in pkg/v1/google, which this project doesn't vendor) -- so the
# resolver got UNAUTHORIZED and every ImageRef kept buildDate=0. Deriving
# the config from the key gives one long-lived credential that both
# consumers can use.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

NAMESPACE="secscan"
DEPLOYMENT="secscan-scanner"
CONTAINER="secscan-scanner"

# Where the credentials land inside the scanner container. DOCKER_CONFIG is
# honoured by both Trivy and go-containerregistry (vendored keychain.go
# calls config.Load($DOCKER_CONFIG) whenever the var is set).
DOCKER_CONFIG_DIR="/etc/secscan/docker"
SA_KEY_DIR="/etc/secscan/gcloud"

# Generated, not taken from ../../gcr -- hence its own name.
DOCKER_SECRET="secscan-registry-docker-config"

# Registry hosts the generated docker config authenticates to. Same three
# ../../gcr's dockerconfigjson covered.
REGISTRY_HOSTS=(
  "gcr.io"
  "us-central1-docker.pkg.dev"
  "us-docker.pkg.dev"
)

SA_ENC="${SCRIPT_DIR}/gcr-gar-sa-key-secret.yaml.enc"

if [ ! -f "$SA_ENC" ]; then
  echo "error: $SA_ENC not found -- run ./encrypt.sh first" >&2
  exit 1
fi

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

if ! openssl enc -d -aes-256-cbc -md sha256 -pbkdf2 -iter 600000 -a \
      -pass fd:3 -in "$SA_ENC" -out "${WORK}/sa.yaml" 3<<<"$KEY" 2>/dev/null \
   || ! grep -q '^kind: Secret' "${WORK}/sa.yaml"; then
  # openssl's padding check catches most wrong keys, but not all -- a
  # decrypt that produced something other than the Secret manifest we put
  # in is a wrong key too.
  echo "Wrong key -- nothing was changed." >&2
  exit 1
fi
unset KEY
echo "Key accepted, credentials decrypted."

# --- read the Secret's identity out of the decrypted manifest ------------
# Not hardcoded: whatever ../../gcr ships is what gets applied.
SA_NAME="$(awk '
  $1 == "metadata:" { in_md = 1; next }
  /^[^[:space:]]/   { in_md = 0 }
  in_md && $1 == "name:" { print $2; exit }
' "${WORK}/sa.yaml")"

SA_KEY_FILE="$(awk '
  $1 == "data:" { in_data = 1; next }
  /^[^[:space:]]/ { in_data = 0 }
  in_data && $1 ~ /:$/ { k = $1; sub(/:$/, "", k); print k; exit }
' "${WORK}/sa.yaml")"

for v in SA_NAME SA_KEY_FILE; do
  if [ -z "${!v}" ]; then
    echo "error: could not read $v out of the decrypted manifest" >&2
    exit 1
  fi
done
echo "  SA key secret : ${SA_NAME} (${SA_KEY_FILE})"

# --- generate the docker config from that key ----------------------------
# "_json_key" + the raw service-account JSON as the password is GCR/GAR's
# documented long-lived basic auth, as opposed to the ~1h oauth2accesstoken
# ../../gcr ships. Built with python3 so the JSON string escaping (the key
# material contains newlines and quotes) is done by a real JSON encoder.
awk -v k="${SA_KEY_FILE}:" '$1 == k { print $2; exit }' "${WORK}/sa.yaml" \
  | base64 -d > "${WORK}/service-account.json"

if ! head -c 1 "${WORK}/service-account.json" | grep -q '{'; then
  echo "error: decoded service-account key is not JSON" >&2
  exit 1
fi

REGISTRY_HOSTS="${REGISTRY_HOSTS[*]}" python3 - "${WORK}/service-account.json" "${WORK}/config.json" <<'PY'
import base64, json, os, sys

sa_path, out_path = sys.argv[1], sys.argv[2]
with open(sa_path, "rb") as fh:
    sa_bytes = fh.read()

json.loads(sa_bytes)  # refuse to build a config around a malformed key
sa_text = sa_bytes.decode("utf-8")
auth = base64.b64encode(b"_json_key:" + sa_bytes).decode("ascii")

entry = {"username": "_json_key", "password": sa_text, "auth": auth}
cfg = {"auths": {host: entry for host in os.environ["REGISTRY_HOSTS"].split()}}

with open(out_path, "w") as fh:
    json.dump(cfg, fh)
PY
chmod 600 "${WORK}/config.json"
echo "  docker config : ${DOCKER_SECRET} (_json_key for ${REGISTRY_HOSTS[*]})"

# --- apply both Secrets into the secscan namespace -----------------------
# The SA manifest in ../../gcr targets its own namespace; the scanner lives
# in ${NAMESPACE}, so the namespace is rewritten on the way in.
sed "s|^\([[:space:]]*\)namespace:.*|\1namespace: ${NAMESPACE}|" "${WORK}/sa.yaml" \
  | kubectl apply -f -

kubectl -n "$NAMESPACE" create secret generic "$DOCKER_SECRET" \
  --type=kubernetes.io/dockerconfigjson \
  --from-file=.dockerconfigjson="${WORK}/config.json" \
  --dry-run=client -o yaml \
  | kubectl apply -f -

# --- point the scanner at them -------------------------------------------
# Strategic merge patch: volumes/volumeMounts/env/imagePullSecrets all merge
# on `name`, so re-running this is idempotent.
kubectl -n "$NAMESPACE" patch deployment "$DEPLOYMENT" --type=strategic -p "$(cat <<PATCH
{
  "spec": {
    "template": {
      "spec": {
        "imagePullSecrets": [
          {"name": "${DOCKER_SECRET}"}
        ],
        "volumes": [
          {
            "name": "registry-docker-config",
            "secret": {
              "secretName": "${DOCKER_SECRET}",
              "items": [{"key": ".dockerconfigjson", "path": "config.json"}]
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

echo "secscan-scanner is configured with the GCR/GAR service-account key."
echo "The resolver retries buildDate=0 refs every 15s -- they should fill in shortly."
