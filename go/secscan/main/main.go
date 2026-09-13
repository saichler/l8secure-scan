package main

import (
	"os"

	"github.com/saichler/l8bus/go/overlay/vnic"
	l8common "github.com/saichler/l8common/go/common"
	evtservices "github.com/saichler/l8events/go/services"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/secscan/services"
)

func main() {
	resources := l8common.CreateResources("secscan-"+os.Getenv("HOSTNAME"), false)
	scommon.RegisterSecscanTypes(resources)

	nic := vnic.NewVirtualNetworkInterface(resources, nil)
	nic.Start()
	nic.WaitForConnection()

	services.ActivateSecscanServices(scommon.DB_CREDS, scommon.DB_NAME, nic)
	evtservices.ActivateEvents(scommon.DB_CREDS, scommon.DB_NAME, nic)
	resources.Logger().Info("secscan services activated!")
	l8common.WaitForSignal(resources)
}
