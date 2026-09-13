/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ifs

import (
	"github.com/saichler/l8types/go/types/l8notify"
)

// INotify provides the API for dispatching notifications across the Layer 8 system.
// Implementations route requests to the Notify service via the VNic — mirrors IEvents exactly.
type INotify interface {
	// Send posts a notification for dispatch (email/webhook/Slack/etc.) and returns the
	// delivery outcome. attributes is forwarded onto the persisted NotifyRecord.
	Send(channel l8notify.NotifyChannel, endpoint, subject, message string,
		attributes map[string]string) *l8notify.DeliveryResult
	// Set the VNIC
	SetVNic(IVNic)
}
