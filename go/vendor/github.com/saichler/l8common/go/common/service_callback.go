/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package common

import (
	"errors"
	"fmt"
	"github.com/saichler/l8types/go/ifs"
)

// ValidateFunc is a function that validates an entity.
// The entity is passed as interface{} and must be type-asserted by the caller.
type ValidateFunc func(interface{}, ifs.IVNic) error

// ActionValidateFunc is a function that validates an entity with access to the CRUD action.
type ActionValidateFunc func(interface{}, ifs.Action, ifs.IVNic) error

// SetIDFunc is a function that generates/sets the primary key on an entity.
type SetIDFunc func(interface{})

type genericCallback struct {
	typeName         string
	typeCheck        func(interface{}) bool
	setID            SetIDFunc
	validate         ValidateFunc
	actionValidators []ActionValidateFunc
	afterActions     []ActionValidateFunc
}

// NewServiceCallback creates a standard IServiceCallback.
// typeCheck should verify the entity type (e.g., func(v interface{}) bool { _, ok := v.(*MyType); return ok }).
func NewServiceCallback(typeName string, typeCheck func(interface{}) bool, setID SetIDFunc, validate ValidateFunc, actionValidators ...ActionValidateFunc) ifs.IServiceCallback {
	return &genericCallback{
		typeName:         typeName,
		typeCheck:        typeCheck,
		setID:            setID,
		validate:         validate,
		actionValidators: actionValidators,
	}
}

// NewServiceCallbackWithAfter creates a ServiceCallback with both action validators
// and after-actions that run after successful PUT/PATCH persistence.
func NewServiceCallbackWithAfter(typeName string, typeCheck func(interface{}) bool, setID SetIDFunc, validate ValidateFunc, actionValidators []ActionValidateFunc, afterActions []ActionValidateFunc) ifs.IServiceCallback {
	return &genericCallback{
		typeName:         typeName,
		typeCheck:        typeCheck,
		setID:            setID,
		validate:         validate,
		actionValidators: actionValidators,
		afterActions:     afterActions,
	}
}

func (cb *genericCallback) Before(any interface{}, action ifs.Action, cont bool, vnic ifs.IVNic) (interface{}, bool, error) {
	if !cb.typeCheck(any) {
		return nil, false, errors.New("invalid " + cb.typeName + " type")
	}
	if action == ifs.POST {
		cb.setID(any)
	}
	for _, av := range cb.actionValidators {
		if err := av(any, action, vnic); err != nil {
			return nil, false, err
		}
	}
	if cb.validate != nil {
		if err := cb.validate(any, vnic); err != nil {
			return nil, false, err
		}
	}
	return nil, true, nil
}

func (cb *genericCallback) After(any interface{}, action ifs.Action, cont bool, vnic ifs.IVNic) (interface{}, bool, error) {
	if (action != ifs.PUT && action != ifs.PATCH) || len(cb.afterActions) == 0 {
		return nil, true, nil
	}
	if !cb.typeCheck(any) {
		return nil, true, nil
	}
	for _, aa := range cb.afterActions {
		if err := aa(any, action, vnic); err != nil {
			fmt.Println("[cascade] warning:", err.Error())
		}
	}
	return nil, true, nil
}
