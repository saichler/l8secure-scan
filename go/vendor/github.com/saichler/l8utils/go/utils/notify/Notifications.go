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

// Package notify provides utilities for creating and handling distributed state change
// notifications. It supports various notification types (Add, Update, Delete, Replace)
// that can be serialized and transmitted across services for state synchronization.
//
// Notifications use protocol buffer serialization for efficient transmission and
// include property-level change tracking for partial updates.
//
// Key features:
//   - Create notifications for Post, Put, Patch, and Delete operations
//   - Property-level change tracking for fine-grained updates
//   - Protocol buffer integration for cross-service communication
//   - Sequence numbering for ordering and deduplication
package notify

import (
	"errors"
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8reflect/go/reflect/updating"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
)

// CreateNotificationSet creates a new notification set container with routing metadata.
// The notification set holds one or more individual notifications for a single entity.
func CreateNotificationSet(t l8notify.L8NotificationType, serviceName, key string, serviceArea byte, modelType, source string,
	changeCount int, notifySequence uint32) *l8notify.L8NotificationSet {
	notificationSet := &l8notify.L8NotificationSet{}
	notificationSet.ServiceName = serviceName
	notificationSet.ServiceArea = int32(serviceArea)
	notificationSet.ModelType = modelType
	notificationSet.Source = source
	notificationSet.Type = t
	notificationSet.NotificationList = make([]*l8notify.L8Notification, changeCount)
	notificationSet.Sequence = notifySequence
	notificationSet.ModelKey = key
	return notificationSet
}

// CreateAddNotification creates a Post notification for a newly added entity.
// The entity is serialized and stored as the NewValue in the notification.
func CreateAddNotification(any interface{}, serviceName, key string, serviceArea byte, modelType, source string, changeCount int, notifySequence uint32) (*l8notify.L8NotificationSet, error) {
	notificationSet := CreateNotificationSet(l8notify.L8NotificationType_Post, serviceName, key, serviceArea, modelType, source, changeCount, notifySequence)
	obj := object.NewEncode()
	err := obj.Add(any)
	if err != nil {
		return nil, err
	}
	n := &l8notify.L8Notification{}
	n.NewValue = obj.Data()
	notificationSet.NotificationList[0] = n
	return notificationSet, nil
}

// CreateReplaceNotification creates a Put notification for a fully replaced entity.
// Both the old and new versions are serialized for the receiver to process.
func CreateReplaceNotification(old, new interface{}, serviceName, key string, serviceArea byte, modelType, source string, changeCount int, notifySequence uint32) (*l8notify.L8NotificationSet, error) {
	notificationSet := CreateNotificationSet(l8notify.L8NotificationType_Put, serviceName, key, serviceArea, modelType, source, 1, notifySequence)
	oldObj := object.NewEncode()
	err := oldObj.Add(old)
	if err != nil {
		return nil, err
	}

	newObj := object.NewEncode()
	err = newObj.Add(new)
	if err != nil {
		return nil, err
	}

	n := &l8notify.L8Notification{}
	n.OldValue = oldObj.Data()
	n.NewValue = newObj.Data()
	notificationSet.NotificationList[0] = n
	return notificationSet, nil
}

// CreateDeleteNotification creates a Delete notification for a removed entity.
// The deleted entity is serialized as the OldValue for reference.
func CreateDeleteNotification(any interface{}, serviceName, key string, serviceArea byte, modelType, source string, changeCount int, notifySequence uint32) (*l8notify.L8NotificationSet, error) {
	notificationSet := CreateNotificationSet(l8notify.L8NotificationType_Delete, serviceName, key, serviceArea, modelType, source, 1, notifySequence)
	obj := object.NewEncode()
	err := obj.Add(any)
	if err != nil {
		return nil, err
	}
	n := &l8notify.L8Notification{}
	n.OldValue = obj.Data()
	notificationSet.NotificationList[0] = n
	return notificationSet, nil
}

// CreateUpdateNotification creates a Patch notification with property-level changes.
// Each change includes the property ID and old/new values for that specific property.
func CreateUpdateNotification(changes []*updating.Change, serviceName, key string, serviceArea byte, modelType, source string, changeCount int, notifySequence uint32) (*l8notify.L8NotificationSet, error) {
	notificationSet := CreateNotificationSet(l8notify.L8NotificationType_Patch, serviceName, key, serviceArea, modelType, source, changeCount, notifySequence)
	for i, change := range changes {
		n := &l8notify.L8Notification{}
		n.PropertyId = change.PropertyId()
		if change.OldValue() != nil {
			obj := object.NewEncode()
			err := obj.Add(change.OldValue())
			if err != nil {
				return nil, err
			}
			n.OldValue = obj.Data()
		}
		if change.NewValue() != nil {
			obj := object.NewEncode()
			err := obj.Add(change.NewValue())
			if err != nil {
				return nil, err
			}
			n.NewValue = obj.Data()
		}
		notificationSet.NotificationList[i] = n
	}
	return notificationSet, nil
}

// ItemOf extracts the entity from a notification set by deserializing the appropriate
// value based on notification type. For Patch notifications, reconstructs the entity
// from individual property changes.
func ItemOf(n *l8notify.L8NotificationSet, resources ifs.IResources, isTSDBService bool) (interface{}, []*l8notify.L8TSDBNotification, error) {
	switch n.Type {
	case l8notify.L8NotificationType_Put:
		fallthrough
	case l8notify.L8NotificationType_Post:
		obj := object.NewDecode(n.NotificationList[0].NewValue, 0, resources.Registry())
		v, e := obj.Get()
		return v, nil, e
	case l8notify.L8NotificationType_Delete:
		obj := object.NewDecode(n.NotificationList[0].OldValue, 0, resources.Registry())
		v, e := obj.Get()
		return v, nil, e
	case l8notify.L8NotificationType_Patch:

		info, err := resources.Registry().Info(n.ModelType)
		if err != nil {
			return nil, nil, err
		}
		root, err := info.NewInstance()
		if err != nil {
			return nil, nil, err
		}
		var tsdb []*l8notify.L8TSDBNotification

		for _, notif := range n.NotificationList {
			var value interface{}
			if notif.NewValue != nil {
				obj := object.NewDecode(notif.NewValue, 0, resources.Registry())
				v, e1 := obj.Get()
				if e1 != nil {
					return nil, nil, e1
				}
				value = v
			}
			if isTSDBService {
				l8point, ok := value.(*l8api.L8TimeSeriesPoint)
				if ok {
					newPoint := &l8notify.L8TSDBNotification{
						PropertyId: notif.PropertyId,
						Point:      l8point,
					}
					if tsdb == nil {
						tsdb = []*l8notify.L8TSDBNotification{newPoint}
					} else {
						tsdb = append(tsdb, newPoint)
					}
					continue
				}
			}

			p, e := properties.PropertyOf(notif.PropertyId, resources)
			_, _, e = p.Set(root, value)
			if e != nil {
				return nil, nil, e
			}
		}
		return root, tsdb, nil
	}
	return nil, nil, errors.New("Unknown notification type")
}
