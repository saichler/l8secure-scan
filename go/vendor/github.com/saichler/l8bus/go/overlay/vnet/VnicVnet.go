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
	"fmt"
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8notify"
)

// VnicVnet provides a VNic interface implementation for the VNet itself,
// enabling the VNet to participate in service communication as a virtual endpoint.
// This allows the VNet to send and receive messages like any other VNic in the network.
type VnicVnet struct {
	vnet *VNet
}

// newVnicVnet creates a new VnicVnet wrapper for the given VNet.
func newVnicVnet(vnet *VNet) *VnicVnet {
	v := &VnicVnet{vnet: vnet}
	v.vnet.resources.Events().SetVNic(v)
	return v
}

// Start is not implemented for VnicVnet as it operates through the parent VNet.
func (this *VnicVnet) Start() {
	panic("implement me")
}

// Shutdown is not implemented for VnicVnet as it operates through the parent VNet.
func (this *VnicVnet) Shutdown() {
	panic("implement me")
}

// Name is not implemented for VnicVnet.
func (this *VnicVnet) Name() string {
	panic("implement me")
	return ""
}

// SendMessage is not implemented for VnicVnet; use Unicast or Multicast instead.
func (this *VnicVnet) SendMessage(data []byte) error {
	//panic("implement me")
	return nil
}

// Unicast sends a message to a specific destination VNic by UUID.
func (this *VnicVnet) Unicast(destination string, serviceName string, serviceArea byte, action ifs.Action, data interface{}) error {
	elems := object.New(nil, data)
	bts, err := this.vnet.protocol.CreateMessageFor(destination, serviceName, serviceArea, ifs.P1, ifs.M_All, action,
		this.Resources().SysConfig().LocalUuid, this.Resources().SysConfig().LocalUuid, elems,
		false, false, this.vnet.protocol.NextMessageNumber(), ifs.NotATransaction,
		"", "", -1, -1, -1, -1, -1, 0, false, "")
	if err != nil {
		return err
	}
	if destination == this.Resources().SysConfig().LocalUuid {
		this.vnet.addVnetTask(QHandleData, bts, this)
		return nil
	}
	_, conn := this.vnet.switchTable.conns.getConnection(destination, true)
	if conn == nil {
		return fmt.Errorf("no connection found for destination %s", destination)
	}
	conn.SendMessage(bts)
	return nil
}

// Request sends a request to a destination and waits for a response with timeout.
func (this *VnicVnet) Request(destination string, serviceName string, area byte, action ifs.Action, data interface{}, timeout int, returnAttributes ...string) ifs.IElements {
	if destination == "" {
		externals := this.vnet.switchTable.conns.allExternalVnets()
		for uuid, _ := range externals {
			destination = uuid
			break
		}
	}
	_, conn := this.vnet.switchTable.conns.getConnection(destination, true)
	if conn == nil {
		return object.New(nil, []interface{}{})
	}
	return conn.Request(destination, serviceName, area, action, data, timeout, returnAttributes...)
}

// Reply sends a response back to the source VNic that originated the request.
func (this *VnicVnet) Reply(msg *ifs.Message, elements ifs.IElements) error {
	reply := msg.CloneReply(this.vnet.resources.SysConfig().LocalUuid, this.vnet.resources.SysConfig().RemoteUuid)
	data, err := this.vnet.protocol.CreateMessageForm(reply, elements)
	if err != nil {
		this.vnet.resources.Logger().Error(err)
		return err
	}
	hp := health.HealthOf(msg.Source(), this.vnet.resources)
	alias := " No Alias Yet"
	if hp != nil {
		alias = hp.Alias
	}
	this.vnet.resources.Logger().Debug("Replying to ", msg.Source(), " ", alias)
	_, conn := this.vnet.switchTable.conns.getConnection(msg.Source(), true)
	if conn == nil {
		return fmt.Errorf("no connection found for source %s", msg.Source())
	}
	return conn.SendMessage(data)
}

// Multicast sends a message to all connections hosting the specified service.
func (this *VnicVnet) Multicast(serviceName string, serviceArea byte, action ifs.Action, any interface{}) error {
	var err error
	var data []byte
	myUuid := this.vnet.resources.SysConfig().LocalUuid
	connections := this.vnet.switchTable.connectionsForService(serviceName, serviceArea, myUuid, ifs.M_All)
	//in case this is the first multicast from a vnet to a vnet
	if serviceName >= ifs.SysMsg && len(connections) == 0 {
		connections = this.vnet.switchTable.conns.allExternalVnets()
	}
	for uuid, connection := range connections {
		data, err = this.vnet.protocol.CreateMessageFor(uuid, serviceName, serviceArea, ifs.P1, ifs.M_All, action,
			myUuid, uuid, object.New(nil, any), false, false, this.vnet.protocol.NextMessageNumber(),
			ifs.NotATransaction, "", "", -1, -1, -1, -1,
			-1, 0, false, "")
		if err != nil {
			continue
		}
		e := connection.SendMessage(data)
		if e != nil {
			err = e
		}
	}
	data, err = this.vnet.protocol.CreateMessageFor(myUuid, serviceName, serviceArea, ifs.P1, ifs.M_All, action,
		myUuid, myUuid, object.New(nil, any), false, false, this.vnet.protocol.NextMessageNumber(),
		ifs.NotATransaction, "", "", -1, -1, -1, -1,
		-1, 0, false, "")
	this.vnet.addVnetTask(QHandleData, data, this)
	return err
}

func (this *VnicVnet) RoundRobin(serviceName string, area byte, action ifs.Action, data interface{}) error {
	panic("implement me")
	return nil
}

func (this *VnicVnet) RoundRobinRequest(serviceName string, area byte, action ifs.Action, data interface{}, timeout int, returnAttributes ...string) ifs.IElements {
	panic("implement me")
	return nil
}

func (this *VnicVnet) Proximity(serviceName string, area byte, action ifs.Action, data interface{}) error {
	panic("implement me")
	return nil
}

func (this *VnicVnet) ProximityRequest(serviceName string, area byte, action ifs.Action, data interface{}, timeout int, returnAttributes ...string) ifs.IElements {
	panic("implement me")
	return nil
}

func (this *VnicVnet) Leader(serviceName string, area byte, action ifs.Action, data interface{}) error {
	panic("implement me")
	return nil
}

func (this *VnicVnet) LeaderRequest(serviceName string, area byte, action ifs.Action, data interface{}, timeout int, returnAttributes ...string) ifs.IElements {
	panic("implement me")
	return nil
}

func (this *VnicVnet) Local(serviceName string, area byte, action ifs.Action, data interface{}) error {
	panic("implement me")
	return nil
}

func (this *VnicVnet) LocalRequest(serviceName string, area byte, action ifs.Action, data interface{}, timeout int, returnAttributes ...string) ifs.IElements {
	panic("implement me")
	return nil
}

// Forward sends a message to a destination and returns the response.
func (this *VnicVnet) Forward(msg *ifs.Message, destination string) ifs.IElements {
	pb, err := this.vnet.protocol.ElementsOf(msg)
	if err != nil {
		return object.NewError(err.Error())
	}

	timeout := 15
	if msg.Tr_Timeout() > 0 {
		timeout = int(msg.Tr_Timeout())
	}

	if destination == "" {
		externals := this.vnet.switchTable.conns.allExternalVnets()
		for uuid, _ := range externals {
			destination = uuid
			break
		}
	}
	if destination == this.Resources().SysConfig().LocalUuid {
		return this.Resources().Services().Handle(pb, msg.Action(), msg, this)
	}
	_, conn := this.vnet.switchTable.conns.getConnection(destination, true)
	if conn == nil {
		return object.New(nil, []interface{}{})
	}
	return conn.Request(destination, msg.ServiceName(), msg.ServiceArea(), msg.Action(), pb, timeout)
}

func (this *VnicVnet) ServiceAPI(serviceName string, area byte) ifs.ServiceAPI {
	return vnic.NewAPI(serviceName, area, this, false, false)
}

// Resources returns the IResources from the parent VNet.
func (this *VnicVnet) Resources() ifs.IResources {
	return this.vnet.resources
}

// NotifyServiceAdded broadcasts health updates when services are added.
func (this *VnicVnet) NotifyServiceAdded(serviceNames []string, serviceArea byte) error {
	if this == nil {
		return nil
	}
	curr := health.HealthOf(this.vnet.resources.SysConfig().LocalUuid, this.vnet.resources)
	if curr == nil {
		return nil
	}
	hp := &l8health.L8Health{}
	hp.AUuid = curr.AUuid
	hp.Services = curr.Services
	//mergeServices(hp, this.vnet.resources.SysConfig().Services)
	for _, serviceName := range serviceNames {
		this.Multicast(serviceName, serviceArea, ifs.PATCH, hp)
	}
	return nil
}

func (this *VnicVnet) NotifyServiceRemoved(serviceName string, area byte) error {
	panic("implement me")
	return nil
}

// PropertyChangeNotification forwards property change notifications to the parent VNet.
func (this *VnicVnet) PropertyChangeNotification(set *l8notify.L8NotificationSet) {
	this.vnet.PropertyChangeNotification(set)
}

func (this *VnicVnet) WaitForConnection() {
	panic("implement me")
}

func (this *VnicVnet) Running() bool {
	panic("implement me")
	return false
}

// SetResponse sets the response for a pending request on the source connection.
func (this *VnicVnet) SetResponse(msg *ifs.Message, pb ifs.IElements) {
	_, conn := this.vnet.switchTable.conns.getConnection(msg.Source(), true)
	if conn == nil {
		return
	}
	conn.SetResponse(msg, pb)
}

// IsVnet returns true, indicating this VNic represents a VNet switch.
func (this *VnicVnet) IsVnet() bool {
	return true
}
