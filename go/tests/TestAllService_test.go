package tests

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	_ "github.com/lib/pq"
	evtservices "github.com/saichler/l8events/go/services"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/services"
	"github.com/saichler/l8secure-scan/go/tests/mocks"
	"github.com/saichler/l8types/go/ifs"
)

func TestMain(m *testing.M) {
	setup()
	m.Run()
	tear()
}

func openDBConnection(dbname, user, pass, port string) *sql.DB {
	if port == "" {
		port = "5432"
	}
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"127.0.0.1", port, user, pass, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic(fmt.Errorf("failed to connect to database: %w", err))
	}
	return db
}

func dropAllTables(t *testing.T, vnic ifs.IVNic) {
	_, user, pass, port, err := vnic.Resources().Security().Credential(scommon.DB_CREDS, scommon.DB_NAME, vnic.Resources())
	if err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}
	db := openDBConnection(scommon.DB_NAME, user, pass, port)
	_, err = db.Exec("DROP SCHEMA public CASCADE")
	if err != nil {
		t.Fatalf("Failed to drop schema: %v", err)
	}
	_, err = db.Exec("CREATE SCHEMA public")
	if err != nil {
		t.Fatalf("Failed to recreate schema: %v", err)
	}
	fmt.Println("Cleaned database (dropped and recreated public schema)")
}

// TestAllServices is the single go-test-visible entry point (mirrors
// ../l8alarms/go/tests/TestAllService_test.go's real pattern exactly) --
// every other *_test.go file in this package exports a lowercase
// testXxx(t, ...) helper called from here, sharing the one expensive
// topology/DB/web-server/plugin setup rather than repeating it per test.
func TestAllServices(t *testing.T) {
	servicesVnic := topo.VnicByVnetNum(1, 1)
	webVnic := topo.VnicByVnetNum(3, 3)
	log := webVnic.Resources().Logger()

	// 0. Clean slate
	dropAllTables(t, servicesVnic)

	// 1. Register types (incl. primary-key decorators) on servicesVnic's
	// own resources BEFORE activating services -- same order as
	// go/secscan/main/main.go, same resources throughout (see
	// StartWebserver.go's doc comment for why order/vnic matter here).
	scommon.RegisterSecscanTypes(servicesVnic.Resources())

	// 2. Activate all secscan services + l8events (EventsServiceRequired)
	services.ActivateSecscanServices(scommon.DB_CREDS, scommon.DB_NAME, servicesVnic)
	evtservices.ActivateEvents(scommon.DB_CREDS, scommon.DB_NAME, servicesVnic)

	// 3. Start the web server on a separate vnic (non-blocking)
	port := 9443
	scommon.RegisterSecscanTypes(webVnic.Resources())
	startWebServer(port, webVnic)
	time.Sleep(10 * time.Second)

	// 4. HTTP client pointing at the web server, authenticated as opsadmin
	// (the only role allowed to create Customer rows, PRD §9) -- reuses
	// Phase 6's own mocks.Client verbatim.
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client := mocks.NewClient(fmt.Sprintf("https://localhost:%d", port), httpClient)
	if err := client.Authenticate("opsadmin", "opsadmin"); err != nil {
		log.Fail(t, "Authentication failed: ", err.Error())
		return
	}

	// 5. Seed the "local" customer + categories + security user + this
	// project's own real images (Phase 6, unchanged, reused verbatim).
	if err := mocks.RunSeed(client); err != nil {
		log.Fail(t, "Mock data seeding failed: ", err.Error())
		return
	}

	// 6. Test areas (PRD §19), each seeding whatever additional fixtures
	// it specifically needs on top of the base seed above.
	testIngestion(t, client, servicesVnic)
	testResolver(t, servicesVnic)
	testScanPipeline(t, servicesVnic)
	testReporting(t, client, servicesVnic)
	testSecurity(t, client, servicesVnic)
}
