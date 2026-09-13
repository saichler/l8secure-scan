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

// Collect iterates over all cached items and applies the filter function to each.
// The filter function receives a cloned copy of each item and returns a boolean
// indicating whether to include it and an optional transformed value.
// Returns a map of primary keys to the filtered/transformed values.
func (this *Cache) Collect(f func(interface{}) (bool, interface{})) map[string]interface{} {
	result := map[string]interface{}{}
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	if this.cacheEnabled() {
		for k, v := range this.iCache.cache {
			vClone := cloner.Clone(v)
			ok, elem := f(vClone)
			if ok {
				result[k] = elem
			}
		}
		return result
	}
	return this.store.Collect(f)
}
