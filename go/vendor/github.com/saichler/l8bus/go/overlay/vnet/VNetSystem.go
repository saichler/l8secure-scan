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

package vnet

import (

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8system"
)

// systemMessageReceived handles incoming system control messages for route and service management.
// It processes route additions/removals and service registrations from the network.
func (this *VNet) systemMessageReceived(data []byte, vnic ifs.IVNic) {
	msg, err := this.protocol.MessageOf(data)
	if err != nil {
		this.resources.Logger().Error(err)
		return
	}

	pb, err := this.protocol.ElementsOf(msg)
	if err != nil {
		if msg.Tr_State() != ifs.NotATransaction {
			//This message should not be processed and we should just
			//reply with nil to unblock the transaction
			vnic.Reply(msg, nil)
			return
		}
		this.resources.Logger().Error(err)
		return
	}

	systemMessage := pb.Element().(*l8system.L8SystemMessage)

	switch systemMessage.Action {
	case l8system.L8SystemAction_Routes_Add:
		added := this.switchTable.routeTable.addRoutes(systemMessage.GetRouteTable().Rows)
		this.routesAdded(added)
		return
	case l8system.L8SystemAction_Routes_Remove:
		removed := this.switchTable.routeTable.removeRoutes(systemMessage.GetRouteTable().Rows)
		this.routesRemoved(removed)
		return
	case l8system.L8SystemAction_Service_Add:
		serviceData := systemMessage.GetServiceData()
		this.switchTable.services.addService(serviceData)
		if systemMessage.Publish {
			this.publishSystemMessage(systemMessage)
			//go health.AddServiceToHealth(msg.Source(), serviceData.ServiceName, serviceData.ServiceArea, this.resources)
		}
		return
	default:
		panic("unknown system action")
	}
}

// routesAdded publishes new routes to external VNets when routes are added to the local table.
func (this *VNet) routesAdded(added map[string]string) {
	if len(added) > 0 {
		this.publishRoutes()
	}
}

// routesRemoved handles cleanup when routes are removed, including service deregistration and health removal.
func (this *VNet) routesRemoved(removed map[string]string) {
	if len(removed) > 0 {
		this.switchTable.services.removeService(removed)
		this.publishRemovedRoutes(removed)
		this.removeHealth(removed)
	}
}

// removeHealth deletes health records for VNics that have been disconnected.
func (this *VNet) removeHealth(removed map[string]string) {
	hs, _ := health.HealthService(this.resources)
	for uuid, _ := range removed {
		hp := health.HealthOf(uuid, this.resources)
		if hp != nil {
			hs.Delete(object.New(nil, hp), this.vnic)
		}
	}
}
