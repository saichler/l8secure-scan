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

// Package cache provides a high-performance, thread-safe in-memory cache with optional
// persistent storage integration. It supports CRUD operations (Post, Get, Put, Patch, Delete),
// automatic change notifications, query caching with TTL-based cleanup, and clone-based
// isolation for concurrent access safety.
//
// The cache uses reflection to extract primary and unique keys from stored elements,
// enabling efficient lookups. It integrates with the Layer 8 notification system to
// broadcast state changes across distributed services.
//
// Key features:
//   - Thread-safe operations using sync.RWMutex
//   - Optional persistent storage layer integration
//   - Automatic cloning to prevent external mutation of cached data
//   - Built-in notification generation for Post, Put, Patch, and Delete operations
//   - Query result caching with configurable TTL (default 30 seconds)
//   - Statistics tracking for monitoring cache usage
package cache

import (
	"errors"
	"reflect"
	"sync"

	"github.com/saichler/l8reflect/go/reflect/cloning"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
)

var cloner = cloning.NewCloner()

// Cache is a thread-safe in-memory cache with optional persistent storage backing.
// It supports CRUD operations with automatic notification generation for distributed
// state synchronization. The cache uses cloning to ensure callers cannot mutate
// cached data directly, providing isolation and thread-safety.
type Cache struct {
	iCache               *internalCache
	mtx                  *sync.RWMutex
	cond                 *sync.Cond
	store                ifs.IStorage
	modelType            string
	primaryKeyFieldNames []string
	uniqueKeyFieldNames  []string
	r                    ifs.IResources

	notifySequence uint32
	serviceName    string
	serviceArea    byte
	cleaner        *ttlCleaner
	subs           *subscriptions
}

// NewCache creates a new Cache instance. The sampleElement is used to determine
// the type and key field names for cached items. If initElements are provided and
// the store is empty, they will be used to initialize the cache. The cache automatically
// starts a TTL cleaner goroutine for query cache maintenance.
func NewCache(sampleElement interface{}, initElements []interface{}, store ifs.IStorage, r ifs.IResources) *Cache {
	this := &Cache{}
	this.iCache = newInternalCache()
	this.mtx = &sync.RWMutex{}
	this.cond = sync.NewCond(this.mtx)
	this.store = store
	this.r = r
	this.subs = newSubscriptions()
	this.modelType = reflect.ValueOf(sampleElement).Elem().Type().Name()

	_, _, err := this.KeysFor(sampleElement)
	if err != nil {
		panic("Error in initialized elements " + err.Error())
	}

	loadedFromStore := false

	if this.store != nil {
		items := this.store.Collect(allElementsInCache)
		for _, v := range items {
			pk, uk, _ := this.KeysFor(v)
			this.iCache.put(pk, uk, v)
		}
		if len(items) > 0 {
			loadedFromStore = true
		}
	}

	if !loadedFromStore && this.store != nil {
		for _, item := range initElements {
			pk, _, er := this.KeysFor(item)
			if er != nil {
				r.Logger().Error(er.Error())
				continue
			}
			this.store.Put(pk, item)
		}
	}

	if !loadedFromStore && this.cacheEnabled() && initElements != nil {
		for _, item := range initElements {
			pk, uk, er := this.KeysFor(item)
			if er != nil {
				r.Logger().Error("#2 Init item", " error:", er.Error())
				continue
			}
			this.iCache.put(pk, uk, item)
		}
	}
	addTotalMetadata(this)

	// Start TTL cleaner for query cache
	this.cleaner = newTTLCleaner(this)
	this.cleaner.start()

	return this
}

// SetNotificationsFor configures the cache to generate notifications for the specified
// service. The serviceName and serviceArea identify the service in the notification routing.
func (this *Cache) SetNotificationsFor(serviceName string, serviceArea byte) {
	this.serviceName = serviceName
	this.serviceArea = serviceArea
}

func (this *Cache) cacheEnabled() bool {
	if this.store == nil {
		return true
	}
	return this.store.CacheEnabled()
}

// Size returns the number of items currently in the cache.
func (this *Cache) Size() int {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	return this.iCache.size()
}

// KeysFor extracts the primary key and unique key from the given item using reflection.
// It uses decorator metadata to identify which fields comprise the keys. Returns the
// primary key, unique key, and any error encountered during extraction.
func (this *Cache) KeysFor(any interface{}) (string, string, error) {
	if any == nil {
		return "", "", errors.New("Cannot get keys for nil interface")
	}

	v := reflect.ValueOf(any)
	if v.Kind() != reflect.Ptr {
		return "", "", errors.New("Cannot get keys for non-pointer interface")
	}
	v = v.Elem()

	if this.primaryKeyFieldNames == nil {
		node, _, err := this.r.Introspector().Decorators().NodeFor(any)
		if err != nil {
			return "", "", err
		}
		this.primaryKeyFieldNames, err = this.r.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_Primary)
		if err != nil {
			return "", "", err
		}
		this.uniqueKeyFieldNames, err = this.r.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_Unique)
	}

	pkValue, err := this.r.Introspector().Decorators().KeyForValue(this.primaryKeyFieldNames, v, this.modelType, true)
	ukValue, _ := this.r.Introspector().Decorators().KeyForValue(this.uniqueKeyFieldNames, v, this.modelType, false)
	return pkValue, ukValue, err
}

func allElementsInCache(i interface{}) (bool, interface{}) {
	return true, i
}

// ServiceName returns the service name configured for notifications.
func (this *Cache) ServiceName() string {
	return this.serviceName
}

// ServiceArea returns the service area configured for notifications.
func (this *Cache) ServiceArea() byte {
	return this.serviceArea
}

// Source returns the local UUID identifying this cache instance as the notification source.
func (this *Cache) Source() string {
	return this.r.SysConfig().LocalUuid
}

// ModelType returns the type name of elements stored in this cache.
func (this *Cache) ModelType() string {
	return this.modelType
}

// RegisterSubscription registers a user (identified by AAAId) for real-time
// change notifications on this cache's model type. Called by the service
// handler after Fetch when the query has Register=true.
func (this *Cache) RegisterSubscription(aaaId string, queryHash int32, queryText string) {
	this.subs.register(&Subscription{
		AAAId:     aaaId,
		QueryHash: queryHash,
		QueryText: queryText,
	})
}

// UnregisterSubscription removes a subscription for the given AAAId.
// Called when a WebSocket disconnects or a user navigates away.
func (this *Cache) UnregisterSubscription(aaaId string) {
	this.subs.unregister(aaaId)
}

// Subscribers returns all active subscriptions for this cache's model type.
// Called by the service handler after a mutation to determine which tokens to notify.
func (this *Cache) Subscribers() []*Subscription {
	return this.subs.subscribers()
}

// HasSubscribers returns true if any tokens are subscribed to this cache.
func (this *Cache) HasSubscribers() bool {
	return this.subs.hasSubscribers()
}

// EvictStaleSubscriptions removes subscriptions not refreshed within ttlSeconds.
// Returns the number evicted. Used for testing; in production the TTL cleaner calls this automatically.
func (this *Cache) EvictStaleSubscriptions(ttlSeconds int64) int {
	return this.subs.evictStale(ttlSeconds)
}

// Close stops the TTL cleaner goroutine and releases cache resources.
// This should be called when the cache is no longer needed.
func (this *Cache) Close() {
	if this.cleaner != nil {
		this.cleaner.stop()
	}
}
