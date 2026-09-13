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
	"fmt"
	l8common "github.com/saichler/l8common/go/types/l8common"
	"github.com/saichler/l8types/go/ifs"
	l8api "github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8reflect"
	"reflect"
)

// VB (Validation Builder) chains validators for a ServiceCallback.
// Use NewValidation to start building, chain .Require() calls, then .Build().
// All getter functions accept interface{} — the caller must type-assert inside.
type VB struct {
	typeName         string
	typeCheck        func(interface{}) bool
	setID            SetIDFunc
	validators       []func(interface{}, ifs.IVNic) error
	actionValidators []ActionValidateFunc
	afterActions     []ActionValidateFunc
}

// NewValidation creates a validation builder for a ServiceCallback.
// typeInstance is a zero-value pointer to the entity type (e.g., &fin.Budget{}).
// vnic provides access to the introspector for deriving typeName, typeCheck, and setID.
func NewValidation(typeInstance interface{}, vnic ifs.IVNic) *VB {
	instanceType := reflect.TypeOf(typeInstance)
	typeName := instanceType.Elem().Name()

	typeCheck := func(v interface{}) bool {
		return reflect.TypeOf(v) == instanceType
	}

	// Derive setID from the introspector's primary key decorator
	var setID SetIDFunc
	introspector := vnic.Resources().Introspector()
	if introspector != nil {
		node, ok := introspector.NodeByValue(typeInstance)
		if ok {
			fields, err := introspector.Decorators().Fields(node, l8reflect.L8DecoratorType_Primary)
			if err == nil && len(fields) > 0 {
				pkFieldName := fields[0]
				setID = func(v interface{}) {
					rv := reflect.ValueOf(v).Elem()
					f := rv.FieldByName(pkFieldName)
					if f.IsValid() && f.Kind() == reflect.String && f.String() == "" {
						f.SetString(ifs.NewUuid())
					}
				}
			}
		}
	}
	if setID == nil {
		setID = func(v interface{}) {} // no-op if introspector unavailable
	}

	return &VB{typeName: typeName, typeCheck: typeCheck, setID: setID}
}

// Require adds a required string field validation.
// getter can be func(interface{}) string OR func(*ConcreteType) string.
func (b *VB) Require(getter interface{}, name string) *VB {
	if typed, ok := getter.(func(interface{}) string); ok {
		b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
			return ValidateRequired(typed(e), name)
		})
		return b
	}
	fnVal := reflect.ValueOf(getter)
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
		return ValidateRequired(results[0].String(), name)
	})
	return b
}

// RequireInt64 adds a required int64 field validation.
// getter can be func(interface{}) int64 OR func(*ConcreteType) int64.
func (b *VB) RequireInt64(getter interface{}, name string) *VB {
	if typed, ok := getter.(func(interface{}) int64); ok {
		b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
			return ValidateRequiredInt64(typed(e), name)
		})
		return b
	}
	fnVal := reflect.ValueOf(getter)
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
		return ValidateRequiredInt64(results[0].Int(), name)
	})
	return b
}

// Enum adds an enum field validation using the protobuf _name map.
// Value 0 (UNSPECIFIED) is rejected; unknown values are rejected.
// getter can be func(interface{}) int32 OR func(*ConcreteType) int32 (or any int32-compatible return).
func (b *VB) Enum(getter interface{}, nameMap map[int32]string, name string) *VB {
	if typed, ok := getter.(func(interface{}) int32); ok {
		b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
			return ValidateEnum(typed(e), nameMap, name)
		})
		return b
	}
	fnVal := reflect.ValueOf(getter)
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
		return ValidateEnum(int32(results[0].Int()), nameMap, name)
	})
	return b
}

// Money adds a required money field validation (nil + CurrencyId check).
func (b *VB) Money(getter func(interface{}) *l8common.Money, name string) *VB {
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		return ValidateMoney(getter(e), name)
	})
	return b
}

// MoneyPositive adds a required money field validation with positive amount.
func (b *VB) MoneyPositive(getter func(interface{}) *l8common.Money, name string) *VB {
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		return ValidateMoneyPositive(getter(e), name)
	})
	return b
}

// OptionalMoney validates a money field only when non-nil (skips nil).
func (b *VB) OptionalMoney(getter func(interface{}) *l8common.Money, name string) *VB {
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		m := getter(e)
		if m == nil {
			return nil
		}
		return ValidateMoney(m, name)
	})
	return b
}

// DateNotZero adds a required date (non-zero timestamp) validation.
// getter can be func(interface{}) int64 OR func(*ConcreteType) int64.
func (b *VB) DateNotZero(getter interface{}, name string) *VB {
	if typed, ok := getter.(func(interface{}) int64); ok {
		b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
			return ValidateDateNotZero(typed(e), name)
		})
		return b
	}
	fnVal := reflect.ValueOf(getter)
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
		return ValidateDateNotZero(results[0].Int(), name)
	})
	return b
}

// DateAfter validates that date1 > date2 (skips if either is zero).
// getter1 and getter2 can each be func(interface{}) int64 OR func(*ConcreteType) int64.
func (b *VB) DateAfter(getter1, getter2 interface{}, name1, name2 string) *VB {
	g1 := wrapInt64Getter(getter1)
	g2 := wrapInt64Getter(getter2)
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		d1, d2 := g1(e), g2(e)
		if d1 == 0 || d2 == 0 {
			return nil
		}
		return ValidateDateAfter(d1, d2, name1, name2)
	})
	return b
}

// wrapInt64Getter wraps a typed or untyped int64 getter into func(interface{}) int64.
func wrapInt64Getter(getter interface{}) func(interface{}) int64 {
	if typed, ok := getter.(func(interface{}) int64); ok {
		return typed
	}
	fnVal := reflect.ValueOf(getter)
	return func(e interface{}) int64 {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
		return results[0].Int()
	}
}

// DateRange validates a required *l8common.DateRange field (nil + StartDate < EndDate).
func (b *VB) DateRange(getter func(interface{}) *l8common.DateRange, name string) *VB {
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		return ValidateDateRange(getter(e), name)
	})
	return b
}

// Compute adds a function that derives/computes entity fields before validation.
// fn can be func(interface{}) error OR func(*ConcreteType) error — reflection wraps typed funcs.
func (b *VB) Compute(fn interface{}) *VB {
	if typed, ok := fn.(func(interface{}) error); ok {
		return b.Custom(func(e interface{}, _ ifs.IVNic) error { return typed(e) })
	}
	fnVal := reflect.ValueOf(fn)
	return b.Custom(func(e interface{}, _ ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
		if !results[0].IsNil() {
			return results[0].Interface().(error)
		}
		return nil
	})
}

// Custom adds a custom validation function.
// fn can be:
//   - func(interface{}, ifs.IVNic) error
//   - func(*ConcreteType, ifs.IVNic) error (2 params, reflected)
//   - func(interface{}) error (1 param, vnic ignored)
//   - func(*ConcreteType) error (1 param, reflected, vnic ignored)
func (b *VB) Custom(fn interface{}) *VB {
	if typed, ok := fn.(func(interface{}, ifs.IVNic) error); ok {
		b.validators = append(b.validators, typed)
		return b
	}
	if typed, ok := fn.(func(interface{}) error); ok {
		b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
			return typed(e)
		})
		return b
	}
	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()
	if fnType.NumIn() == 1 {
		b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
			results := fnVal.Call([]reflect.Value{reflect.ValueOf(e)})
			if !results[0].IsNil() {
				return results[0].Interface().(error)
			}
			return nil
		})
		return b
	}
	b.validators = append(b.validators, func(e interface{}, vnic ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e), reflect.ValueOf(vnic)})
		if !results[0].IsNil() {
			return results[0].Interface().(error)
		}
		return nil
	})
	return b
}

// BeforeAction adds an action-aware validator that runs before persistence.
// Unlike Custom, it receives the CRUD action so it can branch on POST/PUT/etc.
// fn can be ActionValidateFunc OR func(*ConcreteType, ifs.Action, ifs.IVNic) error.
func (b *VB) BeforeAction(fn interface{}) *VB {
	if typed, ok := fn.(ActionValidateFunc); ok {
		b.actionValidators = append(b.actionValidators, typed)
		return b
	}
	if typed, ok := fn.(func(interface{}, ifs.Action, ifs.IVNic) error); ok {
		b.actionValidators = append(b.actionValidators, typed)
		return b
	}
	fnVal := reflect.ValueOf(fn)
	b.actionValidators = append(b.actionValidators, func(e interface{}, action ifs.Action, vnic ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e), reflect.ValueOf(action), reflect.ValueOf(vnic)})
		if !results[0].IsNil() {
			return results[0].Interface().(error)
		}
		return nil
	})
	return b
}

// StatusTransition adds a status state-machine validator.
func (b *VB) StatusTransition(cfg *StatusTransitionConfig) *VB {
	b.actionValidators = append(b.actionValidators, cfg.BuildValidator())
	return b
}

// After adds a function to run after successful persistence (PUT/PATCH only).
// fn can be ActionValidateFunc OR func(*ConcreteType, ifs.Action, ifs.IVNic) error.
func (b *VB) After(fn interface{}) *VB {
	if typed, ok := fn.(ActionValidateFunc); ok {
		b.afterActions = append(b.afterActions, typed)
		return b
	}
	if typed, ok := fn.(func(interface{}, ifs.Action, ifs.IVNic) error); ok {
		b.afterActions = append(b.afterActions, typed)
		return b
	}
	fnVal := reflect.ValueOf(fn)
	b.afterActions = append(b.afterActions, func(e interface{}, action ifs.Action, vnic ifs.IVNic) error {
		results := fnVal.Call([]reflect.Value{reflect.ValueOf(e), reflect.ValueOf(action), reflect.ValueOf(vnic)})
		if !results[0].IsNil() {
			return results[0].Interface().(error)
		}
		return nil
	})
	return b
}

// ValidatePeriod validates an L8Period field.
func ValidatePeriod(p *l8api.L8Period, name string) error {
	if p == nil {
		return fmt.Errorf("%s is required", name)
	}
	if p.PeriodType == l8api.L8PeriodType_invalid_period_type {
		return fmt.Errorf("%s type is required", name)
	}
	if p.PeriodYear < 1970 || p.PeriodYear > 2100 {
		return fmt.Errorf("%s year must be between 1970 and 2100", name)
	}
	switch p.PeriodType {
	case l8api.L8PeriodType_Quarterly:
		if p.PeriodValue < l8api.L8PeriodValue_Q1 || p.PeriodValue > l8api.L8PeriodValue_Q4 {
			return fmt.Errorf("%s quarterly value must be Q1-Q4", name)
		}
	case l8api.L8PeriodType_Monthly:
		if p.PeriodValue < l8api.L8PeriodValue_January || p.PeriodValue > l8api.L8PeriodValue_December {
			return fmt.Errorf("%s monthly value must be January-December", name)
		}
	}
	return nil
}

// Period adds a required L8Period field validation.
func (b *VB) Period(getter func(interface{}) *l8api.L8Period, name string) *VB {
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		return ValidatePeriod(getter(e), name)
	})
	return b
}

// OptionalPeriod validates an L8Period field only when non-nil.
func (b *VB) OptionalPeriod(getter func(interface{}) *l8api.L8Period, name string) *VB {
	b.validators = append(b.validators, func(e interface{}, _ ifs.IVNic) error {
		p := getter(e)
		if p == nil {
			return nil
		}
		return ValidatePeriod(p, name)
	})
	return b
}

// Build creates the IServiceCallback from the chained validators.
func (b *VB) Build() ifs.IServiceCallback {
	validate := func(item interface{}, vnic ifs.IVNic) error {
		for _, v := range b.validators {
			if err := v(item, vnic); err != nil {
				return err
			}
		}
		return nil
	}
	if len(b.afterActions) > 0 {
		return NewServiceCallbackWithAfter(b.typeName, b.typeCheck, b.setID, validate,
			b.actionValidators, b.afterActions)
	}
	return NewServiceCallback(b.typeName, b.typeCheck, b.setID, validate, b.actionValidators...)
}
