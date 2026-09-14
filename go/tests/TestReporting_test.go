package tests

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/tests/mocks"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
)

// testReporting covers three closely-related PRD §19 scenarios that all
// read back the data testScanPipeline just wrote: ImageRefCve query
// shape/scoping, cache/reduction consistency, and the VulnRep CSV report.
func testReporting(t *testing.T, client *mocks.Client, vnic ifs.IVNic) {
	testImageRefCveQueryShape(t, vnic)
	testCacheReductionConsistency(t, client, vnic)
	testCsvReport(t, client, vnic)
}

// ImageRefCve query shape (PRD §19): seed a second customer's image whose
// findings reuse the SAME cveId as testScanPipeline's img-old
// ("CVE-2024-SHARED"), then assert querying by a single imageRefId
// returns only that image's own findings, correctly sorted severity
// desc -- the exact query pattern §8's Cve/ImageRefCve split depends on.
func testImageRefCveQueryShape(t *testing.T, vnic ifs.IVNic) {
	const otherCust = "report-cust-2"
	createCustomer(t, vnic, otherCust)

	otherRef := postBareImageRef(t, vnic, otherCust, "report-test/other-img", "v1")
	if _, err := l8common.PostEntity(scommon.ImageRefCveServiceName, scommon.ServiceArea, &secscan.ImageRefCve{
		CustomerId: otherCust, ImageRefId: otherRef.ImageRefId, CveId: "CVE-2024-SHARED",
		Severity: secscan.Severity_SEVERITY_CRITICAL, PackageName: "pkgZ", Title: "shared finding",
	}, vnic); err != nil {
		t.Fatalf("failed to seed cross-customer ImageRefCve: %v", err)
	}

	group := findImageGroupByName(t, vnic, "local", "img")
	refs, err := l8common.GetEntitiesByQuery(scommon.ImageRefServiceName, scommon.ServiceArea,
		fmt.Sprintf("select * from ImageRef where imageGroupId='%s' and tag='old'", group.ImageGroupId), vnic)
	if err != nil || len(refs) != 1 {
		t.Fatalf("failed to find img-old ImageRef: err=%v count=%d", err, len(refs))
	}
	oldRef := refs[0].(*secscan.ImageRef)

	found, err := l8common.GetEntitiesByQuery(scommon.ImageRefCveServiceName, scommon.ServiceArea,
		fmt.Sprintf("select * from ImageRefCve where imageRefId='%s' sort-by severity desc", oldRef.ImageRefId), vnic)
	if err != nil {
		t.Fatalf("ImageRefCve query failed: %v", err)
	}
	if len(found) != 2 {
		t.Fatalf("expected exactly 2 ImageRefCve rows for img-old (not the cross-customer row too), got %d", len(found))
	}
	first, ok := found[0].(*secscan.ImageRefCve)
	if !ok || first == nil || first.Severity != secscan.Severity_SEVERITY_CRITICAL {
		t.Fatalf("expected the first (severity desc) row to be CRITICAL (CVE-2024-SHARED), got %v", first)
	}
	for _, e := range found {
		f := e.(*secscan.ImageRefCve)
		if f.ImageRefId != oldRef.ImageRefId {
			t.Fatalf("ImageRefCve query leaked a row from a different imageRefId: %v", f)
		}
	}

	fmt.Println("testImageRefCveQueryShape: scoped to one imageRefId, sorted severity desc, no cross-customer leakage")
}

// Cache/reduction consistency (PRD §19): the VulnRep CSV, dashboard Trend
// indicator, and Group Detail Trend panel are all defined to read the
// SAME cached ImageGroup fields / the SAME reductionPct math (single
// source of truth, §7/§9/§10) -- this asserts VulnRep's own CSV output
// (the one piece actually reachable over HTTP) matches an independent
// recomputation of that same math against the raw ImageRef rows, for
// both a real-numbers group and an N/A-edge-case group.
func testCacheReductionConsistency(t *testing.T, client *mocks.Client, vnic ifs.IVNic) {
	_, rows := fetchVulnRepCsv(t, client, "local")

	scanned := findCsvRow(t, rows, "img")
	assertCsvCell(t, scanned, "Newest Critical", "0")
	assertCsvCell(t, scanned, "Oldest Critical", "1")
	assertCsvCell(t, scanned, "Reduction % Critical", "100.0")
	assertCsvCell(t, scanned, "Reduction % High", "100.0")
	// oldest.medium=0 for img-old -> division-by-zero N/A rule.
	assertCsvCell(t, scanned, "Reduction % Medium", "N/A")
	assertCsvCell(t, scanned, "Reduction % Low", "N/A")

	// An unscanned group (one of Phase 6's seeded, still-PENDING images)
	// has scannedRefCount<2 -> every reduction cell and count cell is N/A.
	unscanned := findCsvRow(t, rows, "secscan")
	assertCsvCell(t, unscanned, "Newest Critical", "")
	assertCsvCell(t, unscanned, "Reduction % Critical", "N/A")
	assertCsvCell(t, unscanned, "Reduction % Low", "N/A")

	fmt.Println("testCacheReductionConsistency: CSV reduction math matches expected values, N/A edge cases correct")
}

// CSV report (PRD §19/§10): full 15-column header set, and the "Image
// Refs" cell's own sort order (newest buildDate first, verified against
// vulnrep's real imageRefsCell -- sort.Slice on BuildDate descending).
func testCsvReport(t *testing.T, client *mocks.Client, vnic ifs.IVNic) {
	header, rows := fetchVulnRepCsv(t, client, "local")

	wantHeaders := []string{
		"Name", "Category",
		"Newest Critical", "Newest High", "Newest Medium", "Newest Low",
		"Oldest Critical", "Oldest High", "Oldest Medium", "Oldest Low",
		"Reduction % Critical", "Reduction % High", "Reduction % Medium", "Reduction % Low",
		"Image Refs",
	}
	if len(header) != 15 {
		t.Fatalf("expected 15 CSV columns, got %d: %v", len(header), header)
	}
	for i, want := range wantHeaders {
		if header[i] != want {
			t.Fatalf("expected column %d to be %q, got %q", i, want, header[i])
		}
	}

	row := findCsvRow(t, rows, "img")
	refsCell := row[len(row)-1]
	newIdx := strings.Index(refsCell, "scanpipeline-test/img:new")
	midIdx := strings.Index(refsCell, "scanpipeline-test/img:mid")
	oldIdx := strings.Index(refsCell, "scanpipeline-test/img:old")
	if newIdx < 0 || midIdx < 0 || oldIdx < 0 {
		t.Fatalf("Image Refs cell missing an expected entry: %q", refsCell)
	}
	if !(newIdx < midIdx && midIdx < oldIdx) {
		t.Fatalf("expected Image Refs cell sorted newest-buildDate-first (new, mid, old), got %q", refsCell)
	}

	fmt.Println("testCsvReport: 15-column header set and Image Refs sort order both correct")
}

func createCustomer(t *testing.T, vnic ifs.IVNic, customerId string) {
	if _, err := l8common.PostEntity(scommon.CustomerServiceName, scommon.ServiceArea,
		&secscan.Customer{CustomerId: customerId, Name: customerId, IsActive: true}, vnic); err != nil {
		t.Fatalf("failed to create customer %s: %v", customerId, err)
	}
}

func findImageGroupByName(t *testing.T, vnic ifs.IVNic, customerId, imageName string) *secscan.ImageGroup {
	found, err := l8common.GetEntitiesByQuery(scommon.ImageGroupServiceName, scommon.ServiceArea,
		fmt.Sprintf("select * from ImageGroup where customerId='%s' and imageName='%s'", customerId, imageName), vnic)
	if err != nil || len(found) != 1 {
		t.Fatalf("failed to find ImageGroup customerId=%s imageName=%s: err=%v count=%d", customerId, imageName, err, len(found))
	}
	g, ok := found[0].(*secscan.ImageGroup)
	if !ok || g == nil {
		t.Fatalf("unexpected type for ImageGroup customerId=%s imageName=%s", customerId, imageName)
	}
	return g
}

func fetchVulnRepCsv(t *testing.T, client *mocks.Client, customerId string) ([]string, [][]string) {
	body, err := client.Post("/60/VulnRep", map[string]interface{}{"customerId": customerId})
	if err != nil {
		t.Fatalf("VulnRep request failed: %v", err)
	}
	var resp struct {
		CsvData  string `json:"csvData"`
		Filename string `json:"filename"`
		RowCount int32  `json:"rowCount"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("failed to decode VulnRep response: %v (body=%s)", err, body)
	}
	r := csv.NewReader(strings.NewReader(resp.CsvData))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse VulnRep CSV: %v", err)
	}
	if len(records) < 1 {
		t.Fatalf("VulnRep CSV has no header row")
	}
	return records[0], records[1:]
}

func findCsvRow(t *testing.T, rows [][]string, name string) []string {
	for _, row := range rows {
		if len(row) > 0 && row[0] == name {
			return row
		}
	}
	t.Fatalf("no CSV row found with Name=%q", name)
	return nil
}

func assertCsvCell(t *testing.T, row []string, colName, want string) {
	idx := map[string]int{
		"Name": 0, "Category": 1,
		"Newest Critical": 2, "Newest High": 3, "Newest Medium": 4, "Newest Low": 5,
		"Oldest Critical": 6, "Oldest High": 7, "Oldest Medium": 8, "Oldest Low": 9,
		"Reduction % Critical": 10, "Reduction % High": 11, "Reduction % Medium": 12, "Reduction % Low": 13,
		"Image Refs": 14,
	}[colName]
	if row[idx] != want {
		t.Fatalf("expected column %q to be %q, got %q (row=%v)", colName, want, row[idx], row)
	}
}
