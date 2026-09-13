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
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
)

// DeleteStatement generates and prepares a DELETE SQL statement.
// For child tables, it deletes by ParentKey pattern matching.
// For root tables, it uses the query criteria for filtering.
func (this *Statement) DeleteStatement(tx *sql.Tx, parentKeyPattern string) (*sql.Stmt, error) {
	del := strings.New("DELETE FROM ")
	del.Add(this.node.TypeName)

	// If parentKeyPattern is provided, delete by ParentKey pattern (for child tables)
	if parentKeyPattern != "" {
		del.Add(" WHERE ParentKey LIKE '")
		del.Add(parentKeyPattern)
		del.Add("%'")
	} else if this.query != nil && this.query.Criteria() != nil {
		// For root table, use the query criteria
		ok, whereClause := expression(this.query.Criteria(), this.query.RootType().TypeName)
		if ok {
			del.Add(" WHERE ")
			del.Add(whereClause)
		}
	}
	del.Add(";")
	return tx.Prepare(del.String())
}

// DeleteByKeysStatement generates a DELETE statement that removes records
// where ParentKey matches any of the provided key patterns.
// Used for cascading deletes in child tables.
func (this *Statement) DeleteByKeysStatement(tx *sql.Tx, keys []string) (*sql.Stmt, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	del := strings.New("DELETE FROM ")
	del.Add(this.node.TypeName)
	del.Add(" WHERE ")

	first := true
	for _, key := range keys {
		if !first {
			del.Add(" OR ")
		}
		first = false
		del.Add("ParentKey LIKE '")
		del.Add(key)
		del.Add("%'")
	}

	del.Add(";")
	return tx.Prepare(del.String())
}

// Query2DeleteSql generates a DELETE SQL string from a query.
// Applies the query's criteria as a WHERE clause for the root table.
func (this *Statement) Query2DeleteSql(query ifs.IQuery, typeName string) (string, bool) {
	del := strings.New("DELETE FROM ")
	del.Add(typeName)

	if query.Criteria() == nil {
		return del.String(), true
	}

	if typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			del.Add(" WHERE ")
			del.Add(str)
		}
	}
	return del.String(), true
}
