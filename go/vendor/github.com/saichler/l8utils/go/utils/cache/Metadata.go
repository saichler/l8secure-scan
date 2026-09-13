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

const (
	Total = "Total"
)

func (this *Cache) AddMetadataFunc(name string, f func(interface{}) (bool, string)) {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.iCache.addMetadataFunc(name, f)
}

func (this *Cache) Metadata() map[string]float64 {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	result := make(map[string]float64)
	if this.iCache.metadataFunc != nil {
		for _, elem := range this.iCache.cache {
			for name, f := range this.iCache.metadataFunc {
				ok1, _ := f(elem)
				if ok1 {
					result[name]++
				}
			}
		}
	}
	return result
}

func addTotalMetadata(cache *Cache) {
	cache.AddMetadataFunc(Total, TotalStat)
}

func TotalStat(any interface{}) (bool, string) {
	return true, ""
}
