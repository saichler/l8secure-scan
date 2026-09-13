package vulnrep

import (
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = scommon.VulnRepServiceName
	ServiceArea = scommon.ServiceArea
)

// VulnRep is a POST-only action service (PRD §10) generating the
// cross-group CSV vulnerability report. Modeled directly on
// l8services/go/services/csvexport/CsvExport.go: implements
// ifs.IServiceHandler, registered via ifs.NewServiceLevelAgreement +
// web.New/AddEndpoint. The response reuses l8api.L8CsvExportResponse
// (csv_data/filename/row_count) -- the only real, working precedent in
// this framework for a generated-CSV response; there is no raw text/csv
// HTTP body anywhere in the ecosystem (verified).
type VulnRep struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	vnic.Resources().Registry().Register(&secscan.VulnRepRequest{})
	vnic.Resources().Registry().Register(&l8api.L8CsvExportResponse{})

	handler := &VulnRep{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&secscan.VulnRepRequest{}, ifs.POST, &l8api.L8CsvExportResponse{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *VulnRep) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *VulnRep) DeActivate() error {
	return nil
}

func (this *VulnRep) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *VulnRep) WebService() ifs.IWebService {
	return this.sla.WebService()
}
