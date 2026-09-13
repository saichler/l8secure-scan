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
	"sync"
	"time"
)

const DefaultSubscriptionTTL = 300 // 5 minutes

// Subscription represents a user's interest in change notifications
// for a specific model type. Keyed by AAAId (authenticated user identity).
type Subscription struct {
	AAAId     string
	QueryHash int32
	QueryText string
	lastSeen  int64
}

// subscriptions tracks which browser tabs are subscribed to change notifications
// for a single Cache instance (one model type). Thread-safe for concurrent access.
type subscriptions struct {
	mu   sync.RWMutex
	subs map[string]*Subscription
}

func newSubscriptions() *subscriptions {
	return &subscriptions{
		subs: make(map[string]*Subscription),
	}
}

func (this *subscriptions) register(sub *Subscription) {
	this.mu.Lock()
	defer this.mu.Unlock()
	sub.lastSeen = time.Now().Unix()
	this.subs[sub.AAAId] = sub
}

func (this *subscriptions) unregister(aaaId string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	delete(this.subs, aaaId)
}

func (this *subscriptions) subscribers() []*Subscription {
	this.mu.RLock()
	defer this.mu.RUnlock()
	result := make([]*Subscription, 0, len(this.subs))
	for _, sub := range this.subs {
		cp := *sub
		result = append(result, &cp)
	}
	return result
}

func (this *subscriptions) hasSubscribers() bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return len(this.subs) > 0
}

// evictStale removes subscriptions not refreshed within ttlSeconds.
// Returns the number of evicted subscriptions.
func (this *subscriptions) evictStale(ttlSeconds int64) int {
	this.mu.Lock()
	defer this.mu.Unlock()
	now := time.Now().Unix()
	removed := 0
	for aaaId, sub := range this.subs {
		if now-sub.lastSeen > ttlSeconds {
			delete(this.subs, aaaId)
			removed++
		}
	}
	return removed
}
