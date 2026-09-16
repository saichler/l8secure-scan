package mocks

import (
	"fmt"
	"strings"
)

// alreadyExists is true when err is the server's "already exists" rejection
// for a duplicate Customer/user id -- expected and harmless when re-running
// the seeder against a cluster that's already (partially) seeded, e.g. to
// add a new customerSeeds entry without tearing down existing data.
func alreadyExists(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists")
}

// Seed data (PRD §15, scope reduced per explicit user instruction 2026-09-13
// -- see plans/PROGRESS.md: no synthetic/fabricated CVE catalog or scan
// results. Instead seed real customers and register real container images
// (this project's own, plus the sibling ../probler and ../l8erp projects'
// own images) as real ImageRefs via the real ImgRefAdd bulk-ingestion path
// (§6.1) -- these are genuine images, so a real secscan-scanner run
// against them produces real Trivy findings instead of fabricated ones.

const (
	localCustomerID   = "local"
	problerCustomerID = "probler"
	l8erpCustomerID   = "l8erp"
)

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

// ../probler's own images (its go/build-all-images.sh's real docker tags,
// one per go/prob/<service>/build.sh -- verified against those build
// scripts directly, not assumed). NOT probler-security/probler-postgres
// (same secrets-embedding exclusion as this project's own images above).
var problerImageRefs = []string{
	"saichler/probler-admission:latest",
	"saichler/probler-alarms:latest",
	"saichler/probler-collector:latest",
	"saichler/probler-inv-box:latest",
	"saichler/probler-inv-gpu:latest",
	"saichler/probler-inv-k8s:latest",
	"saichler/probler-logagent:latest",
	"saichler/logs-vnet:latest",
	"saichler/probler-maint:latest",
	"saichler/probler-webui2:latest",
	"saichler/probler-orm:latest",
	"saichler/probler-parser:latest",
	"saichler/probler-ptctl:latest",
	"saichler/probler-topo:latest",
	"saichler/probler-vnet:latest",
	"saichler/layer8-webui:latest",
}

// ../l8erp's own images (its go/build-all-images.sh's real docker tags).
// NOT any erp-security/erp-postgres equivalent (same exclusion).
var l8erpImageRefs = []string{
	"saichler/erp-vnet:latest",
	"saichler/erp-logs-vnet:latest",
	"saichler/erp-web:latest",
	"saichler/erp-maint:latest",
	"saichler/erp:latest",
	"saichler/erp-log-agent:latest",
}

var seedCategories = []map[string]interface{}{
	{"customerId": localCustomerID, "name": "Application Services", "colorCode": "#2563eb"},
	{"customerId": localCustomerID, "name": "Base Images", "colorCode": "#64748b"},
}

// Password must satisfy l8secure's default password policy
// (PasswordPolicy.go: min 10 chars, upper+digit+special required, no 3+
// repeated/sequential characters -- "123"/"abc"/"aaa" all rejected) -- a
// real 400 caught only by actually running this against a live server.
// localUserPassword keeps its original name/value -- e2e/fixtures/env.ts
// hardcodes this exact string as CUSTOMER_PASS independently, so it must
// never change.
const localUserPassword = "Vx9!TangoQm"
const problerUserPassword = "Pz4!ProblerX"
const l8erpUserPassword = "Ez6!L8ErpRun"

// One entry per customer this seeds: the Customer record, its customer-role
// security user, and the real images registered under it.
type customerSeed struct {
	customerID   string
	customerName string
	userID       string
	userPassword string
	imageRefs    []string
}

var customerSeeds = []customerSeed{
	{customerID: localCustomerID, customerName: "Local", userID: "local-user", userPassword: localUserPassword, imageRefs: projectImageRefs},
	{customerID: problerCustomerID, customerName: "Probler", userID: "probler-user", userPassword: problerUserPassword, imageRefs: problerImageRefs},
	{customerID: l8erpCustomerID, customerName: "L8ERP", userID: "l8erp-user", userPassword: l8erpUserPassword, imageRefs: l8erpImageRefs},
}

func seedCustomer(client *Client, s customerSeed) error {
	_, err := client.Post("/60/Customer", map[string]interface{}{
		"customerId": s.customerID,
		"name":       s.customerName,
		"isActive":   true,
	})
	if alreadyExists(err) {
		fmt.Printf("  Customer %q already exists, skipping\n", s.customerID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("seed customer %q: %w", s.customerID, err)
	}
	fmt.Printf("  Customer %q created\n", s.customerID)
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

// Security user for a customer (PRD §15 item 8) -- provisioned via the
// Security API (/73/users), never a project-owned endpoint (SecurityRules).
// customer=s.customerID is what makes SecScan.getCurrentCustomerId()
// resolve correctly for this user after login (the
// sessionStorage.userCustomer mechanism wired in Phase 4/the
// l8secure/l8types/l8ui plumbing).
func seedSecurityUser(client *Client, s customerSeed) error {
	_, err := client.Post("/73/users", map[string]interface{}{
		"userId":       s.userID,
		"fullName":     s.customerName + " Customer User",
		"password":     map[string]interface{}{"hash": s.userPassword},
		"roles":        map[string]interface{}{"customer": true},
		"associateIds": []string{s.customerID},
		"customer":     s.customerID,
	})
	if alreadyExists(err) {
		fmt.Printf("  Security user %q already exists, skipping\n", s.userID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("seed security user %q: %w", s.userID, err)
	}
	fmt.Printf("  Security user %q created (password: %s, customer: %s)\n", s.userID, s.userPassword, s.customerID)
	return nil
}

// Registers a customer's real images via the real ImgRefAdd bulk ingestion
// action-service (§6.1) -- the exact same path a real user's "Add Images"
// UI action takes, not a direct ImageGroup/ImageRef POST.
func seedImageRefs(client *Client, s customerSeed) error {
	resp, err := client.Post("/60/ImgRefAdd", map[string]interface{}{
		"customerId":      s.customerID,
		"imageRefStrings": s.imageRefs,
	})
	if err != nil {
		return fmt.Errorf("seed image refs for %q: %w", s.customerID, err)
	}
	fmt.Printf("  ImgRefAdd response: %s\n", resp)
	return nil
}

// RunSeed runs every seeding step in dependency order. Verified against a
// real cluster: opsadmin's role (secscan.json) grants ONLY Customer
// access (PRD §9's "opsadmin only manages Customer records") -- Categories
// and ImgRefAdd are "customer"-role-only actions and return "access
// denied" for opsadmin, a real 400 caught only by actually running this
// against a live server. So after creating each Customer and its
// customer-role security user (both opsadmin-only actions), the client
// re-authenticates AS that new user for everything customer-scoped.
func RunSeed(client *Client) error {
	for _, s := range customerSeeds {
		fmt.Printf("Seeding Customer %q...\n", s.customerID)
		if err := seedCustomer(client, s); err != nil {
			return err
		}

		fmt.Printf("Seeding Security User for %q...\n", s.customerID)
		if err := seedSecurityUser(client, s); err != nil {
			return err
		}

		fmt.Printf("Re-authenticating as %q (customer role)...\n", s.userID)
		if err := client.Authenticate(s.userID, s.userPassword); err != nil {
			return fmt.Errorf("re-authenticate as %q: %w", s.userID, err)
		}

		// Categories are only defined for the "local" customer today --
		// seedCategories already carries customerId per entry, so this
		// just no-ops (empty loop body) for probler/l8erp.
		if s.customerID == localCustomerID {
			fmt.Println("Seeding Categories...")
			if err := seedCategoriesData(client); err != nil {
				return err
			}
		}

		fmt.Printf("Seeding Image Refs for %q (via ImgRefAdd)...\n", s.customerID)
		if err := seedImageRefs(client, s); err != nil {
			return err
		}

		// Re-authenticate as opsadmin fresh, not by restoring the token
		// saved before this loop started -- verified against a real
		// server: the server invalidates an EARLIER token the instant any
		// OTHER user authenticates, even though that token hasn't expired
		// (a fresh opsadmin login works immediately after a customer-user
		// login; restoring the old opsadmin token from before that login
		// gets "access denied" every time). Both of RunSeed's callers
		// (mocks/main.go, TestAllService_test.go) already authenticate as
		// this exact opsadmin/opsadmin account before calling RunSeed.
		if err := client.Authenticate("opsadmin", "opsadmin"); err != nil {
			return fmt.Errorf("re-authenticate as opsadmin: %w", err)
		}
	}

	return nil
}
