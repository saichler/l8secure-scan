# encripted/

The real GCR/GAR service-account key, encrypted, so the scanner can pull
private images without anyone hand-rolling a credential in the UI.

Replaces the earlier approach -- storing a per-registry username/password
through `ISecurityProvider.Credential("registries", host)` and retrying
Trivy with `TRIVY_USERNAME`/`TRIVY_PASSWORD` on an `unauthorized`, entered
through a "Provide Credentials" popup. That path is gone; the PRD had
ruled it out anyway (§3 lists "registry credential vaulting UI" as out of
scope, §18/§24 say the scanner pod carries registry access out-of-band).

`SCAN_STATUS_AUTH_REQUIRED` stays, and now means one thing only: the
credentials this script installs don't cover that image's registry host.
The fix is to widen them here, not to retry.

## Files

| File | What it is |
|---|---|
| `gcr-gar-sa-key-secret.yaml.enc` | `Opaque` Secret -- the GCP service-account JSON key |
| `encrypt.sh` | Re-encrypts it from `../../gcr` after a rotation |
| `apply-registry-credentials.sh` | Asks for the key, decrypts, configures the scanner pod |

Ciphertext is `openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -a`,
base64-armored so it diffs and stores as text in git.

## Only the service-account key is stored here

`../../gcr` also has a `gcr-gar-auth-secret.yaml` (a
`kubernetes.io/dockerconfigjson`). It is deliberately **not** kept here: it
authenticates as `oauth2accesstoken`, a gcloud OAuth access token with a
~1h TTL, so an encrypted copy in git is an expired credential by the time
anyone decrypts it. That is not theoretical -- it is what broke build-date
resolution:

- Trivy hid the problem. It has native GCP support and read the
  service-account key straight from `GOOGLE_APPLICATION_CREDENTIALS`, so
  scans succeeded.
- The resolver did not. `go-containerregistry`'s `authn.DefaultKeychain`
  reads **only** the docker config and has no GCP support (that lives in
  `pkg/v1/google`, which this project doesn't vendor). It presented the
  expired token, got `UNAUTHORIZED`, and every ImageRef stayed at
  `buildDate=0` with the 401 in `scanError`.

So `apply-registry-credentials.sh` **generates** the docker config from the
service-account key instead: `_json_key` as the username with the raw key
JSON as the password, which is GCR/GAR's documented long-lived basic auth.
One credential, no expiry, and both consumers can use it.

## Configuring the scanner

```
./apply-registry-credentials.sh
```

It prompts for the key. On a wrong key it stops before touching the
cluster. On the right key it:

1. decrypts the manifest into a `0700` tmpdir that is wiped on exit
   (including on error and Ctrl-C),
2. generates a `_json_key` docker config from the key for `gcr.io`,
   `us-central1-docker.pkg.dev` and `us-docker.pkg.dev`,
3. applies the service-account Secret (namespace rewritten to `secscan`)
   and the generated `secscan-registry-docker-config` Secret,
4. patches `deployment/secscan-scanner` to mount both and set
   `DOCKER_CONFIG=/etc/secscan/docker` and
   `GOOGLE_APPLICATION_CREDENTIALS=/etc/secscan/gcloud/<sa-key>`, plus an
   `imagePullSecrets` entry,
5. waits for the rollout.

The patch is a strategic merge on `name`, so re-running it is idempotent.

`DOCKER_CONFIG` is what covers both registry consumers: Trivy
(`scanloop/trivy.go`) and go-containerregistry's `authn.DefaultKeychain`
(`resolver/resolver.go`) -- vendored `keychain.go` calls
`config.Load($DOCKER_CONFIG)` whenever the variable is set.
`GOOGLE_APPLICATION_CREDENTIALS` stays set as well; it now points at the
same key, so it is a second path to one credential rather than a second
credential.

The Secret name and its `data` key are read out of the decrypted manifest
at runtime, not hardcoded, so whatever `../../gcr` currently holds is what
gets applied. The registry host list is hardcoded in the script -- it is
the same three the old dockerconfigjson covered.

After a first run, an older `gcr-gar-auth` Secret left over from a previous
version of this script is unused and can be deleted by hand.

## Re-encrypting after a key rotation

```
./encrypt.sh
```

Reads `../../gcr/gcr-gar-sa-key-secret.yaml`, prompts for the key, rewrites
the `.enc` file. Commit that.

## The key

The key is never stored in this repo, and must not be. It is typed into
both scripts. Losing it means re-exporting from `../../gcr` and running
`encrypt.sh` with a new one.
