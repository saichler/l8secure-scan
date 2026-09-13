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

// This file contains decorator management for introspection nodes.
// Decorators add metadata to types for primary keys, unique keys,
// and special behaviors like always-overwrite mode.

package introspecting

import (
	"errors"
	"reflect"

	"github.com/saichler/l8reflect/go/reflect/helping"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// Decorators returns this Introspector as an IDecorators interface.
func (this *Introspector) Decorators() ifs.IDecorators {
	return this
}

// AddPrimaryKeyDecorator marks the specified fields as the primary key for a type.
// Primary keys are used to uniquely identify instances within collections.
// This method is thread-safe.
func (this *Introspector) AddPrimaryKeyDecorator(any interface{}, fields ...string) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	node, _, err := this.nodeFor(any)
	if err != nil || node == nil {
		node, _ = this.inspect(any)
	}
	addDecorator(l8reflect.L8DecoratorType_Primary, fields, node)
	return nil
}

// AddUniqueKeyDecorator marks the specified fields as a unique key.
// Unique keys identify unique instances within collections but are secondary to primary keys.
// This method is thread-safe.
func (this *Introspector) AddUniqueKeyDecorator(any interface{}, fields ...string) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	node, _, err := this.nodeFor(any)
	if err != nil || node == nil {
		return err
	}
	addDecorator(l8reflect.L8DecoratorType_Unique, fields, node)
	return nil
}

// AddNonUniqueKeyDecorator marks the specified fields as a non-unique key.
// Non-unique keys are used for grouping or indexing without uniqueness guarantee.
// This method is thread-safe.
func (this *Introspector) AddNonUniqueKeyDecorator(any interface{}, fields ...string) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	node, _, err := this.nodeFor(any)
	if err != nil || node == nil {
		return err
	}
	addDecorator(l8reflect.L8DecoratorType_NonUnique, fields, node)
	return nil
}

// AddAlwayOverwriteDecorator marks a node to always perform full overwrites during updates.
// When set, updates replace the entire value rather than merging changes.
// This method is thread-safe.
func (this *Introspector) AddAlwayOverwriteDecorator(nodeId string) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	node, ok := this.Node(nodeId)
	if !ok {
		return errors.New(strings2.New("Node for ID ", nodeId, " not found").String())
	}
	addAlwayOverwriteDecorator(node)
	return nil
}

// AddNoNestedInspection marks a type to skip nested introspection.
// The type is registered but its fields are not recursively inspected.
// This method is thread-safe.
func (this *Introspector) AddNoNestedInspection(any interface{}) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	node, _, err := this.nodeFor(any)
	if err != nil {
		return err
	}
	addNoNestedInspection(node)
	return nil
}

// NodeFor retrieves the L8Node and reflect.Value for a given interface.
// Returns an error if the input is nil or invalid.
// This method is thread-safe.
func (this *Introspector) NodeFor(any interface{}) (*l8reflect.L8Node, reflect.Value, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.nodeFor(any)
}

// nodeFor is the internal unlocked version of NodeFor.
// Callers must hold this.mutex before calling.
func (this *Introspector) nodeFor(any interface{}) (*l8reflect.L8Node, reflect.Value, error) {
	if any == nil {
		panic("Node For a nil interface")
	}
	v, e := helping.PtrValue(any)
	if e != nil {
		return nil, v, e
	}
	node, ok := this.Node(v.Type().Name())
	if !ok {
		node, e = this.inspect(any)
		if e != nil {
			return nil, v, e
		}
	}
	return node, v, nil
}

// PrimaryKeyDecoratorValue extracts the primary key value from an instance.
// Returns the concatenated key string built from primary key field values.
func (this *Introspector) PrimaryKeyDecoratorValue(any interface{}) (string, *l8reflect.L8Node, error) {
	node, v, err := this.NodeFor(any)
	if err != nil {
		return "", node, err
	}
	return this.PrimaryKeyDecoratorFromValue(node, v)
}

// UniqueKeyDecoratorValue extracts the unique key value from an instance.
func (this *Introspector) UniqueKeyDecoratorValue(any interface{}) (string, *l8reflect.L8Node, error) {
	node, v, err := this.NodeFor(any)
	if err != nil {
		return "", node, err
	}
	return this.uniqueKeyDecoratorValue(node, v)
}

// NonUniqueKeyDecoratorValue extracts the non-unique key value from an instance.
func (this *Introspector) NonUniqueKeyDecoratorValue(any interface{}) (string, *l8reflect.L8Node, error) {
	node, v, err := this.NodeFor(any)
	if err != nil {
		return "", node, err
	}
	return this.nonUniqueKeyDecoratorValue(node, v)
}

// uniqueKeyDecoratorValue is internal helper for unique key extraction.
func (this *Introspector) uniqueKeyDecoratorValue(node *l8reflect.L8Node, value reflect.Value) (string, *l8reflect.L8Node, error) {
	return this.decoratorKey(node, l8reflect.L8DecoratorType_Unique, value)
}

// nonUniqueKeyDecoratorValue is internal helper for non-unique key extraction.
func (this *Introspector) nonUniqueKeyDecoratorValue(node *l8reflect.L8Node, value reflect.Value) (string, *l8reflect.L8Node, error) {
	return this.decoratorKey(node, l8reflect.L8DecoratorType_NonUnique, value)
}

// PrimaryKeyDecoratorFromValue extracts the primary key from a node and value.
func (this *Introspector) PrimaryKeyDecoratorFromValue(node *l8reflect.L8Node, value reflect.Value) (string, *l8reflect.L8Node, error) {
	return this.decoratorKey(node, l8reflect.L8DecoratorType_Primary, value)
}

// decoratorKey builds a key string from field values based on the decorator type.
// Fields are concatenated with "::" separator and type prefixes.
func (this *Introspector) decoratorKey(node *l8reflect.L8Node, decoratorType l8reflect.L8DecoratorType, value reflect.Value) (string, *l8reflect.L8Node, error) {
	fields, err := this.Fields(node, decoratorType)
	if err != nil {
		return "", node, err
	}

	str := strings2.New()
	str.TypesPrefix = true
	first := true
	for _, field := range fields {
		if !first {
			str.TypesPrefix = false
			str.Add("::")
			str.TypesPrefix = true
		}
		v := value.FieldByName(field).Interface()
		v2 := str.StringOf(v)
		str.Add(v2)
		first = false
	}
	return str.String(), node, nil
}

// Fields returns the field names associated with a decorator type on a node.
func (this *Introspector) Fields(node *l8reflect.L8Node, decoratorType l8reflect.L8DecoratorType) ([]string, error) {
	if node == nil {
		return nil, errors.New("Node is nil")
	}
	decValue := node.Decorators[int32(decoratorType)]
	if decValue == nil {
		return nil, errors.New(strings2.New("Decorator Not Found in ", node.TypeName).String())
	}
	return decValue.Fields, nil
}

// fieldInterface returns the interface value of a named field on value.
// Returns an error (when returnError is true) if value is zero/invalid or the
// field does not exist — previously these cases caused a reflect panic
// ("call of reflect.Value.Interface on zero Value") inside KeyForValue.
func fieldInterface(value reflect.Value, field, typeName string, returnError bool) (interface{}, error) {
	if !value.IsValid() {
		if returnError {
			return nil, errors.New(strings2.New("KeyForValue: zero/invalid value for type ", typeName,
				" (likely a typed-nil pointer) while resolving field ", field).String())
		}
		return nil, nil
	}
	fv := value.FieldByName(field)
	if !fv.IsValid() {
		if returnError {
			return nil, errors.New(strings2.New("KeyForValue: field ", field, " not found on type ", typeName).String())
		}
		return nil, nil
	}
	return fv.Interface(), nil
}

// KeyForValue builds a key string from the specified fields and value.
// Supports 1-3 fields efficiently with a fallback for more fields.
func (this *Introspector) KeyForValue(fields []string, value reflect.Value, typeName string, returnError bool) (string, error) {
	if fields == nil || len(fields) == 0 {
		if returnError {
			return "", errors.New(strings2.New("Primary Key Decorator is empty for type ", typeName).String())
		}
		return "", nil
	}
	vals := make([]interface{}, len(fields))
	for i, field := range fields {
		iv, err := fieldInterface(value, field, typeName, returnError)
		if err != nil {
			return "", err
		}
		vals[i] = iv
	}
	switch len(vals) {
	case 1:
		return strings2.New(vals[0]).String(), nil
	case 2:
		return strings2.New(vals[0], vals[1]).String(), nil
	case 3:
		return strings2.New(vals[0], vals[1], vals[2]).String(), nil
	default:
		result := strings2.New()
		for _, v := range vals {
			result.Add(result.StringOf(v))
		}
		return result.String(), nil
	}
}

// addDecorator is an internal helper to add a decorator with fields to a node.
func addDecorator(decoratorType l8reflect.L8DecoratorType, fields []string, node *l8reflect.L8Node) {
	if node.Decorators == nil {
		node.Decorators = make(map[int32]*l8reflect.L8Decorator)
	}
	node.Decorators[int32(decoratorType)] = &l8reflect.L8Decorator{Fields: fields}
}

// addNoNestedInspection marks a node to skip nested inspection.
func addNoNestedInspection(rnode *l8reflect.L8Node) {
	addDecorator(l8reflect.L8DecoratorType_NoNestedInspection, []string{}, rnode)
}

// NoNestedInspection checks if a type has the no-nested-inspection decorator.
func (this *Introspector) NoNestedInspection(any interface{}) bool {
	return this.BoolDecoratorValueFor(any, l8reflect.L8DecoratorType_NoNestedInspection)
}

// addAlwayOverwriteDecorator marks a node for always-full updates.
func addAlwayOverwriteDecorator(rnode *l8reflect.L8Node) {
	addDecorator(l8reflect.L8DecoratorType_AlwaysFull, []string{}, rnode)
}

// AlwaysFullDecorator checks if a type has the always-full decorator.
func (this *Introspector) AlwaysFullDecorator(any interface{}) bool {
	return this.BoolDecoratorValueFor(any, l8reflect.L8DecoratorType_AlwaysFull)
}

// BoolDecoratorValueFor checks if a decorator type exists on a value's type.
func (this *Introspector) BoolDecoratorValueFor(any interface{}, typ l8reflect.L8DecoratorType) bool {
	node, _, err := this.NodeFor(any)
	if err != nil {
		return false
	}
	return this.BoolDecoratorValueForNode(node, typ)
}

// BoolDecoratorValueForNode checks if a decorator type exists on a node.
func (this *Introspector) BoolDecoratorValueForNode(node *l8reflect.L8Node, typ l8reflect.L8DecoratorType) bool {
	if node == nil {
		return false
	}
	_, err := this.Fields(node, typ)
	if err != nil {
		return false
	}
	return true
}
