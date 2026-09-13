package scanloop

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/saichler/l8secure-scan/go/types/secscan"
)

// trivyReport is the minimal subset of `trivy image --format json` this
// project needs (VulnerabilityID/PkgName/InstalledVersion/FixedVersion/
// Severity/Title, PRD §13.1) -- Trivy is invoked as an external CLI, not
// linked as a Go library, so only these fields are modeled.
type trivyReport struct {
	Results []trivyResult `json:"Results"`
}

type trivyResult struct {
	Vulnerabilities []trivyVuln `json:"Vulnerabilities"`
}

type trivyVuln struct {
	VulnerabilityID  string `json:"VulnerabilityID"`
	PkgName          string `json:"PkgName"`
	InstalledVersion string `json:"InstalledVersion"`
	FixedVersion     string `json:"FixedVersion"`
	Severity         string `json:"Severity"`
	Title            string `json:"Title"`
}

// runTrivy shells out to `trivy image --format json <target>` and parses
// its output. tag takes precedence over digest when both/neither are
// empty is a caller bug (PrepareImageRef always sets at least a tag from
// a parsed reference, or the ref would never have been ingested).
func runTrivy(repoName, tag, digest string) (*trivyReport, error) {
	target := repoName
	switch {
	case tag != "":
		target = repoName + ":" + tag
	case digest != "":
		target = repoName + "@" + digest
	default:
		return nil, errors.New("image reference has neither tag nor digest")
	}

	cmd := exec.Command("trivy", "image", "--format", "json", target)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("trivy scan failed: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	report := &trivyReport{}
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
func countSeverities(report *trivyReport) (total, distinct *secscan.VulnerabilityCounts) {
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
