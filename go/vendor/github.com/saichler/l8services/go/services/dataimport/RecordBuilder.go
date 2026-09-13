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

package dataimport

import (
	"fmt"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8reflect"
	"reflect"
	"strconv"
)

// BuildRecord creates a protobuf struct instance from mapped key-value pairs.
func BuildRecord(mappedValues map[string]string, node *l8reflect.L8Node,
	registry ifs.IRegistry, transforms []*l8api.L8ImportValueTransform,
	defaults map[string]string) (interface{}, []error) {

	info, err := registry.Info(node.TypeName)
	if err != nil {
		return nil, []error{fmt.Errorf("cannot find type %s: %w", node.TypeName, err)}
	}
	instance, err := info.NewInstance()
	if err != nil {
		return nil, []error{fmt.Errorf("cannot create instance of type %s: %w", node.TypeName, err)}
	}

	v := reflect.ValueOf(instance)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var errs []error

	// Apply defaults first, then overwrite with mapped values
	merged := make(map[string]string, len(defaults)+len(mappedValues))
	for k, val := range defaults {
		merged[k] = val
	}
	for k, val := range mappedValues {
		merged[k] = val
	}

	for fieldName, raw := range merged {
		attr := node.Attributes[fieldName]
		if attr == nil {
			continue
		}

		// Apply transforms
		transformed, err := TransformValue(raw, transforms, fieldName)
		if err != nil {
			errs = append(errs, fmt.Errorf("field %s: %w", fieldName, err))
			continue
		}

		goFieldName := attr.FieldName
		if goFieldName == "" {
			goFieldName = fieldName
		}

		field := v.FieldByName(goFieldName)
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		if err := setFieldValue(field, transformed); err != nil {
			errs = append(errs, fmt.Errorf("field %s: %w", fieldName, err))
		}
	}

	return instance, errs
}

func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int32:
		n, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot parse int32 '%s': %w", value, err)
		}
		field.SetInt(n)
	case reflect.Int64:
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse int64 '%s': %w", value, err)
		}
		field.SetInt(n)
	case reflect.Float32:
		f, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return fmt.Errorf("cannot parse float32 '%s': %w", value, err)
		}
		field.SetFloat(f)
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("cannot parse float64 '%s': %w", value, err)
		}
		field.SetFloat(f)
	case reflect.Bool:
		b := value == "true" || value == "1" || value == "yes"
		field.SetBool(b)
	case reflect.Uint32:
		n, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			return fmt.Errorf("cannot parse uint32 '%s': %w", value, err)
		}
		field.SetUint(n)
	case reflect.Uint64:
		n, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse uint64 '%s': %w", value, err)
		}
		field.SetUint(n)
	default:
		// Skip complex types (structs, slices, maps)
	}
	return nil
}

// MapRow applies column mappings to a parsed row, returning target_field→value.
func MapRow(row ParsedRow, mappings []*l8api.L8ImportColumnMapping) map[string]string {
	result := make(map[string]string, len(mappings))
	for _, m := range mappings {
		if m.Skip || m.TargetField == "" {
			continue
		}
		if val, ok := row.Values[m.SourceColumn]; ok {
			result[m.TargetField] = val
		}
	}
	return result
}
