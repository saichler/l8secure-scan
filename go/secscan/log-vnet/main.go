package main

import (
	"github.com/saichler/l8bus/go/overlay/vnet"
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8logfusion/go/agent/logserver"
)

// secscan-log-vnet: the distributed logging backbone (LogServicesRequired,
// PRD §16). Mirrors ../probler/go/prob/log-vnet/main.go's real pattern --
// listens on sysconfig's logConfig.vnetPort (33010 in secscan.json),
// distinct from the main vnetPort every other binary's vnic connects to.
func main() {
	logsDbDirectory := "/data/logsdb/secscan"
	resources := l8common.CreateResources("secscan-log-vnet", true)
	resources.SysConfig().VnetPort = resources.SysConfig().LogConfig.VnetPort
	net := vnet.NewVNet(resources, true)
	net.Start()
	logserver.ActivateLogService(logsDbDirectory, net.VnetVnic())
	resources.Logger().Info("secscan logs vnet started!")
	l8common.WaitForSignal(resources)
}
