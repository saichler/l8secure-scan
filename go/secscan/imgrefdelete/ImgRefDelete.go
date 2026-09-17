package imgrefdelete

import (
	scommon "github.com/saichler/l8secure-scan/go/secscan/common"
	"github.com/saichler/l8secure-scan/go/types/secscan"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = scommon.ImgRefDeleteServiceName
	ServiceArea = scommon.ServiceArea
)

// ImgRefDelete is a POST-only action service (mirrors imgrefadd's own
// pattern) -- not a plain-CRUD entity service, no ORM table of its own.
// Deletes one ImageRef and recomputes its parent ImageGroup's rollup
// cache in the same call, which a plain ORM DELETE on ImageRef can't do
// itself (ImageRefServiceCallback.After() only fires on PUT/PATCH, never
// DELETE -- verified against l8common's genericCallback source).
type ImgRefDelete struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	vnic.Resources().Registry().Register(&secscan.ImgRefDeleteRequest{})
	vnic.Resources().Registry().Register(&secscan.ImgRefDeleteResponse{})

	handler := &ImgRefDelete{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&secscan.ImgRefDeleteRequest{}, ifs.POST, &secscan.ImgRefDeleteResponse{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *ImgRefDelete) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *ImgRefDelete) DeActivate() error {
	return nil
}

func (this *ImgRefDelete) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *ImgRefDelete) WebService() ifs.IWebService {
	return this.sla.WebService()
}
