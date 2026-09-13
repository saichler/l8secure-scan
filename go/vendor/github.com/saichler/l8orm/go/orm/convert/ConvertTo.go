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
package convert

import (
	"bytes"
	"errors"
	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types/l8orms"
	"reflect"
	"strconv"
	strings2 "strings"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

// ConvertTo transforms Go objects into relational data format (L8OrmRData).
// It flattens the object hierarchy into tables, with each struct type becoming a table
// and nested structs stored in separate tables linked by parent keys.
// The action parameter (POST/PATCH) affects how zero values are handled.
func ConvertTo(action ifs.Action, objects ifs.IElements, res ifs.IResources) ifs.IElements {
	if objects == nil {
		return nil
	}

	data := &l8orms.L8OrmRData{}
	data.Tables = make(map[string]*l8orms.L8OrmTable)
	v := reflect.ValueOf(objects.Element())
	typeName, err := TypeOf(v)
	if err != nil {
		return object.NewError(err.Error())
	}
	data.RootTypeName = typeName

	node, ok := res.Introspector().Node(data.RootTypeName)
	if !ok {
		n, err := res.Introspector().Inspect(objects.Element())
		if err != nil {
			return object.NewError(err.Error())
		}
		node = n
	}

	elements := objects.Elements()
	keys := objects.Keys()

	if len(elements) == 1 {
		err := convertTo(action, v, "", "", node, data, res)
		if err != nil {
			return object.NewError(err.Error())
		}
		return object.New(nil, data)
	}

	for i, element := range elements {
		key := ""
		if keys[i] != nil {
			str := strings.New()
			key = str.ToString(reflect.ValueOf(keys[i]))
		}
		err := convertTo(action, reflect.ValueOf(element), "", key, node, data, res)
		if err != nil {
			return object.NewError(err.Error())
		}
	}

	return object.New(nil, data)
}

// TypeOf extracts the element type name from a reflect.Value.
// For slices and maps, it returns the element type name.
// For pointers, it returns the pointed-to type name.
// For structs, it returns the struct type name directly.
func TypeOf(v reflect.Value) (string, error) {
	if !v.IsValid() {
		return "", errors.New("TypeOf: reflect.Value is invalid (nil element)")
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
		return v.Type().Elem().Elem().Name(), nil
	} else if v.Kind() == reflect.Ptr {
		return v.Elem().Type().Name(), nil
	} else if v.Kind() == reflect.Struct {
		return v.Type().Name(), nil
	}
	return "", errors.New("TypeOf: unsupported kind " + v.Kind().String() + " for type " + v.Type().Name())
}

// convertTo recursively converts a single Go value into relational table rows.
// It handles struct fields by storing simple values in columns and recursively
// processing nested structs, slices, and maps into their respective tables.
// For PATCH actions, zero values are skipped to enable partial updates.
func convertTo(action ifs.Action, value reflect.Value, parentKey, myKey string, node *l8reflect.L8Node, data *l8orms.L8OrmRData, res ifs.IResources) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if !value.IsValid() {
		return nil
	}

	table, attributeRows := TableAndRowsCreate(node, data, parentKey)
	SetColumns(table, node)

	row := &l8orms.L8OrmRow{}
	row.ParentKey = parentKey
	row.RecKey = RecKey(node, value, myKey, res)
	row.ColumnValues = make(map[int32][]byte)

	subTableAttributes := make(map[string]*l8reflect.L8Node)
	for attrName, attrNode := range node.Attributes {
		if attrNode.IsStruct {
			if common.IsTimeSeriesType(attrNode.TypeName) {
				extractTsData(value, node, attrName, data, res)
				continue
			}
			subTableAttributes[attrName] = attrNode
			continue
		}
		fieldValue := value.FieldByName(attrName)
		if fieldValue.IsValid() {
			// For PATCH, skip zero/default values
			if action == ifs.PATCH && fieldValue.IsZero() {
				continue
			}
			col := table.Columns[attrName]
			err := SetValueToRow(row, col, fieldValue)
			if err != nil {
				return err
			}
		}
	}

	for attrName, attrNode := range subTableAttributes {
		fieldValue := value.FieldByName(attrName)
		if fieldValue.IsValid() {
			if attrNode.IsMap {
				mapKeys := fieldValue.MapKeys()
				for _, mapKey := range mapKeys {
					mapValue := fieldValue.MapIndex(mapKey)
					mapValueStr := strings.New()
					mapValueStr.TypesPrefix = true
					err := convertTo(action, mapValue, KeyForRow(row), mapValueStr.ToString(mapKey), attrNode, data, res)
					if err != nil {
						return err
					}
				}
			} else if attrNode.IsSlice {
				for i := 0; i < fieldValue.Len(); i++ {
					sliceValue := fieldValue.Index(i)
					err := convertTo(action, sliceValue, KeyForRow(row), strconv.Itoa(i), attrNode, data, res)
					if err != nil {
						return err
					}
				}
			} else {
				err := convertTo(action, fieldValue, KeyForRow(row), "", attrNode, data, res)
				if err != nil {
					return err
				}
			}
		}
	}

	attributeRows.Rows = append(attributeRows.Rows, row)
	return nil
}

// TableAndRowsCreate creates or retrieves the table and attribute rows for a given node.
// It initializes all necessary nested structures in the relational data hierarchy.
// Unlike TableAndRowsGet, this function creates missing structures rather than returning nil.
func TableAndRowsCreate(node *l8reflect.L8Node, data *l8orms.L8OrmRData, parentKey string) (*l8orms.L8OrmTable, *l8orms.L8OrmAttributeRows) {
	table, ok := data.Tables[node.TypeName]
	if !ok {
		table = &l8orms.L8OrmTable{}
		table.Name = node.TypeName
		data.Tables[node.TypeName] = table
	}
	if table.InstanceRows == nil {
		table.InstanceRows = make(map[string]*l8orms.L8OrmInstanceRows)
	}
	instanceRows, ok := table.InstanceRows[parentKey]
	if !ok {
		instanceRows = &l8orms.L8OrmInstanceRows{}
		table.InstanceRows[parentKey] = instanceRows
	}
	if instanceRows.AttributeRows == nil {
		instanceRows.AttributeRows = make(map[string]*l8orms.L8OrmAttributeRows)
	}
	attributeRows, ok := instanceRows.AttributeRows[node.FieldName]
	if !ok {
		attributeRows = &l8orms.L8OrmAttributeRows{}
		instanceRows.AttributeRows[node.FieldName] = attributeRows
	}
	if attributeRows.Rows == nil {
		attributeRows.Rows = make([]*l8orms.L8OrmRow, 0)
	}
	return table, attributeRows
}

// SetValueToRow serializes a field value and stores it in the row's column values.
// The value is serialized using the L8 serialization framework for efficient storage.
func SetValueToRow(row *l8orms.L8OrmRow, col int32, val reflect.Value) error {
	obj := object.NewEncode()
	err := obj.Add(val.Interface())
	if err != nil {
		return err
	}
	row.ColumnValues[col] = obj.Data()
	return nil
}

// RecKey generates the record key for a row in the format "FieldName[key]".
// If the struct has a primary key decorator, that key is used.
// Otherwise, the myKey parameter (slice index or map key) is used.
func RecKey(node *l8reflect.L8Node, value reflect.Value, myKey string, res ifs.IResources) string {
	key, _, _ := res.Introspector().Decorators().PrimaryKeyDecoratorFromValue(node, value)
	if key == "" {
		str := strings.New(node.FieldName)
		str.Add("[")
		str.Add(myKey)
		str.Add("]")
		return str.String()
	} else {
		str := strings.New(node.FieldName)
		str.Add("[")
		str.Add(str.ToString(reflect.ValueOf(key)))
		str.Add("]")
		return str.String()
	}
}

// KeyForRow concatenates the ParentKey and RecKey to form the full hierarchical key.
// This composite key is used as the parent key for child rows in nested structures.
func KeyForRow(row *l8orms.L8OrmRow) string {
	buff := bytes.Buffer{}
	buff.WriteString(row.ParentKey)
	buff.WriteString(row.RecKey)
	return buff.String()
}

func extractTsData(value reflect.Value, node *l8reflect.L8Node, attrName string, data *l8orms.L8OrmRData, res ifs.IResources) {
	fieldValue := value.FieldByName(attrName)
	if !fieldValue.IsValid() || fieldValue.IsNil() || fieldValue.Len() == 0 {
		return
	}

	key, _, _ := res.Introspector().Decorators().PrimaryKeyDecoratorFromValue(node, value)
	if key == "" {
		return
	}

	propertyId := strings2.ToLower(node.TypeName) + "<" + key + ">." + strings2.ToLower(attrName)

	for i := 0; i < fieldValue.Len(); i++ {
		pointVal := fieldValue.Index(i)
		if pointVal.Kind() == reflect.Ptr {
			if pointVal.IsNil() {
				continue
			}
			pointVal = pointVal.Elem()
		}
		point, ok := pointVal.Addr().Interface().(*l8api.L8TimeSeriesPoint)
		if !ok {
			continue
		}
		data.TsData = append(data.TsData, &l8notify.L8TSDBNotification{
			PropertyId: propertyId,
			Point:      point,
		})
	}
}
