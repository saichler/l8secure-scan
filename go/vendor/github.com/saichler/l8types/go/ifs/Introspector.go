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
	"github.com/saichler/l8types/go/types/l8reflect"

	"reflect"
)

// IIntrospector provides runtime type introspection capabilities.
// It builds and caches type metadata trees for efficient reflection operations.
type IIntrospector interface {
	// Inspect analyzes a type and returns its node representation.
	Inspect(interface{}) (*l8reflect.L8Node, error)
	// Node retrieves a cached node by its unique key.
	Node(string) (*l8reflect.L8Node, bool)
	// NodeByType retrieves a node by reflect.Type.
	NodeByType(p reflect.Type) (*l8reflect.L8Node, bool)
	// NodeByTypeName retrieves a node by type name string.
	NodeByTypeName(string) (*l8reflect.L8Node, bool)
	// NodeByValue retrieves a node for a given value's type.
	NodeByValue(interface{}) (*l8reflect.L8Node, bool)
	// Nodes returns all inspected nodes, optionally filtering roots and structs.
	Nodes(bool, bool) []*l8reflect.L8Node
	// Registry returns the type registry.
	Registry() IRegistry
	// Kind returns the reflect.Kind for a node.
	Kind(*l8reflect.L8Node) reflect.Kind
	// Clone creates a deep copy of an object.
	Clone(interface{}) interface{}
	// TableView retrieves a table view definition by type name.
	TableView(string) (*l8reflect.L8TableView, bool)
	// TableViews returns all table view definitions.
	TableViews() []*l8reflect.L8TableView
	// Clean removes a type's cached introspection data.
	Clean(string)
	// Decorators returns the decorator manager.
	Decorators() IDecorators
}

// IDecorators manages field decorators for type introspection.
// Decorators provide metadata about fields like primary keys, unique constraints, etc.
type IDecorators interface {
	// AddPrimaryKeyDecorator marks fields as primary key.
	AddPrimaryKeyDecorator(interface{}, ...string) error
	// AddUniqueKeyDecorator marks fields as unique.
	AddUniqueKeyDecorator(interface{}, ...string) error
	// AddNonUniqueKeyDecorator marks fields as non-unique indexed fields.
	AddNonUniqueKeyDecorator(interface{}, ...string) error
	// AddAlwayOverwriteDecorator marks a field to always overwrite (no merge).
	AddAlwayOverwriteDecorator(string) error
	// AddNoNestedInspection prevents recursive inspection of a type.
	AddNoNestedInspection(interface{}) error

	// PrimaryKeyDecoratorValue extracts the primary key value from an object.
	PrimaryKeyDecoratorValue(interface{}) (string, *l8reflect.L8Node, error)
	// PrimaryKeyDecoratorFromValue extracts primary key from a reflect.Value.
	PrimaryKeyDecoratorFromValue(*l8reflect.L8Node, reflect.Value) (string, *l8reflect.L8Node, error)
	// UniqueKeyDecoratorValue extracts the unique key value from an object.
	UniqueKeyDecoratorValue(interface{}) (string, *l8reflect.L8Node, error)
	// NonUniqueKeyDecoratorValue extracts the non-unique key value from an object.
	NonUniqueKeyDecoratorValue(interface{}) (string, *l8reflect.L8Node, error)
	// NoNestedInspection returns true if nested inspection is disabled.
	NoNestedInspection(interface{}) bool
	// AlwaysFullDecorator returns true if the field always sends full values.
	AlwaysFullDecorator(interface{}) bool

	// NodeFor returns the node and value for an object.
	NodeFor(interface{}) (*l8reflect.L8Node, reflect.Value, error)
	// BoolDecoratorValueFor checks if an object has a specific decorator.
	BoolDecoratorValueFor(interface{}, l8reflect.L8DecoratorType) bool
	// BoolDecoratorValueForNode checks if a node has a specific decorator.
	BoolDecoratorValueForNode(*l8reflect.L8Node, l8reflect.L8DecoratorType) bool
	// Fields returns the field names associated with a decorator.
	Fields(*l8reflect.L8Node, l8reflect.L8DecoratorType) ([]string, error)
	// KeyForValue builds a key string from field values.
	KeyForValue([]string, reflect.Value, string, bool) (string, error)
}
