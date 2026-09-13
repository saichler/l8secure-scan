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

// Package replication provides data replication services for the Layer 8 framework.
// It tracks which service elements are stored on which nodes, enabling distributed
// data placement and retrieval across the cluster.
package replication

import (
	"errors"

	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8services/go/services/dcache"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
)

// Service constants for the replication service registration.
const (
	ServiceType = "ReplicationService"
	ServiceName = "Replicas"
	ServiceArea = byte(0)
)

// ReplicationService manages replication indexes that track which nodes
// store which data elements for each service. It maintains a distributed
// cache of L8ReplicationIndex entries.
type ReplicationService struct {
	cache ifs.IDistributedCache
}

// Activate initializes the replication service by setting up the primary key
// decorator and creating the distributed cache for replication indexes.
func (this *ReplicationService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(&l8services.L8ReplicationIndex{}, "ServiceName", "ServiceArea")
	this.cache = dcache.NewDistributedCache(sla.ServiceName(), sla.ServiceArea(), &l8services.L8ReplicationIndex{},
		nil, vnic, vnic.Resources())
	return nil
}

// DeActivate performs cleanup when the service is shut down.
func (this *ReplicationService) DeActivate() error {
	return nil
}

// Post creates or replaces replication index entries in the cache.
func (this *ReplicationService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	for _, elem := range pb.Elements() {
		this.cache.Post(elem, pb.Notification())
	}
	return nil
}

// Put performs a full replacement of replication index entries.
func (this *ReplicationService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	for _, elem := range pb.Elements() {
		this.cache.Put(elem, pb.Notification())
	}
	return nil
}

// Patch performs a partial update of replication index entries.
func (this *ReplicationService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	for _, elem := range pb.Elements() {
		this.cache.Patch(elem, pb.Notification())
	}
	return nil
}

// Delete removes replication index entries from the cache.
func (this *ReplicationService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	for _, elem := range pb.Elements() {
		this.cache.Delete(elem, pb.Notification())
	}
	return nil
}

// Get retrieves replication index entries (not implemented - returns nil).
func (this *ReplicationService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Failed handles message delivery failures (no-op for replication service).
func (this *ReplicationService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns nil as replication service doesn't use transactions.
func (this *ReplicationService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

// Service returns the replication service handler from the resources.
func Service(r ifs.IResources) ifs.IServiceHandler {
	repService, _ := r.Services().ServiceHandler(ServiceName, ServiceArea)
	return repService
}

// ReplicationIndex retrieves the replication index for a specific service,
// which contains the mapping of keys to their storage locations (node UUIDs).
func ReplicationIndex(serviceName string, serviceArea byte, r ifs.IResources) *l8services.L8ReplicationIndex {
	repService, ok := r.Services().ServiceHandler(ServiceName, ServiceArea)
	if !ok {
		return nil
	}
	replicationService, ok := repService.(*ReplicationService)
	if !ok {
		return nil
	}

	filter := &l8services.L8ReplicationIndex{}
	filter.ServiceName = serviceName
	filter.ServiceArea = int32(serviceArea)

	index, err := replicationService.cache.Get(filter)
	if err == nil {
		return index.(*l8services.L8ReplicationIndex)
	}
	return nil
}

// WebService returns nil as replication service doesn't expose a web interface.
func (this *ReplicationService) WebService() ifs.IWebService {
	return nil
}

// Replication returns false as the replication service itself is not replicated.
func (this *ReplicationService) Replication() bool {
	return false
}

// ReplicationCount returns 0 as the replication service doesn't use replication.
func (this *ReplicationService) ReplicationCount() int {
	return 0
}

// KeyOf returns empty string (not applicable for replication service).
func (this *ReplicationService) KeyOf(pb ifs.IElements, r ifs.IResources) string {
	return ""
}

// ConcurrentGets returns false (no concurrent gets for replication service).
func (this *ReplicationService) ConcurrentGets() bool {
	return false
}

// ReplicationFor looks up the replication locations for a message's data element.
// Returns a map of node UUIDs to their replica numbers for the element's key.
func ReplicationFor(msg *ifs.Message, r ifs.IResources, service ifs.IServiceHandler) (map[string]byte, error) {
	pb, err := protocol.ElementsOf(msg, r)
	if err != nil {
		return nil, err
	}
	key := service.TransactionConfig().KeyOf(pb, r)
	repIndex := ReplicationIndex(msg.ServiceName(), msg.ServiceArea(), r)
	if repIndex == nil {
		return nil, errors.New("Replication Index not found")
	}
	locations, ok := repIndex.Keys[key]
	result := map[string]byte{}
	if !ok {
		return result, nil
	}

	for uuid, rep := range locations.Location {
		result[uuid] = byte(rep)
	}
	return result, nil
}
