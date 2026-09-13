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

package filestore

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/web"
)

const (
	ServiceName = "FileStore"
	ServiceArea = byte(0)
	MaxFileSize = 5 * 1024 * 1024 // 5MB
)

var StorageRoot = "/data/l8files"

type FileStore struct {
	sla *ifs.ServiceLevelAgreement
}

func SetStorageRoot(path string) {
	StorageRoot = path
}

func Activate(vnic ifs.IVNic) {
	vnic.Resources().Registry().Register(&l8api.L8FileUploadRequest{})
	vnic.Resources().Registry().Register(&l8api.L8FileUploadResponse{})
	vnic.Resources().Registry().Register(&l8api.L8FileDownloadRequest{})
	vnic.Resources().Registry().Register(&l8api.L8FileDownloadResponse{})

	handler := &FileStore{}
	sla := ifs.NewServiceLevelAgreement(handler, ServiceName, ServiceArea, false, nil)

	ws := web.New(ServiceName, ServiceArea, 0)
	ws.AddEndpoint(&l8api.L8FileUploadRequest{}, ifs.POST, &l8api.L8FileUploadResponse{})
	ws.AddEndpoint(&l8api.L8FileDownloadRequest{}, ifs.PUT, &l8api.L8FileDownloadResponse{})
	sla.SetWebService(ws)

	vnic.Resources().Services().Activate(sla, vnic)
}

func (this *FileStore) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	this.sla = sla
	return nil
}

func (this *FileStore) DeActivate() error {
	return nil
}
