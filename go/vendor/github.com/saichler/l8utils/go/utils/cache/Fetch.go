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
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// Fetch retrieves a paginated slice of items from the cache matching the query criteria.
// The start parameter specifies the starting index and blockSize determines the page size.
// Results are cloned to prevent external mutation. Metadata is returned only on the first page.
// Query results may be cached internally with TTL-based expiration for performance.
func (this *Cache) Fetch(start, blockSize int, q ifs.IQuery) ([]interface{}, *l8api.L8MetaData) {
	this.mtx.Lock()
	defer this.mtx.Unlock()

	// Register (or refresh) this query as a live subscription whenever the
	// caller opted in with register=true -- unconditional on every matching
	// call, not just the first, so lastSeen keeps refreshing and an actively
	// re-fetching client never goes stale against TTL eviction
	// (l8utils/plans/generic-websocket-change-notifications.md Phase 3).
	this.r.Logger().Info("DEBUG Cache.Fetch modelType=", this.modelType, " register=", q.Register(),
		" aaaId=", q.AAAId(), " text=", q.Text())
	if q.Register() && q.AAAId() != "" {
		this.registerSubscription(q.AAAId(), q)
		this.r.Logger().Info("DEBUG Cache.Fetch registered subscription for aaaId=", q.AAAId())
	}

	values, metadata := this.iCache.fetch(start, blockSize, q, this.r)

	// Aggregate queries return empty slice with results in metadata
	if q.IsAggregate() {
		metadataClone := cloner.Clone(metadata).(*l8api.L8MetaData)
		return values, metadataClone
	}

	result := make([]interface{}, len(values))
	for i, v := range values {
		result[i] = cloner.Clone(v)
	}

	if q.Page() == 0 {
		metadataClone := cloner.Clone(metadata).(*l8api.L8MetaData)
		return result, metadataClone
	}
	return result, nil
}
