// Package scanjob is the stateless action service (plans/scanjob-live-progress.md
// Phase 2) that the Dashboard/mobile "Scan Selected" POSTs to directly. No ORM,
// no cache -- modeled on secscan/imgrefadd (ifs.IServiceHandler, registered via
// ifs.NewServiceLevelAgreement + web.New/AddEndpoint, not l8common.ActivateService).
// It reuses the existing secscan.ScanJob message as both request and response
// shape -- no new proto message needed, since the persistence service
// (secscan/scanjobs) already registers ScanJob/ScanJobList generically
// (scommon.RegisterSecscanTypes).
package scanjob

import (
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = scommon.ScanJobServiceName
	ServiceArea = scommon.ServiceArea
)

// ScanJobHandler -- NOT named ScanJob: the framework's Registry.NewOf
// (l8utils/go/utils/registry/Registry.go) looks up/creates handler
// instances by bare struct name via reflection, with no package
// qualification. RegisterSecscanTypes already registers the protobuf
// secscan.ScanJob message under that same bare name "ScanJob" before this
// service activates -- naming this struct ScanJob too caused
// Registry.NewOf to return a *secscan.ScanJob instead of this handler,
// panicking on the resulting failed ifs.IServiceHandler type assertion
// (found live, deploying to KIND -- plans/scanjob-live-progress.md Phase 6).
type ScanJobHandler struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	handler := &ScanJobHandler{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&secscan.ScanJob{}, ifs.POST, &secscan.ScanJob{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *ScanJobHandler) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *ScanJobHandler) DeActivate() error {
	return nil
}

func (this *ScanJobHandler) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ScanJobHandler) WebService() ifs.IWebService {
	return this.sla.WebService()
}
