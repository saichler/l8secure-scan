# encripted/

The real GCR/GAR registry credentials, encrypted, so the scanner can pull
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
| `gcr-gar-auth-secret.yaml.enc` | `kubernetes.io/dockerconfigjson` Secret -- the Docker registry auth |
| `gcr-gar-sa-key-secret.yaml.enc` | `Opaque` Secret -- the GCP service-account JSON key |
| `encrypt.sh` | Re-encrypts both from `../../gcr` after the credentials change |
| `apply-registry-credentials.sh` | Asks for the key, decrypts, configures the scanner pod |

Ciphertext is `openssl enc -aes-256-cbc -pbkdf2 -iter 600000 -salt -a`,
base64-armored so it diffs and stores as text in git.

## Configuring the scanner

```
./apply-registry-credentials.sh
```

It prompts for the key. On a wrong key it stops before touching the
cluster. On the right key it:

1. decrypts both manifests into a `0700` tmpdir that is wiped on exit
   (including on error and Ctrl-C),
2. applies both Secrets into the `secscan` namespace, rewriting the
   namespace from whatever `../../gcr` ships,
3. patches `deployment/secscan-scanner` to mount them and set
   `DOCKER_CONFIG=/etc/secscan/docker` and
   `GOOGLE_APPLICATION_CREDENTIALS=/etc/secscan/gcloud/<sa-key>`, and adds
   the dockerconfigjson Secret as an `imagePullSecrets` entry,
4. waits for the rollout.

The patch is a strategic merge on `name`, so re-running it is idempotent.

`DOCKER_CONFIG` is the one setting that covers both registry consumers in
the scanner: Trivy (`scanloop/trivy.go`) and go-containerregistry's
`authn.DefaultKeychain` (`resolver/resolver.go`) -- the keychain falls back
to `$DOCKER_CONFIG/config.json` when `$HOME/.docker/config.json` is absent,
which it is in the scanner image.

Nothing is hardcoded from the credentials themselves: the Secret names and
their `data` keys are read out of the decrypted manifests at runtime, so
whatever `../../gcr` currently holds is what gets applied.

## Re-encrypting after a credential rotation

```
./encrypt.sh
```

Reads `../../gcr/gcr-gar-auth-secret.yaml` and
`../../gcr/gcr-gar-sa-key-secret.yaml`, prompts for the key, rewrites the
two `.enc` files. Commit those.

## The key

The key is never stored in this repo, and must not be. It is typed into
both scripts. Losing it means re-exporting from `../../gcr` and running
`encrypt.sh` with a new one.
