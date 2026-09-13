/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dataimport

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceArea = byte(0)
)

// baseHandler provides shared IServiceHandler stubs for custom handlers.
type baseHandler struct {
	sla *ifs.ServiceLevelAgreement
}

func (h *baseHandler) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	h.sla = sla
	return nil
}
func (h *baseHandler) DeActivate() error                                        { return nil }
func (h *baseHandler) Post(ifs.IElements, ifs.IVNic) ifs.IElements              { return object.NewError("not supported") }
func (h *baseHandler) Put(ifs.IElements, ifs.IVNic) ifs.IElements               { return object.NewError("not supported") }
func (h *baseHandler) Patch(ifs.IElements, ifs.IVNic) ifs.IElements             { return object.NewError("not supported") }
func (h *baseHandler) Delete(ifs.IElements, ifs.IVNic) ifs.IElements            { return object.NewError("not supported") }
func (h *baseHandler) Get(ifs.IElements, ifs.IVNic) ifs.IElements               { return object.NewError("not supported") }
func (h *baseHandler) GetCopy(ifs.IElements, ifs.IVNic) ifs.IElements           { return nil }
func (h *baseHandler) Failed(ifs.IElements, ifs.IVNic, *ifs.Message) ifs.IElements { return nil }
func (h *baseHandler) TransactionConfig() ifs.ITransactionConfig                { return nil }
func (h *baseHandler) WebService() ifs.IWebService                              { return h.sla.WebService() }

// Activate registers and activates all data import custom handlers.
// The template CRUD service (ImprtTmpl) is activated separately by the
// application layer since it requires ORM/database access.
func Activate(vnic ifs.IVNic) {
	// Register all types
	vnic.Resources().Registry().Register(&l8api.L8ImportTemplate{})
	vnic.Resources().Registry().Register(&l8api.L8ImportTemplateList{})
	vnic.Resources().Registry().Register(&l8api.L8ImportAIMappingRequest{})
	vnic.Resources().Registry().Register(&l8api.L8ImportAIMappingResponse{})
	vnic.Resources().Registry().Register(&l8api.L8ImportExecuteRequest{})
	vnic.Resources().Registry().Register(&l8api.L8ImportExecuteResponse{})
	vnic.Resources().Registry().Register(&l8api.L8ImportModelInfoRequest{})
	vnic.Resources().Registry().Register(&l8api.L8ImportModelInfoResponse{})
	vnic.Resources().Registry().Register(&l8api.L8ImportTemplateExportRequest{})
	vnic.Resources().Registry().Register(&l8api.L8ImportTemplateExportResponse{})
	vnic.Resources().Registry().Register(&l8api.L8ImportTemplateImportRequest{})
	vnic.Resources().Registry().Register(&l8api.L8ImportTemplateImportResponse{})

	activateAI(vnic)
	activateExecute(vnic)
	activateModelInfo(vnic)
	activateTransfer(vnic)
}

func activateAI(vnic ifs.IVNic) {
	handler := &AIHandler{}
	sla := ifs.NewServiceLevelAgreement(handler, "ImprtAI", ServiceArea, false, nil)
	ws := web.New("ImprtAI", ServiceArea, 0)
	ws.AddEndpoint(&l8api.L8ImportAIMappingRequest{}, ifs.POST, &l8api.L8ImportAIMappingResponse{})
	sla.SetWebService(ws)
	vnic.Resources().Services().Activate(sla, vnic)
}

func activateExecute(vnic ifs.IVNic) {
	handler := &ExecuteHandler{}
	sla := ifs.NewServiceLevelAgreement(handler, "ImprtExec", ServiceArea, false, nil)
	ws := web.New("ImprtExec", ServiceArea, 0)
	ws.AddEndpoint(&l8api.L8ImportExecuteRequest{}, ifs.POST, &l8api.L8ImportExecuteResponse{})
	sla.SetWebService(ws)
	vnic.Resources().Services().Activate(sla, vnic)
}

func activateModelInfo(vnic ifs.IVNic) {
	handler := &ModelInfoHandler{}
	sla := ifs.NewServiceLevelAgreement(handler, "ImprtInfo", ServiceArea, false, nil)
	ws := web.New("ImprtInfo", ServiceArea, 0)
	ws.AddEndpoint(&l8api.L8ImportModelInfoRequest{}, ifs.POST, &l8api.L8ImportModelInfoResponse{})
	sla.SetWebService(ws)
	vnic.Resources().Services().Activate(sla, vnic)
}

func activateTransfer(vnic ifs.IVNic) {
	handler := &TransferHandler{}
	sla := ifs.NewServiceLevelAgreement(handler, "ImprtXfer", ServiceArea, false, nil)
	ws := web.New("ImprtXfer", ServiceArea, 0)
	ws.AddEndpoint(&l8api.L8ImportTemplateExportRequest{}, ifs.POST, &l8api.L8ImportTemplateExportResponse{})
	ws.AddEndpoint(&l8api.L8ImportTemplateImportRequest{}, ifs.PUT, &l8api.L8ImportTemplateImportResponse{})
	sla.SetWebService(ws)
	vnic.Resources().Services().Activate(sla, vnic)
}

// Handler types — each embeds baseHandler for stubs, overrides relevant methods.
type AIHandler struct{ baseHandler }
type ExecuteHandler struct{ baseHandler }
type ModelInfoHandler struct{ baseHandler }
type TransferHandler struct{ baseHandler }
