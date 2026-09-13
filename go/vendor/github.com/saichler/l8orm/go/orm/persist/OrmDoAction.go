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
	"github.com/saichler/l8types/go/types/l8web"
	"reflect"
)

// do executes a database write operation (POST, PUT, PATCH) with callback support.
// It follows the pattern: Before callbacks -> Cache update -> ORM write -> After callbacks.
// Returns an empty response on success, or an error response on failure.
func (this *OrmService) do(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	pb = elemList(pb)
	pbBefore, cont := this.Before(action, pb, vnic)
	if !cont {
		return object.New(nil, &l8web.L8Empty{})
	}
	if pbBefore != nil {
		if pbBefore.Error() != nil {
			return pbBefore
		}
		pb = pbBefore
	}

	// Cache elements before writing to DB
	this.cacheAction(action, pb, vnic)

	err := this.orm.Write(action, pb, vnic.Resources())

	if err != nil {
		return object.NewError(err.Error())
	}
	pbAfter, cont := this.After(action, pb, vnic)
	if !cont {

	}
	if pbAfter != nil {
		return object.New(nil, &l8web.L8Empty{})
	}

	return object.New(nil, &l8web.L8Empty{})
}

// cacheAction updates the cache based on the action type.
// For POST/PUT, caches each element. For PATCH, applies partial updates.
func (this *OrmService) cacheAction(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) {
	if this.cache == nil {
		return
	}
	for _, elem := range pb.Elements() {
		if elem == nil {
			continue
		}
		switch action {
		case ifs.POST, ifs.PUT:
			this.cachePost(elem)
		case ifs.PATCH:
			// For patch, ensure the element exists in cache first
			if _, ok := this.cacheGet(elem); !ok {
				// Cache miss — fetch from DB to populate cache before patching
				q, e := ElementToQuery(pb, this.sla.ServiceItem(), vnic)
				if e == nil {
					result := this.orm.Read(q, vnic.Resources())
					this.cacheElements(result)
				}
			}
			this.cachePatch(elem)
		}
	}
}

func elemList(pb ifs.IElements) ifs.IElements {
	if len(pb.Elements()) == 1 {
		v := reflect.ValueOf(pb.Element())
		if !v.IsValid() {
			return pb
		}
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		fieldList := v.FieldByName("List")
		if fieldList.IsValid() {
			elems := []interface{}{}
			for i := 0; i < fieldList.Len(); i++ {
				elems = append(elems, fieldList.Index(i).Interface())
			}
			return object.New(nil, elems)
		}
	}
	return pb
}
