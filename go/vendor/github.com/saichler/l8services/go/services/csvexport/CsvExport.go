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

package csvexport

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "CsvExport"
	ServiceArea = byte(0)
)

type CsvExport struct {
	sla *ifs.ServiceLevelAgreement
}

func Activate(vnic ifs.IVNic) {
	vnic.Resources().Registry().Register(&l8api.L8CsvExportRequest{})
	vnic.Resources().Registry().Register(&l8api.L8CsvExportResponse{})

	handler := &CsvExport{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&l8api.L8CsvExportRequest{}, ifs.POST, &l8api.L8CsvExportResponse{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *CsvExport) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *CsvExport) DeActivate() error {
	return nil
}
