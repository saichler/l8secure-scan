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
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

func (this *FileStore) Patch(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *FileStore) Delete(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *FileStore) Get(ifs.IElements, ifs.IVNic) ifs.IElements {
	return object.NewError("not supported")
}

func (this *FileStore) GetCopy(ifs.IElements, ifs.IVNic) ifs.IElements {
	return nil
}

func (this *FileStore) Failed(ifs.IElements, ifs.IVNic, *ifs.Message) ifs.IElements {
	return nil
}

func (this *FileStore) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *FileStore) WebService() ifs.IWebService {
	return this.sla.WebService()
}
