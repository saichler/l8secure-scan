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

// This file contains getter methods for extracting values from nested data structures
// through property paths. Supports maps, slices, structs, and pointers.

package properties

import (
	"reflect"
)

// getField retrieves a field from a struct value using the cached field index.
// Falls back to FieldByName if fieldIndex is not set (-1).
func (this *Property) getField(structValue reflect.Value) reflect.Value {
	if this.fieldIndex >= 0 {
		return structValue.Field(this.fieldIndex)
	}
	return structValue.FieldByName(this.node.FieldName)
}

// getMap retrieves values from a map field.
// If a specific key is set in the parent, returns only that entry's field value.
// Otherwise, returns the field value from all map entries.
func (this *Property) getMap(parent reflect.Value) []reflect.Value {
	result := make([]reflect.Value, 0)
	if this.parent.key != nil {
		myValue := parent.MapIndex(reflect.ValueOf(this.parent.key))
		if !myValue.IsValid() {
			return result
		}
		if myValue.Kind() == reflect.Ptr {
			if myValue.IsNil() {
				return result
			}
			myValue = myValue.Elem()
		}
		myValue = this.getField(myValue)
		result = append(result, myValue)
	} else {
		keys := parent.MapKeys()
		for _, key := range keys {
			value := parent.MapIndex(key)
			if value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
			myValue := this.getField(value)
			result = append(result, myValue)
		}
	}
	return result
}

func (this *Property) getSlice(parent reflect.Value) []reflect.Value {
	result := make([]reflect.Value, 0)
	if this.parent.key != nil {
		myValue := parent.Index(this.parent.key.(int))
		if !myValue.IsValid() {
			return result
		}
		if myValue.Kind() == reflect.Ptr {
			if myValue.IsNil() {
				return result
			}
			myValue = myValue.Elem()
		}
		myValue = this.getField(myValue)
		result = append(result, myValue)
	} else {
		for i := 0; i < parent.Len(); i++ {
			value := parent.Index(i)
			if value.Kind() == reflect.Interface {
				value = value.Elem()
			}
			if value.Kind() == reflect.Ptr {
				if value.IsNil() {
					continue
				}
				value = value.Elem()
			}

			myValue := this.getField(value)
			result = append(result, myValue)
		}
	}
	return result
}

func (this *Property) GetValue(any reflect.Value) []reflect.Value {
	if !any.IsValid() {
		return []reflect.Value{}
	}
	if any.Kind() == reflect.Ptr && any.IsNil() {
		return []reflect.Value{}
	}
	if this.parent == nil {
		return []reflect.Value{any}
	}

	parents := this.parent.GetValue(any)
	results := make([]reflect.Value, 0)

	for _, parent := range parents {
		if parent.Kind() == reflect.Ptr {
			parent = parent.Elem()
		}
		if parent.Kind() == reflect.Map {
			mapItems := this.getMap(parent)
			results = append(results, mapItems...)
		} else if parent.Kind() == reflect.Slice {
			sliceItems := this.getSlice(parent)
			results = append(results, sliceItems...)
		} else {
			if parent.IsValid() {
				value := this.getField(parent)
				results = append(results, value)
			}
		}
	}
	return results
}

func (this *Property) Get(any interface{}) (interface{}, error) {
	if any == nil {
		if this == nil {
			panic("nil this")
		}
		if this.resources == nil {
			panic("nil resources")
		}
		if this.resources.Registry() == nil {
			panic("nil registry")
		}
		info, err := this.resources.Registry().Info(this.node.TypeName)
		if err != nil {
			return nil, err
		}
		n, err := info.NewInstance()
		if this.key != nil {
			this.SetPrimaryKey(this.node, n, this.key)
		}
		return n, nil
	}
	values := this.GetValue(reflect.ValueOf(any))
	if len(values) == 0 || !values[0].IsValid() {
		return nil, nil
	}
	if values[0].Kind() == reflect.Ptr && values[0].IsNil() {
		return nil, nil
	}
	if len(values) == 1 {
		return values[0].Interface(), nil
	}
	result := make([]interface{}, len(values))
	for i, v := range values {
		result[i] = v.Interface()
	}
	return result, nil
}

func (this *Property) GetAsValues(any interface{}) []reflect.Value {
	_interface, _ := this.Get(any)
	if _interface == nil {
		return []reflect.Value{reflect.ValueOf(this)}
	}
	value := reflect.ValueOf(_interface)
	if value.Kind() == reflect.Map {
		result := make([]reflect.Value, value.Len())
		keys := value.MapKeys()
		for i, key := range keys {
			item := value.MapIndex(key)
			if item.Kind() == reflect.Interface {
				item = item.Elem()
			}
			result[i] = item
		}
		return result
	} else if value.Kind() == reflect.Slice {
		result := make([]reflect.Value, value.Len())
		for i := 0; i < len(result); i++ {
			item := value.Index(i)
			if item.Kind() == reflect.Interface {
				item = item.Elem()
			}
			result[i] = item
		}
		return result
	} else {
		return []reflect.Value{value}
	}
}
