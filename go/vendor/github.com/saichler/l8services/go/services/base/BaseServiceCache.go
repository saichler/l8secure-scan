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

package base

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

// Collect iterates over all cached elements and applies the filter function f.
// The filter function returns (include, transformedValue) for each element.
// Returns a map of primary keys to elements that passed the filter.
func (this *BaseService) Collect(f func(interface{}) (bool, interface{})) map[string]interface{} {
	if this.cache == nil {
		return nil
	}
	return this.cache.Collect(f)
}

// All returns a map of all cached elements keyed by their primary keys.
// Returns nil if the cache is not initialized.
func (this *BaseService) All() map[string]interface{} {
	if this.cache == nil {
		return nil
	}
	return this.cache.Collect(all)
}

// ServiceName returns the name of the service as registered in the cache.
// Returns empty string if the cache is not initialized.
func (this *BaseService) ServiceName() string {
	if this.cache == nil {
		return ""
	}
	return this.cache.ServiceName()
}

// ServiceArea returns the service area identifier (zone/shard) for this service.
// Returns 0 if the cache is not initialized.
func (this *BaseService) ServiceArea() byte {
	if this.cache == nil {
		return 0
	}
	return this.cache.ServiceArea()
}

// Size returns the number of elements currently stored in the cache.
// Returns 0 if the cache is not initialized.
func (this *BaseService) Size() int {
	if this.cache == nil {
		return 0
	}
	return this.cache.Size()
}

// Fetch retrieves a page of elements from the cache with pagination support.
// Parameters: start (offset), blockSize (page size), q (query filter).
// Returns the matching elements and metadata containing total counts.
func (this *BaseService) Fetch(start, blockSize int, q ifs.IQuery) ([]interface{}, *l8api.L8MetaData) {
	if this.cache == nil {
		return nil, nil
	}
	return this.cache.Fetch(start, blockSize, q)
}

// AddMetadataFunc registers a metadata extraction function with the given name.
// The function is applied to elements to extract metadata values for indexing.
func (this *BaseService) AddMetadataFunc(name string, f func(interface{}) (bool, string)) {
	if this.cache == nil {
		return
	}
	this.cache.AddMetadataFunc(name, f)
}

// all is a filter function that accepts all elements without transformation.
// Used by the All() method to retrieve the complete cache contents.
func all(i interface{}) (bool, interface{}) {
	return true, i
}
