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

// Package updating provides differential update tracking between two instances of the same type.
// It compares old and new values recursively, detects changes, applies updates to the old instance,
// and records a list of Change objects that describe what was modified.

package updating

import (
	"errors"
	"reflect"

	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
)

// Updater performs differential updates between two object instances.
// It tracks changes, applies updates to the old instance, and stores a list of changes.
type Updater struct {
	// changes holds the list of detected changes
	changes []*Change
	// resources provides access to introspection and registry
	resources ifs.IResources
	// nilIsValid when true allows nil/zero values to overwrite existing values
	nilIsValid bool
	// newItemIsFull when true treats the new item as a complete replacement, detecting deletions
	newItemIsFull bool
	// dryRun when true skips mutation of the old instance, only recording changes
	dryRun bool
}

// NewUpdater creates a new Updater with the given configuration.
// isNilValid controls whether nil/zero values can overwrite existing values.
// newItemIsFull controls whether the new item represents a complete replacement.
func NewUpdater(resources ifs.IResources, isNilValid, newItemIsFull bool) *Updater {
	upd := &Updater{}
	upd.resources = resources
	upd.nilIsValid = isNilValid
	upd.newItemIsFull = newItemIsFull
	return upd
}

// Changes returns the list of changes detected during the update operation.
func (this *Updater) Changes() []*Change {
	return this.changes
}

// DryUpdate compares old and new instances and records all modifications without mutating old.
// Returns an error if either value is nil or if type comparison fails.
func (this *Updater) DryUpdate(old, new interface{}) error {
	this.dryRun = true
	defer func() { this.dryRun = false }()
	return this.Update(old, new)
}

// Update compares old and new instances, applies changes to old, and records all modifications.
// Returns an error if either value is nil or if type comparison fails.
func (this *Updater) Update(old, new interface{}) error {
	oldValue := reflect.ValueOf(old)
	newValue := reflect.ValueOf(new)
	if !oldValue.IsValid() || !newValue.IsValid() {
		return errors.New("either old or new are nil or invalid")
	}
	if oldValue.Kind() == reflect.Ptr {
		oldValue = oldValue.Elem()
		newValue = newValue.Elem()
	}
	pKey, node, err := this.resources.Introspector().Decorators().PrimaryKeyDecoratorValue(old)
	if err != nil {
		return err
	}
	prop := properties.NewProperty(node, nil, pKey, oldValue, this.resources)
	return update(prop, node, oldValue, newValue, this)
}

// update is the internal recursive update function that dispatches to type-specific comparators.
func update(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if !newValue.IsValid() {
		return nil
	}
	if newValue.Kind() == reflect.Ptr && newValue.IsNil() && !updates.nilIsValid {
		return nil
	}

	kind := oldValue.Kind()
	comparator := comparators[kind]
	if comparator == nil {
		panic("No comparator for kind:" + kind.String() + ", please add it!")
	}
	return comparator(instance, node, oldValue, newValue, updates)
}

// addUpdate records a change for the given property with old and new values.
func (this *Updater) addUpdate(prop *properties.Property, oldValue, newValue interface{}) {
	if !this.nilIsValid && newValue == nil {
		return
	}
	if this.changes == nil {
		this.changes = make([]*Change, 0)
	}
	this.changes = append(this.changes, NewChange(oldValue, newValue, prop))
}
