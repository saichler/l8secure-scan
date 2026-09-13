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

// AddMetadataFunc registers a metadata extraction function with the given name.
// The function is applied to elements to extract indexed metadata values.
// Used for building secondary indexes and enabling filtered queries.
func (this *DCache) AddMetadataFunc(name string, f func(interface{}) (bool, string)) {
	this.cache.AddMetadataFunc(name, f)
}
