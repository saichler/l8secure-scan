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

// Package manager provides service management functionality for the l8services framework.
// This file handles notification routing and delegation to registered service handlers.
package manager

import (
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8utils/go/utils/notify"
	"reflect"
	"strconv"
)

// Notify processes incoming notifications and routes them to the appropriate service handler.
// It implements the notification handling interface for the ServiceManager.
//
// Parameters:
//   - pb: The elements container holding the notification data (expected to contain *l8notify.L8NotificationSet)
//   - vnic: The virtual network interface card for network operations
//   - msg: The message containing source, service name, and service area information
//   - isTransaction: Flag indicating if this notification is part of a transaction
//
// Returns:
//   - ifs.IElements: The response from the service handler, or an error if processing fails
//
// The function performs the following steps:
//  1. Ignores notifications from the local node (prevents self-notification loops)
//  2. Looks up the appropriate service handler based on service name and area
//  3. Handles failed messages by delegating to the handler's Failed method
//  4. Extracts the notification item and wraps it in elements
//  5. Calls Before/After hooks if the handler implements IServiceHandlerModifier
//  6. Delegates to the appropriate handler method based on notification type
func (this *ServiceManager) Notify(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message, isTransaction bool) ifs.IElements {
	if vnic.Resources().SysConfig().LocalUuid == msg.Source() {
		return object.New(nil, nil)
	}
	notification := pb.Element().(*l8notify.L8NotificationSet)
	h, ok := this.services.get(notification.ServiceName, byte(notification.ServiceArea))
	if !ok {
		return object.NewError("Cannot find active handler for service " + msg.ServiceName() +
			" area " + strconv.Itoa(int(msg.ServiceArea())))
	}
	tsdbHandler, isTSDB := h.(ifs.ITSDBService)
	if msg != nil && msg.FailMessage() != "" {
		return h.Failed(pb, vnic, msg)
	}
	item, tsdbNotifications, err := notify.ItemOf(notification, this.resources, isTSDB)
	if err != nil {
		return object.NewError(err.Error())
	}

	if isTSDB && tsdbNotifications != nil {
		tsdbHandler.AddTSDB(tsdbNotifications)
	}

	if !reflect.ValueOf(item).IsZero() {
		npb := object.NewNotify(item)
		resp := this.delegateNotification(notification.ServiceName, notification.Type, h, npb, item, vnic)
		return resp
	}
	return object.New(nil, item)
}

// delegateNotification routes a notification to the appropriate handler method based on the notification type.
// It acts as a dispatcher that maps L8NotificationType values to corresponding handler operations.
//
// Parameters:
//   - serviceName: The name of the service receiving the notification
//   - notificationType: The type of notification (Post, Put, Patch, or Delete)
//   - h: The service handler that will process the notification
//   - npb: The elements container with the notification payload
//   - item: The raw notification item (used for special handling like health service node deletion)
//   - vnic: The virtual network interface card for network operations
//
// Returns:
//   - ifs.IElements: The response from the handler method, or an error for invalid notification types
//
// Special behavior:
//   - For Delete notifications on the health service, it triggers onNodeDelete to clean up node-related data
func (this *ServiceManager) delegateNotification(serviceName string, notificationType l8notify.L8NotificationType, h ifs.IServiceHandler,
	npb ifs.IElements, item interface{}, vnic ifs.IVNic) ifs.IElements {
	switch notificationType {
	case l8notify.L8NotificationType_Post:
		return h.Post(npb, vnic)
	case l8notify.L8NotificationType_Put:
		return h.Put(npb, vnic)
	case l8notify.L8NotificationType_Patch:
		return h.Patch(npb, vnic)
	case l8notify.L8NotificationType_Delete:
		result := h.Delete(npb, vnic)
		if serviceName == health.ServiceName {
			this.onNodeDelete(item.(*l8health.L8Health).AUuid)
		}
		return result
	default:
		return object.NewError("invalid notification type, ignoring")
	}
}

// NotifyTypeToAction converts an L8NotificationType to its corresponding ifs.Action.
// This utility function provides a mapping between the notification type enum and the action enum,
// allowing consistent action representation across the system.
//
// Parameters:
//   - notifyType: The L8NotificationType to convert
//
// Returns:
//   - ifs.Action: The corresponding action (POST, PUT, PATCH, DELETE), or 0 for unknown types
//
// Mapping:
//   - L8NotificationType_Post   -> ifs.POST
//   - L8NotificationType_Put    -> ifs.PUT
//   - L8NotificationType_Patch  -> ifs.PATCH
//   - L8NotificationType_Delete -> ifs.DELETE
func NotifyTypeToAction(notifyType l8notify.L8NotificationType) ifs.Action {
	switch notifyType {
	case l8notify.L8NotificationType_Post:
		return ifs.POST
	case l8notify.L8NotificationType_Put:
		return ifs.PUT
	case l8notify.L8NotificationType_Patch:
		return ifs.PATCH
	case l8notify.L8NotificationType_Delete:
		return ifs.DELETE
	default:
		return 0
	}
}
