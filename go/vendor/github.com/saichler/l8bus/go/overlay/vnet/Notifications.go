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

	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8system"
)

// PropertyChangeNotification handles property change notifications from services (primarily health),
// broadcasting the notification to local VNics in the network.
func (this *VNet) PropertyChangeNotification(set *l8notify.L8NotificationSet) {
	//only health service will call this callback so check if the notification is from a local source
	//if it is from local source, then just notify local vnics
	protocol.MsgLog.AddLog(set.ServiceName, byte(set.ServiceArea), ifs.Notify)
	vnetUuid := this.resources.SysConfig().LocalUuid
	nextId := this.protocol.NextMessageNumber()
	syncData, _ := this.protocol.CreateMessageFor("", set.ServiceName, byte(set.ServiceArea), ifs.P1, ifs.M_All,
		ifs.Notify, vnetUuid, vnetUuid, object.New(nil, set), false, false,
		nextId, ifs.NotATransaction, "", "",
		-1, -1, -1, -1, -1, 0, false, "")
	this.addVnetTask(QHandleData, syncData, this.vnic)
}

// publishRoutes broadcasts the current route table to all external VNet connections.
func (this *VNet) publishRoutes() {
	vnetUuid := this.resources.SysConfig().LocalUuid
	vnetName := this.resources.SysConfig().LocalAlias

	nextId := this.protocol.NextMessageNumber()

	routeTable := &l8system.L8RouteTable{Rows: this.switchTable.conns.Routes()}
	this.resources.Logger().Debug("Vnet ", vnetName, " publish routes ", len(routeTable.Rows))

	data := &l8system.L8SystemMessage_RouteTable{RouteTable: routeTable}
	routes := &l8system.L8SystemMessage{Action: l8system.L8SystemAction_Routes_Add, Data: data}

	routesData, _ := this.protocol.CreateMessageFor("", ifs.SysMsg, ifs.SysAreaPrimary, ifs.P1, ifs.M_All,
		ifs.POST, vnetUuid, vnetUuid, object.New(nil, routes), false, false,
		nextId, ifs.NotATransaction, "", "",
		-1, -1, -1, -1, -1, 0, false, "")

	allExternal := this.switchTable.conns.allExternalVnets()
	for _, external := range allExternal {
		external.SendMessage(routesData)
	}
}

// publishRemovedRoutes broadcasts route removal messages to all external VNet connections.
func (this *VNet) publishRemovedRoutes(removed map[string]string) {
	vnetUuid := this.resources.SysConfig().LocalUuid
	nextId := this.protocol.NextMessageNumber()

	routeTable := &l8system.L8RouteTable{Rows: removed}
	data := &l8system.L8SystemMessage_RouteTable{RouteTable: routeTable}
	routes := &l8system.L8SystemMessage{Action: l8system.L8SystemAction_Routes_Remove, Data: data}

	routesData, _ := this.protocol.CreateMessageFor("", ifs.SysMsg, ifs.SysAreaPrimary, ifs.P1, ifs.M_All,
		ifs.POST, vnetUuid, vnetUuid, object.New(nil, routes), false, false,
		nextId, ifs.NotATransaction, "", "",
		-1, -1, -1, -1, -1, 0, false, "")

	allExternal := this.switchTable.conns.allExternalVnets()
	for _, external := range allExternal {
		external.SendMessage(routesData)
	}
}

// publishSystemMessage broadcasts a system control message to all external VNet connections.
func (this *VNet) publishSystemMessage(sysmsg *l8system.L8SystemMessage) {
	vnetUuid := this.resources.SysConfig().LocalUuid
	nextId := this.protocol.NextMessageNumber()

	sysmsg.Publish = false

	sysmsgData, _ := this.protocol.CreateMessageFor("", ifs.SysMsg, ifs.SysAreaPrimary, ifs.P1, ifs.M_All,
		ifs.POST, vnetUuid, vnetUuid, object.New(nil, sysmsg), false, false,
		nextId, ifs.NotATransaction, "", "",
		-1, -1, -1, -1, -1, 0, false, "")

	allExternal := this.switchTable.conns.allExternalVnets()
	for _, external := range allExternal {
		external.SendMessage(sysmsgData)
	}
}
