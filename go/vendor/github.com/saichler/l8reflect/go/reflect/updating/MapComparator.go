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

// This file contains the map comparator for detecting changes in map fields.
// Handles map creation, entry additions, modifications, and deletions.

package updating

import (
	"reflect"

	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
)

// mapUpdate compares and updates map values.
// Detects new entries, modified entries, and deleted entries when newItemIsFull is true.
func mapUpdate(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
	if oldValue.IsNil() && newValue.IsNil() {
		return nil
	}
	if oldValue.IsNil() && !newValue.IsNil() {
		updates.addUpdate(instance, nil, newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		return nil
	}
	if !oldValue.IsNil() && newValue.IsNil() && updates.nilIsValid {
		updates.addUpdate(instance, oldValue.Interface(), nil)
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		return nil
	}

	alwaysFullDecorator := instance.Resources().Introspector().Decorators().BoolDecoratorValueForNode(node, l8reflect.L8DecoratorType_AlwaysFull)
	if newValue.IsValid() && !newValue.IsNil() && alwaysFullDecorator {
		updates.addUpdate(instance, nil, newValue.Interface())
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		return nil
	}

	newKeys := newValue.MapKeys()
	for _, key := range newKeys {
		oldKeyValue := oldValue.MapIndex(key)
		newKeyValue := newValue.MapIndex(key)

		if !oldKeyValue.IsValid() {
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), key.Interface(),
				newKeyValue.Interface(), updates.resources)
			updates.addUpdate(subProperty, nil, newKeyValue.Interface())
			if !updates.dryRun {
				oldValue.SetMapIndex(key, newKeyValue)
			}
			continue
		}

		if !node.IsStruct {
			if deepEqual.Equal(oldKeyValue.Interface(), newKeyValue.Interface()) {
				continue
			}
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), key.Interface(), newKeyValue.Interface(), updates.resources)
			updates.addUpdate(subProperty, nil, newKeyValue.Interface())
			if !updates.dryRun {
				oldValue.SetMapIndex(key, newKeyValue)
			}
		} else if oldKeyValue.IsValid() && newKeyValue.IsValid() {
			if deepEqual.Equal(oldKeyValue.Interface(), newKeyValue.Interface()) {
				continue
			}
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), key.Interface(), newKeyValue.Interface(), updates.resources)
			err := structUpdate(subProperty, node, oldKeyValue.Elem(), newKeyValue.Elem(), updates)
			if err != nil {
				return err
			}
		}
	}

	if updates.newItemIsFull {
		oldKeys := oldValue.MapKeys()
		for _, key := range oldKeys {
			newKeyValue := newValue.MapIndex(key)
			oldKeyValue := oldValue.MapIndex(key)
			if !newKeyValue.IsValid() {
				subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), key.Interface(), nil, updates.resources)
				updates.addUpdate(subProperty, oldKeyValue.Interface(), ifs.Deleted_Entry)
				if !updates.dryRun {
					oldValue.SetMapIndex(key, reflect.Value{})
				}
			}
		}
	}
	return nil
}
