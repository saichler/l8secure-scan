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

	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8types/go/types/l8notify"
)

// Patch applies partial updates to an existing cache item. Unlike Put, Patch only modifies
// the fields present in the input value, preserving other existing fields. If the item
// doesn't exist, it creates a new entry. If createNotification is true, generates an
// Update notification containing only the changed properties for distributed sync.
func (this *Cache) Patch(v interface{}, createNotification bool) (*l8notify.L8NotificationSet, *l8notify.L8NotificationSet, error) {
	pk, uk, err := this.KeysFor(v)
	if err != nil {
		return nil, nil, errors.New("Patch error " + err.Error())
	}
	if pk == "" {
		return nil, nil, errors.New("Patch Interface does not contain the Key attributes")
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()
	var n *l8notify.L8NotificationSet
	var e error
	var item interface{}
	var ok bool

	if this.cacheEnabled() {
		item, ok = this.iCache.get(pk, uk)
	} else {
		item, e = this.store.Get(pk)
		ok = e == nil
	}

	//If the item does not exist in the cache
	if !ok {
		//Clone the value for the cache
		vClone := cloner.Clone(v)

		if this.cacheEnabled() {
			//Place the new Item clone in the cache
			this.iCache.put(pk, uk, vClone)
		}

		if this.store != nil {
			//place the new item clone in the store
			e = this.store.Put(pk, vClone)
		}

		if !createNotification {
			return n, nil, e
		}

		//Clone the value for the notification
		//itemClone := cloner.Clone(v)
		n, e = this.createAddNotification(vClone, pk)
		return n, this.createClientNotification(n), e
	}

	//Create a new updater
	patchUpdater := updating.NewUpdater(this.r, false, false)
	//Dry update to detect changes without mutating the cached item
	e = patchUpdater.DryUpdate(item, v)
	if e != nil {
		return n, nil, e
	}

	//if there are no changes, then nothing to do
	changes := patchUpdater.Changes()
	if changes == nil {
		return n, nil, e
	}

	//Apply the changes to the existing item in the cache
	for _, change := range changes {
		change.Apply(item)
	}

	this.iCache.stampChanged()

	if this.store != nil {
		e = this.store.Put(pk, item)
	}

	if !createNotification {
		return n, nil, e
	}

	n, e = this.createUpdateNotification(changes, pk)
	cn := this.createClientNotificationForPatch(item, pk)
	return n, cn, e
}
