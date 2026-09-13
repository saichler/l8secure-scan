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

// Get retrieves a single element from the cache by its primary key.
// The parameter v should contain the key field(s) to match against.
// Returns the found element and nil error, or nil and error if not found.
func (this *DCache) Get(v interface{}) (interface{}, error) {
	return this.cache.Get(v)
}
