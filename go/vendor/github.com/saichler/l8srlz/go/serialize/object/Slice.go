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

// addSlice serializes a Go slice to binary format.
// Format: length (int32), type flag (byte), then elements.
// Byte slices ([]byte) are optimized with direct copy (flag=1).
// Other slices serialize each element individually (flag=0).
// Nil or empty slices are encoded as -1.
func addSlice(any interface{}, data *[]byte, location *int) error {
	if any == nil {
		addInt32(int32(-1), data, location)
		return nil
	}
	slice := reflect.ValueOf(any)
	if slice.Len() == 0 {
		addInt32(int32(-1), data, location)
		return nil
	}

	addInt32(int32(slice.Len()), data, location)
	dataByte, ok := any.([]byte)
	if ok {
		(*data)[*location] = 1
		*location += 1
		checkAndEnlarge(data, location, len(dataByte))
		copy((*data)[*location:*location+len(dataByte)], dataByte)
		*location += len(dataByte)
	} else {
		(*data)[*location] = 0
		*location += 1
		obj := newDecode(data, location, nil)
		for i := 0; i < slice.Len(); i++ {
			element := slice.Index(i).Interface()
			obj.Add(element)
		}
	}
	return nil
}

// getSlice deserializes a slice from binary format.
// Reconstructs the properly typed slice using reflection.
// Handles byte slices with optimized direct copy.
// Infers element type from the first element for typed reconstruction.
func getSlice(data *[]byte, location *int, registry ifs.IRegistry) (interface{}, error) {
	l := getInt32(data, location)
	size := int(l)
	if size == -1 || size == 0 {
		return nil, nil
	}

	if (*data)[*location] == 1 {
		*location += 1
		result := make([]byte, size)
		copy(result, (*data)[*location:*location+size])
		*location += size
		return result, nil
	} else {
		*location += 1
	}

	elems := make([]interface{}, 0)
	var sliceType reflect.Type

	obj := newDecode(data, location, registry)

	for i := 0; i < size; i++ {
		element, _ := obj.Get()
		if i == 0 {
			sliceType = reflect.SliceOf(reflect.ValueOf(element).Type())
		}
		elems = append(elems, element)
	}

	newSlice := reflect.MakeSlice(sliceType, len(elems), len(elems))
	for i := 0; i < int(size); i++ {
		if elems[i] != nil {
			newSlice.Index(i).Set(reflect.ValueOf(elems[i]))
		}
	}

	return newSlice.Interface(), nil
}
