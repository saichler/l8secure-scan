#!/usr/bin/env bash
# Re-encrypts the real GCR/GAR registry credentials from ../../gcr into the
# *.enc files next to this script, so they can be committed to this repo.
#
# Run this by hand whenever the credentials in ../../gcr change. The
# passphrase is never stored -- it is typed in, and the same one has to be
# typed into apply-registry-credentials.sh to get the plaintext back.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
GCR_DIR="$(cd "${SCRIPT_DIR}/../../gcr" && pwd)"

SOURCES=(
  "gcr-gar-auth-secret.yaml"
  "gcr-gar-sa-key-secret.yaml"
)

for src in "${SOURCES[@]}"; do
  if [ ! -f "${GCR_DIR}/${src}" ]; then
    echo "error: ${GCR_DIR}/${src} not found" >&2
    exit 1
  fi
done

read -r -s -p "Encryption key: " KEY
echo
if [ -z "$KEY" ]; then
  echo "error: empty key" >&2
  exit 1
fi

for src in "${SOURCES[@]}"; do
  # -a (base64 armor) so the ciphertext is text and diffs/stores cleanly in
  # git; -pbkdf2 with a high iteration count so a passphrase (not a
  # high-entropy key) is an acceptable input.
  openssl enc -aes-256-cbc -md sha256 -pbkdf2 -iter 600000 -salt -a \
    -pass fd:3 \
    -in "${GCR_DIR}/${src}" \
    -out "${SCRIPT_DIR}/${src}.enc" 3<<<"$KEY"
  echo "encrypted ${src} -> ${src}.enc"
done

unset KEY
echo "Done."
