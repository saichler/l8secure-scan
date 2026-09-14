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
	"errors"
	"github.com/saichler/l8utils/go/utils/queues"
	"net"
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/protocol"
	vnic2 "github.com/saichler/l8bus/go/overlay/vnic"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"github.com/saichler/l8types/go/types/l8system"
	"github.com/saichler/l8types/go/types/l8web"
	resources2 "github.com/saichler/l8utils/go/utils/resources"
	"github.com/saichler/l8utils/go/utils/strings"
)

// VNet represents a Virtual Network switch that manages connections between
// distributed nodes in the Layer8 overlay network. It acts as a central hub
// for message routing, service discovery, and health monitoring.
type VNet struct {
	resources        ifs.IResources
	socket           net.Listener
	running          bool
	ready            bool
	switchTable      *SwitchTable
	protocol         *protocol.Protocol
	discovery        *Discovery
	vnic             *VnicVnet
	vnetServiceTasks *queues.Queue
	vnetSystemTasks  *queues.Queue
	handleDataTasks  *queues.Queue
	healthReport     *queues.Queue
	vnetServices     map[string]bool
	vnetUuid         string
}

// NewVNet creates and initializes a new VNet instance. It registers required
// types, initializes the switch table, protocol handler, and discovery service.
// The hasSecondary parameter enables secondary VNet connectivity for cross-network communication.
func NewVNet(resources ifs.IResources, hasSecondary ...bool) *VNet {
	resources.Registry().Register(&l8system.L8SystemMessage{})
	resources.Registry().Register(&l8web.L8Empty{})
	resources.Registry().Register(&l8health.L8Top{})
	net := &VNet{}
	net.vnetServices = map[string]bool{health.ServiceName: true, "tokens": true, "users": true, "roles": true, "Creds": true, ifs.SystemServiceGroup: true}
	net.vnetServiceTasks = queues.NewQueue("vnetServiceTasks", int(resources2.DEFAULT_QUEUE_SIZE))
	net.vnetSystemTasks = queues.NewQueue("vnetSystemTasks", queues.NO_LIMIT)
	net.handleDataTasks = queues.NewQueue("vnicVnetUnicastTasks", int(resources2.DEFAULT_QUEUE_SIZE))
	net.healthReport = queues.NewQueue("healthReport", int(resources2.DEFAULT_QUEUE_SIZE))
	net.resources = resources
	net.resources.Set(net)
	net.vnic = newVnicVnet(net)
	net.protocol = protocol.New(net.vnic)
	net.running = true
	net.resources.SysConfig().LocalUuid = ifs.NewUuid()
	net.vnetUuid = net.resources.SysConfig().LocalUuid
	net.switchTable = newSwitchTable(net)
	go net.processTasks(net.vnetSystemTasks, net.systemMessageReceived)
	go net.processTasks(net.vnetServiceTasks, net.vnetServiceRequest)
	go net.processTasks(net.handleDataTasks, net.HandleData)
	go net.processTasks(net.healthReport, net.sendHealthReport)

	secService, ok := net.resources.Security().(ifs.ISecurityProviderActivate)
	if ok {
		secService.Activate(net.vnic)
	} else {
	}

	health.Activate(net.vnic, true)
	if hasSecondary != nil && hasSecondary[0] {
		net.resources.SysConfig().RemoteVnet = "X"
		health.Activate(net.vnic, true)
		net.resources.SysConfig().RemoteVnet = ""
	}

	net.discovery = NewDiscovery(net)

	hp := &l8health.L8Health{}
	hp.Alias = net.resources.SysConfig().LocalAlias
	hp.AUuid = net.resources.SysConfig().LocalUuid
	hp.IsVnet = true
	hp.Services = net.resources.SysConfig().Services

	hs, _ := health.HealthService(net.resources)
	hs.Put(object.NewNotify(hp), net.vnic)
	if hasSecondary != nil && hasSecondary[0] {
		net.resources.SysConfig().RemoteVnet = "X"
		hs, _ = health.HealthService(net.resources)
		hs.Put(object.NewNotify(hp), net.vnic)
		net.resources.SysConfig().RemoteVnet = ""
	}
	go net.patchStatistics()
	return net
}

// Start begins the VNet server, binding to the configured port and accepting
// incoming connections. It also initiates peer discovery via UDP broadcast.
func (this *VNet) Start() error {
	var err error
	go this.start(&err)
	for !this.ready && err == nil {
		time.Sleep(time.Millisecond * 50)
	}
	time.Sleep(time.Millisecond * 50)
	this.discovery.Discover()
	return err
}

func (this *VNet) start(err *error) {
	this.resources.Logger().Debug("VNet.start: Starting VNet ")
	if this.resources.SysConfig().VnetPort == 0 {
		er := errors.New("Switch Port does not have a port defined")
		err = &er
		return
	}

	er := this.bind()
	if er != nil {
		err = &er
		return
	}

	for this.running {
		this.ready = true
		conn, e := this.socket.Accept()
		if e != nil && this.running {
			this.resources.Logger().Error("Failed to accept socket connection:", err)
			continue
		}
		if this.running {
			this.resources.Logger().Debug("Accepted socket connection...")
			go this.connect(conn)
		}
	}
	this.resources.Logger().Warning("Vnet ", this.resources.SysConfig().LocalAlias, " has ended")
}

func (this *VNet) bind() error {
	socket, e := net.Listen("tcp", strings.New(":", int(this.resources.SysConfig().VnetPort)).String())
	if e != nil {
		return this.resources.Logger().Error("Unable to bind to port ",
			this.resources.SysConfig().VnetPort, e.Error())
	}
	this.resources.Logger().Debug("Bind Successfully to port ",
		this.resources.SysConfig().VnetPort)
	this.socket = socket
	return nil
}

func (this *VNet) connect(conn net.Conn) {
	sec := this.resources.Security()
	err := sec.CanAccept(conn)
	if err != nil {
		this.resources.Logger().Error(err)
		return
	}

	config := &l8sysconfig.L8SysConfig{MaxDataSize: resources2.DEFAULT_MAX_DATA_SIZE,
		RxQueueSize: resources2.DEFAULT_QUEUE_SIZE,
		TxQueueSize: resources2.DEFAULT_QUEUE_SIZE,
		LocalAlias:  this.resources.SysConfig().LocalAlias,
		LocalUuid:   this.resources.SysConfig().LocalUuid,
		IAmVnet:     true,
		Services: &l8services.L8Services{ServiceToAreas: map[string]*l8services.L8ServiceAreas{
			health.ServiceName: &l8services.L8ServiceAreas{
				Areas: map[int32]bool{0: true},
			}}},
	}

	resources := resources2.NewResources(this.resources.Logger())
	resources.Copy(this.resources)
	resources.Set(this)
	resources.Set(config)

	vnic := vnic2.NewVirtualNetworkInterface(resources, conn)
	vnic.Resources().SysConfig().LocalUuid = this.resources.SysConfig().LocalUuid

	err = sec.ValidateConnection(conn, config)
	if err != nil {
		this.resources.Logger().Error(err)
		return
	}

	this.addHealthForVNic(vnic.Resources().SysConfig())

	vnic.Start()
	this.notifyNewVNic(vnic)
}

func (this *VNet) notifyNewVNic(vnic ifs.IVNic) {
	this.switchTable.addVNic(vnic)
}

// Shutdown gracefully stops the VNet, closing all connections and releasing resources.
func (this *VNet) Shutdown() {
	this.resources.Logger().Debug("Shutdown called!")
	this.running = false
	this.socket.Close()
	this.switchTable.shutdown()
}

// Failed handles message delivery failures by creating and sending a failure
// response back to the originating VNic with the specified error message.
func (this *VNet) Failed(data []byte, vnic ifs.IVNic, failMsg string) {
	msg, err := this.protocol.MessageOf(data)
	if err != nil {
		this.resources.Logger().Error(err)
		return
	}

	failMessage := msg.CloneFail(failMsg, this.resources.SysConfig().RemoteUuid)
	data, _ = failMessage.Marshal(nil, this.resources)

	err = vnic.SendMessage(data)
}

// HandleData processes incoming message data, routing it to the appropriate
// destination based on message headers. It supports unicast, multicast, and
// service-based routing modes.
func (this *VNet) HandleData(data []byte, vnic ifs.IVNic) {
	source, sourceVnet, destination, serviceName, serviceArea, _, multicastMode := ifs.HeaderOf(data)
	protocol.MsgLog.AddLog(serviceName, serviceArea, ifs.Handle)

	if serviceName == ifs.SysMsg && serviceArea == ifs.SysAreaPrimary {
		this.addVnetTask(QSystem, data, vnic)
		return
	}

	if destination != "" {
		//The destination is the vnet
		if destination == this.vnetUuid {
			this.addVnetTask(QService, data, vnic)
			return
		}

		if destination == ifs.DESTINATION_Single {
			destination = this.switchTable.services.serviceFor(serviceName, serviceArea, source, multicastMode)
		}
		//Incase the destination is the vnet after the service sele
		if destination == this.vnetUuid {
			this.addVnetTask(QService, data, vnic)
			return
		}
		//The destination is a single port
		_, p := this.switchTable.conns.getConnection(destination, true)
		if p == nil {
			this.Failed(data, vnic, strings.New("Cannot find destination port for ", destination).String())
			return
		}

		err := p.SendMessage(data)
		if err != nil {
			if !p.Running() {
				uuid := p.Resources().SysConfig().RemoteUuid
				hp := health.HealthOf(uuid, this.resources)
				this.sendHealth(hp)
			}
			this.Failed(data, vnic, strings.New("Error sending data:", err.Error()).String())
			return
		}
	} else {
		connections := this.switchTable.connectionsForService(serviceName, serviceArea, sourceVnet, multicastMode)
		this.uniCastToPorts(connections, data)
		_, ok := this.vnetServices[serviceName]
		if ok && source != this.vnetUuid {
			this.addVnetTask(QService, data, vnic)
		}
		return
	}
}

func (this *VNet) uniCastToPorts(connections map[string]ifs.IVNic, data []byte) {
	for _, port := range connections {
		port.SendMessage(data)
	}
}

// ShutdownVNic handles the disconnection of a VNic, removing its routes,
// services, and health records from the switch table.
func (this *VNet) ShutdownVNic(vnic ifs.IVNic) {
	uuid := vnic.Resources().SysConfig().RemoteUuid
	removed := map[string]string{uuid: ""}
	this.switchTable.routeTable.removeRoutes(removed)
	this.switchTable.services.removeService(removed)
	this.removeHealth(removed)
	this.publishRemovedRoutes(removed)
}

// Resources returns the IResources instance containing configuration,
// registry, security provider, and other shared resources.
func (this *VNet) Resources() ifs.IResources {
	return this.resources
}

func (this *VNet) sendHealth(hp *l8health.L8Health) {
	vnetUuid := this.resources.SysConfig().LocalUuid
	nextId := this.protocol.NextMessageNumber()
	h, _ := this.protocol.CreateMessageFor("", health.ServiceName, 0, ifs.P1, ifs.M_All,
		ifs.POST, vnetUuid, vnetUuid, object.New(nil, hp), false, false,
		nextId, ifs.NotATransaction, "", "",
		-1, -1, -1, -1, -1, 0, false, "")
	this.addVnetTask(QHandleData, h, this.vnic)
}

// VnetVnic returns the internal VNic used by the VNet for its own service communication.
func (this *VNet) VnetVnic() ifs.IVNic {
	return this.vnic
}

func (this *VNet) patchStatistics() {
	keepAliveInterval := this.resources.SysConfig().KeepAliveIntervalSeconds
	if keepAliveInterval <= 30 {
		keepAliveInterval = 30
	}
	for this.running {
		time.Sleep(time.Second * time.Duration(keepAliveInterval))
		hp := health.BaseHealthStats(this.resources)
		hs, ok := health.HealthService(this.resources)
		if ok {
			hs.Patch(object.New(nil, hp), this.vnic)
		}
	}
}
