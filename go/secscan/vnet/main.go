package main

import (
	"os"

	"github.com/saichler/l8bus/go/overlay/vnet"
	l8common "github.com/saichler/l8common/go/common"
)

// secscan-vnet: the virtual network backbone every other secscan binary's
// vnic connects through (PRD §16). Mirrors ../l8alarms/go/alm/vnet/main.go's
// real pattern exactly.
func main() {
	resources := l8common.CreateResources("secscan-vnet-"+os.Getenv("HOSTNAME"), false)
	net := vnet.NewVNet(resources)
	net.Start()
	resources.Logger().Info("secscan vnet started!")
	l8common.WaitForSignal(resources)
}
