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

// This file contains type-specific comparator functions for primitive types.
// Each comparator detects changes between old and new values and records updates.
// Comparators handle int, uint, string, bool, and float types.

package updating

import (
	"reflect"

	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8reflect/go/reflect/cloning"
	"github.com/saichler/l8reflect/go/reflect/properties"
)

// comparators maps reflect.Kind to the appropriate comparison function for that type.
var comparators map[reflect.Kind]func(*properties.Property, *l8reflect.L8Node, reflect.Value, reflect.Value, *Updater) error

// deepEqual is used for comparing complex values like structs and slices.
var deepEqual = cloning.NewDeepEqual()

func init() {
	comparators = make(map[reflect.Kind]func(*properties.Property, *l8reflect.L8Node, reflect.Value, reflect.Value, *Updater) error)
	comparators[reflect.Int] = intUpdate
	comparators[reflect.Int8] = intUpdate
	comparators[reflect.Int16] = intUpdate
	comparators[reflect.Int32] = intUpdate
	comparators[reflect.Int64] = intUpdate

	comparators[reflect.Uint] = uintUpdate
	comparators[reflect.Uint8] = uintUpdate
	comparators[reflect.Uint16] = uintUpdate
	comparators[reflect.Uint32] = uintUpdate
	comparators[reflect.Uint64] = uintUpdate

	comparators[reflect.String] = stringUpdate

	comparators[reflect.Bool] = boolUpdate

	comparators[reflect.Float32] = floatUpdate
	comparators[reflect.Float64] = floatUpdate

	comparators[reflect.Ptr] = ptrUpdate

	comparators[reflect.Struct] = structUpdate

	comparators[reflect.Slice] = sliceUpdate

	comparators[reflect.Map] = mapUpdate
}

// intUpdate compares and updates signed integer values.
func intUpdate(property *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if oldValue.Int() != newValue.Int() && (newValue.Int() != 0 || updates.nilIsValid) {
		updates.addUpdate(property, oldValue.Interface(), newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
	}
	return nil
}

// uintUpdate compares and updates unsigned integer values.
func uintUpdate(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if oldValue.Uint() != newValue.Uint() && (newValue.Uint() != 0 || updates.nilIsValid) {
		updates.addUpdate(instance, oldValue.Interface(), newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
	}
	return nil
}

// stringUpdate compares and updates string values.
func stringUpdate(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if oldValue.String() != newValue.String() && (newValue.String() != "" || updates.nilIsValid) {
		updates.addUpdate(instance, oldValue.Interface(), newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
	}
	return nil
}

// boolUpdate compares and updates boolean values.
// Unlike other primitives, false is a meaningful value for bools (not a "zero"),
// so any difference is always recorded regardless of nilIsValid.
func boolUpdate(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if newValue.Bool() == oldValue.Bool() {
		return nil
	}
	updates.addUpdate(instance, oldValue.Interface(), newValue.Interface())
	if !updates.dryRun {
		oldValue.Set(newValue)
	}
	return nil
}

// floatUpdate compares and updates floating-point values.
func floatUpdate(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if oldValue.Float() != newValue.Float() && (newValue.Float() != 0 || updates.nilIsValid) {
		updates.addUpdate(instance, oldValue.Interface(), newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
	}
	return nil
}
