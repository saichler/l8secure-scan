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
	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/types/l8api"
	"reflect"
	"strconv"
	"strings"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

// ConvertFrom transforms relational data (L8OrmRData) back into Go objects.
// It reconstructs the original object hierarchy from the flat table structure,
// handling nested structs, slices, and maps. The metadata parameter carries
// pagination and count information to be included in the result.
func ConvertFrom(o ifs.IElements, metadata *l8api.L8MetaData, res ifs.IResources) ifs.IElements {
	data := o.Element().(*l8orms.L8OrmRData)
	node, ok := res.Introspector().Node(data.RootTypeName)
	if !ok {
		return object.NewError("No node for type " + data.RootTypeName)
	}
	resp, e := convertFrom(node, "", data, res)
	if e != nil {
		return object.NewError(e.Error())
	}
	return object.NewQueryResult(resp, metadata)
}

// convertFrom recursively converts a table's rows into Go struct instances.
// It processes the node's attributes, populating simple fields from column values
// and recursively converting nested struct fields from their respective tables.
func convertFrom(node *l8reflect.L8Node, parentKey string, data *l8orms.L8OrmRData, res ifs.IResources) (interface{}, error) {
	table, attributeRows := TableAndRowsGet(node, data, parentKey)

	//No data for this attribute
	if table == nil {
		return nil, nil
	}

	info, e := res.Registry().Info(node.TypeName)
	if e != nil {
		return nil, e
	}

	rows := make(map[string]interface{}, 0)
	// Track insertion order for root queries to preserve sort order
	orderedInstances := make([]interface{}, 0, len(attributeRows.Rows))
	subTableAttributes := make(map[string]*l8reflect.L8Node)
	subAttributesFull := false
	for _, row := range attributeRows.Rows {
		instance, err := info.NewInstance()
		if err != nil {
			return nil, err
		}
		value := reflect.ValueOf(instance).Elem()
		for attrName, attrNode := range node.Attributes {
			if attrNode.IsStruct {
				if common.IsTimeSeriesType(attrNode.TypeName) {
					continue
				}
				if !subAttributesFull {
					subTableAttributes[attrName] = attrNode
				}
				continue
			}
			colIndex, ok := table.Columns[attrName]
			if !ok {
				continue
			}
			e = SetValueFromRow(colIndex, attrName, value, row, res.Registry())
			if e != nil {
				return nil, e
			}
		}

		for attrName, attrNode := range subTableAttributes {
			v, e := convertFrom(attrNode, KeyForRow(row), data, res)
			if e != nil {
				return nil, e
			}
			if v == nil {
				continue
			}
			SetValueFromSubTable(attrName, value, v)
		}

		subAttributesFull = true
		rows[row.RecKey] = instance
		orderedInstances = append(orderedInstances, instance)
	}

	if node.IsSlice {
		return convertRowsToSlice(rows, node, res.Registry())
	}
	if node.IsMap {
		v, e := convertRowsToMap(rows, node, res.Registry())
		if e != nil {
			panic(e)
		}
		if v == nil {
			panic(node.FieldName)
		}
		return v, e
	}
	// For root queries (parentKey is empty), return ordered slice to preserve sort order.
	// For child struct queries, return map (used by SetValueFromSubTable).
	if parentKey == "" {
		return orderedInstances, nil
	}
	return rows, nil
}

// convertRowsToSlice transforms a map of rows into a typed slice.
// The row keys contain slice indices in the format "FieldName[index]",
// which are used to place elements in the correct slice positions.
func convertRowsToSlice(rows map[string]interface{}, node *l8reflect.L8Node, r ifs.IRegistry) (interface{}, error) {
	info, e := r.Info(node.TypeName)
	if e != nil {
		return nil, e
	}

	slice := reflect.MakeSlice(reflect.SliceOf(reflect.New(info.Type()).Type()), len(rows), len(rows))
	for key, value := range rows {
		index, e := GetSliceIndex(key)
		if e != nil {
			return nil, e
		}
		slice.Index(index).Set(reflect.ValueOf(value))
	}
	return slice.Interface(), nil
}

// convertRowsToMap transforms a map of rows into a typed Go map.
// The row keys contain map keys in the format "FieldName[mapKey]",
// which are parsed and converted to the appropriate key type.
func convertRowsToMap(rows map[string]interface{}, node *l8reflect.L8Node, r ifs.IRegistry) (interface{}, error) {
	valInfo, e := r.Info(node.TypeName)
	if e != nil {
		return nil, e
	}
	keyInfo, e := r.Info(node.KeyTypeName)
	if e != nil {
		return nil, e
	}

	newMap := reflect.MakeMap(reflect.MapOf(keyInfo.Type(), reflect.New(valInfo.Type()).Type()))

	for key, value := range rows {
		index, e := GetMapIndex(key, r)
		if e != nil {
			return nil, e
		}
		newMap.SetMapIndex(index, reflect.ValueOf(value))
	}
	return newMap.Interface(), nil
}

// GetMapIndex extracts and converts a map key from the RecKey string format.
// The key is extracted from the bracketed portion of the string (e.g., "Field[key]" -> "key")
// and converted to a reflect.Value of the appropriate type.
func GetMapIndex(mapKey string, r ifs.IRegistry) (reflect.Value, error) {
	index1 := strings.LastIndex(mapKey, "[")
	index2 := strings.LastIndex(mapKey, "]")
	return strings2.FromString(mapKey[index1+1:index2], r)
}

// GetSliceIndex extracts a slice index from the RecKey string format.
// The index is extracted from the bracketed portion of the string (e.g., "Field[5]" -> 5)
// and converted to an integer.
func GetSliceIndex(sliceKey string) (int, error) {
	index1 := strings.LastIndex(sliceKey, "[")
	index2 := strings.LastIndex(sliceKey, "]")
	index, e := strconv.Atoi(sliceKey[index1+1 : index2])
	return index, e
}

// SetValueFromRow deserializes a column value from a row and sets it on the struct field.
// It handles type conversions between the serialized byte format and the target Go type.
func SetValueFromRow(colIndex int32, attrName string, value reflect.Value, row *l8orms.L8OrmRow, r ifs.IRegistry) error {
	fieldValue := value.FieldByName(attrName)
	data := row.ColumnValues[colIndex]
	if data == nil || len(data) == 0 {
		return nil
	}
	obj := object.NewDecode(data, 0, r)
	v, e := obj.Get()
	if e != nil {
		return e
	}
	if fieldValue.Kind() == reflect.Int32 {
		fieldValue.SetInt(reflect.ValueOf(v).Int())
	} else {
		setFieldValue := reflect.ValueOf(v)
		if fieldValue.Kind() == setFieldValue.Kind() {
			if setFieldValue.Kind() == reflect.Slice &&
				setFieldValue.Type() != fieldValue.Type() &&
				fieldValue.Type().Elem().Kind() == reflect.Int32 {
				converted := reflect.MakeSlice(fieldValue.Type(), setFieldValue.Len(), setFieldValue.Len())
				for i := 0; i < setFieldValue.Len(); i++ {
					converted.Index(i).SetInt(setFieldValue.Index(i).Int())
				}
				fieldValue.Set(converted)
			} else {
				fieldValue.Set(setFieldValue)
			}
		} else {
			if setFieldValue.Kind() != reflect.String || setFieldValue.String() != "" {
				panic("Not set" + setFieldValue.String())
			}
		}
	}
	return nil
}

// SetValueFromSubTable sets a struct field value from converted sub-table data.
// For single struct fields, it extracts the value from the map wrapper.
// For slices and maps, it sets the value directly.
func SetValueFromSubTable(attrName string, value reflect.Value, i interface{}) {
	fieldValue := value.FieldByName(attrName)
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Map && fieldValue.Kind() != reflect.Map {
		mapKeys := v.MapKeys()
		fieldValue.Set(reflect.ValueOf(v.MapIndex(mapKeys[0]).Interface()))
		return
	}
	fieldValue.Set(reflect.ValueOf(i))
}

// TableAndRowsGet retrieves the table and attribute rows for a given node and parent key.
// Returns nil if the table doesn't exist or has no data for the specified parent.
// This is used during conversion to locate the relevant data for each struct type.
func TableAndRowsGet(node *l8reflect.L8Node, data *l8orms.L8OrmRData, parentKey string) (*l8orms.L8OrmTable, *l8orms.L8OrmAttributeRows) {
	table := data.Tables[node.TypeName]
	if table == nil {
		return nil, nil
	}
	if table.InstanceRows == nil {
		return nil, nil
	}
	instanceRows, ok := table.InstanceRows[parentKey]
	if !ok {
		return nil, nil
	}
	if instanceRows.AttributeRows == nil {
		return nil, nil
	}
	attributeRows, ok := instanceRows.AttributeRows[node.FieldName]
	if !ok {
		return nil, nil
	}
	if attributeRows.Rows == nil {
		return nil, nil
	}
	return table, attributeRows
}
