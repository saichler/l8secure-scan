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

// This file contains struct and pointer comparators for detecting changes.
// Handles recursive comparison of struct fields and pointer dereferencing.

package updating

import (
	"errors"
	"reflect"

	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8reflect/go/reflect/properties"
)

// ptrUpdate compares and updates pointer values.
// Handles nil-to-value, value-to-nil, and delegates to nested comparison for valid pointers.
func ptrUpdate(property *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if oldValue.IsNil() && !newValue.IsNil() {
		updates.addUpdate(property, nil, newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		return nil
	}
	if !oldValue.IsNil() && newValue.IsNil() && updates.nilIsValid {
		updates.addUpdate(property, oldValue, nil)
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		return nil
	}
	if oldValue.IsNil() && newValue.IsNil() {
		return nil
	}
	return update(property, node, oldValue.Elem(), newValue.Elem(), updates)
}

// structUpdate compares and updates struct values by recursively comparing each field.
func structUpdate(property *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if !oldValue.IsValid() && newValue.IsValid() {
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		updates.addUpdate(property, nil, newValue.Interface())
		return nil
	}
	if oldValue.IsValid() && !newValue.IsValid() && updates.nilIsValid {
		newValue.Set(reflect.New(oldValue.Type()).Elem())
		updates.addUpdate(property, oldValue.Interface(), newValue.Interface())
		return nil
	}
	if !oldValue.IsValid() && !newValue.IsValid() {
		return nil
	}

	if !newValue.IsValid() && oldValue.IsValid() {
		newValue = reflect.New(oldValue.Type()).Elem()
		newValue.Set(reflect.New(oldValue.Type()).Elem())
	}

	if oldValue.Type().Name() != newValue.Type().Name() {
		return errors.New("Mismatch type, old=" + oldValue.Type().Name() + ", new=" + newValue.Type().Name())
	}
	for _, attr := range node.Attributes {
		oldFldValue := oldValue.FieldByName(attr.FieldName)
		newFldValue := newValue.FieldByName(attr.FieldName)

		// Time series slices are append-only: record the whole new slice
		// as a single change and let timeSeriesAppend handle merging.
		// Never drill down into individual L8TimeSeriesPoint fields.
		if attr.IsSlice && attr.TypeName == "L8TimeSeriesPoint" {
			if !newFldValue.IsValid() || newFldValue.IsNil() || newFldValue.Len() == 0 {
				continue
			}
			if oldFldValue.IsValid() && !oldFldValue.IsNil() &&
				deepEqual.Equal(oldFldValue.Interface(), newFldValue.Interface()) {
				continue
			}
			subInstance := properties.NewProperty(attr, property, nil, oldFldValue, updates.resources)
			updates.addUpdate(subInstance, nil, newFldValue.Interface())
			if !updates.dryRun {
				oldFldValue.Set(newFldValue)
			}
			continue
		}

		subInstance := properties.NewProperty(attr, property, nil, oldFldValue, updates.resources)
		err := update(subInstance, attr, oldFldValue, newFldValue, updates)
		if err != nil {
			return err
		}
	}
	return nil
}
