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

package ifs

import (
	"reflect"

	"github.com/saichler/l8types/go/types/l8api"
)

// IRegistry provides a type registry for dynamic type instantiation.
// Allows registering types by name and creating instances dynamically,
// which is essential for deserializing data without compile-time type knowledge.
type IRegistry interface {
	// Register registers a type from an instance, extracting the type information.
	Register(interface{}) (bool, error)
	// RegisterType registers a type directly from a reflect.Type.
	RegisterType(reflect.Type) (bool, error)
	// Info retrieves the registration info for a type by name.
	Info(string) (IInfo, error)
	// RegisterEnums registers enum string-to-int32 mappings.
	RegisterEnums(map[string]int32)
	// Enum returns the int32 value for an enum string.
	Enum(string) int32
	// UnRegister removes a type from the registry.
	UnRegister(string) (bool, error)
	// NewOf creates a new instance of the same type as the provided instance.
	NewOf(interface{}) interface{}
	// TypeList returns a list of all registered type names.
	TypeList() *l8api.L8TypeList
}

// IInfo provides information about a registered type.
type IInfo interface {
	// Type returns the reflect.Type for this registration.
	Type() reflect.Type
	// Name returns the registered type name.
	Name() string
	// Serializer returns a serializer for this type in the given mode.
	Serializer(SerializerMode) ISerializer
	// AddSerializer adds a custom serializer for this type.
	AddSerializer(ISerializer)
	// NewInstance creates a new zero-value instance of this type.
	NewInstance() (interface{}, error)
}
