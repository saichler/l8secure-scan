package tests

import (
	"errors"
	"fmt"
	"testing"
	"time"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/resolver"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// testResolver exercises PRD §19's metadata-resolution scenario: seeds
// ImageRefs with buildDate=0 (the natural state right after ingestion,
// §6.1 Phase A), injects a fixture registry lookup (resolver.LookupCreated,
// the exported seam added this phase -- TestLocationAndApproach: this
// test only ever calls resolver.Run, the real exported entry point), runs
// the resolver loop briefly, and asserts buildDate populates (success
// case) and scanError is set / the row stays visibly unresolved (failure
// case) -- never silently stuck with no explanation.
func testResolver(t *testing.T, vnic ifs.IVNic) {
	const custID = "local"
	const fixedCreated = int64(1700000000) // 2023-11-14T22:13:20Z, arbitrary fixed fixture value

	okRef := postBareImageRef(t, vnic, custID, "registry.example.com/resolver-test/ok-image", "v1")
	failRef := postBareImageRef(t, vnic, custID, "registry.example.com/resolver-test/fail-image", "v1")

	origLookup := resolver.LookupCreated
	defer func() { resolver.LookupCreated = origLookup }()
	resolver.LookupCreated = func(repoName, tag, digest string) (int64, error) {
		if repoName == okRef.RepoName {
			return fixedCreated, nil
		}
		return 0, errors.New("fixture: registry lookup failed")
	}

	stop := make(chan struct{})
	go resolver.Run(vnic, stop)
	time.Sleep(3 * time.Second)
	close(stop)

	assertBuildDate(t, vnic, okRef.ImageRefId, fixedCreated, "")
	assertBuildDateFailed(t, vnic, failRef.ImageRefId)

	fmt.Println("testResolver: buildDate resolved on success, scanError set on failure")
}

func postBareImageRef(t *testing.T, vnic ifs.IVNic, custID, repoName, tag string) *secscan.ImageRef {
	created, err := l8common.PostEntity(scommon.ImageRefServiceName, scommon.ServiceArea,
		&secscan.ImageRef{CustomerId: custID, RepoName: repoName, Tag: tag}, vnic)
	if err != nil {
		t.Fatalf("failed to seed ImageRef %s:%s: %v", repoName, tag, err)
	}
	ref, ok := created.(*secscan.ImageRef)
	if !ok || ref == nil {
		t.Fatalf("unexpected type creating ImageRef %s:%s", repoName, tag)
	}
	if ref.BuildDate != 0 {
		t.Fatalf("expected freshly-ingested ImageRef to have buildDate=0, got %d", ref.BuildDate)
	}
	return ref
}

func assertBuildDate(t *testing.T, vnic ifs.IVNic, refId string, want int64, wantErr string) {
	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageRef %s: %v", refId, err)
	}
	ref, ok := result.(*secscan.ImageRef)
	if !ok || ref == nil {
		t.Fatalf("ImageRef %s not found", refId)
	}
	if ref.BuildDate != want {
		t.Fatalf("expected ImageRef %s buildDate=%d, got %d", refId, want, ref.BuildDate)
	}
	if ref.ScanError != wantErr {
		t.Fatalf("expected ImageRef %s scanError=%q, got %q", refId, wantErr, ref.ScanError)
	}
}

func assertBuildDateFailed(t *testing.T, vnic ifs.IVNic, refId string) {
	result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: refId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageRef %s: %v", refId, err)
	}
	ref, ok := result.(*secscan.ImageRef)
	if !ok || ref == nil {
		t.Fatalf("ImageRef %s not found", refId)
	}
	if ref.BuildDate != 0 {
		t.Fatalf("expected failed-lookup ImageRef %s to keep buildDate=0, got %d", refId, ref.BuildDate)
	}
	if ref.ScanError == "" {
		t.Fatalf("expected failed-lookup ImageRef %s to have a non-empty scanError", refId)
	}
}
