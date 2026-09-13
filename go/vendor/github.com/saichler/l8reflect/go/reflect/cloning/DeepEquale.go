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

package cloning

import (
	"bytes"
	"reflect"
)

// DeepEqual provides deep equality comparison for Go data structures.
// It compares values by recursively examining their contents rather than
// just comparing memory addresses. It supports all Go primitive types,
// slices, maps, structs, and pointers.
type DeepEqual struct {
	// comparators maps each reflect.Kind to its corresponding comparison function
	comparators map[reflect.Kind]func(reflect.Value, reflect.Value) bool
}

// NewDeepEqual creates and initializes a new DeepEqual instance.
// The returned DeepEqual is ready to compare any Go data structures.
func NewDeepEqual() *DeepEqual {
	de := &DeepEqual{}
	de.initCloners()
	return de
}

// initCloners initializes the comparison function registry with handlers for all supported Go types.
func (this *DeepEqual) initCloners() {
	this.comparators = make(map[reflect.Kind]func(reflect.Value, reflect.Value) bool)
	this.comparators[reflect.Int] = this.intComp
	this.comparators[reflect.Int8] = this.intComp
	this.comparators[reflect.Int16] = this.intComp
	this.comparators[reflect.Int32] = this.intComp
	this.comparators[reflect.Int64] = this.intComp

	this.comparators[reflect.Uint] = this.uintComp
	this.comparators[reflect.Uint8] = this.uintComp
	this.comparators[reflect.Uint16] = this.uintComp
	this.comparators[reflect.Uint32] = this.uintComp
	this.comparators[reflect.Uint64] = this.uintComp

	this.comparators[reflect.String] = this.stringComp

	this.comparators[reflect.Bool] = this.boolComp

	this.comparators[reflect.Float32] = this.floatComp
	this.comparators[reflect.Float64] = this.floatComp

	this.comparators[reflect.Ptr] = this.ptrComp

	this.comparators[reflect.Struct] = this.structComp

	this.comparators[reflect.Slice] = this.sliceComp

	this.comparators[reflect.Map] = this.mapComp

}

// Equal compares two values for deep equality.
// Returns true if both values have identical contents, false otherwise.
// Handles nil values, different kinds, and recursively compares composite types.
func (this *DeepEqual) Equal(aSide, zSide interface{}) bool {
	aSideValue := reflect.ValueOf(aSide)
	zSideValue := reflect.ValueOf(zSide)
	return this.equal(aSideValue, zSideValue)
}

// equal is the internal recursive comparison function that dispatches to type-specific comparators.
func (this *DeepEqual) equal(aSideValue, zSideValue reflect.Value) bool {
	if aSideValue.IsValid() && !zSideValue.IsValid() {
		return false
	}
	if !aSideValue.IsValid() && zSideValue.IsValid() {
		return false
	}
	if !aSideValue.IsValid() && !zSideValue.IsValid() {
		return true
	}
	if aSideValue.Kind() != zSideValue.Kind() {
		return false
	}

	kind := aSideValue.Kind()
	comparator := this.comparators[kind]
	if comparator == nil {
		panic("No comparator for kind:" + kind.String() + ", please add it!")
	}
	return comparator(aSideValue, zSideValue)
}

// Type-specific comparator functions for primitive and composite types

// intComp compares two integer values (int, int32, int64).
func (this *DeepEqual) intComp(aSideValue, zSideValue reflect.Value) bool {
	return aSideValue.Int() == zSideValue.Int()
}

// uintComp compares two unsigned integer values (uint, uint32, uint64).
func (this *DeepEqual) uintComp(aSideValue, zSideValue reflect.Value) bool {
	return aSideValue.Uint() == zSideValue.Uint()
}

// stringComp compares two string values.
func (this *DeepEqual) stringComp(aSideValue, zSideValue reflect.Value) bool {
	return aSideValue.String() == zSideValue.String()
}

// boolComp compares two boolean values.
func (this *DeepEqual) boolComp(aSideValue, zSideValue reflect.Value) bool {
	return aSideValue.Bool() == zSideValue.Bool()
}

// floatComp compares two floating-point values (float32, float64).
func (this *DeepEqual) floatComp(aSideValue, zSideValue reflect.Value) bool {
	return aSideValue.Float() == zSideValue.Float()
}

// ptrComp compares two pointer values by recursively comparing their pointed-to values.
// Handles nil pointers appropriately.
func (this *DeepEqual) ptrComp(aSideValue, zSideValue reflect.Value) bool {
	if aSideValue.IsNil() && !zSideValue.IsNil() {
		return false
	}
	if !aSideValue.IsNil() && zSideValue.IsNil() {
		return false
	}
	if aSideValue.IsNil() && zSideValue.IsNil() {
		return true
	}
	return this.equal(aSideValue.Elem(), zSideValue.Elem())
}

// structComp compares two struct values field by field.
// Skips fields matching SkipFieldByName criteria.
// Returns false if struct types don't match.
func (this *DeepEqual) structComp(aSideValue, zSideValue reflect.Value) bool {
	if aSideValue.Type().Name() != zSideValue.Type().Name() {
		return false
	}
	for i := 0; i < aSideValue.Type().NumField(); i++ {
		fieldName := aSideValue.Type().Field(i).Name
		if SkipFieldByName(fieldName) {
			continue
		}
		aFieldValue := aSideValue.Field(i)
		zFieldValue := zSideValue.Field(i)
		eq := this.equal(aFieldValue, zFieldValue)
		if !eq {
			return false
		}
	}
	return true
}

// sliceComp compares two slice values element by element.
// Returns false if lengths differ or any element differs.
func (this *DeepEqual) sliceComp(aSideValue, zSideValue reflect.Value) bool {
	if aSideValue.IsNil() && !zSideValue.IsNil() {
		return false
	}
	if !aSideValue.IsNil() && zSideValue.IsNil() {
		return false
	}
	if aSideValue.IsNil() && zSideValue.IsNil() {
		return true
	}

	// Byte slices are compared atomically.
	if aSideValue.Type().Elem().Kind() == reflect.Uint8 {
		return bytes.Equal(aSideValue.Bytes(), zSideValue.Bytes())
	}

	if aSideValue.Len() != zSideValue.Len() {
		return false
	}

	for i := 0; i < aSideValue.Len(); i++ {
		aSideCel := aSideValue.Index(i)
		zSideCel := zSideValue.Index(i)
		eq := this.equal(aSideCel, zSideCel)
		if !eq {
			return false
		}
	}
	return true
}

// mapComp compares two map values by comparing all key-value pairs.
// Returns false if map sizes differ or any key-value pair differs.
func (this *DeepEqual) mapComp(aSideValue, zSideValue reflect.Value) bool {
	if aSideValue.IsNil() && !zSideValue.IsNil() {
		return false
	}
	if !aSideValue.IsNil() && zSideValue.IsNil() {
		return false
	}
	if aSideValue.IsNil() && zSideValue.IsNil() {
		return true
	}
	mapKeysAside := aSideValue.MapKeys()
	mapKeysZside := zSideValue.MapKeys()

	if len(mapKeysAside) != len(mapKeysZside) {
		return false
	}

	for _, key := range mapKeysAside {
		aSideV := aSideValue.MapIndex(key)
		zSideV := zSideValue.MapIndex(key)
		eq := this.equal(aSideV, zSideV)
		if !eq {
			return false
		}
	}
	return true
}
