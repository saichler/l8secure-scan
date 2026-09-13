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

package plugins

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
)

// ServiceName is the identifier used to register and lookup the plugin service.
const (
	ServiceName     = "Plugin"
	ServiceTypeName = "PluginService"
)

// PluginService handles plugin distribution and loading across the Layer8 network.
// It receives plugin data via POST requests and loads them into the local VNic.
type PluginService struct {
}

// Activate registers the plugin type with the registry when the service starts.
func (this *PluginService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8web.L8Plugin{})
	return nil
}

// DeActivate is called when the service is stopped.
func (this *PluginService) DeActivate() error {
	return nil
}

// Post handles incoming plugin data and loads it into the local VNic.
func (this *PluginService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	plugin := pb.Element().(*l8web.L8Plugin)
	err := LoadPlugin(plugin, vnic)
	if err != nil {
		vnic.Resources().Logger().Error(err.Error())
	}
	return object.New(err, nil)
}
func (this *PluginService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *PluginService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *PluginService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *PluginService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}
func (this *PluginService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return object.New(nil, nil)
}
func (this *PluginService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

func (this *PluginService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

func (this *PluginService) WebService() ifs.IWebService {
	return nil
}
