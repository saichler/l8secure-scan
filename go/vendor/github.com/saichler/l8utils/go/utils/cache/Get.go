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
	"errors"

	"github.com/saichler/l8utils/go/utils/strings"
)

// Get retrieves an item from the cache by extracting its key from the provided value.
// Returns a cloned copy of the cached item to prevent external mutation.
// If the item is not found, returns an error. Thread-safe for concurrent access.
func (this *Cache) Get(v interface{}) (interface{}, error) {
	var item interface{}
	var e error
	var pk string
	var uk string
	var ok bool

	pk, uk, e = this.KeysFor(v)
	if e != nil && uk == "" {
		return item, e
	}

	if pk == "" && uk == "" {
		e = errors.New("Interface does not contain the Key attributes")
		return item, e
	}

	this.mtx.RLock()
	defer this.mtx.RUnlock()

	if this.cacheEnabled() {
		item, ok = this.iCache.get(pk, uk)
		if ok {
			itemClone := cloner.Clone(item)
			return itemClone, e
		}
	} else {
		item, e = this.store.Get(pk)
		if e == nil {
			return item, e
		}
		e = errors.New(strings.New("Cache:", this.serviceName, ":", this.serviceArea, " ", e.Error()).String())
		return item, e
	}
	e = errors.New("Not found in the cache")
	return item, e
}
