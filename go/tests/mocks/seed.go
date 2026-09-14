package mocks

import "fmt"

// Seed data (PRD §15, scope reduced per explicit user instruction 2026-09-13
// -- see plans/PROGRESS.md: no synthetic/fabricated CVE catalog or scan
// results. Instead seed exactly one customer ("local") and register this
// project's own real container images (PRD §16's own binary/base-image
// table) as real ImageRefs via the real ImgRefAdd bulk-ingestion path
// (§6.1) -- these are genuine images, so a real secscan-scanner run
// against them produces real Trivy findings instead of fabricated ones.

const localCustomerID = "local"

// This project's own 6 project-specific images only (PRD §16), all tagged
// :latest, matching ../l8secure/build-images.sh's own tagging convention.
// NOT the 2 base/infra images (secscan-security, secscan-postgres) --
// those embed compiled secrets (hashed passwords, JWT signing key/secret,
// TLS private key) and must never be pushed to the public registry or
// otherwise surfaced, seed data included (explicit user instruction).
var projectImageRefs = []string{
	"saichler/secscan:latest",
	"saichler/secscan-web:latest",
	"saichler/secscan-vnet:latest",
	"saichler/secscan-scanner:latest",
	"saichler/secscan-log-vnet:latest",
	"saichler/secscan-log-agent:latest",
}

var seedCategories = []map[string]interface{}{
	{"customerId": localCustomerID, "name": "Application Services", "colorCode": "#2563eb"},
	{"customerId": localCustomerID, "name": "Base Images", "colorCode": "#64748b"},
}

func seedCustomer(client *Client) error {
	_, err := client.Post("/60/Customer", map[string]interface{}{
		"customerId": localCustomerID,
		"name":       "Local",
		"isActive":   true,
	})
	if err != nil {
		return fmt.Errorf("seed customer: %w", err)
	}
	fmt.Printf("  Customer %q created\n", localCustomerID)
	return nil
}

func seedCategoriesData(client *Client) error {
	for _, cat := range seedCategories {
		_, err := client.Post("/60/ImgCat", cat)
		if err != nil {
			return fmt.Errorf("seed category %v: %w", cat["name"], err)
		}
		fmt.Printf("  Category %q created\n", cat["name"])
	}
	return nil
}

// Security user for the "local" customer (PRD §15 item 8) -- provisioned
// via the Security API (/73/users), never a project-owned endpoint
// (SecurityRules). customer=localCustomerID is what makes
// SecScan.getCurrentCustomerId() resolve correctly for this user after
// login (the sessionStorage.userCustomer mechanism wired in Phase 4/the
// l8secure/l8types/l8ui plumbing).
//
// localUserPassword must satisfy l8secure's default password policy
// (PasswordPolicy.go: min 10 chars, upper+digit+special required, no 3+
// repeated/sequential characters -- "123"/"abc"/"aaa" all rejected) -- a
// real 400 caught only by actually running this against a live server.
const localUserPassword = "Vx9!TangoQm"

func seedSecurityUser(client *Client) error {
	_, err := client.Post("/73/users", map[string]interface{}{
		"userId":   "local-user",
		"fullName": "Local Customer User",
		"password": map[string]interface{}{"hash": localUserPassword},
		"roles":    map[string]interface{}{"customer": true},
		"associateIds": []string{
			localCustomerID,
		},
		"customer": localCustomerID,
	})
	if err != nil {
		return fmt.Errorf("seed security user: %w", err)
	}
	fmt.Printf("  Security user \"local-user\" created (password: %s, customer: %s)\n", localUserPassword, localCustomerID)
	return nil
}

// Registers this project's own real images via the real ImgRefAdd bulk
// ingestion action-service (§6.1) -- the exact same path a real user's
// "Add Images" UI action takes, not a direct ImageGroup/ImageRef POST.
func seedImageRefs(client *Client) error {
	resp, err := client.Post("/60/ImgRefAdd", map[string]interface{}{
		"customerId":      localCustomerID,
		"imageRefStrings": projectImageRefs,
	})
	if err != nil {
		return fmt.Errorf("seed image refs: %w", err)
	}
	fmt.Printf("  ImgRefAdd response: %s\n", resp)
	return nil
}

// RunSeed runs every seeding step in dependency order. Verified against a
// real cluster: opsadmin's role (secscan.json) grants ONLY Customer
// access (PRD §9's "opsadmin only manages Customer records") -- Categories
// and ImgRefAdd are "customer"-role-only actions and return "access
// denied" for opsadmin, a real 400 caught only by actually running this
// against a live server. So after creating the Customer and the
// customer-role security user (both opsadmin-only actions), the client
// re-authenticates AS that new user for everything customer-scoped.
func RunSeed(client *Client) error {
	fmt.Println("Seeding Customer...")
	if err := seedCustomer(client); err != nil {
		return err
	}

	fmt.Println("Seeding Security User...")
	if err := seedSecurityUser(client); err != nil {
		return err
	}

	// Save the opsadmin token so it's restored before returning -- callers
	// (e.g. go/tests' TestAllServices) keep using the same *Client
	// afterward and expect it to still be opsadmin, not local-user.
	opsadminToken := client.token

	fmt.Println("Re-authenticating as the local-user (customer role)...")
	if err := client.Authenticate("local-user", localUserPassword); err != nil {
		return fmt.Errorf("re-authenticate as local-user: %w", err)
	}

	fmt.Println("Seeding Categories...")
	if err := seedCategoriesData(client); err != nil {
		return err
	}

	fmt.Println("Seeding Image Refs (this project's own images, via ImgRefAdd)...")
	if err := seedImageRefs(client); err != nil {
		return err
	}

	client.token = opsadminToken
	return nil
}
