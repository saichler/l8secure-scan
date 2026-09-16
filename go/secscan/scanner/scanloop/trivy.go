package scanloop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
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

// runTrivyCLI shells out to `trivy image --format json <target>` and
// parses its output. tag takes precedence over digest when both/neither
// are empty is a caller bug (PrepareImageRef always sets at least a tag
// from a parsed reference, or the ref would never have been ingested).
func runTrivyCLI(repoName, tag, digest string) (*TrivyReport, error) {
	target := repoName
	switch {
	case tag != "":
		target = repoName + ":" + tag
	case digest != "":
		target = repoName + "@" + digest
	default:
		return nil, errors.New("image reference has neither tag nor digest")
	}

	trivyMu.Lock()
	defer trivyMu.Unlock()

	cmd := exec.Command("trivy", "image", "--format", "json", target)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		if isImageNotFound(stderrText) {
			return nil, fmt.Errorf("%w: %s", ErrImageNotFound, stderrText)
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
