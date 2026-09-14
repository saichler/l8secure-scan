package tests

import (
	"github.com/saichler/l8bus/go/overlay/health"
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8web/go/web/server"
)

// startWebServer mirrors go/secscan/ui/main/main.go's real, working setup
// (Phase 7) on an already-existing test vnic instead of creating a new
// one -- the same pattern ../l8alarms/go/tests/StartWebserver.go uses for
// its own project. Types must already be registered on nic.Resources()
// before this is called (see TestAllServices).
func startWebServer(port int, nic ifs.IVNic) ifs.IWebServer {
	domain, private, _ := nic.Resources().Certificate()

	serverConfig := &server.RestServerConfig{
		Host:           "localhost",
		Port:           port,
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

	nic.Resources().Logger().Info("SecScan Test Web Server Started!")

	go svr.Start()

	return svr
}
