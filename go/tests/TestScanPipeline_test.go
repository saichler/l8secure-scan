package tests

import (
	"fmt"
	"testing"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/scanloop"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// testScanPipeline exercises PRD §19's scan-pipeline scenario end to end,
// via a fixture Trivy payload injected through scanloop.RunTrivy (the
// exported seam added this phase). Scanning is triggered the same way the
// real Dashboard/mobile "Scan Selected" button does -- POST to the
// stateless ScanJob action service, which kicks off scanloop.Run in the
// background itself (plans/scanjob-live-progress.md) -- not a direct call
// to scanloop.Run from the test:
//   - total_counts/distinct_counts and ImageRef status transitions
//   - ImageGroup.newestCounts/oldestCounts updating correctly, including
//     the case where a newly-completed scan's buildDate is OLDER than the
//     group's current cached oldest (the cache must move, not just append)
//   - Cve find-or-create doesn't duplicate an already-catalogued CVE
//     across two different images' findings
//   - ImageRefCve rows carry the scanned ImageRef's own customerId
func testScanPipeline(t *testing.T, vnic ifs.IVNic) {
	const custID = "local"
	const (
		tOld = int64(1600000000) // oldest
		tMid = int64(1650000000)
		tNew = int64(1700000000) // newest
	)

	// Same repoName (-> same deriveImageName, PRD §5) across all three, so
	// they land in ONE ImageGroup as different build tags of one logical
	// image -- distinct tags keep PrepareImageRef's exact-duplicate dedupe
	// (customerId+repoName+tag+digest) from rejecting the 2nd/3rd POST.
	const repo = "scanpipeline-test/img"
	refOld := seedResolvedImageRef(t, vnic, custID, repo, "old", tOld)
	refMid := seedResolvedImageRef(t, vnic, custID, repo, "mid", tMid)
	refNew := seedResolvedImageRef(t, vnic, custID, repo, "new", tNew)

	if refOld.ImageGroupId != refMid.ImageGroupId || refMid.ImageGroupId != refNew.ImageGroupId {
		t.Fatalf("expected all three refs to share one ImageGroup (same imageName), got %s/%s/%s",
			refOld.ImageGroupId, refMid.ImageGroupId, refNew.ImageGroupId)
	}
	groupId := refOld.ImageGroupId

	// Keyed by tag, not repoName -- all three refs above share one
	// repoName (same logical image, different build tags).
	fixtures := map[string]*scanloop.TrivyReport{
		refOld.Tag: {Results: []scanloop.TrivyResult{{Vulnerabilities: []scanloop.TrivyVuln{
			{VulnerabilityID: "CVE-2024-SHARED", Severity: "CRITICAL", PkgName: "pkgA", Title: "shared finding"},
			{VulnerabilityID: "CVE-2024-OLD-1", Severity: "HIGH", PkgName: "pkgB", Title: "old-only finding"},
		}}}},
		refMid.Tag: {Results: []scanloop.TrivyResult{{Vulnerabilities: []scanloop.TrivyVuln{
			{VulnerabilityID: "CVE-2024-SHARED", Severity: "CRITICAL", PkgName: "pkgC", Title: "shared finding"},
			{VulnerabilityID: "CVE-2024-DUP", Severity: "MEDIUM", PkgName: "pkgD", Title: "dup finding"},
			{VulnerabilityID: "CVE-2024-DUP", Severity: "MEDIUM", PkgName: "pkgE", Title: "dup finding"},
		}}}},
		refNew.Tag: {Results: []scanloop.TrivyResult{}}, // clean scan
	}

	origRunTrivy := scanloop.RunTrivy
	defer func() { scanloop.RunTrivy = origRunTrivy }()
	scanloop.RunTrivy = func(repoName, tag, digest string) (*scanloop.TrivyReport, error) {
		report, ok := fixtures[tag]
		if !ok {
			return nil, fmt.Errorf("fixture: no Trivy fixture for tag %s", tag)
		}
		return report, nil
	}

	// Job A: scan mid + new first, leaving old unscanned -- oldest-among-
	// scanned should be "mid" at this point. postScanJob's POST triggers
	// scanning immediately in the background (no poll delay) -- sleep long
	// enough for both images' fixture "scans" to finish.
	postScanJob(t, vnic, custID, []string{refMid.ImageRefId, refNew.ImageRefId})
	time.Sleep(4 * time.Second)

	assertImageRefScanned(t, vnic, refMid.ImageRefId, 1, 2)
	assertImageRefScanned(t, vnic, refNew.ImageRefId, 0, 0)

	groupAfterA := fetchImageGroup(t, vnic, groupId)
	if groupAfterA.ScannedRefCount != 2 {
		t.Fatalf("expected scannedRefCount=2 after job A, got %d", groupAfterA.ScannedRefCount)
	}
	if groupAfterA.NewestCounts == nil || groupAfterA.NewestCounts.Critical != 0 {
		t.Fatalf("expected newest (img-new, clean) counts all zero after job A, got %v", groupAfterA.NewestCounts)
	}
	if groupAfterA.OldestCounts == nil || groupAfterA.OldestCounts.Critical != 1 || groupAfterA.OldestCounts.Medium != 2 {
		t.Fatalf("expected oldest-among-scanned (img-mid) counts critical=1/medium=2 after job A, got %v", groupAfterA.OldestCounts)
	}

	// Job B: scan "old", whose buildDate is OLDER than "mid" -- the
	// oldest-cache must MOVE to "old", not stay pinned on "mid".
	postScanJob(t, vnic, custID, []string{refOld.ImageRefId})
	time.Sleep(4 * time.Second)

	assertImageRefScanned(t, vnic, refOld.ImageRefId, 1, 0)

	groupAfterB := fetchImageGroup(t, vnic, groupId)
	if groupAfterB.ScannedRefCount != 3 {
		t.Fatalf("expected scannedRefCount=3 after job B, got %d", groupAfterB.ScannedRefCount)
	}
	if groupAfterB.OldestCounts == nil || groupAfterB.OldestCounts.Critical != 1 || groupAfterB.OldestCounts.High != 1 || groupAfterB.OldestCounts.Medium != 0 {
		t.Fatalf("expected oldest cache to MOVE to img-old (critical=1/high=1/medium=0) after job B, got %v", groupAfterB.OldestCounts)
	}
	if groupAfterB.NewestCounts == nil || groupAfterB.NewestCounts.Critical != 0 {
		t.Fatalf("expected newest (img-new) to stay unchanged after job B, got %v", groupAfterB.NewestCounts)
	}

	// Cve find-or-create: CVE-2024-SHARED appeared in both img-old and
	// img-mid's findings -- must resolve to exactly one catalog row.
	sharedCves, err := l8common.GetEntitiesByQuery(scommon.CveServiceName, scommon.ServiceArea,
		"select * from Cve where cveId='CVE-2024-SHARED'", vnic)
	if err != nil {
		t.Fatalf("failed to query Cve catalog: %v", err)
	}
	if len(sharedCves) != 1 {
		t.Fatalf("expected exactly 1 Cve row for CVE-2024-SHARED (find-or-create dedup), got %d", len(sharedCves))
	}

	// ImageRefCve rows carry the scanned ImageRef's own customerId, not
	// any scanner-identity value.
	findings, err := l8common.GetEntitiesByQuery(scommon.ImageRefCveServiceName, scommon.ServiceArea,
		fmt.Sprintf("select * from ImageRefCve where imageRefId='%s'", refOld.ImageRefId), vnic)
	if err != nil {
		t.Fatalf("failed to query ImageRefCve for img-old: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 ImageRefCve rows for img-old, got %d", len(findings))
	}
	for _, e := range findings {
		f, ok := e.(*secscan.ImageRefCve)
		if !ok || f == nil || f.CustomerId != custID {
			t.Fatalf("expected ImageRefCve.customerId == %q, got %v", custID, f)
		}
	}

	fmt.Println("testScanPipeline: counts/status transitions, oldest-cache movement, Cve dedup, and customerId all correct")
}

func seedResolvedImageRef(t *testing.T, vnic ifs.IVNic, custID, repoName, tag string, buildDate int64) *secscan.ImageRef {
	ref := postBareImageRef(t, vnic, custID, repoName, tag)
	ref.BuildDate = buildDate
	if err := l8common.PutEntity(scommon.ImageRefServiceName, scommon.ServiceArea, ref, vnic); err != nil {
		t.Fatalf("failed to set buildDate on %s: %v", repoName, err)
	}
	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: ref.ImageRefId}, vnic)
	if err != nil {
		t.Fatalf("failed to re-fetch %s after setting buildDate: %v", repoName, err)
	}
	fresh, ok := result.(*secscan.ImageRef)
	if !ok || fresh == nil {
		t.Fatalf("unexpected type re-fetching %s", repoName)
	}
	return fresh
}

func postScanJob(t *testing.T, vnic ifs.IVNic, custID string, imageRefIds []string) {
	_, err := l8common.PostEntity(scommon.ScanJobServiceName, scommon.ServiceArea,
		&secscan.ScanJob{CustomerId: custID, ImageRefIds: imageRefIds}, vnic)
	if err != nil {
		t.Fatalf("failed to POST ScanJob for %v: %v", imageRefIds, err)
	}
}

func fetchImageGroup(t *testing.T, vnic ifs.IVNic, groupId string) *secscan.ImageGroup {
	result, err := l8common.GetEntity(scommon.ImageGroupServiceName, scommon.ServiceArea, &secscan.ImageGroup{ImageGroupId: groupId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageGroup %s: %v", groupId, err)
	}
	g, ok := result.(*secscan.ImageGroup)
	if !ok || g == nil {
		t.Fatalf("ImageGroup %s not found", groupId)
	}
	return g
}

func assertImageRefScanned(t *testing.T, vnic ifs.IVNic, refId string, wantCritical, wantMedium int32) {
	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageRef %s: %v", refId, err)
	}
	ref, ok := result.(*secscan.ImageRef)
	if !ok || ref == nil {
		t.Fatalf("ImageRef %s not found", refId)
	}
	if ref.ScanStatus != secscan.ScanStatus_SCAN_STATUS_COMPLETED {
		t.Fatalf("expected ImageRef %s scanStatus=COMPLETED, got %v", refId, ref.ScanStatus)
	}
	if ref.TotalCounts == nil || ref.TotalCounts.Critical != wantCritical || ref.TotalCounts.Medium != wantMedium {
		t.Fatalf("expected ImageRef %s total_counts critical=%d/medium=%d, got %v", refId, wantCritical, wantMedium, ref.TotalCounts)
	}
}
