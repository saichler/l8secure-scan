package tests

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/resolver"
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
//
// It also covers the "latest" fallback that runs on the way to MISSING
// (scanloop/latest_fallback.go), across all three of its branches at once,
// one repo each:
//
//	no-latest    registry has no "latest" either -> nothing added
//	has-latest   registry does have it           -> added
//	already-has  we already hold a "latest" ref  -> nothing added, and the
//	                                                registry is never asked
//
// The last one is the point of the ordering: the cheap local check has to
// short-circuit before the network call, so this asserts no lookup was
// even attempted for that repo. Safe to assert because the test harness
// (TestAllService_test.go) activates the services and scanjob but never
// starts resolver.Run, so the only thing calling resolver.LookupCreated
// here is the fallback itself.
func testMissingImage(t *testing.T, vnic ifs.IVNic) {
	const custID = "local"
	const repoNoLatest = "missingimage-test/no-latest"
	const repoHasLatest = "missingimage-test/has-latest"
	const repoAlreadyHas = "missingimage-test/already-has"

	now := time.Now().Unix()
	ghostNoLatest := seedResolvedImageRef(t, vnic, custID, repoNoLatest, "ghost", now)
	ghostHasLatest := seedResolvedImageRef(t, vnic, custID, repoHasLatest, "ghost", now)
	ghostAlreadyHas := seedResolvedImageRef(t, vnic, custID, repoAlreadyHas, "ghost", now)
	// The "already covered" case: this repo's latest ref exists up front.
	postBareImageRef(t, vnic, custID, repoAlreadyHas, scanloop.LatestTag)

	origRunTrivy := scanloop.RunTrivy
	defer func() { scanloop.RunTrivy = origRunTrivy }()
	scanloop.RunTrivy = func(repoName, tag, digest string) (*scanloop.TrivyReport, error) {
		return nil, fmt.Errorf("%w: manifest unknown", scanloop.ErrImageNotFound)
	}

	origLookup := resolver.LookupCreated
	defer func() { resolver.LookupCreated = origLookup }()
	var mu sync.Mutex
	var lookedUp []string
	resolver.LookupCreated = func(repoName, tag, digest string) (int64, error) {
		mu.Lock()
		lookedUp = append(lookedUp, repoName+":"+tag)
		mu.Unlock()
		if repoName == repoHasLatest && tag == scanloop.LatestTag {
			return now, nil
		}
		return 0, errors.New("fixture: no such tag in the registry")
	}

	postScanJob(t, vnic, custID, []string{
		ghostNoLatest.ImageRefId, ghostHasLatest.ImageRefId, ghostAlreadyHas.ImageRefId,
	})
	time.Sleep(4 * time.Second)

	// The fallback must never change the outcome for the ref that was
	// actually scanned -- all three stay MISSING regardless of branch.
	for _, ref := range []*secscan.ImageRef{ghostNoLatest, ghostHasLatest, ghostAlreadyHas} {
		assertScanStatusMissing(t, vnic, ref.ImageRefId)
	}

	if n := countRefs(t, vnic, custID, repoHasLatest, scanloop.LatestTag); n != 1 {
		t.Fatalf("expected the registry-backed %s:%s to have been added exactly once, found %d",
			repoHasLatest, scanloop.LatestTag, n)
	}
	if n := countRefs(t, vnic, custID, repoNoLatest, scanloop.LatestTag); n != 0 {
		t.Fatalf("expected no %s:%s to be added (the registry doesn't have it), found %d",
			repoNoLatest, scanloop.LatestTag, n)
	}
	if n := countRefs(t, vnic, custID, repoAlreadyHas, scanloop.LatestTag); n != 1 {
		t.Fatalf("expected %s:%s to stay at exactly one ref (no duplicate), found %d",
			repoAlreadyHas, scanloop.LatestTag, n)
	}

	mu.Lock()
	attempted := append([]string(nil), lookedUp...)
	mu.Unlock()
	for _, l := range attempted {
		if strings.HasPrefix(l, repoAlreadyHas+":") {
			t.Fatalf("expected no registry lookup for %s -- it already had a %s ref, so the local check should have short-circuited first; lookups were %v",
				repoAlreadyHas, scanloop.LatestTag, attempted)
		}
	}

	fmt.Println("testMissingImage: unresolvable refs land MISSING, and the latest-tag fallback adds, skips and short-circuits correctly")
}

func assertScanStatusMissing(t *testing.T, vnic ifs.IVNic, refId string) {
	t.Helper()
	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageRef %s: %v", refId, err)
	}
	got, ok := result.(*secscan.ImageRef)
	if !ok || got == nil {
		t.Fatalf("ImageRef %s not found", refId)
	}
	if got.ScanStatus != secscan.ScanStatus_SCAN_STATUS_MISSING {
		t.Fatalf("expected ImageRef %s scanStatus=MISSING, got %v", refId, got.ScanStatus)
	}
	if got.ScanError == "" {
		t.Fatalf("expected ImageRef %s to carry a scanError explaining why it's missing", refId)
	}
}

// countRefs counts real ImageRef rows for (customer, repo, tag). Skips nil
// entries for the same reason PrepareImageRef's dedupe loop does: the
// local-handler fast path returns a one-element slice holding a single nil
// for a genuine zero-match query.
func countRefs(t *testing.T, vnic ifs.IVNic, customerId, repoName, tag string) int {
	t.Helper()
	query := fmt.Sprintf("select * from ImageRef where customerId='%s' and repoName='%s' and tag='%s'",
		customerId, repoName, tag)
	rows, err := l8common.GetEntitiesByQuery(scommon.ImageRefServiceName, scommon.ServiceArea, query, vnic)
	if err != nil {
		t.Fatalf("failed to query ImageRefs for %s:%s: %v", repoName, tag, err)
	}
	n := 0
	for _, e := range rows {
		if r, ok := e.(*secscan.ImageRef); ok && r != nil {
			n++
		}
	}
	return n
}
