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
	"sync/atomic"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

type internalCache struct {
	cache           map[string]interface{}
	UniqueToPrimary map[string]string
	PrimaryToUnique map[string]string
	hasExtraKeys    bool
	stamp           int64
	queries         map[int64]*internalQuery
	metadataFunc    map[string]func(interface{}) (bool, string)
}

func newInternalCache() *internalCache {
	iq := &internalCache{}
	iq.cache = make(map[string]interface{})
	iq.queries = make(map[int64]*internalQuery)
	iq.UniqueToPrimary = make(map[string]string)
	iq.PrimaryToUnique = make(map[string]string)
	return iq
}

func newMetadata() *l8api.L8MetaData {
	metadata := &l8api.L8MetaData{}
	metadata.KeyCount = &l8api.L8Count{}
	metadata.KeyCount.Counts = make(map[string]float64)
	metadata.ValueCount = make(map[string]*l8api.L8Count)
	return metadata
}

func addToMetadata(value interface{}, metadataFunc map[string]func(interface{}) (bool, string), metadata *l8api.L8MetaData) {
	if metadataFunc != nil {
		for count, f := range metadataFunc {
			ok1, v := f(value)
			if ok1 {
				metadata.KeyCount.Counts[count]++
				if v != "" {
					vCount, ok2 := metadata.ValueCount[count]
					if !ok2 {
						vCount = &l8api.L8Count{}
						vCount.Counts = make(map[string]float64)
						metadata.ValueCount[count] = vCount
					}
					vCount.Counts[v]++
				}
			}
		}
	}
}

func (this *internalCache) put(pk, uk string, value interface{}) {
	_, ok := this.cache[pk]
	this.cache[pk] = value
	this.putUnique(pk, uk)
	if !ok {
		this.bumpStamp()
	}
}

// bumpStamp advances the cache's write-generation counter, invalidating any
// cached query whose own stamp no longer matches (see fetch()). Deliberately
// NOT time.Now().Unix(): that has only 1-second resolution, so two writes
// (or a write and a query-cache refresh) landing in the same wall-clock
// second would collide on the same stamp value and the second write would
// be silently invisible to any already-cached query until some later write
// happened to land in a different second — an intermittent stale-read bug
// under any fast-moving workload (e.g. a write immediately followed by a
// query for it). All callers of put/delete/stampChanged already hold the
// owning Cache's mutex (see Cache.Post/Patch/Delete/Fetch), so a plain
// increment is safe without its own atomicity.
func (this *internalCache) bumpStamp() {
	this.stamp++
}

func (this *internalCache) get(pk, uk string) (interface{}, bool) {
	if pk == "" && uk == "" {
		return nil, false
	}
	if pk == "" && uk != "" {
		pk = this.UniqueToPrimary[uk]
	}
	item, ok := this.cache[pk]
	return item, ok
}

func (this *internalCache) delete(pk, uk string) (interface{}, bool) {
	item, ok := this.cache[pk]
	if !ok {
		return item, ok
	}
	delete(this.cache, pk)
	this.deleteUnique(pk, uk)
	this.bumpStamp()
	return item, ok
}

func (this *internalCache) stampChanged() {
	this.bumpStamp()
}

func (this *internalCache) size() int {
	return len(this.cache)
}

func hashString(s string) int32 {
	var h int32
	for _, c := range s {
		h = 31*h + int32(c)
	}
	return h
}

func (this *internalCache) fetch(start, blockSize int, q ifs.IQuery, r ifs.IResources) ([]interface{}, *l8api.L8MetaData) {
	if q.IsAggregate() {
		return this.fetchAggregate(q)
	}

	aaaId := q.AAAId()
	hash := int64(q.Hash())
	if aaaId != "" {
		hash = hash<<32 | int64(hashString(aaaId))
	}

	dq, ok := this.queries[hash]
	if !ok {
		dq = newInternalQuery(q)
		this.queries[hash] = dq
	}

	atomic.StoreInt64(&dq.lastUsed, time.Now().Unix())

	if dq.stamp != this.stamp {
		dq.prepare(this.cache, this.stamp, q.Descending(), this.metadataFunc, r, aaaId)
	}

	result := make([]interface{}, 0)
	for i := start; i < len(dq.data); i++ {
		key := dq.data[i]
		value, ok := this.cache[key]
		if ok {
			result = append(result, value)
		}
		if blockSize == 0 {
			continue
		}
		if len(result) >= blockSize {
			break
		}
	}
	return result, dq.metadata
}

func (this *internalCache) addMetadataFunc(name string, f func(interface{}) (bool, string)) {
	if this.metadataFunc == nil {
		this.metadataFunc = make(map[string]func(interface{}) (bool, string))
	}
	this.metadataFunc[name] = f
}
