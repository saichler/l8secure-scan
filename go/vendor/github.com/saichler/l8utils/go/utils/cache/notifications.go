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

// createClientNotification builds the browser-facing notification, targeted
// only at subscribers whose registered query actually matches value (the
// new/changed record for Post/Put, or the removed record for Delete) --
// never a blind broadcast to every subscriber
// (l8utils/plans/generic-websocket-change-notifications.md Phase 2). Returns
// nil (not a cn with an empty/nil AaaIds) when nobody matches: an empty
// AaaIds map is what tells WebSocketManager.OnNotification to broadcast to
// every connected client, which would be wrong here.
func (this *Cache) createClientNotification(delta *l8notify.L8NotificationSet, value interface{}) *l8notify.L8NotificationSet {
	this.r.Logger().Info("DEBUG createClientNotification modelType=", this.modelType,
		" delta-nil=", delta == nil, " hasSubscribers=", this.HasSubscribers())
	if delta == nil || !this.HasSubscribers() {
		return nil
	}
	aaaIds := this.matchingSubscriberAaaIds(value)
	this.r.Logger().Info("DEBUG createClientNotification matched aaaIds=", aaaIds)
	if len(aaaIds) == 0 {
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
	cn.AaaIds = aaaIds
	return cn
}

func (this *Cache) createClientNotificationForPatch(item interface{}, key string) *l8notify.L8NotificationSet {
	if !this.HasSubscribers() {
		return nil
	}
	aaaIds := this.matchingSubscriberAaaIds(item)
	if len(aaaIds) == 0 {
		return nil
	}
	n, e := this.createAddNotification(item, key)
	if e != nil {
		return nil
	}
	n.Type = l8notify.L8NotificationType_Patch
	n.AaaIds = aaaIds
	return n
}

// matchingSubscriberAaaIds returns the AAAId of every registered subscriber
// whose query matches value, or nil if none do.
func (this *Cache) matchingSubscriberAaaIds(value interface{}) map[string]bool {
	subs := this.Subscribers()
	this.r.Logger().Info("DEBUG matchingSubscriberAaaIds modelType=", this.modelType, " subs=", len(subs))
	if len(subs) == 0 {
		return nil
	}
	var ids map[string]bool
	for _, s := range subs {
		matched := s.Query != nil && s.Query.Match(value)
		this.r.Logger().Info("DEBUG matchingSubscriberAaaIds sub aaaId=", s.AAAId, " query-nil=", s.Query == nil, " matched=", matched)
		if s.Query == nil || !s.Query.Match(value) {
			continue
		}
		if ids == nil {
			ids = make(map[string]bool, len(subs))
		}
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
