package scanloop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/saichler/l8secure-scan/go/types/secscan"
)

// TrivyReport is the minimal subset of `trivy image --format json` this
// project needs (VulnerabilityID/PkgName/InstalledVersion/FixedVersion/
// Severity/Title, PRD §13.1) -- Trivy is invoked as an external CLI, not
// linked as a Go library, so only these fields are modeled. Exported (and
// RunTrivy below is a reassignable var) so PRD §19's scan-pipeline tests
// can inject a fixture Trivy payload deterministically without calling
// any unexported function (TestLocationAndApproach) -- tests only ever
// drive this through the exported scanloop.Run entry point.
type TrivyReport struct {
	Results []TrivyResult `json:"Results"`
}

type TrivyResult struct {
	Vulnerabilities []TrivyVuln `json:"Vulnerabilities"`
}

type TrivyVuln struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
}

// RunTrivy scans one image and defaults to the real `trivy` CLI
// (runTrivyCLI below). Tests reassign it to a fixture function.
var RunTrivy = runTrivyCLI

// ErrImageNotFound wraps a RunTrivy error when the image itself couldn't be
// pulled (bad tag/digest, deleted from the registry, repo doesn't exist) --
// as opposed to Trivy running but failing for some other reason (a real
// tool crash, a malformed report, an unrelated registry outage). image.go
// checks errors.Is(err, ErrImageNotFound) to mark the ImageRef MISSING
// instead of FAILED.
var ErrImageNotFound = errors.New("image not found")

// imageNotFoundMarkers are substrings Trivy/the underlying registry client
// emit on stderr when the image reference itself can't be resolved, not
// when Trivy ran but hit some other error. Verified against real Trivy CLI
// output (docker.io + registry v2 API error bodies) for a nonexistent
// repo/tag/digest -- deliberately excludes auth-only failures ("denied",
// "unauthorized") since those mean the image may well exist, just
// inaccessible with these credentials, which is a real FAILED, not MISSING.
var imageNotFoundMarkers = []string{
	"manifest unknown",
	"manifest_unknown",
	"unable to find the specified image",
	"no such image",
	"name unknown",
	"name_unknown",
	"not found",
}

func isImageNotFound(stderrText string) bool {
	lower := strings.ToLower(stderrText)
	for _, marker := range imageNotFoundMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// ErrAuthRequired wraps a RunTrivy error when the registry rejected the
// pull for lacking (or having wrong) credentials -- the counterpart
// imageNotFoundMarkers' own comment already calls out as excluded from
// MISSING. image.go checks errors.Is(err, ErrAuthRequired) to look up a
// stored credential (ifs.ISecurityProvider.Credential, "registries" group,
// keyed by common.RegistryHost(ref.RepoName)) and retry once before
// marking the ImageRef SCAN_STATUS_AUTH_REQUIRED instead of FAILED.
var ErrAuthRequired = errors.New("registry authentication required")

var authDeniedMarkers = []string{
	"denied",
	"unauthorized",
}

func isAuthDenied(stderrText string) bool {
	lower := strings.ToLower(stderrText)
	for _, marker := range authDeniedMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// trivyMu serializes actual `trivy` CLI invocations. scanloop.go scans up
// to imagePoolSize (4) images concurrently, but Trivy's local cache
// directory (vulnerability DB + the fanal filesystem/layer-analysis cache)
// is one shared bbolt-backed store on disk -- running the CLI concurrently
// against it produces a real, confirmed-live failure: "Failed to acquire
// cache or database lock ... unable to initialize fs cache: cache may be
// in use by another process: timeout". Serializing only the CLI exec
// (not all of scanOneImage) keeps the ImageRef fetch/parse/persist steps
// around it concurrent -- those don't touch Trivy's cache at all.
var trivyMu sync.Mutex

// TrivyLocalImageDirEnv names a directory of pre-saved `docker save`
// tarballs, keyed by sanitized image reference (see localImageTarPath) --
// checked before falling back to Trivy's own image-src chain
// (containerd/remote). Real fix for two separate, confirmed-live
// problems: (1) Docker Hub's anonymous pull rate limit exhausting on
// repeated scans of images this host already has locally (this project's
// own images, plus its sibling ../probler, ../l8erp projects'), and (2) a
// genuine Trivy v0.74.0 + containerd v2.2.1 incompatibility (its
// containerd image source only reliably reads a layer's content on the
// FIRST containerd-sourced scan in a scanner pod's lifetime -- every
// subsequent one fails with "unable to populate: unable to open: failed
// to copy the image: ... not found", reproduced even for images actively
// running as real pods, and unaffected by clearing containerd's
// leases/snapshots). Scanning a local tarball via `--input` bypasses both
// the registry and containerd entirely, so neither problem can occur.
// Unset (the default on non-KIND deployments) -- normal registry-pull
// scanning is unaffected.
const TrivyLocalImageDirEnv = "TRIVY_LOCAL_IMAGE_DIR"

// localImageTarPath returns the path a pre-saved tarball for this exact
// image reference would live at, if TRIVY_LOCAL_IMAGE_DIR is set -- ""
// otherwise. Matches the naming convention the one-time host-side
// `docker save` population step uses (k8s/kind-preload-images.sh).
func localImageTarPath(target string) string {
	dir := os.Getenv(TrivyLocalImageDirEnv)
	if dir == "" {
		return ""
	}
	safeName := strings.NewReplacer("/", "_", ":", "_", "@", "_").Replace(target)
	return filepath.Join(dir, safeName+".tar")
}

// runTrivyCLI shells out to `trivy image --format json <target>` (or
// `--input <tarball>` when a pre-saved local tarball exists for target,
// see localImageTarPath) and parses its output. tag takes precedence over
// digest when both/neither are empty is a caller bug (PrepareImageRef
// always sets at least a tag from a parsed reference, or the ref would
// never have been ingested).
//
// username/password are optional (empty on the normal first attempt) --
// image.go's scanOneImage passes a stored "registries" credential
// (common.RegistryHost-keyed) on its one retry after an ErrAuthRequired,
// via Trivy's documented TRIVY_USERNAME/TRIVY_PASSWORD env vars (single-
// registry basic auth for one invocation, not a docker-config-wide
// setting).
func runTrivyCLI(repoName, tag, digest, username, password string) (*TrivyReport, error) {
	target := repoName
	switch {
	case tag != "":
		target = repoName + ":" + tag
	case digest != "":
		target = repoName + "@" + digest
	default:
		return nil, errors.New("image reference has neither tag nor digest")
	}

	args := []string{"image", "--format", "json"}
	if tarPath := localImageTarPath(target); tarPath != "" {
		if _, statErr := os.Stat(tarPath); statErr == nil {
			args = append(args, "--input", tarPath)
		} else {
			args = append(args, target)
		}
	} else {
		args = append(args, target)
	}

	trivyMu.Lock()
	defer trivyMu.Unlock()

	cmd := exec.Command("trivy", args...)
	if username != "" || password != "" {
		cmd.Env = append(os.Environ(), "TRIVY_USERNAME="+username, "TRIVY_PASSWORD="+password)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if isImageNotFound(stderrText) {
			return nil, fmt.Errorf("%w: %s", ErrImageNotFound, stderrText)
		}
		if isAuthDenied(stderrText) {
			return nil, fmt.Errorf("%w: %s", ErrAuthRequired, stderrText)
		}
		return nil, fmt.Errorf("trivy scan failed: %v: %s", err, stderrText)
	}

	report := &TrivyReport{}
	if err := json.Unmarshal(stdout.Bytes(), report); err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %v", err)
	}
	return report, nil
}

// mapSeverity maps Trivy's severity string to secscan.Severity. Trivy's
// occasional "UNKNOWN" (or anything else unrecognized) has no matching
// bucket -- returns SEVERITY_UNSPECIFIED, and callers must skip such
// findings (enum zero must stay an invalid state, per ProtobufRules; not
// force-mapped into LOW).
func mapSeverity(s string) secscan.Severity {
	switch strings.ToUpper(s) {
	case "CRITICAL":
		return secscan.Severity_SEVERITY_CRITICAL
	case "HIGH":
		return secscan.Severity_SEVERITY_HIGH
	case "MEDIUM":
		return secscan.Severity_SEVERITY_MEDIUM
	case "LOW":
		return secscan.Severity_SEVERITY_LOW
	default:
		return secscan.Severity_SEVERITY_UNSPECIFIED
	}
}

func addSeverity(c *secscan.VulnerabilityCounts, sev secscan.Severity) {
	switch sev {
	case secscan.Severity_SEVERITY_CRITICAL:
		c.Critical++
	case secscan.Severity_SEVERITY_HIGH:
		c.High++
	case secscan.Severity_SEVERITY_MEDIUM:
		c.Medium++
	case secscan.Severity_SEVERITY_LOW:
		c.Low++
	}
}

// countSeverities computes total_counts (every finding, duplicates across
// packages included) and distinct_counts (unique VulnerabilityID) per
// severity, PRD §13.1 step 2.
func countSeverities(report *TrivyReport) (total, distinct *secscan.VulnerabilityCounts) {
	total = &secscan.VulnerabilityCounts{}
	distinct = &secscan.VulnerabilityCounts{}
	seen := map[string]bool{}

	for _, res := range report.Results {
		for _, v := range res.Vulnerabilities {
			sev := mapSeverity(v.Severity)
			if sev == secscan.Severity_SEVERITY_UNSPECIFIED {
				continue
			}
			addSeverity(total, sev)
			if !seen[v.VulnerabilityID] {
				seen[v.VulnerabilityID] = true
				addSeverity(distinct, sev)
			}
		}
	}
	return total, distinct
}
