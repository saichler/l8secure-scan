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

package strings

import (
	"errors"
	"github.com/saichler/l8types/go/ifs"
	"reflect"
	"strconv"
	"strings"
)

// Global map that map a type/kind to a method that converts string to that type
var fromstrings = make(map[reflect.Kind]func(string, []reflect.Kind, ifs.IRegistry) (reflect.Value, error))
var zeroValue = reflect.Zero(reflect.TypeOf(""))

const (
	errorValue = "Failed to convert string to instance:"
)

// initialize the map
func init() {
	fromstrings[reflect.String] = stringFromString
	fromstrings[reflect.Int] = intFromString
	fromstrings[reflect.Int8] = int8FromString
	fromstrings[reflect.Int16] = int16FromString
	fromstrings[reflect.Int32] = int32FromString
	fromstrings[reflect.Int64] = int64FromString
	fromstrings[reflect.Uint] = uintFromString
	fromstrings[reflect.Uint8] = uint8FromString
	fromstrings[reflect.Uint16] = uint16FromString
	fromstrings[reflect.Uint32] = uint32FromString
	fromstrings[reflect.Uint64] = uint64FromString
	fromstrings[reflect.Float32] = float32FromString
	fromstrings[reflect.Float64] = float64FromString
	fromstrings[reflect.Bool] = boolFromString
	fromstrings[reflect.Ptr] = ptrFromString
	fromstrings[reflect.Slice] = sliceFromString
	fromstrings[reflect.Map] = mapFromString
	fromstrings[reflect.Interface] = interfaceFromString
	fromstrings[reflect.Struct] = structFromString
}

// Comvert string to string
func stringFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	return reflect.ValueOf(str), nil
}

// Convert string to int
func intFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(0), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(0), err
	}
	return reflect.ValueOf(i), nil
}

// Convert string to int8
func int8FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(int8(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(int8(0)), err
	}
	return reflect.ValueOf(int8(i)), nil
}

// Convert string to int16
func int16FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(int16(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(int16(0)), err
	}
	return reflect.ValueOf(int16(i)), nil
}

// Convert string to int32
func int32FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(int32(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(int32(0)), err
	}
	return reflect.ValueOf(int32(i)), nil
}

// Convert string to int64
func int64FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(int64(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(int64(0)), err
	}
	return reflect.ValueOf(int64(i)), nil
}

// Convert string to uint
func uintFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(uint(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(uint(0)), err
	}
	return reflect.ValueOf(uint(i)), nil
}

// Convert string to uint8
func uint8FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf([]byte{0}), nil
	}
	return reflect.ValueOf([]byte(str)), nil
}

// Convert string to uint16
func uint16FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(uint16(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(uint16(0)), err
	}
	return reflect.ValueOf(uint16(i)), nil
}

// Convert string to uint32
func uint32FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(uint32(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(uint32(0)), err
	}
	return reflect.ValueOf(uint32(i)), nil
}

// Convert string to uint64
func uint64FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(uint64(0)), nil
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return reflect.ValueOf(uint64(0)), err
	}
	return reflect.ValueOf(uint64(i)), nil
}

// Convert string to bool
func boolFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(false), nil
	}
	i, err := strconv.ParseBool(str)
	if err != nil {
		return reflect.ValueOf(false), err
	}
	return reflect.ValueOf(i), nil
}

// Convert string to float32
func float32FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(float32(0)), nil
	}
	f, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return reflect.ValueOf(float32(0)), err
	}
	return reflect.ValueOf(float32(f)), nil
}

// Convert string to float64
func float64FromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" {
		return reflect.ValueOf(float64(0)), nil
	}
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return reflect.ValueOf(float64(0)), err
	}
	return reflect.ValueOf(float64(f)), nil
}

// Convert string to pointer
func ptrFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	f := fromstrings[kinds[0]]
	if f != nil {
		v, err := f(str, kinds[1:], registry)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		if !v.IsValid() {
			return v, err
		}
		newPtr := reflect.New(v.Type())
		newPtr.Elem().Set(v)
		return newPtr, nil
	}
	return reflect.ValueOf(nil), errors.New("ptr cloud not be created for kind " + kinds[0].String())
}

// Convert string to interface
func interfaceFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	f := fromstrings[kinds[0]]
	if f != nil {
		v, err := f(str, kinds[1:], registry)
		if err != nil {
			return reflect.ValueOf(nil), err
		}
		newVal := reflect.New(v.Type())
		newVal.Elem().Set(v)
		return newVal.Elem(), nil
	}
	return reflect.ValueOf(nil), errors.New("interface cloud not be created for kind " + kinds[0].String())
}

// Convert string to map
func mapFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	str = strings.TrimSpace(str)
	str = str[1 : len(str)-1]
	items := splitOnUnescaped(str, ',')
	var newMap *reflect.Value
	for _, item := range items {
		index := indexOfUnescaped(item, '=')
		if index == -1 {
			return reflect.ValueOf(nil),
				errors.New("map item '" + item + "' does not contain a '=' sign")
		}
		keyStr := unescapeString(strings.TrimSpace(item[0:index]))
		valueStr := unescapeString(strings.TrimSpace(item[index+1:]))
		keyF := fromstrings[kinds[0]]
		valueF := fromstrings[kinds[1]]
		if keyF == nil || valueF == nil {
			return reflect.ValueOf(nil),
				errors.New("map item '" + item + "' cannot find either the key type or the value type converter")
		}
		keyV, err := keyF(keyStr, kinds[2:], registry)
		if err != nil {
			return reflect.ValueOf(nil),
				errors.New("map key item '" + item + " error:" + err.Error())
		}
		valueV, err := valueF(valueStr, kinds[2:], registry)
		if err != nil {
			return reflect.ValueOf(nil),
				errors.New("map value item '" + item + " error:" + err.Error())
		}
		if newMap == nil {
			m := reflect.MakeMap(reflect.MapOf(keyV.Type(), valueV.Type()))
			newMap = &m
		}
		newMap.SetMapIndex(keyV, valueV)
	}
	return *newMap, nil
}

// Convert string to slice
func sliceFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	str = strings.TrimSpace(str)
	// if it is byte array, it will not have square brackets
	if len(str) > 1 && str[0] == '[' {
		str = str[1 : len(str)-1]
	}
	items := splitOnUnescaped(str, ',')

	itemF := fromstrings[kinds[0]]
	if itemF == nil {
		return reflect.ValueOf(nil), errors.New("slice cannot find converter item kind " + kinds[0].String())
	}

	defaultValue, err := itemF("", kinds[1:], registry)
	if err != nil {
		return reflect.ValueOf(nil), errors.New("slice error: " + err.Error())
	}

	if str == "" {
		return reflect.MakeSlice(reflect.SliceOf(defaultValue.Type()), 0, 0), nil
	}

	//Special case for byte array
	if defaultValue.Kind() == reflect.Uint8 {
		newSlice := reflect.MakeSlice(reflect.SliceOf(defaultValue.Type()), len(str), len(str))
		for i, v := range str {
			newSlice.Index(i).Set(reflect.ValueOf(byte(v)))
		}
		return newSlice, nil
	}

	newSlice := reflect.MakeSlice(reflect.SliceOf(defaultValue.Type()), len(items), len(items))

	for i, item := range items {
		v, err := itemF(unescapeString(item), kinds[1:], registry)
		if err != nil {
			return reflect.ValueOf(nil), errors.New("slice item '" + item + "' error:" + err.Error())
		}
		newSlice.Index(i).Set(v)
	}
	return newSlice, nil
}

func structFromString(str string, kinds []reflect.Kind, registry ifs.IRegistry) (reflect.Value, error) {
	if registry == nil {
		return reflect.ValueOf(nil), errors.New("registry cannot be nil")
	}
	if str == "<Nil>" {
		return reflect.ValueOf(nil), nil
	}
	typeInfo, e := registry.Info(str)
	if e != nil {
		return reflect.ValueOf(nil), errors.New("registry info for '" + str + "' error:" + e.Error())
	}

	v, e := typeInfo.NewInstance()
	if e != nil {
		return reflect.ValueOf(nil), e
	}
	return reflect.ValueOf(v), nil
}

// Convert string to an instance
func InstanceOf(str string, registry ifs.IRegistry) (interface{}, error) {
	v, e := FromString(str, registry)
	if e != nil {
		return nil, e
	}
	if v.IsValid() {
		return v.Interface(), nil
	}
	return nil, nil
}

// Conver string to a reflect.value
func FromString(str string, registry ifs.IRegistry) (reflect.Value, error) {
	if str == "" || str == "{0}" || str == "nil" {
		return zeroValue, nil
	}
	v, k := parseStringForKinds(str)
	f := fromstrings[k[0]]
	if f == nil {
		return reflect.ValueOf(nil), errors.New("cannot find converter for " + k[0].String())
	}
	return f(v, k[1:], registry)
}

// Extract the kinds from the prefix of the string
func parseStringForKinds(str string) (string, []reflect.Kind) {
	if len(str) < 3 {
		return "", nil
	}
	if str[0] != '{' {
		return "", nil
	}
	index := strings.Index(str, "}")
	if index == -1 {
		return "", nil
	}
	types := str[1:index]
	result := str[index+1:]
	k := parseKinds(types)
	return result, k
}

// extract the kinds to a list of reflect.Kind
func parseKinds(types string) []reflect.Kind {
	split := strings.Split(types, ",")
	kinds := make([]reflect.Kind, len(split))
	for i, v := range split {
		k, e := strconv.Atoi(v)
		if e != nil {
			return []reflect.Kind{}
		}
		kinds[i] = reflect.Kind(k)
	}
	return kinds
}
