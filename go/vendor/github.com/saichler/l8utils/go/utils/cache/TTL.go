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

package cache

import (
	"sync/atomic"
	"time"
)

const (
	DefaultQueryTTL      = 30 // seconds
	DefaultCleanInterval = 10 // seconds
)

// QueryCount returns the number of cached queries (for testing/monitoring)
func (this *Cache) QueryCount() int {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	return len(this.iCache.queries)
}

// CleanupQueriesNow manually triggers query cleanup with the given TTL (for testing)
func (this *Cache) CleanupQueriesNow(ttlSeconds int64) int {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	return this.iCache.cleanupQueries(ttlSeconds)
}

// cleanupQueries removes queries that haven't been used for ttlSeconds
func (this *internalCache) cleanupQueries(ttlSeconds int64) int {
	now := time.Now().Unix()
	removed := 0
	for hash, q := range this.queries {
		if now-atomic.LoadInt64(&q.lastUsed) > ttlSeconds {
			delete(this.queries, hash)
			removed++
		}
	}
	return removed
}

type ttlCleaner struct {
	cache    *Cache
	ttl      int64
	interval time.Duration
	running  atomic.Bool
	stopCh   chan struct{}
}

func newTTLCleaner(cache *Cache) *ttlCleaner {
	return &ttlCleaner{
		cache:    cache,
		ttl:      DefaultQueryTTL,
		interval: DefaultCleanInterval * time.Second,
		stopCh:   make(chan struct{}),
	}
}

func (t *ttlCleaner) start() {
	if t.running.Swap(true) {
		return // already running
	}
	go t.run()
}

func (t *ttlCleaner) stop() {
	if t.running.Swap(false) {
		close(t.stopCh)
	}
}

func (t *ttlCleaner) run() {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.cache.mtx.Lock()
			removed := t.cache.iCache.cleanupQueries(t.ttl)
			t.cache.mtx.Unlock()
			if removed > 0 && t.cache.r != nil {
				t.cache.r.Logger().Debug("TTL cleanup removed", " queries:", removed)
			}
			evicted := t.cache.subs.evictStale(DefaultSubscriptionTTL)
			if evicted > 0 && t.cache.r != nil {
				t.cache.r.Logger().Debug("TTL cleanup evicted", " subscriptions:", evicted)
			}
		case <-t.stopCh:
			return
		}
	}
}
