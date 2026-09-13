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

// This file contains collection functions for gathering all instances of a type
// from a nested data structure. Useful for extracting all entities of a specific type.

package properties

import (
	"reflect"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
)

// Collect traverses a data structure and collects all instances matching typeName.
// Returns a map keyed by property ID to the collected instances.
func Collect(root interface{}, r ifs.IResources, typeName string) map[string]interface{} {
	rootKey, node, err := r.Introspector().Decorators().PrimaryKeyDecoratorValue(root)
	if err != nil {
		return nil
	}
	result := make(map[string]interface{}, 0)
	collect(root, node, typeName, nil, rootKey, result, r)
	return result
}

func collect(any interface{}, node *l8reflect.L8Node, typeName string,
	parent *Property, key interface{}, elems map[string]interface{}, r ifs.IResources) {
	if any == nil {
		return
	}
	val := reflect.ValueOf(any)
	if !val.IsValid() {
		return
	}
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	typ := val.Type()
	myProperty := NewProperty(node, parent, key, any, r)
	if typ.Name() == typeName {
		id, _ := myProperty.PropertyId()
		elems[id] = any
		return
	}

	if node.Attributes != nil {
		for _, attr := range node.Attributes {
			if attr.IsMap {
				value := val.FieldByName(attr.FieldName)
				if value.IsValid() {
					keys := value.MapKeys()
					for i := 0; i < len(keys); i++ {
						collect(value.MapIndex(keys[i]).Interface(), attr, typeName, myProperty, keys[i].Interface(), elems, r)
					}
				}
			} else if attr.IsSlice {
				value := val.FieldByName(attr.FieldName)
				if value.IsValid() {
					for i := 0; i < value.Len(); i++ {
						collect(value.Index(i).Interface(), attr, typeName, myProperty, i, elems, r)
					}
				}
			} else if attr.IsStruct {
				value := val.FieldByName(attr.FieldName)
				if value.IsValid() {
					collect(value.Interface(), attr, typeName, myProperty, nil, elems, r)
				}
			}
		}
	}
}
