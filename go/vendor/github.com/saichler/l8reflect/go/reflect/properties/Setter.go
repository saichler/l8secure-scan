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

// This file contains setter methods for modifying values in nested data structures
// through property paths. Handles creation of intermediate objects as needed.

package properties

import (
	"errors"
	"reflect"
	"strings"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// Set sets a value at this property path, creating intermediate objects as needed.
// Returns (parentValue, rootValue, error). Creates new instances for nil containers.
func (this *Property) Set(any interface{}, value interface{}) (interface{}, interface{}, error) {
	if this == nil {
		return nil, nil, errors.New("property is nil, cannot instantiate")
	}
	if this.parent == nil {
		if any == nil {
			info, err := this.resources.Registry().Info(this.node.TypeName)
			if err != nil {
				return nil, nil, err
			}
			newAny, err := info.NewInstance()
			if err != nil {
				return nil, nil, err
			}
			any = newAny
		}
		if this.key != nil {
			this.SetPrimaryKey(this.node, any, this.key)
		}
		return any, any, nil
	}
	parent, root, err := this.parent.Set(any, value)
	if err != nil {
		return nil, nil, err
	}
	if any == nil {
		any = root
	}
	parentValue := reflect.ValueOf(parent)
	if parentValue.Kind() == reflect.Ptr {
		parentValue = parentValue.Elem()
	}

	//Special case for setting a value to the map
	if this.node.IsMap && parentValue.Kind() == reflect.Map {
		if this.IsLeaf() {
			parentValue.SetMapIndex(reflect.ValueOf(this.key), reflect.ValueOf(this.value))
		}
		return this.value, any, nil
	} else if parentValue.Kind() == reflect.Map {
		parentValue = parentValue.MapIndex(reflect.ValueOf(this.key))
		// If the map entry doesn't exist, parentValue will be a zero Value.
		// We need to check this and return an error, as we cannot navigate further.
		if !parentValue.IsValid() {
			pid, _ := this.PropertyId()
			return nil, nil, errors.New("map entry does not exist for property " + pid)
		}
	}

	//Special case where the model is setting the same reference
	//in different attributes, which is incorrect.
	if parentValue.Kind() == reflect.Slice {
		pid, _ := this.PropertyId()
		strValue, ok := value.(string)
		if ok && strValue == ifs.Deleted_Entry {
			this.resources.Logger().Error("The model contain same reference in a map and a slice, pid=" + pid)
			return nil, nil, nil
		}
	}
	
	myValue := parentValue.FieldByName(this.node.FieldName)
	info, err := this.resources.Registry().Info(this.node.TypeName)
	if err != nil {
		return nil, nil, err
	}
	typ := info.Type()
	if this.node.IsMap {
		v, e := this.mapSet(myValue, reflect.ValueOf(value))
		return v, any, e
	} else if this.node.IsSlice && this.node.TypeName == "L8TimeSeriesPoint" {
		v, e := this.timeSeriesAppend(myValue, reflect.ValueOf(value))
		return v, any, e
	} else if this.node.IsSlice {
		v, e := this.sliceSet(myValue, reflect.ValueOf(value))
		return v, any, e
	} else if this.resources.Introspector().Kind(this.node) == reflect.Struct {
		// Handle setting to nil
		if value == nil {
			if myValue.IsValid() && myValue.CanSet() {
				myValue.Set(reflect.Zero(myValue.Type()))
			}
			return nil, any, err
		}

		if !myValue.IsValid() || myValue.IsNil() {
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Ptr &&
				!v.IsNil() && v.Elem().Type().Name() == typ.Name() {
				myValue.Set(reflect.ValueOf(value))
			} else {
				newInstance := reflect.New(typ)
				if v.Kind() == reflect.String {
					serializer := info.Serializer(ifs.STRING)
					if serializer != nil {
						inst, _ := serializer.Unmarshal([]byte(v.String()), this.Resources())
						if inst != nil {
							newInstance = reflect.ValueOf(inst)
						}
					}
				}
				if myValue.CanSet() {
					myValue.Set(newInstance)
				} else {
					p, _ := this.PropertyId()
					return nil, any, errors.New("Cannot set value to " + p)
				}
			}
		} else {
			// Handle replacing existing struct pointer with new value
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Ptr &&
				!v.IsNil() && v.Elem().Type().Name() == typ.Name() {
				myValue.Set(reflect.ValueOf(value))
			}
		}
		return myValue.Interface(), any, err
	} else if reflect.ValueOf(value).Kind() == reflect.Int32 || myValue.Kind() == reflect.Int32 {
		if value == nil {
			myValue.SetInt(0)
			return value, any, err
		}
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.String {
			value = this.resources.Registry().Enum(value.(string))
		}
		myValue.SetInt(reflect.ValueOf(value).Int())
		return value, any, err
	} else {
		if value != nil {
			v := reflect.ValueOf(value)
			if v.Kind() != myValue.Kind() {
				v = ConvertValue(myValue, v)
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						pid, _ := this.PropertyId()
						this.resources.Logger().Error("panic in Set: property=", pid, " value=", value, " error=", r)
					}
				}()
				myValue.Set(v)
			}()
		}
		return value, any, err
	}
}

func (this *Property) SetPrimaryKey(node *l8reflect.L8Node, any interface{}, anyKey interface{}) {
	if anyKey == nil {
		return
	}
	keyString := anyKey.(string)
	tokens := strings.Split(keyString, "::")
	fieldsValues := make([]interface{}, len(tokens))
	for i, token := range tokens {
		vv, _ := strings2.FromString(token, nil)
		fieldsValues[i] = vv.Interface()
	}

	value := reflect.ValueOf(any)
	if !value.IsValid() {
		return
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}

	fields, err := this.resources.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_Primary)
	if err == nil {
		for i, attr := range fields {
			fld := value.FieldByName(attr)
			fld.Set(reflect.ValueOf(fieldsValues[i]))
		}
	}
}
