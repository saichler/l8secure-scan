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
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// Fetch retrieves a paginated subset of elements from the cache.
// Parameters: start (offset index), blockSize (page size), q (query filter).
// Returns matching elements and metadata containing total counts.
func (this *DCache) Fetch(start, blockSize int, q ifs.IQuery) ([]interface{}, *l8api.L8MetaData) {
	return this.cache.Fetch(start, blockSize, q)
}
