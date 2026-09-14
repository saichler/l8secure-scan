package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/tests/mocks"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

type imgRefAddItemJSON struct {
	Ref    string `json:"ref"`
	Reason string `json:"reason"`
}

type imgRefAddResponseJSON struct {
	Created []string            `json:"created"`
	Skipped []imgRefAddItemJSON `json:"skipped"`
	Errors  []imgRefAddItemJSON `json:"errors"`
}

// testIngestion exercises PRD §19's bulk-ingestion (ImgRefAdd) scenario:
// a mixed batch of valid new refs (across differing repo hosts/tags, to
// verify imageName-derivation-based group reuse), an exact duplicate, and
// an unparseable line -- asserts the created/skipped/errors split.
func testIngestion(t *testing.T, client *mocks.Client, vnic ifs.IVNic) {
	const custID = "local"

	batch := []string{
		"registry-a.example.com/team/ingest-widget:v1",
		"registry-b.example.com/other/INGEST-WIDGET:v2", // same imageName (last segment, lower-cased) -> same group
		"registry-a.example.com/team/ingest-widget:v1",  // exact duplicate of the first line
		"@sha256:deadbeef", // unparseable: empty repo before '@'
	}

	body, err := client.Post("/60/ImgRefAdd", map[string]interface{}{
		"customerId":      custID,
		"imageRefStrings": batch,
	})
	if err != nil {
		t.Fatalf("ImgRefAdd request failed: %v", err)
	}

	var resp imgRefAddResponseJSON
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("failed to decode ImgRefAdd response: %v (body=%s)", err, body)
	}

	if len(resp.Created) != 2 {
		t.Fatalf("expected 2 created refs, got %d (%v)", len(resp.Created), resp.Created)
	}
	if len(resp.Skipped) != 1 {
		t.Fatalf("expected 1 skipped (duplicate) ref, got %d (%v)", len(resp.Skipped), resp.Skipped)
	}
	if len(resp.Errors) != 1 {
		t.Fatalf("expected 1 errored (unparseable) ref, got %d (%v)", len(resp.Errors), resp.Errors)
	}
	if resp.Errors[0].Ref != "@sha256:deadbeef" {
		t.Fatalf("expected the unparseable line to be the '@sha256:deadbeef' one, got %q", resp.Errors[0].Ref)
	}

	// Both created refs must derive the SAME imageName ("ingest-widget",
	// lower-cased last path segment) and therefore share one ImageGroup,
	// despite differing repo hosts/tags/case (PRD §5, §19).
	var refs []*secscan.ImageRef
	for _, id := range resp.Created {
		result, err := l8common.GetEntity(scommon.ImageRefServiceName, scommon.ServiceArea, &secscan.ImageRef{ImageRefId: id}, vnic)
		if err != nil {
			t.Fatalf("failed to fetch created ImageRef %s: %v", id, err)
		}
		ref, ok := result.(*secscan.ImageRef)
		if !ok || ref == nil {
			t.Fatalf("ImageRef %s not found after creation", id)
		}
		refs = append(refs, ref)
	}
	if len(refs) == 2 && refs[0].ImageGroupId != refs[1].ImageGroupId {
		t.Fatalf("expected both created refs to share one ImageGroup, got %s and %s",
			refs[0].ImageGroupId, refs[1].ImageGroupId)
	}

	group, err := l8common.GetEntity(scommon.ImageGroupServiceName, scommon.ServiceArea, &secscan.ImageGroup{ImageGroupId: refs[0].ImageGroupId}, vnic)
	if err != nil {
		t.Fatalf("failed to fetch ImageGroup: %v", err)
	}
	g, ok := group.(*secscan.ImageGroup)
	if !ok || g == nil || g.ImageName != "ingest-widget" {
		t.Fatalf("expected ImageGroup.imageName == \"ingest-widget\", got %v", g)
	}

	fmt.Println("testIngestion: created/skipped/errors split correct, group reuse across hosts/tags/case confirmed")
}
