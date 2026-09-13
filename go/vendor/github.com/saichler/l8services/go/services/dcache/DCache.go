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

// Package dcache provides a distributed cache implementation for the Layer 8 services framework.
// It wraps the core cache with notification support and listener integration for property
// change events, enabling distributed data synchronization across service instances.
package dcache

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8utils/go/utils/cache"
	"github.com/saichler/l8utils/go/utils/queues"
)

// DCache implements the IDistributedCache interface, providing a thread-safe
// distributed cache with notification support. It wraps the core cache and
// manages a notification queue for broadcasting property change events.
type DCache struct {
	cache     *cache.Cache
	listener  ifs.IServiceCacheListener
	resources ifs.IResources
	nQueue    *queues.Queue
	running   bool
}

// NewDistributedCache creates a new distributed cache instance without persistent storage.
// Parameters: serviceName (identifier), serviceArea (zone/shard), sample (type template),
// initElements (initial data), listener (notification receiver), resources (system resources).
func NewDistributedCache(serviceName string, serviceArea byte, sample interface{}, initElements []interface{},
	listener ifs.IServiceCacheListener, resources ifs.IResources) ifs.IDistributedCache {
	return NewDistributedCacheWithStorage(serviceName, serviceArea, sample, initElements, listener, resources, nil)
}

// NewDistributedCacheNoSync creates a distributed cache without synchronization features.
// Functionally identical to NewDistributedCache; exists for API clarity.
func NewDistributedCacheNoSync(serviceName string, serviceArea byte, sample interface{}, initElements []interface{},
	listener ifs.IServiceCacheListener, resources ifs.IResources) ifs.IDistributedCache {
	return NewDistributedCacheWithStorage(serviceName, serviceArea, sample, initElements, listener, resources, nil)
}

// NewDistributedCacheWithStorage creates a distributed cache with optional persistent storage.
// Initializes the cache, sets up the notification queue, and starts the notification
// processing goroutine if a listener is provided.
func NewDistributedCacheWithStorage(serviceName string, serviceArea byte, sample interface{}, initElements []interface{},
	listener ifs.IServiceCacheListener, resources ifs.IResources, store ifs.IStorage) ifs.IDistributedCache {
	this := &DCache{}
	this.cache = cache.NewCache(sample, initElements, store, resources)
	this.listener = listener
	this.resources = resources
	this.cache.SetNotificationsFor(serviceName, serviceArea)
	this.nQueue = queues.NewQueue("Nitifiction Queue", 50000)
	this.running = true
	if this.listener != nil {
		go this.processNotificationQueue()
	}
	return this
}

// processNotificationQueue runs as a background goroutine that continuously
// dequeues notification sets and forwards them to the registered listener.
// Stops when this.running becomes false.
func (this *DCache) processNotificationQueue() {
	for this.running {
		set, ok := this.nQueue.Next().(*l8notify.L8NotificationSet)
		if ok {
			this.listener.PropertyChangeNotification(set)
		}
	}
}

// Shutdown stops the cache by setting running to false and adding a nil
// element to unblock the notification queue processing goroutine.
func (this *DCache) Shutdown() {
	this.running = false
	this.nQueue.Add(nil)
}

// ServiceName returns the name of this distributed cache service.
func (this *DCache) ServiceName() string {
	return this.cache.ServiceName()
}

// ServiceArea returns the service area identifier (zone/shard) for this cache.
func (this *DCache) ServiceArea() byte {
	return this.cache.ServiceArea()
}

// Size returns the number of elements currently stored in the cache.
func (this *DCache) Size() int {
	return this.cache.Size()
}

// Cache returns the underlying Cache instance for direct access when needed.
func (this *DCache) Cache() *cache.Cache {
	return this.cache
}
