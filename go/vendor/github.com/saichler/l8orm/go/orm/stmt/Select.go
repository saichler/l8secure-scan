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
package stmt

import (
	"database/sql"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8utils/go/utils/strings"
	"reflect"
)

// SelectStatement returns a prepared SELECT statement for this table.
// The statement is created lazily and cached for subsequent calls.
func (this *Statement) SelectStatement(tx *sql.Tx) (*sql.Stmt, error) {
	if this.selectStmt == nil {
		err := this.createSelectStatement(tx)
		if err != nil {
			return nil, err
		}
	}
	return this.selectStmt, nil
}

// createSelectStatement generates and prepares the SELECT SQL statement.
// If a query is provided, it generates SQL from the query; otherwise it
// selects all columns from the table.
func (this *Statement) createSelectStatement(tx *sql.Tx) error {
	var sel *strings.String
	if this.query != nil {
		s, ok := this.Query2Sql(this.query, this.node.TypeName)
		if !ok {
			return nil
		}
		sel = strings.New(s)
	} else {
		sel = strings.New("Select ")
		if this.fields == nil {
			this.fields, this.values = fieldsOf(this.node)
		}
		first := true
		for _, fieldName := range this.fields {
			if !first {
				sel.Add(",")
			}
			first = false
			sel.Add(fieldName)
		}
		sel.Add(" from ").Add(this.node.TypeName)
		sel.Add(";")
	}
	st, err := tx.Prepare(sel.String())
	if err != nil {
		return err
	}
	this.selectStmt = st
	return nil
}

// Row scans a single row from a SQL result set into an L8OrmRow.
// It handles type conversions and serializes values for storage.
func (this *Statement) Row(rows *sql.Rows) (*l8orms.L8OrmRow, error) {
	args, err := this.NewArgs()
	vals := make([]interface{}, len(args))
	row := &l8orms.L8OrmRow{}
	if err != nil {
		return nil, err
	}
	err = rows.Scan(args...)
	if err != nil {
		return nil, err
	}
	if err == nil {
		for i, arg := range args {
			vals[i] = reflect.ValueOf(arg).Elem().Interface()
		}
		row.ParentKey = vals[0].(string)
		row.RecKey = vals[1].(string)
	} else {
		return nil, err
	}
	for i := 2; i < len(vals); i++ {
		attr := this.node.Attributes[this.fields[i]]
		pos := this.columns[this.fields[i]]
		var value interface{}
		if attr.IsMap || attr.IsSlice {
			v, e := strings.FromString(vals[i].(string), this.registy)
			if e != nil {
				return nil, e
			}
			value = v.Interface()
		} else {
			value = vals[i]
		}
		obj := object.NewEncode()
		obj.Add(value)
		if row.ColumnValues == nil {
			row.ColumnValues = make(map[int32][]byte)
		}
		row.ColumnValues[pos] = obj.Data()
	}
	return row, nil
}

// NewArgs creates a slice of pointers for scanning row values from SQL results.
// Each pointer is typed according to the corresponding column's type.
func (this *Statement) NewArgs() ([]interface{}, error) {
	args := make([]interface{}, len(this.fields))
	parentKey := ""
	recKey := ""
	args[0] = &parentKey
	args[1] = &recKey
	for i := 2; i < len(this.fields); i++ {
		attr := this.node.Attributes[this.fields[i]]
		typ := reflect.TypeOf("")
		if !attr.IsSlice && !attr.IsMap {
			info, e := this.registy.Info(attr.TypeName)
			if e != nil {
				return nil, e
			}
			typ = info.Type()
		}
		args[i] = reflect.New(typ).Interface()
	}
	return args, nil
}
