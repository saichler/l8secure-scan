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

package introspecting

import (
	"reflect"

	"github.com/saichler/l8reflect/go/reflect/helping"
	"github.com/saichler/l8types/go/types/l8reflect"
)

// addAttribute creates a new L8Node for a field and adds it to the parent node's attributes.
// Registers the type in the registry and establishes parent-child relationship.
func (this *Introspector) addAttribute(node *l8reflect.L8Node, _type reflect.Type, _fieldName string) *l8reflect.L8Node {
	this.registry.RegisterType(_type)
	if node != nil && node.Attributes == nil {
		node.Attributes = make(map[string]*l8reflect.L8Node)
	}

	subNode := &l8reflect.L8Node{}
	subNode.TypeName = _type.Name()
	subNode.Parent = node
	subNode.FieldName = _fieldName

	if node != nil {
		node.Attributes[subNode.FieldName] = subNode
	}
	return subNode
}

// fixClone updates a cloned node tree with correct parent references and cache keys.
// Called recursively to fix all nodes in a cloned subtree.
func (this *Introspector) fixClone(clone *l8reflect.L8Node, parent *l8reflect.L8Node, fieldName string) {
	clone.Parent = parent
	clone.FieldName = fieldName
	clone.CachedKey = ""
	nodePath := helping.NodeCacheKey(clone)
	this.pathToNode.Put(nodePath, clone)
	if clone.Attributes != nil {
		for k, v := range clone.Attributes {
			this.fixClone(v, clone, k)
		}
	}
}

// addNode creates a new node for a type or returns a clone of an existing non-leaf node.
// Returns (node, true) if an existing node was cloned, (node, false) for new nodes.
func (this *Introspector) addNode(_type reflect.Type, _parent *l8reflect.L8Node, _fieldName string) (*l8reflect.L8Node, bool) {
	exist, ok := this.typeToNode.Get(_type.Name())
	if ok && !helping.IsLeaf(exist) {
		clone := this.cloner.Clone(exist).(*l8reflect.L8Node)
		this.fixClone(clone, _parent, _fieldName)
		if _parent != nil {
			if _parent.Attributes == nil {
				_parent.Attributes = make(map[string]*l8reflect.L8Node)
			}
			_parent.Attributes[_fieldName] = clone
		}
		return clone, true
	}

	node := this.addAttribute(_parent, _type, _fieldName)
	nodePath := helping.NodeCacheKey(node)
	_, ok = this.pathToNode.Get(nodePath)
	if ok {
		return nil, false
	}
	this.pathToNode.Put(nodePath, node)
	if _type.Kind() == reflect.Struct {
		this.typeToNode.Put(node.TypeName, node)
	}
	return node, false
}

// inspectStruct recursively inspects a struct type and builds its node tree.
// Iterates through all exported fields, handling slices, maps, pointers, and primitives.
func (this *Introspector) inspectStruct(_type reflect.Type, _parent *l8reflect.L8Node, _fieldName string) *l8reflect.L8Node {
	localNode, isClone := this.addNode(_type, _parent, _fieldName)
	if isClone {
		f, ok := _type.FieldByName(_fieldName)
		if ok && f.Type.Kind() == reflect.Slice {
			localNode.IsSlice = true
		} else {
			localNode.IsSlice = false
		}
		if ok && f.Type.Kind() == reflect.Map {
			localNode.IsMap = true
		} else {
			localNode.IsMap = false
		}
		return localNode
	}
	localNode.IsStruct = true
	this.registry.RegisterType(_type)
	for index := 0; index < _type.NumField(); index++ {
		field := _type.Field(index)
		if helping.IgnoreName(field.Name) {
			continue
		}
		if field.Type.Kind() == reflect.Slice {
			this.inspectSlice(field.Type, localNode, field.Name)
		} else if field.Type.Kind() == reflect.Map {
			this.inspectMap(field.Type, localNode, field.Name)
		} else if field.Type.Kind() == reflect.Ptr {
			subnode := this.inspectPtr(field.Type.Elem(), localNode, field.Name)
			this.typeToNode.Put(subnode.TypeName, subnode)
		} else {
			this.addNode(field.Type, localNode, field.Name)
		}
	}
	this.addTableView(localNode)
	return localNode
}

// inspectPtr handles pointer type inspection by delegating to the appropriate handler.
// Currently only supports pointers to structs.
func (this *Introspector) inspectPtr(_type reflect.Type, _parent *l8reflect.L8Node, _fieldName string) *l8reflect.L8Node {
	switch _type.Kind() {
	case reflect.Struct:
		return this.inspectStruct(_type, _parent, _fieldName)
	}
	panic("unknown ptr kind " + _type.Kind().String())
}

// inspectMap inspects a map type and creates appropriate nodes.
// Handles maps with struct pointer values specially by inspecting the struct.
func (this *Introspector) inspectMap(_type reflect.Type, _parent *l8reflect.L8Node, _fieldName string) *l8reflect.L8Node {
	if _type.Elem().Kind() == reflect.Ptr && _type.Elem().Elem().Kind() == reflect.Struct {
		subNode := this.inspectStruct(_type.Elem().Elem(), _parent, _fieldName)
		subNode.IsMap = true
		subNode.IsStruct = true
		subNode.KeyTypeName = _type.Key().Name()
		if _parent.Attributes == nil {
			_parent.Attributes = make(map[string]*l8reflect.L8Node)
		}
		_parent.Attributes[_fieldName] = subNode
		return subNode
	} else {
		subNode, _ := this.addNode(_type.Elem(), _parent, _fieldName)
		subNode.IsMap = true
		subNode.KeyTypeName = _type.Key().Name()
		return subNode
	}
}

// inspectSlice inspects a slice type and creates appropriate nodes.
// Handles slices of struct pointers specially by inspecting the struct.
func (this *Introspector) inspectSlice(_type reflect.Type, _parent *l8reflect.L8Node, _fieldName string) *l8reflect.L8Node {
	if _type.Elem().Kind() == reflect.Ptr && _type.Elem().Elem().Kind() == reflect.Struct {
		subNode := this.inspectStruct(_type.Elem().Elem(), _parent, _fieldName)
		subNode.IsSlice = true
		subNode.IsStruct = true
		if _parent.Attributes == nil {
			_parent.Attributes = make(map[string]*l8reflect.L8Node)
		}
		_parent.Attributes[_fieldName] = subNode
		return subNode
	} else {
		subNode, _ := this.addNode(_type.Elem(), _parent, _fieldName)
		subNode.IsSlice = true
		return subNode
	}
}
