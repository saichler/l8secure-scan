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

// testMissingImage exercises the MISSING ScanStatus (proto/secscan.proto
// SCAN_STATUS_MISSING): scanning an ImageRef whose reference can't be
// resolved (bad tag/digest, deleted from the registry) must land the
// ImageRef in MISSING, not FAILED -- via scanloop.ErrImageNotFound, the
// same fixture-injection seam testScanPipeline uses for a real Trivy
// report, here returning an error instead.
func testMissingImage(t *testing.T, vnic ifs.IVNic) {
	const custID = "local"
	const repo = "missingimage-test/img"
	ref := seedResolvedImageRef(t, vnic, custID, repo, "ghost", time.Now().Unix())

	origRunTrivy := scanloop.RunTrivy
	defer func() { scanloop.RunTrivy = origRunTrivy }()
	scanloop.RunTrivy = func(repoName, tag, digest, username, password string) (*scanloop.TrivyReport, error) {
		return nil, fmt.Errorf("%w: manifest unknown", scanloop.ErrImageNotFound)
	}

	postScanJob(t, vnic, custID, []string{ref.ImageRefId})
	time.Sleep(3 * time.Second)

	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: ref.ImageRefId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageRef %s: %v", ref.ImageRefId, err)
	}
	got, ok := result.(*secscan.ImageRef)
	if !ok || got == nil {
		t.Fatalf("ImageRef %s not found", ref.ImageRefId)
	}
	if got.ScanStatus != secscan.ScanStatus_SCAN_STATUS_MISSING {
		t.Fatalf("expected ImageRef %s scanStatus=MISSING, got %v", ref.ImageRefId, got.ScanStatus)
	}
	if got.ScanError == "" {
		t.Fatalf("expected ImageRef %s to carry a scanError explaining why it's missing", ref.ImageRefId)
	}

	fmt.Println("testMissingImage: an unresolvable image reference lands ScanStatus=MISSING, not FAILED")
}
