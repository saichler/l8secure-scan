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

// This file contains the slice comparator for detecting changes in slice fields.
// Handles slice creation, element updates, additions, and size reductions.

package updating

import (
	"bytes"
	"reflect"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8reflect/go/reflect/properties"
)

// sliceUpdate compares and updates slice values.
// Detects element changes, new elements, and deleted elements when newItemIsFull is true.
func sliceUpdate(instance *properties.Property, node *l8reflect.L8Node, oldValue, newValue reflect.Value, updates *Updater) error {
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
		updates.addUpdate(instance, oldValue, nil)
		if !updates.dryRun {
			oldValue.Set(newValue)
		}
		return nil
	}

	// Byte slices are treated as atomic values — replace old with new if different.
	if oldValue.Type().Elem().Kind() == reflect.Uint8 {
		oldBytes := oldValue.Bytes()
		newBytes := newValue.Bytes()
		if !bytes.Equal(oldBytes, newBytes) {
			updates.addUpdate(instance, oldBytes, newBytes)
			if !updates.dryRun {
				oldValue.Set(newValue)
			}
		}
		return nil
	}

	size := newValue.Len()
	if size > oldValue.Len() {
		size = oldValue.Len()
	}

	for i := 0; i < size; i++ {
		oldIndexValue := oldValue.Index(i)
		newIndexValue := newValue.Index(i)
		if !node.IsStruct {
			if oldIndexValue.IsValid() && deepEqual.Equal(oldIndexValue.Interface(), newIndexValue.Interface()) {
				continue
			}
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), i,
				newIndexValue.Interface(), updates.resources)
			updates.addUpdate(subProperty, nil, newIndexValue.Interface())
			if !updates.dryRun {
				oldIndexValue.Set(newIndexValue)
			}
		} else if !oldIndexValue.IsValid() || oldIndexValue.IsNil() {
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property),
				i, newIndexValue.Interface(), updates.resources)
			updates.addUpdate(subProperty, nil, newIndexValue.Interface())
			if !updates.dryRun {
				oldIndexValue.Set(newIndexValue)
			}
		} else if oldIndexValue.IsValid() && newIndexValue.IsValid() {
			if deepEqual.Equal(oldIndexValue.Interface(), newIndexValue.Interface()) {
				continue
			}
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property),
				i, newIndexValue.Interface(), updates.resources)
			err := structUpdate(subProperty, node, oldIndexValue.Elem(), newIndexValue.Elem(), updates)
			if err != nil {
				return err
			}
		}
	}

	vInfo, err := instance.Resources().Registry().Info(instance.Node().TypeName)
	if err != nil {
		return err
	}

	if size < oldValue.Len() && updates.newItemIsFull {
		var newSlice reflect.Value
		if node.IsStruct {
			newSlice = reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(vInfo.Type())), size, size)
		} else {
			newSlice = reflect.MakeSlice(reflect.SliceOf(vInfo.Type()), size, size)
		}

		for i := 0; i < size; i++ {
			newSlice.Index(i).Set(oldValue.Index(i))
		}
		subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), size,
			nil, updates.resources)
		updates.addUpdate(subProperty, nil, ifs.Deleted_Entry)
		if !updates.dryRun {
			oldValue.Set(newSlice)
		}
	} else if newValue.Len() > oldValue.Len() {
		newSlice := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(vInfo.Type())), newValue.Len(), newValue.Len())
		for i := 0; i < size; i++ {
			newSlice.Index(i).Set(oldValue.Index(i))
		}
		for i := size; i < newValue.Len(); i++ {
			newV := newValue.Index(i)
			newSlice.Index(i).Set(newV)
			subProperty := properties.NewProperty(node, instance.Parent().(*properties.Property), i,
				newV.Interface(), updates.resources)
			updates.addUpdate(subProperty, nil, newV.Interface())
		}
		if !updates.dryRun {
			oldValue.Set(newSlice)
		}
	}

	return nil
}
