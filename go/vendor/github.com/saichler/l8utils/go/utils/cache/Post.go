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

// Post adds a new item to the cache. If the item already exists, it performs a replace
// operation instead. The input value is cloned before storage to prevent external mutation.
// If createNotification is true, generates an Add or Replace notification for distributed sync.
// Returns the delta notification, client notification (if subscribers exist), and any error.
func (this *Cache) Post(v interface{}, createNotification bool) (*l8notify.L8NotificationSet, *l8notify.L8NotificationSet, error) {
	pk, uk, err := this.KeysFor(v)
	if err != nil {
		return nil, nil, err
	}
	if pk == "" {
		return nil, nil, errors.New("Post Interface does not contain the Key attributes ")
	}

	//Make sure we clone the input value, so the caller don't have a reference to the cache element
	v = cloner.Clone(v)

	var n *l8notify.L8NotificationSet
	var e error
	var item interface{}
	var ok bool

	this.mtx.Lock()
	defer this.mtx.Unlock()

	if this.cacheEnabled() {
		item, ok = this.iCache.get(pk, uk)
	} else {
		item, e = this.store.Get(pk)
		ok = e == nil
	}

	//If the item does not exist in the cache
	if !ok {
		//First clone the value so we can use it in the notification.
		itemClone := cloner.Clone(v)
		if this.cacheEnabled() {
			//Place the value in the cache
			this.iCache.put(pk, uk, v)
		}
		if this.store != nil {
			e = this.store.Put(pk, v)
			if e != nil {
				return n, nil, e
			}
		}
		//Create the notification using the clone outside the current go routine
		if createNotification {
			n, e = this.createAddNotification(itemClone, pk)
			return n, this.createClientNotification(n), e
		}
		return n, nil, e
	}

	//From this point onwards, it the Put implementation, e.g. item exist and is full
	//Clone the instance so it won't be able to be updated outside the scope
	vClone := cloner.Clone(v)

	if this.cacheEnabled() {
		//Place the value in the cache
		this.iCache.put(pk, uk, vClone)
	}

	if this.store != nil {
		e = this.store.Put(pk, vClone)
		if e != nil {
			return n, nil, e
		}
	}

	if !createNotification {
		return n, nil, e
	}

	//From this point onward, the item is no longer in the cache
	//so we don't need to clone it

	//Create a new updater
	putUpdater := updating.NewUpdater(this.r, true, true)

	//update the item clone with the new element where nil is valid
	e = putUpdater.Update(item, vClone)
	if e != nil {
		return n, nil, e
	}

	//if there are changes, then nothing to do
	changes := putUpdater.Changes()
	if changes == nil || len(changes) == 0 {
		return nil, nil, nil
	}

	n, e = this.createReplaceNotification(item, v, pk)
	return n, this.createClientNotification(n), e
}
