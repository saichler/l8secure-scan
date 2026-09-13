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

package vnic

import (
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8notify"
)

// NotifyServiceAdded sends a health update notification when new services are registered on this VNic.
func (this *VirtualNetworkInterface) NotifyServiceAdded(serviceNames []string, serviceArea byte) error {
	curr := health.HealthOf(this.resources.SysConfig().LocalUuid, this.resources)

	hp := &l8health.L8Health{}
	hp.AUuid = curr.AUuid
	hp.Services = curr.Services
	//mergeServices(hp, this.resources.SysConfig().Services)
	//send notification for health service
	err := this.Unicast(this.resources.SysConfig().RemoteUuid, health.ServiceName, 0, ifs.PATCH, hp)

	return err
}

// NotifyServiceRemoved sends a health update notification when a service is unregistered from this VNic.
func (this *VirtualNetworkInterface) NotifyServiceRemoved(serviceName string, serviceArea byte) error {
	curr := health.HealthOf(this.resources.SysConfig().LocalUuid, this.resources)
	hp := &l8health.L8Health{}
	hp.AUuid = curr.AUuid
	hp.Services = curr.Services
	//mergeServices(hp, this.resources.SysConfig().Services)
	ifs.RemoveService(hp.Services, serviceName, int32(serviceArea))
	return this.Unicast(this.resources.SysConfig().RemoteUuid, health.ServiceName, serviceArea, ifs.PATCH, hp)
}

// PropertyChangeNotification broadcasts property change notifications to service subscribers.
func (this *VirtualNetworkInterface) PropertyChangeNotification(set *l8notify.L8NotificationSet) {
	protocol.MsgLog.AddLog(set.ServiceName, byte(set.ServiceArea), ifs.Notify)
	this.Multicast(set.ServiceName, byte(set.ServiceArea), ifs.Notify, set)
}

/*
func mergeServices(hp *l8health.L8Health, services *l8services.L8Services) {
	if hp.Services == nil {
		hp.Services = services
		return

	}
	for serviceName, serviceAreas := range services.ServiceToAreas {
		_, ok := hp.Services.ServiceToAreas[serviceName]
		if !ok {
			hp.Services.ServiceToAreas[serviceName] = serviceAreas
			continue
		}
		if hp.Services.ServiceToAreas[serviceName].Areas == nil {
			hp.Services.ServiceToAreas[serviceName].Areas = serviceAreas.Areas
			continue
		}
		for svArea, score := range serviceAreas.Areas {
			serviceArea := svArea
			hp.Services.ServiceToAreas[serviceName].Areas[serviceArea] = score
		}
	}
}*/
