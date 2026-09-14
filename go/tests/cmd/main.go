package main

import (
	"flag"

	"github.com/saichler/l8secure-scan/go/tests/mocks"
)

func main() {
	address := flag.String("address", "https://localhost:2790", "Server address")
	user := flag.String("user", "opsadmin", "Username")
	password := flag.String("password", "opsadmin", "Password")
	insecure := flag.Bool("insecure", true, "Skip TLS certificate verification")
	flag.Parse()

	mocks.RunMockGenerator(*address, *user, *password, *insecure)
}
