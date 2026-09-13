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

// Package introspecting provides runtime type introspection and metadata extraction.
// It analyzes Go struct types to build a tree representation (L8Node) of their structure,
// including nested types, maps, slices, and decorators for primary keys and other metadata.
//
// Key features:
//   - Automatic type tree construction from Go structs
//   - Decorator support for primary keys, unique keys, and custom behaviors
//   - Table view generation for data representation
//   - Type registry integration for cross-system type management
package introspecting

import (
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/saichler/l8reflect/go/reflect/cloning"
	"github.com/saichler/l8reflect/go/reflect/helping"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/maps"
)

// Introspector provides runtime type analysis and metadata extraction for Go structs.
// It builds and maintains a tree representation of type structures with caching
// for efficient repeated access.
type Introspector struct {
	// pathToNode maps dot-separated paths to their corresponding L8Node
	pathToNode *RNodeMap
	// typeToNode maps type names to their corresponding L8Node
	typeToNode *RNodeMap
	// registry stores type information for serialization/deserialization
	registry ifs.IRegistry
	// cloner provides deep cloning for node duplication
	cloner *cloning.Cloner
	// tableViews stores table view representations of types
	tableViews *maps.SyncMap
	mutex      *sync.Mutex
}

// NewIntrospect creates a new Introspector with the given type registry.
// The Introspector is ready to inspect Go structs and build type metadata.
func NewIntrospect(registry ifs.IRegistry) *Introspector {
	introspector := &Introspector{}
	introspector.mutex = &sync.Mutex{}
	introspector.registry = registry
	introspector.cloner = cloning.NewCloner()
	introspector.pathToNode = NewIntrospectNodeMap()
	introspector.typeToNode = NewIntrospectNodeMap()
	introspector.tableViews = maps.NewSyncMap()
	return introspector
}

// Registry returns the type registry associated with this Introspector.
func (this *Introspector) Registry() ifs.IRegistry {
	return this.registry
}

// Inspect analyzes a Go struct and returns its L8Node representation.
// The node tree is cached for subsequent lookups.
// Returns an error if the input is nil or not a struct type.
// This method is thread-safe.
func (this *Introspector) Inspect(any interface{}) (*l8reflect.L8Node, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.inspect(any)
}

// inspect is the internal unlocked version of Inspect.
// Callers must hold this.mutex before calling.
func (this *Introspector) inspect(any interface{}) (*l8reflect.L8Node, error) {
	if any == nil {
		return nil, errors.New("Cannot introspect a nil value")
	}

	_, t := helping.ValueAndType(any)
	if t.Kind() == reflect.Slice && t.Kind() == reflect.Map {
		t = t.Elem().Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, errors.New("Cannot introspect a value that is not a struct")
	}
	localNode, ok := this.pathToNode.Get(strings.ToLower(t.Name()))
	if ok {
		return localNode, nil
	}
	return this.inspectStruct(t, nil, ""), nil
}

// Node retrieves an L8Node by its dot-separated path (case-insensitive).
func (this *Introspector) Node(path string) (*l8reflect.L8Node, bool) {
	return this.pathToNode.Get(strings.ToLower(path))
}

// NodeByValue retrieves an L8Node for the type of the given value.
func (this *Introspector) NodeByValue(any interface{}) (*l8reflect.L8Node, bool) {
	val := reflect.ValueOf(any)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return this.NodeByType(val.Type())
}

// NodeByType retrieves an L8Node for the given reflect.Type.
func (this *Introspector) NodeByType(typ reflect.Type) (*l8reflect.L8Node, bool) {
	return this.NodeByTypeName(typ.Name())
}

// NodeByTypeName retrieves an L8Node by type name.
func (this *Introspector) NodeByTypeName(name string) (*l8reflect.L8Node, bool) {
	return this.typeToNode.Get(name)
}

// Nodes returns a list of L8Nodes, optionally filtered by leaf or root status.
// Set onlyLeafs=true to return only leaf nodes (no children).
// Set onlyRoots=true to return only root nodes (no parent).
func (this *Introspector) Nodes(onlyLeafs, onlyRoots bool) []*l8reflect.L8Node {
	if onlyLeafs && onlyRoots {
		panic("Nodes: onlyLeafs and onlyRoots cannot both be true — no node can be both a leaf and a root")
	}
	filter := func(any interface{}) bool {
		n := any.(*l8reflect.L8Node)
		if onlyLeafs && !helping.IsLeaf(n) {
			return false
		}
		if onlyRoots && !helping.IsRoot(n) {
			return false
		}
		return true
	}

	return this.pathToNode.NodesList(filter)
}

// Kind returns the reflect.Kind for the type represented by the given node.
func (this *Introspector) Kind(node *l8reflect.L8Node) reflect.Kind {
	info, err := this.registry.Info(node.TypeName)
	if err != nil {
		panic(err.Error())
	}
	return info.Type().Kind()
}

// Clone performs a deep clone of the given value using the internal cloner.
func (this *Introspector) Clone(any interface{}) interface{} {
	return this.cloner.Clone(any)
}

// addTableView creates and stores a table view representation for a node.
// A table view separates leaf columns from nested subtables.
func (this *Introspector) addTableView(node *l8reflect.L8Node) {
	tv := &l8reflect.L8TableView{Table: node, Columns: make([]*l8reflect.L8Node, 0), SubTables: make([]*l8reflect.L8Node, 0)}
	for _, attr := range node.Attributes {
		if helping.IsLeaf(attr) {
			tv.Columns = append(tv.Columns, attr)
		} else {
			tv.SubTables = append(tv.SubTables, attr)
		}
	}
	this.tableViews.Put(node.TypeName, tv)
}

// TableView retrieves a table view by type name.
func (this *Introspector) TableView(name string) (*l8reflect.L8TableView, bool) {
	tv, ok := this.tableViews.Get(name)
	if !ok {
		return nil, ok
	}
	return tv.(*l8reflect.L8TableView), ok
}

// TableViews returns all registered table views.
func (this *Introspector) TableViews() []*l8reflect.L8TableView {
	list := this.tableViews.ValuesAsList(reflect.TypeOf(&l8reflect.L8TableView{}), nil)
	return list.([]*l8reflect.L8TableView)
}

// Clean removes a type and all its nested types from the introspector caches.
// This method is thread-safe.
func (this *Introspector) Clean(typeName string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	node, ok := this.NodeByTypeName(typeName)
	if !ok {
		return
	}
	this.clean(node)
}

func (this *Introspector) clean(node *l8reflect.L8Node) {
	if node.Attributes != nil {
		for _, attr := range node.Attributes {
			this.clean(attr)
		}
	}
	this.typeToNode.Del(node.TypeName)
	this.pathToNode.Del(helping.NodeCacheKey(node))
}
