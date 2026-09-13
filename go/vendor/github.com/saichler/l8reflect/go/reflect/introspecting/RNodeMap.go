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

	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/maps"
)

// Type reference for L8Node used in list conversions
var _node *l8reflect.L8Node
var _nodeType = reflect.TypeOf(_node)

// RNodeMap is a thread-safe map for storing L8Node references.
// It wraps a SyncMap to provide type-safe access to introspection nodes.
type RNodeMap struct {
	impl *maps.SyncMap
}

// NewIntrospectNodeMap creates a new empty RNodeMap.
func NewIntrospectNodeMap() *RNodeMap {
	m := &RNodeMap{}
	m.impl = maps.NewSyncMap()
	return m
}

// Put stores an L8Node with the given key. Returns true if successful.
func (this *RNodeMap) Put(key string, value *l8reflect.L8Node) bool {
	return this.impl.Put(key, value)
}

// Get retrieves an L8Node by key. Returns the node and true if found.
func (this *RNodeMap) Get(key string) (*l8reflect.L8Node, bool) {
	value, ok := this.impl.Get(key)
	if value != nil {
		return value.(*l8reflect.L8Node), ok
	}
	return nil, ok
}

// Contains checks if a key exists in the map.
func (this *RNodeMap) Contains(key string) bool {
	return this.impl.Contains(key)
}

// NodesList returns all nodes as a slice, optionally filtered by the provided function.
func (this *RNodeMap) NodesList(filter func(v interface{}) bool) []*l8reflect.L8Node {
	return this.impl.ValuesAsList(_nodeType, filter).([]*l8reflect.L8Node)
}

// Iterate calls the provided function for each key-value pair in the map.
func (this *RNodeMap) Iterate(do func(k, v interface{})) {
	this.impl.Iterate(do)
}

// Del removes a node by key. Returns true if the key existed.
func (this *RNodeMap) Del(key string) bool {
	_, ok := this.impl.Delete(key)
	return ok
}
