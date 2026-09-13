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

// Package helping provides utility functions for reflection operations.
// It includes helpers for value/type extraction, node navigation, and field filtering.
package helping

import (
	"errors"
	"reflect"
	"strings"

	"github.com/saichler/l8types/go/types/l8reflect"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// ValueAndType extracts the reflect.Value and reflect.Type from an interface.
// If the value is a pointer, it dereferences it to get the underlying value.
// Returns the dereferenced value and its type.
func ValueAndType(any interface{}) (reflect.Value, reflect.Type) {
	v := reflect.ValueOf(any)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	return v, t
}

// PtrValue extracts and validates a pointer value from an interface.
// Returns an error if the input is nil, invalid, not a pointer, or a nil pointer.
// Returns the dereferenced value on success.
func PtrValue(any interface{}) (reflect.Value, error) {
	if any == nil {
		return reflect.Value{}, errors.New("[PtrValueAndType] input is nil")
	}
	value := reflect.ValueOf(any)
	if !value.IsValid() {
		return reflect.Value{}, errors.New("[PtrValueAndType] Value is invalid")
	}
	if value.Kind() != reflect.Ptr {
		return reflect.Value{}, errors.New("[PtrValueAndType] Value is not Ptr")
	}
	if value.IsNil() {
		return reflect.Value{}, errors.New("[PtrValueAndType] Value is nil")
	}
	value = value.Elem()
	return value, nil
}

// IsLeaf checks if an L8Node is a leaf node (has no child attributes).
// A leaf node represents a primitive type or a type without nested fields.
func IsLeaf(node *l8reflect.L8Node) bool {
	if node.Attributes == nil || len(node.Attributes) == 0 {
		return true
	}
	return false
}

// IsRoot checks if an L8Node is a root node (has no parent).
// The root node is the top-level node in the introspection tree.
func IsRoot(node *l8reflect.L8Node) bool {
	if node.Parent == nil {
		return true
	}
	return false
}

// IgnoreName determines if a field should be ignored during processing.
// Fields are ignored if they match any of the following criteria:
//   - Field name is "DoNotCompare"
//   - Field name is "DoNotCopy"
//   - Field name starts with "XXX" (protobuf internal fields)
//   - Field name starts with a lowercase letter (unexported/private fields)
func IgnoreName(fieldName string) bool {
	if fieldName == "DoNotCompare" {
		return true
	}
	if fieldName == "DoNotCopy" {
		return true
	}
	if len(fieldName) > 3 && fieldName[0:3] == "XXX" {
		return true
	}
	if fieldName[0:1] == strings.ToLower(fieldName[0:1]) {
		return true
	}
	return false
}

// PropertyNodeKey extracts the node key from an instance ID by removing key segments.
// Key segments are enclosed in angle brackets (<key>).
// For example, "person.addresses<home>.street" becomes "person.addresses.street".
func PropertyNodeKey(instanceId string) string {
	buff := strings2.New()
	open := false
	for _, c := range instanceId {
		if c == '<' {
			open = true
		} else if c == '>' {
			open = false
		} else if !open {
			buff.Add(string(c))
		}
	}
	return buff.String()
}

// NodeCacheKey generates a unique cache key for an L8Node based on its path.
// The key is built by traversing from the root to the current node,
// concatenating lowercase type/field names separated by dots.
// The result is cached in the node for subsequent lookups.
func NodeCacheKey(node *l8reflect.L8Node) string {
	if node.CachedKey != "" {
		return node.CachedKey
	}
	if node.Parent == nil {
		return strings.ToLower(node.TypeName)
	}
	buff := strings2.New()
	buff.Add(NodeCacheKey(node.Parent))
	buff.Add(".")
	buff.Add(strings.ToLower(node.FieldName))
	node.CachedKey = buff.String()
	return node.CachedKey
}
