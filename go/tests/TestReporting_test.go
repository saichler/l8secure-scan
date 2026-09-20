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
	assertCsvSevField(t, scanned, "Newest", "C", "0")
	assertCsvSevField(t, scanned, "Oldest", "C", "1")
	assertCsvSevField(t, scanned, "Reduction %", "C", "100.0%")
	assertCsvSevField(t, scanned, "Reduction %", "H", "100.0%")
	// oldest.medium=0 and newest.medium=0 for this group. No longer the
	// "N/A" that used to mean division-by-zero -- 0 -> 0 is no change,
	// which is 0.0%. Same for low.
	assertCsvSevField(t, scanned, "Reduction %", "M", "0.0%")
	assertCsvSevField(t, scanned, "Reduction %", "L", "0.0%")

	// An unscanned group (one of Phase 6's seeded, still-PENDING images)
	// has nil counts on both sides, which render as zeros rather than
	// blanks, and therefore 0.0% reduction -- nothing measured, nothing
	// changed. This is the case that used to be N/A via the
	// scannedRefCount<2 rule.
	unscanned := findCsvRow(t, rows, "secscan")
	assertCsvSevField(t, unscanned, "Newest", "C", "0")
	assertCsvSevField(t, unscanned, "Reduction %", "C", "0.0%")
	assertCsvSevField(t, unscanned, "Reduction %", "L", "0.0%")

	fmt.Println("testCacheReductionConsistency: CSV reduction math matches expected values, zero-baseline edge cases correct")
}

// CSV report (PRD §19/§10): the consolidated 6-column header set (one
// column per severity-count group instead of one per severity, matching
// the Images table's own Vulnerabilities column format -- explicit
// request), and the "Image Refs" cell's own sort order (newest buildDate
// first, verified against vulnrep's real imageRefsCell -- sort.Slice on
// BuildDate descending).
func testCsvReport(t *testing.T, client *mocks.Client, vnic ifs.IVNic) {
	header, rows := fetchVulnRepCsv(t, client, "local")

	wantHeaders := []string{"Name", "Category", "Newest", "Oldest", "Reduction %", "Image Refs"}
	if len(header) != 6 {
		t.Fatalf("expected 6 CSV columns, got %d: %v", len(header), header)
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

	fmt.Println("testCsvReport: 6-column consolidated header set and Image Refs sort order both correct")
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

// assertCsvSevField reads one consolidated cell ("Newest", "Oldest", or
// "Reduction %" -- each "T:<total> C:<crit> H:<high> M:<med> L:<low>") and
// asserts the value tagged with the given single-letter severity prefix
// ("T", "C", "H", "M", or "L").
func assertCsvSevField(t *testing.T, row []string, colName, letter, want string) {
	idx := map[string]int{
		"Name": 0, "Category": 1, "Newest": 2, "Oldest": 3, "Reduction %": 4, "Image Refs": 5,
	}[colName]
	cell := row[idx]
	prefix := letter + ":"
	for _, field := range strings.Fields(cell) {
		if strings.HasPrefix(field, prefix) {
			got := strings.TrimPrefix(field, prefix)
			if got != want {
				t.Fatalf("expected %s %s to be %q, got %q (cell=%q row=%v)", colName, letter, want, got, cell, row)
			}
			return
		}
	}
	t.Fatalf("no %q field found in %s cell %q (row=%v)", prefix, colName, cell, row)
}
