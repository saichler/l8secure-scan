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
	"github.com/saichler/l8types/go/types/l8notify"
)

// Put replaces an existing item in the cache or adds it if not present.
// This is an alias for Post as they share the same implementation logic.
// If createNotification is true, generates an appropriate notification for distributed sync.
func (this *Cache) Put(v interface{}, createNotification bool) (*l8notify.L8NotificationSet, *l8notify.L8NotificationSet, error) {
	//Seems that the post is handling also a put situation, where the item is replaced
	return this.Post(v, createNotification)
}
