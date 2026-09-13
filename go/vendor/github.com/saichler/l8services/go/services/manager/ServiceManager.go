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

// Package manager provides the central orchestration layer for Layer 8 services.
// It handles service registration, activation, request routing, leader election,
// participant management, transaction coordination, and Map-Reduce operations.
package manager

import (
	"bytes"
	"reflect"
	"strconv"
	"sync"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8services/go/services/replication"
	"github.com/saichler/l8services/go/services/transaction/states"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8services"
)

// ServiceManager is the central coordinator for all services in a Layer 8 node.
// It manages service lifecycle, routes requests to appropriate handlers, coordinates
// distributed transactions, handles leader election, and tracks service participants.
type ServiceManager struct {
	services            *ServicesMap
	trManager           *states.TransactionManager
	resources           ifs.IResources
	leaderElection      *LeaderElection
	participantRegistry *ParticipantRegistry
	electionDebouncer   *ElectionDebouncer
	serviceToGroup      sync.Map // serviceKey → groupName string (only non-identity mappings)
}

// NewServices creates a new ServiceManager with all required subsystems initialized.
// It sets up the services map, transaction manager, leader election, participant registry,
// and registers required protocol types with the registry.
func NewServices(resources ifs.IResources) ifs.IServices {
	sp := &ServiceManager{}
	sp.services = NewServicesMap()
	sp.resources = resources
	sp.trManager = states.NewTransactionManager(sp)
	sp.leaderElection = NewLeaderElection(sp)
	sp.participantRegistry = NewParticipantRegistry(sp.resolveGroup)
	sp.electionDebouncer = NewElectionDebouncer(sp.leaderElection)
	_, err := sp.resources.Registry().Register(&l8notify.L8NotificationSet{})
	if err != nil {
		panic(err)
	}
	sp.resources.Registry().Register(&l8services.L8Transaction{})
	sp.resources.Registry().Register(&replication.ReplicationService{})
	return sp
}

// RegisterServiceHandlerType registers a service handler type with the type registry
// to enable dynamic instantiation of handlers during service activation.
func (this *ServiceManager) RegisterServiceHandlerType(handler ifs.IServiceHandler) {
	this.resources.Registry().Register(handler)
}

// Handle is the main entry point for processing incoming service requests.
// It performs security checks, routes to participant registry or leader election handlers,
// initiates transactions for stateful services, and delegates to service handlers.
func (this *ServiceManager) Handle(pb ifs.IElements, action ifs.Action, msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	if vnic == nil {
		return object.NewError("Handle: vnic cannot be nil")
	}
	if msg == nil {
		return object.NewError("Handle: message cannot be nil")
	}
	err := vnic.Resources().Security().CanDoAction(vnic, action, pb, vnic.Resources().SysConfig().LocalUuid, msg.AAAId())
	if err != nil {
		return object.NewError(err.Error())
	}

	// Handle participant registry actions
	if action >= ifs.ServiceRegister && action <= ifs.ServiceQuery {
		vnic.Resources().Logger().Debug("Routing to participant registry, action:", action)
		return this.participantRegistry.handleRegistry(action, vnic, msg)
	}

	// Handle leader election actions
	if action >= ifs.ElectionRequest && action <= ifs.LeaderChallenge {
		return this.leaderElection.handleElection(action, vnic, msg)
	}

	if msg.Action() == ifs.EndPoints {
		this.sendEndPoints(vnic)
		return nil
	}

	h, ok := this.services.get(msg.ServiceName(), msg.ServiceArea())
	if !ok {
		hp := health.HealthOf(vnic.Resources().SysConfig().LocalUuid, vnic.Resources())
		alias := "Unknown"
		if hp != nil {
			alias = hp.Alias
		}
		return object.NewError(alias + " - Cannot find active handler for service " + msg.ServiceName() +
			" area " + strconv.Itoa(int(msg.ServiceArea())))
	}

	if msg.FailMessage() != "" {
		return h.Failed(pb, vnic, msg)
	}

	isStartTransaction := h.TransactionConfig() != nil && msg.Action() < ifs.ElectionRequest && this.GetLeader(msg.ServiceName(), msg.ServiceArea()) != ""
	if isStartTransaction {
		if msg.Tr_State() == ifs.NotATransaction {
			return this.trManager.Create(msg, vnic)
		}
		return this.trManager.Run(msg, vnic)
	}
	resp := this.handle(h, pb, action, msg, vnic)
	scope := vnic.Resources().Security().ScopeView(vnic, resp, vnic.Resources().SysConfig().LocalUuid, msg.AAAId())
	if scope != nil {
		return scope
	}
	return resp
}

// updateReplicationIndex updates the replication index for a service element,
// recording which replica stores which key for data distribution tracking.
func (this *ServiceManager) updateReplicationIndex(serviceName string, serviceArea byte, key string, replica byte, r ifs.IResources) {
	index := replication.ReplicationIndex(serviceName, serviceArea, r)
	if index.Keys == nil {
		index.Keys = make(map[string]*l8services.L8ReplicationKey)
	}
	if index.Keys[key] == nil {
		index.Keys[key] = &l8services.L8ReplicationKey{}
		index.Keys[key].Location = make(map[string]int32)
	}
	index.Keys[key].Location[r.SysConfig().LocalUuid] = int32(replica)
	repService := replication.Service(r)
	repService.Patch(object.New(nil, index), nil)
}

// TransactionHandle processes requests within a transaction context.
// It delegates to the service handler and updates the replication index
// on successful operations for services with replication enabled.
func (this *ServiceManager) TransactionHandle(pb ifs.IElements, action ifs.Action, msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	this.resources.Logger().Debug("Transaction Handle:", msg.ServiceName(), ",", msg.ServiceArea(), ",", action)
	h, _ := this.services.get(msg.ServiceName(), msg.ServiceArea())
	if h == nil {
		this.resources.Logger().Debug("Transaction Handle: No handler for service "+msg.ServiceName(), "-", msg.ServiceArea())
	}
	resp := this.handle(h, pb, action, msg, vnic)
	if resp == nil {
		panic("Transaction Handler " + reflect.ValueOf(h).Elem().Type().Name() + " action " + strconv.Itoa(int(action)) + " resp is nil")
	}
	// Apply ScopeView to transactional responses (same as non-transactional path)
	if resp.Error() == nil && action == ifs.GET {
		scope := vnic.Resources().Security().ScopeView(vnic, resp, vnic.Resources().SysConfig().LocalUuid, msg.AAAId())
		if scope != nil {
			resp = scope
		}
	}
	if resp.Error() == nil && h.TransactionConfig().Replication() {
		key := h.TransactionConfig().KeyOf(resp, vnic.Resources())
		this.updateReplicationIndex(msg.ServiceName(), msg.ServiceArea(), key, msg.Tr_Replica(), vnic.Resources())
	}
	return resp
}

// onNodeDelete handles cleanup when a node is removed from the cluster,
// unregistering the node from all service participant lists.
func (this *ServiceManager) onNodeDelete(uuid string) {
	this.participantRegistry.UnregisterParticipantFromAll(uuid)
	this.resources.Logger().Debug("Unregistered all services for failed node", uuid)
}

// ServiceHandler retrieves the registered handler for a specific service and area.
// Returns the handler and true if found, nil and false otherwise.
func (this *ServiceManager) ServiceHandler(serviceName string, serviceArea byte) (ifs.IServiceHandler, bool) {
	return this.services.get(serviceName, serviceArea)
}

// sendEndPoints broadcasts all registered web service endpoints to the network
// via multicast, enabling service discovery for HTTP/REST endpoints.
func (this *ServiceManager) sendEndPoints(vnic ifs.IVNic) {
	webServices := this.services.webServices()
	for _, ws := range webServices {
		vnic.Resources().Logger().Debug("Sent Webservice multicast for ", ws.ServiceName(), " area ", ws.ServiceArea())
		vnic.Multicast(ifs.WebService, 0, ifs.POST, ws.Serialize())
	}
}

// cacheKey generates a unique key for caching by combining service name and area.
func cacheKey(serviceName string, serviceArea byte) string {
	buff := bytes.Buffer{}
	buff.WriteString(serviceName)
	buff.WriteString("--")
	buff.WriteString(strconv.Itoa(int(serviceArea)))
	return buff.String()
}

// resolveGroup maps a service name+area to its group name.
// The area is always preserved — only the name can change.
// Returns the original name when no group mapping exists (zero overhead fast path).
func (this *ServiceManager) resolveGroup(serviceName string, serviceArea byte) (string, byte) {
	key := cacheKey(serviceName, serviceArea)
	if groupName, ok := this.serviceToGroup.Load(key); ok {
		return groupName.(string), 0
	}
	return serviceName, serviceArea
}

// Services returns a structured list of all registered services and their areas.
func (this *ServiceManager) Services() *l8services.L8Services {
	return this.services.serviceList()
}

// StartElection triggers a leader election for the specified service and area.
func (this *ServiceManager) StartElection(serviceName string, serviceArea byte, vnic ifs.IVNic) {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	this.leaderElection.StartElectionForService(gName, gArea, vnic)
}

// GetLeader returns the UUID of the current leader for the specified service and area.
// Returns empty string if no leader is elected.
func (this *ServiceManager) GetLeader(serviceName string, serviceArea byte) string {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	return this.leaderElection.GetLeader(gName, gArea)
}

// IsLeader checks if the specified UUID is the current leader for the service and area.
func (this *ServiceManager) IsLeader(serviceName string, serviceArea byte, uuid string) bool {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	return this.leaderElection.IsLeader(gName, gArea, uuid)
}

// RegisterParticipant adds a node to the participant list for a service and area.
func (this *ServiceManager) RegisterParticipant(serviceName string, serviceArea byte, uuid string) {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	this.participantRegistry.RegisterParticipant(gName, gArea, uuid)
}

// UnregisterParticipant removes a node from the participant list for a service and area.
func (this *ServiceManager) UnregisterParticipant(serviceName string, serviceArea byte, uuid string) {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	this.participantRegistry.UnregisterParticipant(gName, gArea, uuid)
}

// GetParticipants returns a map of all participant UUIDs for the service and area.
func (this *ServiceManager) GetParticipants(serviceName string, serviceArea byte) map[string]byte {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	return this.participantRegistry.GetParticipants(gName, gArea)
}

// IsParticipant checks if a UUID is registered as a participant for the service and area.
func (this *ServiceManager) IsParticipant(serviceName string, serviceArea byte, uuid string) bool {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	return this.participantRegistry.IsParticipant(gName, gArea, uuid)
}

// ParticipantCount returns the number of registered participants for the service and area.
func (this *ServiceManager) ParticipantCount(serviceName string, serviceArea byte) int {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	return this.participantRegistry.ParticipantCount(gName, gArea)
}

// RoundRobinParticipants selects participants in round-robin fashion for load distribution.
// Returns a map of selected participant UUIDs to their replica numbers.
func (this *ServiceManager) RoundRobinParticipants(serviceName string, serviceArea byte, replications int) map[string]byte {
	gName, gArea := this.resolveGroup(serviceName, serviceArea)
	return this.participantRegistry.RoundRobinParticipants(gName, gArea, replications)
}
