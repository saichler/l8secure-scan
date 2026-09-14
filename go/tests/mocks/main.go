package mocks

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"
)

// RunMockGenerator seeds a "local" customer, a couple of categories, one
// customer-role security user, and this project's own real container
// images (registered via the real ImgRefAdd bulk-ingestion path) against a
// running secscan-web server. Authenticates as opsadmin (the only role
// allowed to create Customer rows, PRD §9).
func RunMockGenerator(address, user, password string, insecure bool) {
	fmt.Printf("SecScan Mock Data Seeder\n")
	fmt.Printf("========================\n")
	fmt.Printf("Server: %s\n", address)
	fmt.Printf("User: %s\n", user)
	if insecure {
		fmt.Printf("TLS: Insecure (certificate verification disabled)\n")
	}
	fmt.Printf("\n")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	if insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client := NewClient(address, httpClient)

	if err := client.Authenticate(user, password); err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Authentication successful\n\n")

	if err := RunSeed(client); err != nil {
		fmt.Printf("Seeding failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nDone.\n")
}
