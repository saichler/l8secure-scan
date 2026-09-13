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

// Package registry provides a type registration system for dynamic instance creation
// and type lookup. It maintains a thread-safe mapping of type names to reflect.Type,
// enabling runtime instantiation of registered types without compile-time knowledge.
//
// The registry automatically registers primitive types and core Layer 8 types at
// initialization. Custom types can be registered using Register or RegisterType methods.
//
// Key features:
//   - Thread-safe type registration and lookup
//   - Dynamic instance creation via NewOf method
//   - Enum value registration and retrieval
//   - Integration with Layer 8 type system (l8api, l8notify, l8reflect, etc.)
package registry

import (
	"errors"
	"reflect"
	"sync"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"github.com/saichler/l8types/go/types/l8system"
	"github.com/saichler/l8types/go/types/l8web"
)

// RegistryImpl is the main implementation of the type registry.
// It maintains separate maps for types and enum values.
type RegistryImpl struct {
	types *TypesMap
	enums map[string]int32
	mtx   *sync.RWMutex
}

// NewRegistry creates a new registry with pre-registered primitive types and
// core Layer 8 types. The registry is ready to use immediately after creation.
func NewRegistry() *RegistryImpl {
	registry := &RegistryImpl{}
	registry.types = NewTypesMap()
	registry.enums = make(map[string]int32)
	registry.mtx = new(sync.RWMutex)
	registry.registerPrimitives()
	registry.registerBase()
	return registry
}

func (this *RegistryImpl) registerPrimitives() {
	this.RegisterType(reflect.TypeOf(int8(0)))
	this.RegisterType(reflect.TypeOf(int16(0)))
	this.RegisterType(reflect.TypeOf(int32(0)))
	this.RegisterType(reflect.TypeOf(int64(0)))
	this.RegisterType(reflect.TypeOf(""))
	this.RegisterType(reflect.TypeOf(true))
	this.RegisterType(reflect.TypeOf(float32(0)))
	this.RegisterType(reflect.TypeOf(float64(0)))
}

func (this *RegistryImpl) registerBase() {
	this.Register(&l8api.L8Query{})
	this.Register(&l8api.L8MetaData{})
	this.Register(&l8api.AuthToken{})
	this.Register(&l8api.AuthUser{})
	this.Register(&l8health.L8Health{})
	this.Register(&l8health.L8HealthList{})
	this.Register(&l8web.L8Empty{})
	this.Register(&l8notify.L8NotificationSet{})
	this.Register(&l8reflect.L8Node{})
	this.Register(&l8reflect.L8TableView{})
	this.Register(&l8web.L8WebService{})
	this.Register(&l8services.L8Services{})
	this.Register(&l8services.L8ReplicationIndex{})
	this.Register(&l8services.L8Transaction{})
	this.Register(&l8sysconfig.L8SysConfig{})
	this.Register(&l8system.L8SystemMessage{})
	this.Register(&l8web.L8Plugin{})
}

// Register adds a type to the registry using a sample instance.
// Accepts either a value or pointer; extracts the underlying type automatically.
func (this *RegistryImpl) Register(any interface{}) (bool, error) {
	v := reflect.ValueOf(any)
	if !v.IsValid() {
		return false, errors.New("invalid input")
	}
	if v.Kind() == reflect.Ptr {
		return this.RegisterType(v.Elem().Type())
	}
	return this.RegisterType(v.Type())
}

// RegisterType adds a reflect.Type directly to the registry.
// Returns true if this is a new registration, false if the type already exists.
func (this *RegistryImpl) RegisterType(t reflect.Type) (bool, error) {
	if t == nil {
		return false, errors.New("Cannot register a nil type")
	}
	if t.Name() == "rtype" {
		return false, nil
	}
	return this.types.Put(t.Name(), t)
}

// UnRegister removes a type from the registry by name.
func (this *RegistryImpl) UnRegister(typeName string) (bool, error) {
	if typeName == "" {
		return false, errors.New("Cannot unregister a blank type")
	}
	return this.types.Del(typeName), nil
}

// Info retrieves type information by name. Returns an error if the type is not registered.
func (this *RegistryImpl) Info(name string) (ifs.IInfo, error) {
	typ, ok := this.types.Get(name)
	if !ok {
		return nil, errors.New("Unknown Type: " + name)
	}
	return typ, nil
}

// RegisterEnums adds a map of enum name-value pairs to the registry.
func (this *RegistryImpl) RegisterEnums(enums map[string]int32) {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	for name, value := range enums {
		this.enums[name] = value
	}
}

// Enum retrieves an enum value by name. Returns 0 if not found.
func (this *RegistryImpl) Enum(name string) int32 {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	return this.enums[name]
}

// NewOf creates a new instance of the same type as the provided sample.
// Registers the type if not already registered, then creates a new zero-value instance.
func (this *RegistryImpl) NewOf(any interface{}) interface{} {
	this.Register(any)
	typeName := reflect.ValueOf(any).Elem().Type().Name()
	info, _ := this.Info(typeName)
	r, _ := info.NewInstance()
	return r
}

// TypeList returns all registered types as an L8TypeList for serialization/introspection.
func (this *RegistryImpl) TypeList() *l8api.L8TypeList {
	return this.types.TypeList()
}
