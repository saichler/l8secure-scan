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

// This file contains map-specific setter logic for property operations.
// Handles map creation, entry insertion, update, and deletion through property paths.

package properties

import (
	"errors"
	"reflect"

	"github.com/saichler/l8types/go/ifs"
)

// mapSet handles setting values within map fields.
// Supports replacing entire maps, setting individual entries by key,
// creating new maps, and handling entry deletions with ifs.Deleted_Entry marker.
func (this *Property) mapSet(myMapValue reflect.Value, newMapValue reflect.Value) (interface{}, error) {
	var vInfo ifs.IInfo
	var kInfo ifs.IInfo
	var err error

	vInfo, err = this.resources.Registry().Info(this.node.TypeName)
	if err != nil {
		return nil, err
	}

	kInfo, err = this.resources.Registry().Info(this.node.KeyTypeName)
	if err != nil {
		return nil, err
	}

	// If myMapValue is a zero Value (not addressable), we cannot set it.
	// This happens when the parent struct doesn't exist (e.g., missing map entry).
	/*
		if !myMapValue.IsValid() {
			pid, _ := this.PropertyId()
			return nil, errors.New("cannot set map value: parent struct does not exist for property " + pid)
		}*/

	//create the map if it is nil
	if !myMapValue.IsValid() {
		if this.node.IsStruct {
			myMapValue = reflect.MakeMap(reflect.MapOf(kInfo.Type(), reflect.PointerTo(vInfo.Type())))
		} else {
			myMapValue = reflect.MakeMap(reflect.MapOf(kInfo.Type(), vInfo.Type()))
		}
	}

	//create the map if it is nil
	if myMapValue.IsNil() {
		if this.node.IsStruct {
			myMapValue.Set(reflect.MakeMap(reflect.MapOf(kInfo.Type(), reflect.PointerTo(vInfo.Type()))))
		} else {
			myMapValue.Set(reflect.MakeMap(reflect.MapOf(kInfo.Type(), vInfo.Type())))
		}
	}

	// Handle setting entire map to nil explicitly
	// This must be checked before creating an empty map for invalid newMapValue
	if this.key == nil && !newMapValue.IsValid() {
		myMapValue.SetZero()
		return nil, nil
	}

	//create the map if it is nil
	if !newMapValue.IsValid() {
		if this.node.IsStruct {
			newMapValue = reflect.MakeMap(reflect.MapOf(kInfo.Type(), reflect.PointerTo(vInfo.Type())))
		} else {
			newMapValue = reflect.MakeMap(reflect.MapOf(kInfo.Type(), vInfo.Type()))
		}
	}

	//This means the entire map is new
	if this.key == nil {
		if newMapValue.Kind() != reflect.Map {
			return nil, errors.New("invalid map type " + newMapValue.Kind().String() + " for map " + myMapValue.Type().String())
		}
		myMapValue.Set(newMapValue)
		return myMapValue.Interface(), nil
	}

	mapKey := reflect.ValueOf(this.key)
	oKeyValue := myMapValue.MapIndex(mapKey)
	//in this case, the newMapValue isn't a map, it is a value
	//this.value = newMapValue.Interface()
	nKeyValue := newMapValue

	//This map entry was marked for deletion so delete it
	if this.isLeaf && nKeyValue.Kind() == reflect.String && nKeyValue.String() == ifs.Deleted_Entry {
		myMapValue.SetMapIndex(mapKey, reflect.Value{})
		return myMapValue.Interface(), err
	}

	//If this node is a struct & this property is not the leaf
	//we need to return the old struct instance to keep drilling down to the updated property.
	if this.node.IsStruct && !this.IsLeaf() {
		//if the old value is not valid, create it
		if !oKeyValue.IsValid() {
			typeName := newMapValue.Type().Name()
			if newMapValue.Kind() == reflect.Ptr {
				typeName = newMapValue.Elem().Type().Name()
			}
			if typeName == vInfo.Type().Name() {
				myMapValue.SetMapIndex(mapKey, newMapValue)
				oKeyValue = newMapValue
			} else {
				o, _ := vInfo.NewInstance()
				oKeyValue = reflect.ValueOf(o)
				myMapValue.SetMapIndex(mapKey, oKeyValue)
			}
		}
		return oKeyValue.Interface(), nil
	}

	myMapValue.SetMapIndex(mapKey, nKeyValue)

	return myMapValue.Interface(), err
}
