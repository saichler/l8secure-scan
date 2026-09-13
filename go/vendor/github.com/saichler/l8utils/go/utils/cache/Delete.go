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

	"github.com/saichler/l8types/go/types/l8notify"
)

// Delete removes an item from the cache by extracting its key from the provided value.
// If createNotification is true, generates a Delete notification for distributed sync.
// Returns an error if the item does not exist in the cache.
func (this *Cache) Delete(v interface{}, createNotification bool) (*l8notify.L8NotificationSet, *l8notify.L8NotificationSet, error) {
	pk, uk, err := this.KeysFor(v)
	if err != nil {
		return nil, nil, err
	}
	if pk == "" {
		return nil, nil, errors.New("Interface does not contain the Key attributes")
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()

	var n *l8notify.L8NotificationSet
	var e error
	var item interface{}
	var ok bool

	if this.cacheEnabled() {
		item, ok = this.iCache.delete(pk, uk)
		if !ok {
			return n, nil, errors.New("Delete Key " + pk + " not found")
		}
	}

	if this.store != nil {
		item, e = this.store.Delete(pk)
		if e != nil {
			return n, nil, e
		}
	}

	if !createNotification {
		return n, nil, e
	}

	n, e = this.createDeleteNotification(item, pk)
	return n, this.createClientNotification(n), e
}
