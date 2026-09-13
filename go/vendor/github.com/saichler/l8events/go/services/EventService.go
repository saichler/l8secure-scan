/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package services

import (
	"errors"
	"time"

	common "github.com/saichler/l8common/go/common"
	"github.com/saichler/l8types/go/ifs"
	evt "github.com/saichler/l8types/go/types/l8events"
)

const (
	EventsServiceName = "Events"
	EventsServiceArea = byte(76)
)

func ActivateEvents(creds, dbname string, vnic ifs.IVNic) {
	common.ActivateService(common.ServiceConfig{
		ServiceName: EventsServiceName, ServiceArea: EventsServiceArea,
		PrimaryKey: "EventId", Voter: true, NonUniqueKeys: []string{"OccurredAt"},
		Replication: boolPtr(false), Callback: &EventCallback{},
	}, &evt.EventRecord{}, &evt.EventRecordList{}, creds, dbname, vnic)
}

func boolPtr(b bool) *bool { return &b }

type EventCallback struct{}

func (this *EventCallback) Before(elem interface{}, action ifs.Action, isNotification bool, vnic ifs.IVNic) (interface{}, bool, error) {
	if action == ifs.GET {
		return nil, true, nil
	}
	event, ok := elem.(*evt.EventRecord)
	if !ok {
		return nil, true, errors.New("invalid event type")
	}

	switch action {
	case ifs.POST:
		common.GenerateID(&event.EventId)
		event.ReceivedAt = time.Now().Unix()
		if event.OccurredAt == 0 {
			event.OccurredAt = event.ReceivedAt
		}
		if event.State == evt.EventState_EVENT_STATE_UNSPECIFIED {
			event.State = evt.EventState_EVENT_STATE_NEW
		}
		return event, true, nil
	case ifs.PUT:
		return nil, true, errors.New("events are immutable, PUT is not allowed")
	case ifs.PATCH:
		return event, true, nil
	}

	return nil, true, nil
}

func (this *EventCallback) After(elem interface{}, action ifs.Action, notify bool, vnic ifs.IVNic) (interface{}, bool, error) {
	return nil, true, nil
}
