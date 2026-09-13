// (c) 2025 Sharon Aicler (saichler@gmail.com)
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

package common

import (
	"errors"
	l8common "github.com/saichler/l8common/go/types/l8common"
	"github.com/saichler/l8types/go/ifs"
	"reflect"
	"time"
)

// SafeCast safely casts an interface{} and returns an error if nil.
func SafeCast(element interface{}) (interface{}, error) {
	if element == nil {
		return nil, errors.New("entity not found")
	}
	return element, nil
}

// ValidateRequired checks if a string field is non-empty
func ValidateRequired(value, fieldName string) error {
	if value == "" {
		return errors.New(fieldName + " is required")
	}
	return nil
}

// ValidateRequiredInt64 checks if an int64 field is non-zero
func ValidateRequiredInt64(value int64, fieldName string) error {
	if value == 0 {
		return errors.New(fieldName + " is required")
	}
	return nil
}

// ValidateEnum validates an enum value against its name map.
// Uses the protobuf-generated _name maps (e.g., Gender_name).
// Value 0 (UNSPECIFIED) is considered invalid.
func ValidateEnum(value interface{}, nameMap map[int32]string, enumName string) error {
	v := int32(reflect.ValueOf(value).Int())
	if v == 0 {
		return errors.New(enumName + " must be specified")
	}
	if _, ok := nameMap[v]; !ok {
		return errors.New("invalid " + enumName + " value")
	}
	return nil
}

// ValidateDateInPast checks if a Unix timestamp is in the past
func ValidateDateInPast(timestamp int64, fieldName string) error {
	if timestamp > time.Now().Unix() {
		return errors.New(fieldName + " must be in the past")
	}
	return nil
}

// ValidateDateNotZero checks if date is set (non-zero)
func ValidateDateNotZero(timestamp int64, fieldName string) error {
	if timestamp == 0 {
		return errors.New(fieldName + " is required")
	}
	return nil
}

// ValidateDateAfter checks if date1 is after date2
func ValidateDateAfter(date1, date2 int64, field1Name, field2Name string) error {
	if date1 <= date2 {
		return errors.New(field1Name + " must be after " + field2Name)
	}
	return nil
}

// ValidateMinimumAge checks if DateOfBirth results in minimum age
func ValidateMinimumAge(dateOfBirth int64, minAge int, fieldName string) error {
	birthTime := time.Unix(dateOfBirth, 0)
	now := time.Now()
	age := now.Year() - birthTime.Year()
	// Adjust if birthday hasn't occurred this year
	if now.YearDay() < birthTime.YearDay() {
		age--
	}
	if age < minAge {
		return errors.New(fieldName + " indicates employee is under minimum working age")
	}
	return nil
}

// ValidateConditionalRequired validates field2 is required when condition is true
func ValidateConditionalRequired(condition bool, value, conditionDesc, fieldName string) error {
	if condition && value == "" {
		return errors.New(fieldName + " is required when " + conditionDesc)
	}
	return nil
}

// ValidateConditionalRequiredInt64 validates an int64 field is required when condition is true
func ValidateConditionalRequiredInt64(condition bool, value int64, conditionDesc, fieldName string) error {
	if condition && value == 0 {
		return errors.New(fieldName + " is required when " + conditionDesc)
	}
	return nil
}

// ValidateMoney validates a required money field (nil check + CurrencyId required).
// Amount is NOT checked — it can be zero or negative for balances/adjustments.
func ValidateMoney(money *l8common.Money, fieldName string) error {
	if money == nil {
		return errors.New(fieldName + " is required")
	}
	if money.CurrencyId == "" {
		return errors.New(fieldName + ".CurrencyId is required")
	}
	return nil
}

// ValidateMoneyPositive validates a required money field with positive amount.
// Use for prices, costs, and totals that must be greater than zero.
func ValidateMoneyPositive(money *l8common.Money, fieldName string) error {
	if err := ValidateMoney(money, fieldName); err != nil {
		return err
	}
	if money.Amount <= 0 {
		return errors.New(fieldName + ".Amount must be positive")
	}
	return nil
}

// ValidateDateRange validates a required DateRange (nil check + StartDate < EndDate).
func ValidateDateRange(dateRange *l8common.DateRange, fieldName string) error {
	if dateRange == nil {
		return errors.New(fieldName + " is required")
	}
	if dateRange.StartDate == 0 {
		return errors.New(fieldName + ".StartDate is required")
	}
	if dateRange.EndDate == 0 {
		return errors.New(fieldName + ".EndDate is required")
	}
	if dateRange.StartDate >= dateRange.EndDate {
		return errors.New(fieldName + ".StartDate must be before EndDate")
	}
	return nil
}

// GenerateID assigns a new UUID to the given ID pointer if it is empty.
// Used to auto-generate primary keys on POST (Add) operations.
func GenerateID(id *string) {
	if *id == "" {
		*id = ifs.NewUuid()
	}
}
