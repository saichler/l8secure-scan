package main

import (
	"os"

	"github.com/saichler/l8bus/go/overlay/vnic"
	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/scanjob"
	"github.com/saichler/l8secure-scan/go/secscan/scanner/resolver"
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
	// Types must be registered locally even though this process owns no
	// ORM table -- verified against a real cluster: l8ql's query
	// interpreter needs the type registered on THIS process's own
	// introspector/registry to build/parse a query at all (GetEntitiesByQuery
	// against ScanJob/ImageRef failed with "Cannot find node for table X"
	// without this), not just on the backend that actually persists it.
	scommon.RegisterSecscanTypes(resources)

	nic := vnic.NewVirtualNetworkInterface(resources, nil)
	nic.Start()
	nic.WaitForConnection()

	stop := make(chan struct{})
	go resolver.Run(nic, stop)

	// Stateless ScanJob action service (plans/scanjob-live-progress.md) --
	// no poll/claim loop anymore; scanning is invoked directly from its
	// Post handler.
	scanjob.Activate(nic)

	resources.Logger().Info("secscan-scanner started!")
	l8common.WaitForSignal(resources)
	close(stop)
}
