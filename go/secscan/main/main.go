package main

import (
	"os"

	"github.com/saichler/l8bus/go/overlay/vnic"
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8common/go/system"
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
	// Required system services (Events, Notify, IntegCfg) come from l8common,
	// never by activating them individually.
	system.Activate(scommon.DB_CREDS, scommon.DB_NAME, nic)
	resources.Logger().Info("secscan services activated!")
	l8common.WaitForSignal(resources)
}
