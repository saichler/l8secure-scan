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

// This file contains zero-allocation callback-based getter methods for extracting
// values from nested data structures. Use ForEachValue when you need to iterate
// over values without allocating slices.

package properties

import (
	"reflect"
)

// ForEachValue calls fn for each value matching this property path.
// If fn returns false, iteration stops early.
// This is a zero-allocation alternative to GetValue for hot paths.
func (this *Property) ForEachValue(any reflect.Value, fn func(reflect.Value) bool) {
	if !any.IsValid() || (any.Kind() == reflect.Ptr && any.IsNil()) {
		return
	}
	if this.parent == nil {
		fn(any)
		return
	}

	this.parent.ForEachValue(any, func(parent reflect.Value) bool {
		if parent.Kind() == reflect.Ptr {
			parent = parent.Elem()
		}

		switch parent.Kind() {
		case reflect.Map:
			return this.forEachMapValue(parent, fn)
		case reflect.Slice:
			return this.forEachSliceValue(parent, fn)
		default:
			if parent.IsValid() {
				return fn(this.getField(parent))
			}
		}
		return true
	})
}

// forEachMapValue iterates over map values calling fn for each field value.
func (this *Property) forEachMapValue(parent reflect.Value, fn func(reflect.Value) bool) bool {
	if this.parent.key != nil {
		myValue := parent.MapIndex(reflect.ValueOf(this.parent.key))
		if !myValue.IsValid() {
			return true
		}
		if myValue.Kind() == reflect.Ptr {
			if myValue.IsNil() {
				return true
			}
			myValue = myValue.Elem()
		}
		return fn(this.getField(myValue))
	}

	iter := parent.MapRange()
	for iter.Next() {
		value := iter.Value()
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}
		if !fn(this.getField(value)) {
			return false
		}
	}
	return true
}

// forEachSliceValue iterates over slice values calling fn for each field value.
func (this *Property) forEachSliceValue(parent reflect.Value, fn func(reflect.Value) bool) bool {
	if this.parent.key != nil {
		myValue := parent.Index(this.parent.key.(int))
		if !myValue.IsValid() {
			return true
		}
		if myValue.Kind() == reflect.Ptr {
			if myValue.IsNil() {
				return true
			}
			myValue = myValue.Elem()
		}
		return fn(this.getField(myValue))
	}

	for i := 0; i < parent.Len(); i++ {
		value := parent.Index(i)
		if value.Kind() == reflect.Interface {
			value = value.Elem()
		}
		if value.Kind() == reflect.Ptr {
			if value.IsNil() {
				continue
			}
			value = value.Elem()
		}
		if !fn(this.getField(value)) {
			return false
		}
	}
	return true
}
