// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"errors"
	"github.com/saichler/l8types/go/ifs"
)

// DeActivate removes a service from the registry and shuts it down.
// It notifies the network of the service removal if the listener is a VNIC.
func (this *ServiceManager) DeActivate(serviceName string, serviceArea byte, r ifs.IResources, l ifs.IServiceCacheListener) error {

	if serviceName == "" {
		return errors.New("Service name is empty")
	}

	handler, ok := this.services.del(serviceName, serviceArea)
	if !ok {
		return errors.New("Can't find service " + serviceName)
	}

	defer handler.DeActivate()

	ifs.RemoveService(this.resources.SysConfig().Services, serviceName, int32(serviceArea))
	vnic, ok := l.(ifs.IVNic)
	if ok {
		vnic.NotifyServiceRemoved(serviceName, serviceArea)
	}
	return nil
}
