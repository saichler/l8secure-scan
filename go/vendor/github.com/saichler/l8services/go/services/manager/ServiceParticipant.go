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
	"sync"

	"github.com/saichler/l8types/go/ifs"
)

// participantSet holds the set of participant UUIDs for a service
// and tracks round-robin usage for load distribution.
type participantSet struct {
	uuids  map[string]struct{}
	rrUsed map[string]struct{}
	mtx    sync.RWMutex
}

// GroupResolver resolves a service name and area to its group name and area.
type GroupResolver func(serviceName string, serviceArea byte) (string, byte)

// ParticipantRegistry manages service participants across the distributed system.
// It tracks which nodes are participating in each service for coordination and routing.
type ParticipantRegistry struct {
	participants  sync.Map // key: serviceKey string -> *participantSet
	groupResolver GroupResolver
}

// NewParticipantRegistry creates a new empty ParticipantRegistry.
func NewParticipantRegistry(resolver GroupResolver) *ParticipantRegistry {
	return &ParticipantRegistry{groupResolver: resolver}
}

// handleRegistry routes participant registry messages to appropriate handlers.
func (pr *ParticipantRegistry) handleRegistry(action ifs.Action, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	switch action {
	case ifs.ServiceRegister:
		return pr.handleServiceRegister(vnic, msg)
	case ifs.ServiceUnregister:
		return pr.handleServiceUnregister(vnic, msg)
	case ifs.ServiceQuery:
		return pr.handleServiceQuery(vnic, msg)
	}
	return nil
}

// resolveGroupKey resolves the message's service name/area to the group key.
func (pr *ParticipantRegistry) resolveGroupKey(msg *ifs.Message) string {
	if pr.groupResolver != nil {
		gName, gArea := pr.groupResolver(msg.ServiceName(), msg.ServiceArea())
		return makeServiceKey(gName, gArea)
	}
	return makeServiceKey(msg.ServiceName(), msg.ServiceArea())
}

// handleServiceRegister adds the message source as a participant for the service.
func (pr *ParticipantRegistry) handleServiceRegister(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	localUuid := vnic.Resources().SysConfig().LocalUuid

	// Skip self-multicast — already registered locally in triggerElections
	if msg.Source() == localUuid {
		return nil
	}

	key := pr.resolveGroupKey(msg)
	ps := pr.getOrCreateParticipantSet(key)

	vnic.Resources().Logger().Debug("Registering participant", msg.Source(), "for", msg.ServiceName(), "area", msg.ServiceArea())

	ps.mtx.Lock()
	ps.uuids[msg.Source()] = struct{}{}
	ps.mtx.Unlock()

	vnic.Resources().Logger().Debug("Registered participant", msg.Source(), "for", msg.ServiceName(), "area", msg.ServiceArea())
	return nil
}

// handleServiceUnregister removes the message source from the participant list.
func (pr *ParticipantRegistry) handleServiceUnregister(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	key := pr.resolveGroupKey(msg)
	ps := pr.getParticipantSet(key)
	if ps == nil {
		return nil
	}

	vnic.Resources().Logger().Debug("Unregistering participant", msg.Source(), "for", msg.ServiceName(), "area", msg.ServiceArea())

	ps.mtx.Lock()
	delete(ps.uuids, msg.Source())
	ps.mtx.Unlock()

	vnic.Resources().Logger().Debug("Unregistered participant", msg.Source(), "for", msg.ServiceName(), "area", msg.ServiceArea())
	return nil
}

// handleServiceQuery responds if this node is a participant for the queried service.
func (pr *ParticipantRegistry) handleServiceQuery(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	localUuid := vnic.Resources().SysConfig().LocalUuid

	// Skip self-multicast to avoid unnecessary messages
	if msg.Source() == localUuid {
		return nil
	}

	key := pr.resolveGroupKey(msg)
	ps := pr.getParticipantSet(key)

	vnic.Resources().Logger().Debug("Service query from", msg.Source(), "for", msg.ServiceName(), "area", msg.ServiceArea())

	if ps != nil {
		ps.mtx.RLock()
		_, isParticipant := ps.uuids[localUuid]
		ps.mtx.RUnlock()

		if isParticipant {
			vnic.Resources().Logger().Debug("Responding to query, I am a participant")
			// Respond that we are a participant
			vnic.Unicast(msg.Source(), msg.ServiceName(), msg.ServiceArea(), ifs.ServiceRegister, nil)
		}
	}

	return nil
}

// RegisterParticipant adds a UUID to the participant set for a service.
func (pr *ParticipantRegistry) RegisterParticipant(serviceName string, serviceArea byte, uuid string) {
	key := makeServiceKey(serviceName, serviceArea)
	ps := pr.getOrCreateParticipantSet(key)

	ps.mtx.Lock()
	ps.uuids[uuid] = struct{}{}
	ps.mtx.Unlock()
}

// UnregisterParticipant removes a UUID from the participant set for a service.
func (pr *ParticipantRegistry) UnregisterParticipant(serviceName string, serviceArea byte, uuid string) {
	key := makeServiceKey(serviceName, serviceArea)
	ps := pr.getParticipantSet(key)
	if ps == nil {
		return
	}

	ps.mtx.Lock()
	delete(ps.uuids, uuid)
	ps.mtx.Unlock()
}

// RoundRobinParticipants selects participants using round-robin for even distribution.
// Resets the usage tracking when all participants have been used in the current cycle.
func (pr *ParticipantRegistry) RoundRobinParticipants(serviceName string, serviceArea byte, replications int) map[string]byte {
	key := makeServiceKey(serviceName, serviceArea)
	ps := pr.getParticipantSet(key)
	if ps == nil {
		return map[string]byte{}
	}

	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	totalParticipants := len(ps.uuids)
	if totalParticipants == 0 {
		return map[string]byte{}
	}

	// Initialize rrUsed map if needed
	if ps.rrUsed == nil {
		ps.rrUsed = make(map[string]struct{})
	}

	// If all participants have been used, reset for new cycle
	if len(ps.rrUsed) >= totalParticipants {
		ps.rrUsed = make(map[string]struct{})
	}

	replica := byte(0)

	// If replications >= total participants, return all
	if replications >= totalParticipants {
		result := make(map[string]byte, totalParticipants)
		for uuid := range ps.uuids {
			result[uuid] = replica
			replica++
		}
		return result
	}

	// Select participants that haven't been used yet in this cycle
	result := make(map[string]byte, replications)
	for uuid := range ps.uuids {
		if len(result) >= replications {
			break
		}
		// Skip if already used in this cycle
		if _, used := ps.rrUsed[uuid]; !used {
			result[uuid] = replica
			replica++
			ps.rrUsed[uuid] = struct{}{}
		}
	}

	return result
}

// GetParticipants returns all participants for a service as a map of UUIDs.
func (pr *ParticipantRegistry) GetParticipants(serviceName string, serviceArea byte) map[string]byte {
	key := makeServiceKey(serviceName, serviceArea)
	ps := pr.getParticipantSet(key)
	if ps == nil {
		return map[string]byte{}
	}

	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	participants := make(map[string]byte, len(ps.uuids))
	for uuid := range ps.uuids {
		participants[uuid] = 0
	}

	return participants
}

// IsParticipant checks if a UUID is registered as a participant for a service.
func (pr *ParticipantRegistry) IsParticipant(serviceName string, serviceArea byte, uuid string) bool {
	key := makeServiceKey(serviceName, serviceArea)
	ps := pr.getParticipantSet(key)
	if ps == nil {
		return false
	}

	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	_, exists := ps.uuids[uuid]
	return exists
}

// ParticipantCount returns the number of participants for a service.
func (pr *ParticipantRegistry) ParticipantCount(serviceName string, serviceArea byte) int {
	key := makeServiceKey(serviceName, serviceArea)
	ps := pr.getParticipantSet(key)
	if ps == nil {
		return 0
	}

	ps.mtx.RLock()
	defer ps.mtx.RUnlock()

	return len(ps.uuids)
}

// UnregisterParticipantFromAll removes a UUID from all service participant sets.
// Used when a node leaves the cluster.
func (pr *ParticipantRegistry) UnregisterParticipantFromAll(uuid string) {
	pr.participants.Range(func(key, value interface{}) bool {
		ps := value.(*participantSet)
		ps.mtx.Lock()
		delete(ps.uuids, uuid)
		ps.mtx.Unlock()
		return true
	})
}

// getParticipantSet retrieves the participant set for a key, or nil if not found.
func (pr *ParticipantRegistry) getParticipantSet(key string) *participantSet {
	ps, ok := pr.participants.Load(key)
	if !ok {
		return nil
	}
	return ps.(*participantSet)
}

// getOrCreateParticipantSet retrieves or creates a participant set for a key.
func (pr *ParticipantRegistry) getOrCreateParticipantSet(key string) *participantSet {
	newPs := &participantSet{
		uuids: make(map[string]struct{}),
	}
	actual, _ := pr.participants.LoadOrStore(key, newPs)
	return actual.(*participantSet)
}
