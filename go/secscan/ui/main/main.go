package main

import (
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/vnic"
	l8common "github.com/saichler/l8common/go/common"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	"github.com/saichler/l8web/go/web/server"
)

// secscan-web: the UI server binary (PRD §16), serving go/secscan/ui/web/
// (built in Phases 4/5) and the REST API every desktop/mobile fetch call
// hits. Mirrors ../l8alarms/go/alm/ui/main/main.go's real, working pattern
// exactly -- the closest-scale real precedent for this exact binary shape.
func main() {
	resources := l8common.CreateResources("secscan-web", false)
	scommon.RegisterSecscanTypes(resources)

	nic := vnic.NewVirtualNetworkInterface(resources, nil)
	nic.Start()
	nic.WaitForConnection()

	domain, private, _ := nic.Resources().Certificate()

	serverConfig := &server.RestServerConfig{
		Host:           ipsegment.MachineIP,
		Port:           2790,
		Authentication: true,
		Prefix:         scommon.PREFIX,
		CertDomain:     domain,
		CertPrivate:    private,
	}
	svr, err := server.NewRestServer(serverConfig)
	if err != nil {
		panic(err)
	}

	hs, ok := nic.Resources().Services().ServiceHandler(health.ServiceName, 0)
	if ok {
		ws := hs.WebService()
		svr.RegisterWebService(ws, nic)
	}

	sla := ifs.NewServiceLevelAgreement(&server.WebService{}, ifs.WebService, 0, false, nil)
	sla.SetArgs(svr, nic)
	nic.Resources().Services().Activate(sla, nic)

	nic.Resources().Logger().Info("Web Server Started!")

	svr.Start()
}
