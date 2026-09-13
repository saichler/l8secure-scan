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

package dcache

import (
	"github.com/saichler/l8types/go/types/l8notify"
)

// Post creates or replaces an element in the distributed cache. Returns a notification
// set describing the changes made. The optional sourceNotification parameter, when true,
// suppresses notification generation (used during replication to avoid loops).
func (this *DCache) Post(v interface{}, sourceNotification ...bool) (*l8notify.L8NotificationSet, error) {
	createNotification := !(sourceNotification != nil && len(sourceNotification) > 0 && sourceNotification[0])
	n, _, e := this.cache.Post(v, createNotification)
	if this.listener != nil && createNotification && e == nil && n != nil {
		this.nQueue.Add(n)
	}
	return n, e
}
