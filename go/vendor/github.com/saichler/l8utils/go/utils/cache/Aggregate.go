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

// fetchAggregate handles aggregate queries by computing results in-memory.
// It collects all cached objects, filters by WHERE, computes aggregates
// (with GROUP BY), applies HAVING, and packs results into metadata.
// Returns an empty slice and metadata with aggregate results in Counts.
func (this *internalCache) fetchAggregate(q ifs.IQuery) ([]interface{}, *l8api.L8MetaData) {
	// Collect all cached objects
	items := make([]interface{}, 0, len(this.cache))
	for _, v := range this.cache {
		items = append(items, v)
	}

	// Filter by WHERE criteria
	filtered := q.Filter(items, false)

	// Compute aggregates (handles GROUP BY internally)
	groups := q.Aggregate(filtered)

	// Filter by HAVING clause
	groups = filterByHaving(groups, q.Having())

	// Pack results into metadata
	metadata := newMetadata()
	PackAggregateResults(groups, q.Aggregates(), q.GroupBy(), metadata)

	return []interface{}{}, metadata
}
