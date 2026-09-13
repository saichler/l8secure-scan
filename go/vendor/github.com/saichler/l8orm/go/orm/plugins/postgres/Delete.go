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
package postgres

import (
	"database/sql"
	"errors"
	"github.com/saichler/l8orm/go/orm/convert"
	"github.com/saichler/l8orm/go/orm/stmt"
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/ifs"
	"strings"
)

// DeleteRelational removes records matching a query from the database.
// It maintains referential integrity by first deleting child table records
// (using ParentKey pattern matching) before deleting root table records.
func (this *Postgres) DeleteRelational(query ifs.IQuery) error {
	data, err := convert.NewRelationsDataForQuery(query)
	if err != nil {
		return err
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()

	var tx *sql.Tx
	var er error

	tx, er = this.db.Begin()
	if er != nil {
		return er
	}

	defer func() {
		if er != nil {
			er = tx.Rollback()
		} else {
			er = tx.Commit()
		}
	}()

	// First, read the root table keys to know what to delete from child tables
	rootTableName := query.RootType().TypeName
	rootKeys, er := this.readRootKeys(tx, query, data)
	if er != nil {
		return er
	}

	// If no matching records found, nothing to delete
	if len(rootKeys) == 0 {
		return nil
	}

	// Delete from child tables first (to maintain referential integrity)
	for tableName, table := range data.Tables {
		if strings.EqualFold(tableName, rootTableName) {
			continue // Skip root table, delete it last
		}

		node, ok := this.res.Introspector().NodeByTypeName(tableName)
		if !ok {
			er = errors.New("table not found " + tableName)
			return er
		}

		statement := stmt.NewStatement(node, table.Columns, query, this.res.Registry())
		deleteStmt, err := statement.DeleteByKeysStatement(tx, rootKeys)
		if err != nil {
			er = err
			return er
		}
		if deleteStmt == nil {
			continue
		}

		_, err = deleteStmt.Exec()
		if err != nil {
			er = err
			return er
		}
	}

	// Finally, delete from root table
	rootNode, ok := this.res.Introspector().NodeByTypeName(rootTableName)
	if !ok {
		er = errors.New("root table not found " + rootTableName)
		return er
	}

	rootTable := data.Tables[rootTableName]
	rootStatement := stmt.NewStatement(rootNode, rootTable.Columns, query, this.res.Registry())
	rootDeleteStmt, err := rootStatement.DeleteStatement(tx, "")
	if err != nil {
		er = err
		return er
	}

	_, er = rootDeleteStmt.Exec()
	return er
}

// readRootKeys fetches the composite keys (ParentKey + RecKey) of records to be deleted.
// These keys are used to identify related child records for cascading deletion.
func (this *Postgres) readRootKeys(tx *sql.Tx, query ifs.IQuery, data *l8orms.L8OrmRData) ([]string, error) {
	rootTableName := query.RootType().TypeName
	rootTable := data.Tables[rootTableName]

	node, ok := this.res.Introspector().NodeByTypeName(rootTableName)
	if !ok {
		return nil, errors.New("root table not found " + rootTableName)
	}

	statement := stmt.NewStatement(node, rootTable.Columns, query, this.res.Registry())
	selectStmt, err := statement.SelectStatement(tx)
	if err != nil {
		return nil, err
	}
	if selectStmt == nil {
		return nil, nil
	}

	rows, err := selectStmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var parentKey, recKey string
		// We only need ParentKey and RecKey, scan into temp variables for other columns
		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		values := make([]interface{}, len(cols))
		values[0] = &parentKey
		values[1] = &recKey
		for i := 2; i < len(cols); i++ {
			var temp interface{}
			values[i] = &temp
		}

		err = rows.Scan(values...)
		if err != nil {
			return nil, err
		}

		// The key for child tables is ParentKey + RecKey
		fullKey := parentKey + recKey
		keys = append(keys, fullKey)
	}

	return keys, nil
}

// Delete removes records matching the query and invalidates the query cache.
// This is the main entry point for deletion operations from the IORM interface.
func (this *Postgres) Delete(q ifs.IQuery, resources ifs.IResources) error {
	// Invalidate the index cache on delete
	defer this.invalidateIndex()
	return this.DeleteRelational(q)
}
