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

// Put performs a full replacement update of an element in the cache.
// Delegates to Post since the underlying cache handles both create and replace.
// The optional sourceNotification parameter suppresses notification generation.
func (this *DCache) Put(v interface{}, sourceNotification ...bool) (*l8notify.L8NotificationSet, error) {
	return this.Post(v, sourceNotification...)
}
