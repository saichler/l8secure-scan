package tests

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/saichler/l8secure-scan/go/tests/mocks"
	"github.com/saichler/l8types/go/ifs"
)

// testSecurity covers PRD §19's row-level-scoping and ScanJob
// data-integrity scenarios.
func testSecurity(t *testing.T, opsadminClient *mocks.Client, vnic ifs.IVNic) {
	testRowScoping(t, opsadminClient)
	testScanJobIntegrity(t, opsadminClient)
}

// Row-level scoping: querying as the "local" customer-role user
// (Phase 6's seeded local-user) must return zero cross-tenant leakage
// (only customerId="local" rows); querying as opsadmin must see
// everything, including Customer management.
func testRowScoping(t *testing.T, opsadminClient *mocks.Client) {
	custClient := newTestClient(t, opsadminClient.BaseURL())
	if err := custClient.Authenticate("local-user", "localpass"); err != nil {
		t.Fatalf("failed to authenticate as local-user: %v", err)
	}

	groups := queryList(t, custClient, "/60/ImgGroup", "select * from ImageGroup")
	if len(groups) == 0 {
		t.Fatalf("expected local-user to see at least its own ImageGroups, got 0")
	}
	for _, g := range groups {
		if g["customerId"] != "local" {
			t.Fatalf("row-scoping leak: local-user saw a row for customerId=%v", g["customerId"])
		}
	}

	// A customer-role user has no Customer-management access (PRD §9,
	// §14 -- only opsadmin's role grants the Customer service).
	if _, err := custClient.Get("/60/Customer", `{"text":"select * from Customer"}`); err == nil {
		t.Fatalf("expected local-user to be denied Customer access, but the request succeeded")
	}

	// opsadmin must see every customer's groups (at least "local" and
	// "report-cust-2", both seeded earlier in this test run) plus full
	// Customer management access.
	allGroups := queryList(t, opsadminClient, "/60/ImgGroup", "select * from ImageGroup")
	seen := map[string]bool{}
	for _, g := range allGroups {
		if custID, ok := g["customerId"].(string); ok {
			seen[custID] = true
		}
	}
	if !seen["local"] || !seen["report-cust-2"] {
		t.Fatalf("expected opsadmin to see groups for both local and report-cust-2, saw: %v", seen)
	}
	customers := queryList(t, opsadminClient, "/60/Customer", "select * from Customer")
	if len(customers) == 0 {
		t.Fatalf("expected opsadmin to have Customer management access, got 0 customers")
	}

	fmt.Println("testRowScoping: customer-role user sees only its own rows and no Customer access; opsadmin sees everything")
}

// ScanJob data-integrity check (PRD §9/§19): a request whose imageRefIds
// don't all belong to the request's own customerId is rejected by
// ScanJobServiceCallback -- a sanity check on the request's own internal
// consistency, not a cross-tenant security boundary (see PRD §4 for why
// write-side identity validation isn't attempted).
func testScanJobIntegrity(t *testing.T, client *mocks.Client) {
	group := findImageGroupByNameViaClient(t, client, "report-cust-2")
	refs := queryList(t, client, "/60/ImageRef", fmt.Sprintf("select * from ImageRef where imageGroupId='%s'", group))
	if len(refs) == 0 {
		t.Fatalf("expected report-cust-2 to have at least one ImageRef")
	}
	foreignRefId, _ := refs[0]["imageRefId"].(string)

	_, err := client.Post("/60/ScanJob", map[string]interface{}{
		"customerId":  "local",
		"imageRefIds": []string{foreignRefId},
	})
	if err == nil {
		t.Fatalf("expected ScanJob POST with a foreign-customer imageRefId to be rejected, but it succeeded")
	}

	fmt.Println("testScanJobIntegrity: ScanJob request referencing another customer's imageRefId was rejected")
}

func findImageGroupByNameViaClient(t *testing.T, client *mocks.Client, customerId string) string {
	groups := queryList(t, client, "/60/ImgGroup", fmt.Sprintf("select * from ImageGroup where customerId='%s'", customerId))
	if len(groups) == 0 {
		t.Fatalf("expected at least one ImageGroup for customerId=%s", customerId)
	}
	id, _ := groups[0]["imageGroupId"].(string)
	return id
}

func newTestClient(t *testing.T, baseURL string) *mocks.Client {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	return mocks.NewClient(baseURL, httpClient)
}

func queryList(t *testing.T, client *mocks.Client, endpoint, query string) []map[string]interface{} {
	q, _ := json.Marshal(map[string]string{"text": query})
	body, err := client.Get(endpoint, string(q))
	if err != nil {
		t.Fatalf("GET %s failed: %v", endpoint, err)
	}
	var resp struct {
		List []map[string]interface{} `json:"list"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("failed to decode GET %s response: %v (body=%s)", endpoint, err, body)
	}
	return resp.List
}
