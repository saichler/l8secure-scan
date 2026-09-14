package main

import (
	"fmt"
	"os"

	"github.com/saichler/l8bus/go/overlay/vnic"
	l8common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8logfusion/go/agent/logs"
	"github.com/saichler/l8logfusion/go/types/l8logf"
	"github.com/saichler/l8utils/go/utils/ipsegment"
)

// secscan-log-agent: per-node log collector, forwards to secscan-log-vnet
// (LogServicesRequired, PRD §16). Mirrors ../l8erp/go/logs/agent/main.go's
// real pattern exactly. LOGPATH default matches LogServicesRequired's
// "log-agent LOGPATH = /data/logs/<project>".
func main() {
	logsDirectory := "/data/logs/secscan"
	ip := os.Getenv("NODE_IP")
	if ip == "" {
		fmt.Println("Env variable NODE_IP is not set, using machine ip")
		ip = ipsegment.MachineIP
	}

	logpath := os.Getenv("LOGPATH")
	if logpath == "" {
		fmt.Println("Env variable LOGPATH is not set, using " + logsDirectory)
		logpath = logsDirectory
	}

	logfile := os.Getenv("LOGFILE")
	if logfile == "" {
		fmt.Println("Env variable LOGFILE is not set, using *")
		logfile = "*"
	}

	r := l8common.CreateResources("secscan-log-agent", true)
	r.SysConfig().RemoteVnet = ip

	nic := vnic.NewVirtualNetworkInterface(r, nil)
	nic.Start()
	nic.WaitForConnection()

	lc := &l8logf.L8LogConfig{Path: logpath, Name: logfile}
	collector := logs.NewLogCollector(lc, nic)
	collector.Collect()
}
