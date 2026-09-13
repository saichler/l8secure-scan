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
package object

import (
	"github.com/saichler/l8types/go/ifs"
	"reflect"
)

// addMap serializes a Go map to binary format.
// Format: count (int32) followed by key-value pairs.
// Nil or empty maps are encoded as -1.
// Uses reflection to iterate over any map type.
func addMap(any interface{}, data *[]byte, location *int) error {
	if any == nil {
		addInt32(int32(-1), data, location)
		return nil
	}
	mapp := reflect.ValueOf(any)
	if mapp.Len() == 0 {
		addInt32(int32(-1), data, location)
		return nil
	}

	addInt32(int32(mapp.Len()), data, location)

	obj := newDecode(data, location, nil)
	keys := mapp.MapKeys()

	for _, key := range keys {
		obj.Add(key.Interface())
		element := mapp.MapIndex(key).Interface()
		obj.Add(element)
	}

	return nil
}

// getMap deserializes a map from binary format.
// It reconstructs the typed map using reflection, inferring key and value
// types from the first non-nil entry. Handles nil values correctly.
func getMap(data *[]byte, location *int, registry ifs.IRegistry) (interface{}, error) {
	l := getInt32(data, location)
	size := int(l)
	if size == -1 || size == 0 {
		return nil, nil
	}

	enc := newDecode(data, location, registry)
	mapp := make(map[interface{}]interface{})
	var mapKeyType reflect.Type
	var mapValueType reflect.Type
	found := false

	for i := 0; i < int(size); i++ {
		key, _ := enc.Get()
		value, _ := enc.Get()
		if !found && key != nil && value != nil {
			found = true
			mapKeyType = reflect.ValueOf(key).Type()
			mapValueType = reflect.ValueOf(value).Type()
		}
		mapp[key] = value
	}
	newMap := reflect.MakeMap(reflect.MapOf(mapKeyType, mapValueType))
	for k, v := range mapp {
		if v == nil {
			newValue := reflect.New(mapValueType)
			newValue.Elem().Set(reflect.Zero(newValue.Elem().Type()))
			newMap.SetMapIndex(reflect.ValueOf(k), newValue.Elem())
		} else {
			newMap.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
	}
	return newMap.Interface(), nil
}
