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

type ScanJob struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	handler := &ScanJob{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&secscan.ScanJob{}, ifs.POST, &secscan.ScanJob{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *ScanJob) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *ScanJob) DeActivate() error {
	return nil
}

func (this *ScanJob) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ScanJob) WebService() ifs.IWebService {
	return this.sla.WebService()
}
