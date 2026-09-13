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

// Package object provides high-performance binary serialization and deserialization
// for Go objects. It supports primitive types, complex types (slices, maps, structs),
// and Protocol Buffers messages with automatic type detection and buffer management.
//
// The package is designed for microservices environments where efficient data
// transmission is critical. It uses a compact binary format with type prefixing
// to enable proper deserialization without prior schema knowledge.
//
// Basic usage:
//
//	// Serialize any Go object
//	data, err := object.DataOf(myObject)
//
//	// Deserialize back to original type
//	result, err := object.ElemOf(data, registry)
package object

import (
	"encoding/base64"
	"errors"
	"go/types"
	"reflect"

	"github.com/saichler/l8types/go/ifs"
)

// Object is the main serialization engine that handles encoding and decoding
// of Go objects to/from binary format. It maintains an internal buffer and
// location pointer for efficient sequential read/write operations.
//
// Object uses exponential buffer growth to minimize memory allocations during
// serialization of large or numerous objects.
type Object struct {
	data     *[]byte       // Internal byte buffer for serialized data
	location *int          // Current read/write position in the buffer
	registry ifs.IRegistry // Type registry for deserializing complex types
}

// Primitive defines the interface for serializing/deserializing primitive types.
// Primitive types don't require a registry for deserialization.
type Primitive interface {
	add(interface{}, *[]byte, *int)
	get(*[]byte, *int) interface{}
}

// Complex defines the interface for serializing/deserializing complex types.
// Complex types may require a registry to resolve type information during deserialization.
type Complex interface {
	add(interface{}, *[]byte, *int) error
	get(*[]byte, *int, ifs.IRegistry) (interface{}, error)
}

// NewEncode creates a new Object configured for serialization (encoding).
// It initializes an internal buffer of 1024 bytes which grows automatically
// as needed using exponential growth strategy.
func NewEncode() *Object {
	obj := &Object{}
	data := make([]byte, 1024)
	location := 0
	obj.data = &data
	obj.location = &location
	return obj
}

// NewDecode creates a new Object configured for deserialization (decoding).
// It wraps the provided byte slice and uses the registry for type resolution
// when deserializing complex types like Protocol Buffers messages.
//
// Parameters:
//   - data: The byte slice containing serialized data
//   - location: Starting position in the data slice (usually 0)
//   - registry: Type registry for resolving type names to Go types
func NewDecode(data []byte, location int, registry ifs.IRegistry) *Object {
	obj := &Object{}
	obj.data = &data
	obj.location = &location
	obj.registry = registry
	return obj
}

// newDecode is an internal constructor that accepts pointers to enable
// shared state across nested deserialization operations.
func newDecode(data *[]byte, location *int, registry ifs.IRegistry) *Object {
	obj := &Object{}
	obj.data = data
	obj.location = location
	obj.registry = registry
	return obj
}

// Data returns the serialized byte slice containing all data written so far.
// The returned slice is a view into the internal buffer up to the current location.
func (this *Object) Data() []byte {
	return (*this.data)[0:*this.location]
}

// Location returns the current read/write position in the internal buffer.
func (this *Object) Location() int {
	return *this.location
}

// Add serializes the given value and appends it to the internal buffer.
// It automatically detects the type of the value and uses the appropriate
// serialization strategy. The type information is prefixed to enable
// proper deserialization.
//
// Supported types:
//   - Primitives: int, int32, int64, uint32, uint64, float32, float64, string, bool, byte
//   - Complex: slices, maps, pointers to structs (Protocol Buffers)
//
// Returns an error if the type is not supported.
func (this *Object) Add(any interface{}) error {

	switch v := any.(type) {
	case int:
		this.addKind(reflect.Int)
		addInt(v, this.data, this.location)
		return nil
	case uint32:
		this.addKind(reflect.Uint32)
		addUInt32(v, this.data, this.location)
		return nil
	case uint64:
		this.addKind(reflect.Uint64)
		addUInt64(v, this.data, this.location)
		return nil
	case int32:
		this.addKind(reflect.Int32)
		addInt32(v, this.data, this.location)
		return nil
	case int64:
		this.addKind(reflect.Int64)
		addInt64(v, this.data, this.location)
		return nil
	case float32:
		this.addKind(reflect.Float32)
		addFloat32(v, this.data, this.location)
		return nil
	case float64:
		this.addKind(reflect.Float64)
		addFloat64(v, this.data, this.location)
		return nil
	case string:
		this.addKind(reflect.String)
		addString(v, this.data, this.location)
		return nil
	case bool:
		this.addKind(reflect.Bool)
		addBool(v, this.data, this.location)
		return nil
	case byte:
		this.addKind(reflect.Uint8)
		addByte(v, this.data, this.location)
		return nil
	case types.Slice:
		this.addKind(reflect.Slice)
		return addSlice(v, this.data, this.location)
	case types.Map:
		this.addKind(reflect.Map)
		return addMap(v, this.data, this.location)
	default:
		kind := reflect.ValueOf(any).Kind()
		switch kind {
		case reflect.Invalid:
			fallthrough
		case reflect.Ptr:
			this.addKind(reflect.Ptr)
			return addStruct(v, this.data, this.location)
		case reflect.Slice:
			this.addKind(reflect.Slice)
			return addSlice(v, this.data, this.location)
		case reflect.Map:
			this.addKind(reflect.Map)
			return addMap(v, this.data, this.location)
		}
	}
	kind := reflect.ValueOf(any).Kind()
	//Special case for enums impl in protocol buffers
	if kind == reflect.Int32 {
		this.addKind(reflect.Int32)
		addInt32(int32(reflect.ValueOf(any).Int()), this.data, this.location)
		return nil
	}
	//Special case for named types with underlying uint8
	if kind == reflect.Uint8 {
		this.addKind(reflect.Uint8)
		addByte(byte(reflect.ValueOf(any).Uint()), this.data, this.location)
		return nil
	}
	//panic("Did not find any Object for kind " + kind.String())
	return errors.New("Did not find any Object for kind " + kind.String())
}

// Get deserializes and returns the next value from the internal buffer.
// It reads the type prefix first to determine the appropriate deserialization
// strategy, then returns the value as an interface{}.
//
// For complex types like Protocol Buffers messages, the registry must be
// properly configured with the type information.
//
// Returns the deserialized value and nil error on success, or nil and
// an error if deserialization fails.
func (this *Object) Get() (interface{}, error) {
	kind := this.getKind()
	switch kind {
	case reflect.Int:
		return getInt(this.data, this.location), nil
	case reflect.Uint32:
		return getUInt32(this.data, this.location), nil
	case reflect.Uint64:
		return getUInt64(this.data, this.location), nil
	case reflect.Int32:
		return getInt32(this.data, this.location), nil
	case reflect.Int64:
		return getInt64(this.data, this.location), nil
	case reflect.Float32:
		return getFloat32(this.data, this.location), nil
	case reflect.Float64:
		return getFloat64(this.data, this.location), nil
	case reflect.String:
		return getString(this.data, this.location), nil
	case reflect.Uint8:
		return getByte(this.data, this.location), nil
	case reflect.Bool:
		return getBool(this.data, this.location), nil
	case reflect.Slice:
		return getSlice(this.data, this.location, this.registry)
	case reflect.Map:
		return getMap(this.data, this.location, this.registry)
	case reflect.Invalid:
		fallthrough
	case reflect.Ptr:
		return getStruct(this.data, this.location, this.registry)
	}
	return nil, errors.New("Did not find any Object for kind " + kind.String())
}

// addKind writes the reflect.Kind as an int32 prefix to enable type identification
// during deserialization.
func (this *Object) addKind(kind reflect.Kind) {
	addInt32(int32(kind), this.data, this.location)
}

// getKind reads and returns the reflect.Kind prefix from the current buffer position.
func (this *Object) getKind() reflect.Kind {
	i := getInt32(this.data, this.location)
	return reflect.Kind(i)
}

// Base64 returns the serialized data as a Base64-encoded string.
// This is useful for transmitting binary data over text-based protocols
// like HTTP or JSON.
func (this *Object) Base64() string {
	return base64.StdEncoding.EncodeToString(this.Data())
}

/*
func (this *Object) appendBytes(data []byte, l int) {
	if this.location+len(data) > len(this.data) {
		newData := make([]byte, this.location+len(data)+512)
		copy(newData[0:len(this.data)], this.data)
		this.data = newData
	}
	copy(this.data[this.location:this.location+l], data)
	this.location += l
}*/

// FromBase64 decodes a Base64-encoded string back to raw bytes.
// This is the inverse operation of Object.Base64() and is useful
// for receiving serialized data from text-based protocols.
func FromBase64(b64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(b64)

}

// DataOf is a convenience function that serializes any element to bytes
// in a single call. It creates a new encoder, adds the element, and
// returns the resulting byte slice.
//
// Returns nil, nil if elem is nil.
func DataOf(elem interface{}) ([]byte, error) {
	if elem == nil {
		return nil, nil
	}
	obj := NewEncode()
	err := obj.Add(elem)
	return obj.Data(), err
}

// ElemOf is a convenience function that deserializes bytes back to
// the original element in a single call. It creates a new decoder
// and returns the deserialized value.
//
// Parameters:
//   - data: The serialized byte slice
//   - r: Type registry for resolving complex types
//
// Returns nil, nil if data is nil.
func ElemOf(data []byte, r ifs.IRegistry) (interface{}, error) {
	if data == nil {
		return nil, nil
	}
	location := 0
	obj := NewDecode(data, location, r)
	return obj.Get()
}

// checkAndEnlarge ensures the buffer has enough capacity for the next write.
// It uses exponential growth (doubling) with a minimum threshold to minimize
// allocations while avoiding excessive memory usage.
//
// Parameters:
//   - data: Pointer to the byte slice buffer
//   - location: Pointer to the current write position
//   - need: Number of bytes needed for the next write operation
func checkAndEnlarge(data *[]byte, location *int, need int) {
	if *location+need > len(*data) {
		// Exponential growth with minimum threshold
		newCap := len(*data) * 2
		if newCap < *location+need+512 {
			newCap = *location + need + 512
		}
		tmp := make([]byte, newCap)
		copy(tmp, *data)
		*data = tmp
	}
}
