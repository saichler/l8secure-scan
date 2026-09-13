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

// This file defines the Change type which represents a single modification
// detected during an update operation. Changes track the property path,
// old value, and new value for each detected difference.

package updating

import (
	"github.com/saichler/l8reflect/go/reflect/properties"
	"github.com/saichler/l8utils/go/utils/strings"
)

// Change represents a single modification detected during an update operation.
// It stores the property path where the change occurred along with old and new values.
type Change struct {
	// property is the path to the modified location
	property *properties.Property
	// oldValue is the value before the update
	oldValue interface{}
	// newValue is the value after the update
	newValue interface{}
}

// String returns a human-readable representation of the change.
func (this *Change) String() (string, error) {
	id, err := this.property.PropertyId()
	if err != nil {
		return "", err
	}
	str := strings.New(id)

	str.Add(" - Old=").Add(str.StringOf(this.oldValue)).
		Add(" New=").Add(str.StringOf(this.newValue))
	return str.String(), nil
}

// Apply applies this change to the given object, setting the new value at the property path.
func (this *Change) Apply(any interface{}) {
	this.property.Set(any, this.newValue)
}

// PropertyId returns the property path identifier for this change.
func (this *Change) PropertyId() string {
	id, _ := this.property.PropertyId()
	return id
}

// OldValue returns the value before the change was applied.
func (this *Change) OldValue() interface{} {
	return this.oldValue
}

// NewValue returns the value after the change was applied.
func (this *Change) NewValue() interface{} {
	return this.newValue
}

// NewChange creates a new Change with the given old value, new value, and property path.
func NewChange(old, new interface{}, property *properties.Property) *Change {
	change := &Change{}
	change.oldValue = old
	change.newValue = new
	change.property = property
	return change
}
