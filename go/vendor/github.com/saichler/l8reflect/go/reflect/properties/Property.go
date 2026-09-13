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

// Package properties provides path-based access to nested data structures.
// It allows getting and setting values in complex objects using dot-notation
// property paths like "person.addresses<home>.street".
//
// Key features:
//   - Path-based property access with map/slice key notation
//   - Get and set operations on nested structures
//   - Property tree navigation with parent-child relationships
//   - Value collection across entire object hierarchies
package properties

import (
	"errors"
	"reflect"
	"strings"

	"github.com/saichler/l8reflect/go/reflect/helping"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// Property represents a path to a specific location in a data structure.
// It maintains a chain of parent properties to enable navigation from
// the root to any nested value.
type Property struct {
	// parent is the Property one level up in the hierarchy
	parent *Property
	// node is the L8Node describing this property's type
	node *l8reflect.L8Node
	// key is the map/slice key for indexed properties
	key interface{}
	// value is the current value at this property (if retrieved)
	value interface{}
	// id is the cached property path string
	id string
	// displayId is the human-readable property path
	displayId string
	// isLeaf indicates if this is a leaf property (no children accessed)
	isLeaf bool
	// resources provides access to introspection and registry
	resources ifs.IResources
	// fieldIndex is the cached struct field index for this property's field
	// in its parent struct. Pre-computed to avoid FieldByName linear search.
	fieldIndex int
}

// NewProperty creates a new Property with the given parameters.
// Sets up parent-child relationship and marks parent as non-leaf.
func NewProperty(node *l8reflect.L8Node, parent *Property, key interface{}, value interface{}, resources ifs.IResources) *Property {
	property := &Property{}
	property.parent = parent
	property.node = node
	property.key = key
	property.value = value
	property.resources = resources
	property.isLeaf = true
	property.fieldIndex = -1
	if parent != nil {
		parent.isLeaf = false
	}
	// Pre-compute field index to avoid FieldByName linear search
	property.fieldIndex = property.computeFieldIndex()
	return property
}

// PropertyOf parses a property path string and returns the corresponding Property.
// Example path: "person.addresses<home>.street"
func PropertyOf(propertyId string, resources ifs.IResources) (*Property, error) {
	propertyKey := helping.PropertyNodeKey(propertyId)
	node, ok := resources.Introspector().Node(propertyKey)
	if !ok {
		return nil, errors.New("Unknown attribute " + propertyKey)
	}
	return newProperty(node, propertyId, resources)
}

// Parent returns the parent property in the path hierarchy.
func (this *Property) Parent() ifs.IProperty {
	return this.parent
}

// Node returns the L8Node associated with this property.
func (this *Property) Node() *l8reflect.L8Node {
	return this.node
}

// Key returns the map/slice key for this property (nil for non-indexed properties).
func (this *Property) Key() interface{} {
	return this.key
}

// Value returns the value stored at this property.
func (this *Property) Value() interface{} {
	return this.value
}

// Resources returns the resources instance for introspection and registry access.
func (this *Property) Resources() ifs.IResources {
	return this.resources
}

// setKeyValue extracts and sets the key from a property path.
// Returns the remaining prefix path after extracting the key.
func (this *Property) setKeyValue(propertyId string) (string, error) {
	id := propertyId
	dIndex := strings.LastIndex(propertyId, ".")
	if dIndex == -1 {
		return "", nil
	}
	beIndex := strings.LastIndex(propertyId, ">")
	if beIndex == -1 {
		return "", nil
	}

	if dIndex > beIndex {
		prefix := propertyId[0:dIndex]
		return prefix, nil
	}

	bsIndex := strings.LastIndex(propertyId, "<")
	if dIndex > bsIndex {
		id = propertyId[:bsIndex]
		dIndex = strings.LastIndex(id, ".")
	}
	prefix := propertyId[0:dIndex]
	suffix := propertyId[dIndex+1:]
	bbIndex := strings.LastIndex(suffix, "<")
	if bbIndex == -1 {
		return prefix, nil
	}

	v := suffix[bbIndex+1 : len(suffix)-1]
	k, e := strings2.FromString(v, this.resources.Registry())
	if e != nil {
		return "", e
	}
	this.key = k.Interface()
	return prefix, nil
}

// IsString returns true if this property holds a string value.
func (this *Property) IsString() bool {
	if this.node.TypeName == reflect.String.String() {
		return true
	}
	return false
}

// PropertyId generates and caches the unique path string for this property.
// Format: "typename.field<key>.subfield<key>"
func (this *Property) PropertyId() (string, error) {
	if this.id != "" {
		return this.id, nil
	}
	buff := strings2.New()
	if this.parent == nil {
		buff.Add(strings.ToLower(this.node.TypeName))
		buff.Add(this.node.CachedKey)
	} else {
		pi, err := this.parent.PropertyId()
		if err != nil {
			return "", err
		}
		buff.Add(pi)
		buff.Add(".")
		buff.Add(strings.ToLower(this.node.FieldName))
	}
	if this.key != nil {
		keyStr := strings2.New()
		keyStr.TypesPrefix = true
		buff.Add("<")
		buff.Add(keyStr.StringOf(this.key))
		buff.Add(">")
	}
	this.id = buff.String()
	return this.id, nil
}

// PropertyDisplayId generates a human-readable display version of the property path.
// Format: "[TypeName][Field Key][SubField Key]"
func (this *Property) PropertyDisplayId() string {
	if this.displayId != "" {
		return this.displayId
	}
	buff := strings2.New()
	if this.parent == nil {
		buff.Add("[")
		buff.Add(this.node.TypeName)
		buff.Add(this.node.CachedKey)
		buff.Add("]")
		return buff.String()
	} else {
		pi := this.parent.PropertyDisplayId()
		buff.Add(pi)
		buff.Add("[")
		buff.Add(this.node.FieldName)
	}
	if this.key != nil {
		keyStr := strings2.New()
		buff.Add(" ")
		buff.Add(keyStr.StringOf(this.key))
	}
	buff.Add("]")
	this.displayId = buff.String()
	return this.displayId
}

// IsLeaf returns true if no child properties have been accessed through this property.
func (this *Property) IsLeaf() bool {
	return this.isLeaf
}

// computeFieldIndex finds the struct field index for this property's FieldName
// in the parent type. Returns -1 if the field cannot be found (e.g., root property).
func (this *Property) computeFieldIndex() int {
	if this.node == nil || this.node.Parent == nil || this.node.FieldName == "" {
		return -1
	}
	if this.resources == nil || this.resources.Registry() == nil {
		return -1
	}
	parentTypeName := this.node.Parent.TypeName
	info, err := this.resources.Registry().Info(parentTypeName)
	if err != nil {
		return -1
	}
	parentType := info.Type()
	if parentType.Kind() == reflect.Ptr {
		parentType = parentType.Elem()
	}
	for i := 0; i < parentType.NumField(); i++ {
		if parentType.Field(i).Name == this.node.FieldName {
			return i
		}
	}
	return -1
}

// newProperty is an internal constructor that builds a Property from a path string.
// Recursively builds the parent chain from the property path.
func newProperty(node *l8reflect.L8Node, propertyPath string, resources ifs.IResources) (*Property, error) {
	property := &Property{}
	property.isLeaf = true
	property.node = node
	property.resources = resources
	property.fieldIndex = -1
	if node.Parent != nil {
		prefix, err := property.setKeyValue(propertyPath)
		if err != nil {
			return nil, err
		}
		pi, err := newProperty(node.Parent, prefix, resources)
		if err != nil {
			return nil, err
		}
		property.parent = pi
		pi.isLeaf = false
		// Pre-compute field index to avoid FieldByName linear search
		property.fieldIndex = property.computeFieldIndex()
	} else {
		index1 := strings.Index(propertyPath, "<")
		index2 := strings.Index(propertyPath, ">")
		if index1 != -1 && index2 != -1 && index2 > index1 {
			k, e := strings2.FromString(propertyPath[index1+1:index2], property.resources.Registry())
			if e != nil {
				return nil, e
			}
			property.key = k.Interface()
		}
	}
	return property, nil
}
