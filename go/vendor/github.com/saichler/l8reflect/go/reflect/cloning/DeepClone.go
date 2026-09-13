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

// Package cloning provides deep cloning and equality comparison utilities for Go data structures.
// It supports all Go primitive types, slices, maps, structs, pointers, interfaces, channels,
// and functions. The package handles circular references to prevent infinite loops during cloning.
//
// Key features:
//   - Type-safe deep cloning of any Go data structure
//   - Circular reference detection and handling
//   - Customizable field filtering (skip fields by name patterns)
//   - Support for all Go primitive and composite types
package cloning

import (
	"reflect"
	"strconv"
	"strings"
)

// Cloner provides deep cloning functionality for Go data structures.
// It maintains a registry of type-specific cloning functions and handles
// circular references through pointer tracking.
type Cloner struct {
	// cloners maps each reflect.Kind to its corresponding cloning function
	cloners map[reflect.Kind]func(reflect.Value, string, map[string]reflect.Value) reflect.Value
}

// NewCloner creates and initializes a new Cloner instance.
// The returned Cloner is ready to clone any Go data structure.
func NewCloner() *Cloner {
	cloner := &Cloner{}
	cloner.initCloners()
	return cloner
}

// initCloners initializes the cloning function registry with handlers for all supported Go types.
// Each handler is responsible for cloning values of a specific reflect.Kind.
func (this *Cloner) initCloners() {
	this.cloners = make(map[reflect.Kind]func(reflect.Value, string, map[string]reflect.Value) reflect.Value)
	this.cloners[reflect.Int] = this.intCloner
	this.cloners[reflect.Int8] = this.int8Cloner
	this.cloners[reflect.Int16] = this.int16Cloner
	this.cloners[reflect.Int32] = this.int32Cloner
	this.cloners[reflect.Int64] = this.int64Cloner
	this.cloners[reflect.Uint] = this.uintCloner
	this.cloners[reflect.Uint8] = this.uint8Cloner
	this.cloners[reflect.Uint16] = this.uint16Cloner
	this.cloners[reflect.Uint32] = this.uint32Cloner
	this.cloners[reflect.Uint64] = this.uint64Cloner
	this.cloners[reflect.Float32] = this.float32Cloner
	this.cloners[reflect.Float64] = this.float64Cloner
	this.cloners[reflect.Complex64] = this.complex64Cloner
	this.cloners[reflect.Complex128] = this.complex128Cloner
	this.cloners[reflect.Bool] = this.boolCloner
	this.cloners[reflect.String] = this.stringCloner
	this.cloners[reflect.Array] = this.arrayCloner
	this.cloners[reflect.Slice] = this.sliceCloner
	this.cloners[reflect.Map] = this.mapCloner
	this.cloners[reflect.Ptr] = this.ptrCloner
	this.cloners[reflect.Struct] = this.structCloner
	this.cloners[reflect.Interface] = this.interfaceCloner
	this.cloners[reflect.Chan] = this.chanCloner
	this.cloners[reflect.Func] = this.funcCloner
}

// Clone performs a deep clone of the provided value and returns the cloned copy.
// It handles all Go types including primitives, slices, maps, structs, and pointers.
// Circular references are detected and handled to prevent infinite recursion.
// Fields matching certain patterns (DoNotCompare, DoNotCopy, XXX prefix, private) are skipped.
// Returns nil if the input is nil or if the cloned value is invalid.
func (this *Cloner) Clone(any interface{}) interface{} {
	if any == nil {
		return nil
	}
	value := reflect.ValueOf(any)
	stopLoop := make(map[string]reflect.Value)
	valueClone := this.clone(value, "", stopLoop)
	if !valueClone.IsValid() {
		return nil
	}
	return valueClone.Interface()
}

// clone is the internal recursive cloning function that dispatches to type-specific cloners.
// The stopLoop map tracks pointer addresses to detect and handle circular references.
func (this *Cloner) clone(value reflect.Value, fieldName string, stopLoop map[string]reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	kind := value.Kind()
	cloner := this.cloners[kind]
	if cloner == nil {
		panic("No cloner for kind:" + kind.String() + ":" + fieldName)
	}
	return cloner(value, fieldName, stopLoop)
}

// sliceCloner creates a deep copy of a slice value, recursively cloning each element.
// Returns the original value if the slice is nil.
func (this *Cloner) sliceCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	if value.IsNil() {
		return value
	}
	newSlice := reflect.MakeSlice(reflect.SliceOf(value.Type().Elem()), value.Len(), value.Len())
	for i := 0; i < value.Len(); i++ {
		elem := value.Index(i)
		elemClone := this.clone(elem, name, stopLoop)
		dst := newSlice.Index(i)
		if elemClone.Kind() == reflect.Int32 {
			dst.SetInt(elemClone.Int())
		} else {
			dst.Set(elemClone)
		}
	}
	return newSlice
}

// ptrCloner creates a deep copy of a pointer value.
// It tracks pointer addresses in stopLoop to detect and handle circular references.
// If a pointer has already been cloned, returns the existing clone to break cycles.
func (this *Cloner) ptrCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	if value.IsNil() {
		return value
	}

	p := strconv.Itoa(int(value.Pointer()))
	exist, ok := stopLoop[p]
	if ok {
		return exist
	}

	newPtr := reflect.New(value.Elem().Type())
	stopLoop[p] = newPtr

	newPtr.Elem().Set(this.clone(value.Elem(), name, stopLoop))

	return newPtr
}

// structCloner creates a deep copy of a struct value.
// It iterates through all fields, skipping those matching SkipFieldByName criteria,
// and recursively clones each eligible field.
func (this *Cloner) structCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	cloneStruct := reflect.New(value.Type()).Elem()
	structType := value.Type()
	for i := 0; i < structType.NumField(); i++ {
		fieldValue := value.Field(i)
		fieldName := structType.Field(i).Name
		if SkipFieldByName(fieldName) {
			continue
		}
		cloned := this.clone(fieldValue, structType.Field(i).Name, stopLoop)
		if cloned.Kind() == reflect.Int32 {
			cloneStruct.Field(i).SetInt(cloned.Int())
		} else {
			cloneStruct.Field(i).Set(cloned)
		}
	}
	return cloneStruct
}

// mapCloner creates a deep copy of a map value, recursively cloning each key-value pair.
// Returns the original value if the map is nil.
func (this *Cloner) mapCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	if value.IsNil() {
		return value
	}
	mapKeys := value.MapKeys()
	mapClone := reflect.MakeMapWithSize(value.Type(), len(mapKeys))
	for _, key := range mapKeys {
		mapElem := value.MapIndex(key)
		mapElemClone := this.clone(mapElem, name, stopLoop)
		if mapElemClone.Kind() == reflect.Int32 {
			v := reflect.New(value.Type().Elem()).Elem()
			v.SetInt(mapElemClone.Int())
			mapClone.SetMapIndex(key, v)
		} else {
			mapClone.SetMapIndex(key, mapElemClone)
		}
	}
	return mapClone
}

func (this *Cloner) intCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Int()
	return reflect.ValueOf(int(i))
}

func (this *Cloner) uintCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Uint()
	return reflect.ValueOf(uint(i))
}

func (this *Cloner) uint32Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Uint()
	return reflect.ValueOf(uint32(i))
}

func (this *Cloner) uint64Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Uint()
	return reflect.ValueOf(uint64(i))
}

func (this *Cloner) float32Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Float()
	return reflect.ValueOf(float32(i))
}

func (this *Cloner) float64Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Float()
	return reflect.ValueOf(float64(i))
}

func (this *Cloner) boolCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	b := value.Bool()
	return reflect.ValueOf(b)
}

func (this *Cloner) int32Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Int()
	return reflect.ValueOf(int32(i))
}

func (this *Cloner) int64Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Int()
	return reflect.ValueOf(int64(i))
}

func (this *Cloner) stringCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	s := value.String()
	return reflect.ValueOf(s)
}

func (this *Cloner) int8Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Int()
	return reflect.ValueOf(int8(i))
}

func (this *Cloner) int16Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Int()
	return reflect.ValueOf(int16(i))
}

func (this *Cloner) uint8Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Uint()
	return reflect.ValueOf(uint8(i))
}

func (this *Cloner) uint16Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	i := value.Uint()
	return reflect.ValueOf(uint16(i))
}

func (this *Cloner) complex64Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	c := value.Complex()
	return reflect.ValueOf(complex64(c))
}

func (this *Cloner) complex128Cloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	c := value.Complex()
	return reflect.ValueOf(complex128(c))
}

// arrayCloner creates a deep copy of an array value, recursively cloning each element.
func (this *Cloner) arrayCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	arrayType := value.Type()
	newArray := reflect.New(arrayType).Elem()
	for i := 0; i < value.Len(); i++ {
		elem := value.Index(i)
		elemClone := this.clone(elem, name, stopLoop)
		dst := newArray.Index(i)
		if elemClone.Kind() == reflect.Int32 {
			dst.SetInt(elemClone.Int())
		} else {
			dst.Set(elemClone)
		}
	}
	return newArray
}

// interfaceCloner clones the concrete value inside an interface.
// Returns the original value if the interface is nil.
func (this *Cloner) interfaceCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	if value.IsNil() {
		return value
	}
	// Get the concrete value inside the interface
	concreteValue := value.Elem()
	// Clone the concrete value
	clonedConcrete := this.clone(concreteValue, name, stopLoop)
	// Return it wrapped in the same interface type
	return clonedConcrete
}

func (this *Cloner) chanCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	if value.IsNil() {
		return value
	}
	// Create a new channel of the same type
	// Note: We can't clone channel contents, so we just create a new empty channel
	chanType := value.Type()
	newChan := reflect.MakeChan(chanType, 0)
	return newChan
}

func (this *Cloner) funcCloner(value reflect.Value, name string, stopLoop map[string]reflect.Value) reflect.Value {
	if value.IsNil() {
		return value
	}
	// Functions can't be truly cloned in Go, so we return the same function
	return value
}

// SkipFieldByName determines if a struct field should be skipped during cloning/comparison.
// Fields are skipped if they match any of the following criteria:
//   - Field name is "DoNotCompare"
//   - Field name is "DoNotCopy"
//   - Field name starts with "XXX" (protobuf internal fields)
//   - Field name starts with a lowercase letter (unexported/private fields)
func SkipFieldByName(fieldName string) bool {
	if fieldName == "DoNotCompare" {
		return true
	}
	if fieldName == "DoNotCopy" {
		return true
	}
	if len(fieldName) > 3 && fieldName[0:3] == "XXX" {
		return true
	}
	if fieldName[0:1] == strings.ToLower(fieldName[0:1]) {
		return true
	}
	return false
}
