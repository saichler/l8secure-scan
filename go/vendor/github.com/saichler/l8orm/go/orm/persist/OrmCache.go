/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package persist

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"reflect"
)

// cacheGet retrieves an element from the cache by its primary key.
// Returns the cached element and true on hit, or nil and false on miss.
// If cache is nil, always returns miss.
func (this *OrmService) cacheGet(element interface{}) (interface{}, bool) {
	if this.cache == nil {
		return nil, false
	}
	item, err := this.cache.Get(element)
	if err != nil {
		return nil, false
	}
	return item, true
}

// cachePost adds or replaces an element in the cache.
// Notifications are disabled since the ORM service layer handles its own callbacks.
// No-op if cache is nil.
func (this *OrmService) cachePost(element interface{}) {
	if this.cache == nil {
		return
	}
	this.cache.Post(element, false)
}

// cachePatch applies a partial update to an element in the cache.
// Notifications are disabled since the ORM service layer handles its own callbacks.
// No-op if cache is nil.
func (this *OrmService) cachePatch(element interface{}) {
	if this.cache == nil {
		return
	}
	this.cache.Patch(element, false)
}

// cacheDelete removes an element from the cache.
// Notifications are disabled since the ORM service layer handles its own callbacks.
// No-op if cache is nil. Returns any error from the underlying cache so the caller
// can log it with service/type context instead of silently swallowing failures.
func (this *OrmService) cacheDelete(element interface{}) error {
	if this.cache == nil {
		return nil
	}
	_, _, err := this.cache.Delete(element, false)
	return err
}

// cacheFetch retrieves a paginated slice of elements from the cache matching a query.
// Returns the elements as IElements with metadata, or nil if cache is nil or empty.
func (this *OrmService) cacheFetch(query ifs.IQuery) ifs.IElements {
	if this.cache == nil || this.cache.Size() == 0 {
		return nil
	}
	start := int(query.Page()) * int(query.Limit())
	blockSize := int(query.Limit())
	if blockSize == 0 {
		blockSize = 100
	}
	values, metadata := this.cache.Fetch(start, blockSize, query)
	if query.IsAggregate() {
		return object.NewQueryResult(values, metadata)
	}
	if len(values) == 0 && start == 0 {
		return nil
	}
	return object.NewQueryResult(values, metadata)
}

// cacheElements caches all elements from an IElements result.
// Used after DB reads to populate the cache.
// No-op if cache is nil or elements is nil/has error.
func (this *OrmService) cacheElements(elements ifs.IElements) {
	if this.cache == nil || elements == nil || elements.Error() != nil {
		return
	}
	for _, elem := range elements.Elements() {
		if elem != nil {
			this.cachePost(elem)
		}
	}
}

// fetchFromDbAndCache reads from the database and caches each result element.
// Returns the IElements result from the DB read.
func (this *OrmService) fetchFromDbAndCache(query ifs.IQuery, resources ifs.IResources) ifs.IElements {
	result := this.orm.Read(query, resources)
	this.cacheElements(result)
	return result
}

// loadCacheInitElements reads all records from the database for cache initialization.
// Returns the elements as []interface{} to pass to NewCache's initElements parameter.
// If the tables don't exist yet, returns nil so the cache starts empty.
func (this *OrmService) loadCacheInitElements(vnic ifs.IVNic) []interface{} {
	typeName := reflect.TypeOf(this.sla.ServiceItem()).Elem().Name()
	qe, err := object.NewQuery("select * from "+typeName, vnic.Resources())
	if err != nil {
		return nil
	}
	q, err := qe.Query(vnic.Resources())
	if err != nil {
		return nil
	}
	result := this.orm.Read(q, vnic.Resources())
	if result == nil || result.Error() != nil {
		return nil
	}
	elements := result.Elements()
	filtered := make([]interface{}, 0, len(elements))
	for _, elem := range elements {
		if elem != nil {
			filtered = append(filtered, elem)
		}
	}
	return filtered
}

// cacheMetadata returns the cache metadata (counts) or nil if cache is nil.
func (this *OrmService) cacheMetadata() *l8api.L8MetaData {
	if this.cache == nil {
		return nil
	}
	counts := this.cache.Metadata()
	metadata := &l8api.L8MetaData{}
	metadata.KeyCount = &l8api.L8Count{}
	metadata.KeyCount.Counts = counts
	return metadata
}
