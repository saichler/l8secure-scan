package imgrefadd

import (
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = scommon.ImgRefAddServiceName
	ServiceArea = scommon.ServiceArea
)

// ImgRefAdd is a POST-only action service (PRD §6.1) -- not a plain-CRUD
// entity service, no ORM table of its own. Modeled directly on
// l8services/go/services/csvexport/CsvExport.go: implements
// ifs.IServiceHandler, registered via ifs.NewServiceLevelAgreement +
// web.New/AddEndpoint, not common.ActivateService.
type ImgRefAdd struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	vnic.Resources().Registry().Register(&secscan.ImgRefAddRequest{})
	vnic.Resources().Registry().Register(&secscan.ImgRefAddResponse{})

	handler := &ImgRefAdd{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&secscan.ImgRefAddRequest{}, ifs.POST, &secscan.ImgRefAddResponse{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *ImgRefAdd) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *ImgRefAdd) DeActivate() error {
	return nil
}

func (this *ImgRefAdd) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ImgRefAdd) WebService() ifs.IWebService {
	return this.sla.WebService()
}
