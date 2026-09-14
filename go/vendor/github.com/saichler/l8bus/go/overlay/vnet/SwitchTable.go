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
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	"github.com/saichler/l8utils/go/utils/strings"
)

// SwitchTable manages the routing and connection state for a VNet switch.
// It maintains mappings between node UUIDs, their connections, and the services they provide.
type SwitchTable struct {
	conns         *Connections
	services      *Services
	routeTable    *RouteTable
	switchService *VNet
	desc          string
}

func newSwitchTable(switchService *VNet) *SwitchTable {
	switchTable := &SwitchTable{}
	vnetUuid := switchService.resources.SysConfig().LocalUuid
	switchTable.routeTable = newRouteTable(vnetUuid)
	switchTable.conns = newConnections(vnetUuid, switchTable.routeTable, switchService.resources.Logger())
	switchTable.services = newServices(switchTable.routeTable)
	switchTable.switchService = switchService
	switchTable.desc = strings.New("SwitchTable (", switchService.resources.SysConfig().LocalUuid, ") - ").String()
	go switchTable.monitor()
	return switchTable
}

func (this *SwitchTable) addVNic(vnic ifs.IVNic) {
	config := vnic.Resources().SysConfig()
	//check if this port is local to the machine, e.g. not belong to public subnet
	isLocal := ipsegment.IpSegment.IsLocal(config.Address)
	isExternalVnic := config.RemoteVnet != ""
	if isExternalVnic {
		this.conns.addExternalVnic(config.RemoteUuid, vnic)
	} else
	// If it is local, add it to the internal map
	if isLocal && !config.ForceExternal {
		this.conns.addInternal(config.RemoteUuid, vnic)
		this.switchService.addVnetTask(QHealthReport, []byte(config.RemoteUuid), this.switchService.vnic)
	} else {
		// otherwise, add it to the external connections
		this.conns.addExternalVnet(config.RemoteUuid, vnic)
		//When this is an external vnet, we need to re-publish the services
		this.switchService.resources.Services().TriggerElections(this.switchService.vnic)
	}
	this.switchService.publishRoutes()
}

func (this *SwitchTable) connectionsForService(serviceName string, serviceArea byte, sourceSwitch string, mode ifs.MulticastMode) map[string]ifs.IVNic {
	isHope0 := this.switchService.resources.SysConfig().LocalUuid == sourceSwitch
	result := make(map[string]ifs.IVNic)
	switch mode {
	case ifs.M_All:
		uuidMap := this.services.serviceUuids(serviceName, serviceArea)
		for uuid, _ := range uuidMap {
			usedUuid, vnic := this.conns.getConnection(uuid, isHope0)
			if vnic != nil {
				result[usedUuid] = vnic
			}
		}
		return result
	default:
		uuid := this.services.serviceFor(serviceName, serviceArea, sourceSwitch, mode)
		if uuid != "" {
			usedUuid, vnic := this.conns.getConnection(uuid, true)
			if vnic != nil {
				result[usedUuid] = vnic
			} else {
				this.switchService.resources.Logger().Error("Cannot find vnic for uuid:", uuid, ":", usedUuid)
			}
			return result
		} else {
			this.switchService.resources.Logger().Error("Cannot find uuid for service ", serviceName, ":", serviceArea)
		}
	}
	return this.connectionsForService(serviceName, serviceArea, sourceSwitch, ifs.M_All)
}

func (this *SwitchTable) shutdown() {
	conns := this.conns.all()
	for _, conn := range conns {
		conn.Shutdown()
	}
}

func (this *SwitchTable) monitor() {
	if true {
		return
	}
	for this.switchService.running {
		time.Sleep(time.Second * 15)

		allDown := this.conns.allDownConnections()
		for uuid, _ := range allDown {
			this.conns.shutdownConnection(uuid)
			hp := health.HealthOf(uuid, this.switchService.resources)
			if hp.Status != l8health.L8HealthState_Down {
				this.switchService.resources.Logger().Debug("Update health status to Down")
				hp.Status = l8health.L8HealthState_Down
				hs, _ := health.HealthService(this.switchService.resources)
				hs.Patch(object.New(nil, hp), this.switchService.vnic)
			}
		}
	}
}
