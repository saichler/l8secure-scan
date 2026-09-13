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

// Collect iterates over all cached elements and applies the filter function f.
// The filter function returns (include, transformedValue) for each element.
// Returns a map of primary keys to elements that passed the filter.
func (this *DCache) Collect(f func(interface{}) (bool, interface{})) map[string]interface{} {
	return this.cache.Collect(f)
}
