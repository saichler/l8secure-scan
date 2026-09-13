/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ifs

import (
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8services"
)

// IServices manages service registration, activation, and message routing.
// It is the central service registry for the Layer 8 system.
type IServices interface {
	// RegisterServiceHandlerType registers a service handler type for later activation.
	// This ensures the handler code is linked and available.
	RegisterServiceHandlerType(IServiceHandler)
	// Activate starts a service with the given SLA configuration.
	Activate(*ServiceLevelAgreement, IVNic) (IServiceHandler, error)
	// DeActivate stops a service and removes it from the registry.
	DeActivate(string, byte, IResources, IServiceCacheListener) error
	// Handle routes a message to the appropriate service handler.
	Handle(IElements, Action, *Message, IVNic) IElements
	// TransactionHandle routes a transactional message to the handler.
	TransactionHandle(IElements, Action, *Message, IVNic) IElements
	// Notify handles property change notifications for a service.
	Notify(IElements, IVNic, *Message, bool) IElements
	// ServiceHandler returns the handler for a service by name and area.
	ServiceHandler(string, byte) (IServiceHandler, bool)
	// Services returns the registry of all active services.
	Services() *l8services.L8Services
	// GetLeader returns the UUID of the leader for a service.
	GetLeader(string, byte) string
	// GetParticipants returns all participants hosting a service.
	GetParticipants(string, byte) map[string]byte
	// RoundRobinParticipants returns a subset of participants for round-robin selection.
	RoundRobinParticipants(string, byte, int) map[string]byte
	// TriggerElections initiates leader election for all services.
	TriggerElections(nic IVNic)
}

// IServiceHandler handles incoming requests for a specific service.
// Implements the CRUD operations and lifecycle management for a service.
type IServiceHandler interface {
	// Activate initializes the service handler with its SLA configuration.
	Activate(*ServiceLevelAgreement, IVNic) error
	// DeActivate shuts down the service handler.
	DeActivate() error
	// Post handles create operations.
	Post(IElements, IVNic) IElements
	// Put handles replace operations.
	Put(IElements, IVNic) IElements
	// Patch handles partial update operations.
	Patch(IElements, IVNic) IElements
	// Delete handles delete operations.
	Delete(IElements, IVNic) IElements
	// Get handles read operations.
	Get(IElements, IVNic) IElements
	// Failed handles failed message delivery.
	Failed(IElements, IVNic, *Message) IElements
	// TransactionConfig returns the transaction configuration for this handler.
	TransactionConfig() ITransactionConfig
	// WebService returns the web service configuration if enabled.
	WebService() IWebService
}

// IServiceHandlerCache extends IServiceHandler with caching capabilities.
// Provides in-memory caching with query and metadata support.
type IServiceHandlerCache interface {
	IServiceHandler
	// Collect applies a filter function and returns matching elements.
	Collect(f func(interface{}) (bool, interface{})) map[string]interface{}
	// All returns all cached elements.
	All() map[string]interface{}
	// Fetch retrieves paginated results with optional query filtering.
	Fetch(int, int, IQuery) ([]interface{}, *l8api.L8MetaData)
	// AddMetadataFunc registers a metadata extraction function.
	AddMetadataFunc(string, func(interface{}) (bool, string))
	// ServiceName returns the service name.
	ServiceName() string
	// ServiceArea returns the service area.
	ServiceArea() byte
	// Size returns the number of cached elements.
	Size() int
}

// IMapReduceService defines the interface for map-reduce operations.
// Used to aggregate results from distributed queries.
type IMapReduceService interface {
	// Merge combines results from multiple service areas into a single result.
	Merge(map[string]IElements) IElements
}

// IServiceCacheListener receives notifications about cache changes.
type IServiceCacheListener interface {
	// PropertyChangeNotification is called when properties change in the cache.
	PropertyChangeNotification(set *l8notify.L8NotificationSet)
}

// IDistributedCache provides a distributed caching interface with CRUD operations.
// Supports notification generation for property changes.
type IDistributedCache interface {
	// Post creates a new entry, optionally broadcasting the notification.
	Post(interface{}, ...bool) (*l8notify.L8NotificationSet, error)
	// Put replaces an entry, optionally broadcasting the notification.
	Put(interface{}, ...bool) (*l8notify.L8NotificationSet, error)
	// Patch partially updates an entry, optionally broadcasting the notification.
	Patch(interface{}, ...bool) (*l8notify.L8NotificationSet, error)
	// Delete removes an entry, optionally broadcasting the notification.
	Delete(interface{}, ...bool) (*l8notify.L8NotificationSet, error)
	// Get retrieves an entry by key.
	Get(interface{}) (interface{}, error)
	// Collect applies a filter function and returns matching elements.
	Collect(f func(interface{}) (bool, interface{})) map[string]interface{}
	// ServiceName returns the service name.
	ServiceName() string
	// ServiceArea returns the service area.
	ServiceArea() byte
	// Fetch retrieves paginated results with optional query filtering.
	Fetch(int, int, IQuery) ([]interface{}, *l8api.L8MetaData)
	// AddMetadataFunc registers a metadata extraction function.
	AddMetadataFunc(string, func(interface{}) (bool, string))
}

// IReplicationCache handles replicated data storage across multiple nodes.
type IReplicationCache interface {
	// Post creates a new entry on replica N.
	Post(interface{}, int) error
	// Put replaces an entry on replica N.
	Put(interface{}, int) error
	// Patch partially updates an entry on replica N.
	Patch(interface{}, int) error
	// Delete removes an entry from replica N.
	Delete(interface{}, int) error
	// Get retrieves an entry from replica N.
	Get(interface{}, int) (interface{}, error)
}

// ITransactionConfig defines transaction behavior for a service.
type ITransactionConfig interface {
	// Replication returns true if replication is enabled.
	Replication() bool
	// ReplicationCount returns the number of replicas.
	ReplicationCount() int
	// KeyOf extracts the key from elements for transaction tracking.
	KeyOf(IElements, IResources) string
	// Voter returns true if this service participates in voting.
	Voter() bool
}

type ITSDBService interface {
	AddTSDB(notifications []*l8notify.L8TSDBNotification)
	GetTSDB(propertyId string, start, end int64) []*l8api.L8TimeSeriesPoint
}
