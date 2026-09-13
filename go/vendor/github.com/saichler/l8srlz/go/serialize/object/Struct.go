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
	"errors"
	"github.com/saichler/l8types/go/ifs"
	"google.golang.org/protobuf/proto"
	"reflect"
)

// addStruct serializes a Protocol Buffers message to binary format.
// Format: size (int32), type name (string), protobuf bytes.
// Special cases:
//   - nil: size = -1
//   - empty message: size = -2
//
// Uses Google's protobuf library for the actual message serialization.
func addStruct(any interface{}, data *[]byte, location *int) error {
	if any == nil {
		addInt32(int32(-1), data, location)
		return nil
	}

	val := reflect.ValueOf(any)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			addInt32(int32(-1), data, location)
			return nil
		}
		val = val.Elem()
	}

	typeName := val.Type().Name()

	pb := any.(proto.Message)
	pbData, err := proto.Marshal(pb)
	if err != nil {
		return errors.New("Failed To marshal proto " + typeName + " in protobuf object:" + err.Error())
	}

	size := len(pbData)
	checkAndEnlarge(data, location, 8+len(typeName)+size)
	if size == 0 {
		addInt32(int32(-2), data, location)
	} else {
		addInt32(int32(len(pbData)), data, location)
	}
	addString(typeName, data, location)
	if size > 0 {
		copy((*data)[*location:*location+len(pbData)], pbData)
		*location += len(pbData)
	}
	return nil
}

// getStruct deserializes a Protocol Buffers message from binary format.
// Uses the registry to look up the type by name and create a new instance.
// The protobuf bytes are then unmarshaled into the instance.
//
// Returns an error if the type is not registered or unmarshaling fails.
func getStruct(data *[]byte, location *int, registry ifs.IRegistry) (interface{}, error) {
	l := getInt32(data, location)
	size := int(l)

	if size == -1 || size == 0 {
		return nil, nil
	}

	typeName := getString(data, location)

	var info ifs.IInfo
	var err error
	var pb interface{}

	info, err = registry.Info(typeName)
	if err != nil {
		//panic("Unknown proto name " + typeName + " in registry, please register it.")
		return nil, errors.New("Unknown proto name " + typeName + " in registry, please register it.")
	}

	pb, err = info.NewInstance()
	if err != nil {
		return nil, errors.New("Error proto name " + typeName + " in registry, cannot instantiate.")
	}
	//if the size is -2 it is an empty interface
	if size == -2 {
		return pb, nil
	}

	protoData := make([]byte, size)
	copy(protoData, (*data)[*location:*location+size])

	err = proto.Unmarshal(protoData, pb.(proto.Message))
	if err != nil {
		return []byte{}, errors.New("Failed To unmarshal proto " + typeName + ":" + err.Error())
	}
	*location += size

	return pb, nil
}
