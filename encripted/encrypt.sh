#!/usr/bin/env bash
# Re-encrypts the real GCR/GAR service-account key from ../../gcr into the
# .enc file next to this script, so it can be committed to this repo.
#
# Only the service-account key. ../../gcr's dockerconfigjson is NOT stored
# here: it authenticates as "oauth2accesstoken", a gcloud OAuth access token
# with a ~1h TTL, so an encrypted copy in git is an expired credential by
# the time anyone decrypts it. apply-registry-credentials.sh generates a
# docker config from this key instead (_json_key basic auth, long-lived).
#
# Run this by hand whenever the key in ../../gcr is rotated. The passphrase
# is never stored -- it is typed in, and the same one has to be typed into
# apply-registry-credentials.sh to get the plaintext back.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GCR_DIR="$(cd "${SCRIPT_DIR}/../../gcr" && pwd)"

SOURCE="gcr-gar-sa-key-secret.yaml"

if [ ! -f "${GCR_DIR}/${SOURCE}" ]; then
  echo "error: ${GCR_DIR}/${SOURCE} not found" >&2
  exit 1
fi

read -r -s -p "Encryption key: " KEY
echo
if [ -z "$KEY" ]; then
  echo "error: empty key" >&2
  exit 1
fi

# -a (base64 armor) so the ciphertext is text and diffs/stores cleanly in
# git; -pbkdf2 with a high iteration count so a passphrase (not a
# high-entropy key) is an acceptable input.
openssl enc -aes-256-cbc -md sha256 -pbkdf2 -iter 600000 -salt -a \
  -pass fd:3 \
  -in "${GCR_DIR}/${SOURCE}" \
  -out "${SCRIPT_DIR}/${SOURCE}.enc" 3<<<"$KEY"
echo "encrypted ${SOURCE} -> ${SOURCE}.enc"

unset KEY
echo "Done."
