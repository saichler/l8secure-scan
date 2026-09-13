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
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8utils/go/utils/notify"
)

func (this *Cache) createNotificationSet(t l8notify.L8NotificationType, key string, changeCount int) *l8notify.L8NotificationSet {
	defer func() { this.notifySequence++ }()
	return notify.CreateNotificationSet(t, this.serviceName, key, this.serviceArea, this.modelType, this.Source(), changeCount, this.notifySequence)
}

func (this *Cache) createClientNotification(delta *l8notify.L8NotificationSet) *l8notify.L8NotificationSet {
	if delta == nil || !this.HasSubscribers() {
		return nil
	}
	cn := &l8notify.L8NotificationSet{}
	cn.ServiceName = delta.ServiceName
	cn.ServiceArea = delta.ServiceArea
	cn.ModelType = delta.ModelType
	cn.ModelKey = delta.ModelKey
	cn.Type = delta.Type
	cn.Source = delta.Source
	cn.NotificationList = delta.NotificationList
	cn.AaaIds = this.subscriberAaaIds()
	return cn
}

func (this *Cache) createClientNotificationForPatch(item interface{}, key string) *l8notify.L8NotificationSet {
	if !this.HasSubscribers() {
		return nil
	}
	n, e := this.createAddNotification(item, key)
	if e != nil {
		return nil
	}
	n.Type = l8notify.L8NotificationType_Patch
	n.AaaIds = this.subscriberAaaIds()
	return n
}

func (this *Cache) subscriberAaaIds() map[string]bool {
	subs := this.Subscribers()
	if len(subs) == 0 {
		return nil
	}
	ids := make(map[string]bool, len(subs))
	for _, s := range subs {
		ids[s.AAAId] = true
	}
	return ids
}

func (this *Cache) createAddNotification(any interface{}, key string) (*l8notify.L8NotificationSet, error) {
	defer func() { this.notifySequence++ }()
	return notify.CreateAddNotification(any, this.serviceName, key, this.serviceArea, this.modelType, this.Source(), 1, this.notifySequence)
}

func (this *Cache) createReplaceNotification(old, new interface{}, key string) (*l8notify.L8NotificationSet, error) {
	defer func() { this.notifySequence++ }()
	return notify.CreateReplaceNotification(old, new, this.serviceName, key, this.serviceArea, this.modelType, this.Source(), 1, this.notifySequence)
}

func (this *Cache) createDeleteNotification(any interface{}, key string) (*l8notify.L8NotificationSet, error) {
	defer func() { this.notifySequence++ }()
	return notify.CreateDeleteNotification(any, this.serviceName, key, this.serviceArea, this.modelType, this.Source(), 1, this.notifySequence)
}

func (this *Cache) createUpdateNotification(changes []*updating.Change, key string) (*l8notify.L8NotificationSet, error) {
	defer func() { this.notifySequence++ }()
	return notify.CreateUpdateNotification(changes, this.serviceName, key, this.serviceArea, this.modelType, this.Source(), len(changes), this.notifySequence)
}
