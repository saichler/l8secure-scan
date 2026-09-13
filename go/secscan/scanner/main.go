package main

import (
	"os"

	"github.com/saichler/l8bus/go/overlay/vnic"
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/resolver"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/scanloop"
)

// secscan-scanner is a stateless worker (PRD §13) -- it owns no ORM and
// serves no HTTP; it only calls the secscan backend remotely over vnic
// (SingleOwnerDatabaseTable). Single replica for v1 (PRD §13's
// concurrency note). No login/credential step: verified that raw vnic
// calls aren't AAA-checked in this framework (plans/PROGRESS.md) -- the
// same CreateResources/vnic pattern every other binary in this project
// uses is all that's needed here too.
func main() {
	resources := l8common.CreateResources("secscan-scanner-"+os.Getenv("HOSTNAME"), false)

	nic := vnic.NewVirtualNetworkInterface(resources, nil)
	nic.Start()
	nic.WaitForConnection()

	stop := make(chan struct{})
	go resolver.Run(nic, stop)
	go scanloop.Run(nic, stop)

	resources.Logger().Info("secscan-scanner started!")
	l8common.WaitForSignal(resources)
	close(stop)
}
