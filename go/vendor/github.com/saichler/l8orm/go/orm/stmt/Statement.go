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

// Package stmt provides SQL statement builders for PostgreSQL database operations.
// It generates SELECT, INSERT, UPDATE, and DELETE statements from query objects
// and L8 reflection metadata, handling column mapping and value serialization.
package stmt

import (
	"database/sql"
	"github.com/saichler/l8orm/go/types/l8orms"
	"reflect"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

// Statement manages SQL statement generation and execution for a single table.
// It caches prepared statements and handles column-to-field mapping.
type Statement struct {
	fields  []string            // Ordered list of field names (ParentKey, RecKey, then attributes)
	values  map[string]int      // Field name to parameter position mapping
	columns map[string]int32    // Column name to index mapping from table schema
	registy ifs.IRegistry       // Type registry for deserialization
	node    *l8reflect.L8Node   // Type metadata for the table
	query   ifs.IQuery          // Query for filtering and projection

	insertStmt   *sql.Stmt      // Cached prepared INSERT statement
	selectStmt   *sql.Stmt      // Cached prepared SELECT statement
	updateStmt   *sql.Stmt      // Cached prepared UPDATE statement
	metaDataStmt *sql.Stmt      // Cached prepared COUNT statement
}

// NewStatement creates a new Statement for the given type node and column schema.
func NewStatement(node *l8reflect.L8Node, columns map[string]int32, query ifs.IQuery, registy ifs.IRegistry) *Statement {
	return &Statement{node: node, columns: columns, registy: registy, query: query}
}

// RowValues extracts the parameter values from a row for SQL statement execution.
// For PATCH actions, zero values are converted to nil to trigger COALESCE behavior.
func (this *Statement) RowValues(action ifs.Action, row *l8orms.L8OrmRow) ([]interface{}, error) {
	result := make([]interface{}, len(this.values))
	result[0] = row.ParentKey
	result[1] = row.RecKey
	for attrName, attr := range this.node.Attributes {
		if attr.IsStruct {
			continue
		}
		fieldPos := this.values[attrName]
		rowPos := this.columns[attrName]
		data, exists := row.ColumnValues[rowPos]
		if !exists || len(data) == 0 {
			result[fieldPos-1] = nil
			continue
		}
		val, err := getValueForPostgres(data, this.registy)
		if err != nil {
			return nil, err
		}
		if action == ifs.PATCH && isZeroValue(val) {
			result[fieldPos-1] = nil
		} else {
			result[fieldPos-1] = val
		}
	}
	return result, nil
}

// isZeroValue checks if a value is the zero value for its type.
// Used by PATCH operations to determine which fields to skip.
func isZeroValue(val interface{}) bool {
	if val == nil {
		return true
	}
	v := reflect.ValueOf(val)
	return v.IsZero()
}

// fieldsOf extracts the ordered field list and position map from a node.
// ParentKey and RecKey are always first (positions 1 and 2), followed by attributes.
func fieldsOf(node *l8reflect.L8Node) ([]string, map[string]int) {
	fields := []string{"ParentKey", "RecKey"}
	values := map[string]int{"ParentKey": 1, "RecKey": 2}
	index := 3
	for attrName, attr := range node.Attributes {
		if attr.IsStruct {
			continue
		}
		fields = append(fields, attrName)
		values[attrName] = index
		index++
	}
	return fields, values
}

// getValueForPostgres deserializes a byte array and converts it to a PostgreSQL-compatible value.
// Slices and maps are serialized to a string format with type prefixes.
func getValueForPostgres(data []byte, r ifs.IRegistry) (interface{}, error) {
	obj := object.NewDecode(data, 0, r)
	val, err := obj.Get()
	if err != nil {
		return nil, err
	}
	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return "nil", nil
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
		str := strings.New()
		str.TypesPrefix = true
		val = str.ToString(v)
	}
	return val, nil
}
