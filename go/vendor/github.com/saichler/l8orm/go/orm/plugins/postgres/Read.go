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
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"strings"
)

// ReadRelational executes a query and returns raw relational data.
// It fetches data from all tables in the query's type hierarchy and
// returns the results as L8OrmRData along with metadata (record counts).
func (this *Postgres) ReadRelational(query ifs.IQuery) (*l8orms.L8OrmRData, *l8api.L8MetaData, error) {
	data, err := convert.NewRelationsDataForQuery(query)
	if err != nil {
		return nil, nil, err
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()

	rootNode, ok := this.res.Introspector().NodeByTypeName(query.RootType().TypeName)
	if ok {
		err = this.verifyTables(rootNode)
		if err != nil {
			return nil, nil, err
		}
	}

	var tx *sql.Tx
	var er error

	tx, er = this.db.Begin()
	if er != nil {
		return nil, nil, er
	}

	defer func() {
		if er != nil {
			er = tx.Rollback()
		} else {
			er = tx.Commit()
		}
	}()

	var rootTableStatement *stmt.Statement

	for tableName, table := range data.Tables {
		node, ok := this.res.Introspector().NodeByTypeName(tableName)
		if !ok {
			return nil, nil, errors.New("table not found " + data.RootTypeName)
		}
		statement := stmt.NewStatement(node, table.Columns, query, this.res.Registry())
		st, err := statement.SelectStatement(tx)
		if err != nil {
			return nil, nil, err
		}
		if st == nil {
			continue
		}

		if strings.ToLower(tableName) == strings.ToLower(query.RootType().TypeName) {
			rootTableStatement = statement
		}

		rows, err := st.Query()
		if err != nil {
			return nil, nil, err
		}
		dataRow, err := this.readRows(rows, statement)
		if err != nil {
			return nil, nil, err
		}
		for _, row := range dataRow {
			fldName := nameOfField(row.RecKey)
			if table.InstanceRows == nil {
				table.InstanceRows = make(map[string]*l8orms.L8OrmInstanceRows)
			}
			if table.InstanceRows[row.ParentKey] == nil {
				table.InstanceRows[row.ParentKey] = &l8orms.L8OrmInstanceRows{}
			}
			if table.InstanceRows[row.ParentKey].AttributeRows == nil {
				table.InstanceRows[row.ParentKey].AttributeRows = make(map[string]*l8orms.L8OrmAttributeRows)
			}
			if table.InstanceRows[row.ParentKey].AttributeRows[fldName] == nil {
				table.InstanceRows[row.ParentKey].AttributeRows[fldName] = &l8orms.L8OrmAttributeRows{}
			}
			if table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows == nil {
				table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows = make([]*l8orms.L8OrmRow, 0)
			}
			attrRows := table.InstanceRows[row.ParentKey].AttributeRows[fldName]
			attrRows.Rows = append(attrRows.Rows, row)
		}
	}
	return data, rootTableStatement.MetaData(tx), nil
}

// nameOfField extracts the field name from a RecKey by removing the bracketed portion.
// For example, "MyField[123]" returns "MyField".
func nameOfField(recKey string) string {
	index := strings.Index(recKey, "[")
	if index == -1 {
		return recKey
	}
	return recKey[0:index]
}

// readRows scans all rows from a SQL result set into L8OrmRow structures.
func (this *Postgres) readRows(rows *sql.Rows, statement *stmt.Statement) ([]*l8orms.L8OrmRow, error) {
	result := make([]*l8orms.L8OrmRow, 0)
	for rows.Next() {
		row, err := statement.Row(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, nil
}

// Read executes a query and returns the results as Go objects.
// For paginated queries (with Limit > 0), it uses the in-memory index cache.
// For non-paginated queries, it performs a direct database read.
func (this *Postgres) Read(q ifs.IQuery, resources ifs.IResources) ifs.IElements {
	// Aggregate queries use a dedicated path (no ParentKey/RecKey scanning)
	if q.IsAggregate() {
		return this.readAggregate(q)
	}
	// Check if this query benefits from indexing (has Limit for pagination)
	if q.Limit() > 0 {
		return this.readWithIndex(q, resources)
	}
	// No pagination - use direct read
	relData, metadata, err := this.ReadRelational(q)
	if err != nil {
		return object.NewError(err.Error())
	}
	return this.populateTsFields(convert.ConvertFrom(object.New(nil, relData), metadata, resources), resources)
}

// readWithIndex uses the in-memory primary index for paginated queries.
// It caches the full query result's RecKeys and serves page requests from cache.
// Cache entries are per-user (AAA ID combined into hash) and invalidated on writes.
func (this *Postgres) readWithIndex(q ifs.IQuery, resources ifs.IResources) ifs.IElements {
	aaaId := q.AAAId()
	hash := int64(q.Hash())
	if aaaId != "" {
		hash = hash<<32 | int64(hashString(aaaId))
	}

	this.indexMtx.RLock()
	cached, exists := this.indexQueries[hash]
	currentStamp := this.indexStamp
	this.indexMtx.RUnlock()

	if exists && cached.stamp == currentStamp {
		cached.touch()
		return this.readByRecKeys(q, cached.pageKeys(q.Page(), q.Limit()), cached.metadata, resources)
	}

	recKeys, metadata, err := this.readRecKeys(q)
	if err != nil {
		return object.NewError(err.Error())
	}

	if aaaId != "" && resources.Security() != nil {
		recKeys, metadata = this.filterRecKeysBySecurity(q, recKeys, resources, aaaId)
	}

	this.indexMtx.Lock()
	cached = &cachedQuery{
		recKeys:  recKeys,
		stamp:    currentStamp,
		lastUsed: currentStamp,
		metadata: metadata,
	}
	this.indexQueries[hash] = cached
	this.indexMtx.Unlock()

	return this.readByRecKeys(q, cached.pageKeys(q.Page(), q.Limit()), metadata, resources)
}

// filterRecKeysBySecurity fetches full objects for the RecKeys, applies ScopeItem
// to each, and returns only the RecKeys that pass the security filter.
func (this *Postgres) filterRecKeysBySecurity(q ifs.IQuery, recKeys []string, resources ifs.IResources, aaaId string) ([]string, *l8api.L8MetaData) {
	if len(recKeys) == 0 {
		return recKeys, &l8api.L8MetaData{}
	}

	uuid := ""
	if resources.SysConfig() != nil {
		uuid = resources.SysConfig().LocalUuid
	}

	elements := this.readByRecKeys(q, recKeys, nil, resources)
	if elements == nil || elements.Error() != nil {
		return recKeys, &l8api.L8MetaData{}
	}

	elems := elements.Elements()
	filteredKeys := make([]string, 0, len(recKeys))

	for i, elem := range elems {
		if elem == nil {
			continue
		}
		if i >= len(recKeys) {
			break
		}
		scoped := resources.Security().ScopeItem(resources, elem, uuid, aaaId)
		if scoped != nil {
			filteredKeys = append(filteredKeys, recKeys[i])
		}
	}

	metadata := &l8api.L8MetaData{
		KeyCount: &l8api.L8Count{
			Counts: map[string]float64{
				"Total": float64(len(filteredKeys)),
			},
		},
	}

	return filteredKeys, metadata
}

// readRecKeys fetches only RecKeys for the root table (for cache population).
// This lightweight query is used to populate the pagination index without
// fetching all column data.
func (this *Postgres) readRecKeys(query ifs.IQuery) ([]string, *l8api.L8MetaData, error) {
	this.mtx.Lock()
	defer this.mtx.Unlock()

	node, ok := this.res.Introspector().NodeByTypeName(query.RootType().TypeName)
	if !ok {
		return nil, nil, errors.New("table not found " + query.RootType().TypeName)
	}

	err := this.verifyTables(node)
	if err != nil {
		return nil, nil, err
	}

	tx, er := this.db.Begin()
	if er != nil {
		return nil, nil, er
	}

	defer func() {
		if er != nil {
			er = tx.Rollback()
		} else {
			er = tx.Commit()
		}
	}()

	statement := stmt.NewStatement(node, nil, query, this.res.Registry())
	sqlStr := statement.Query2RecKeysSql(query, query.RootType().TypeName)

	rows, err := tx.Query(sqlStr)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	recKeys := make([]string, 0)
	for rows.Next() {
		var recKey string
		err := rows.Scan(&recKey)
		if err != nil {
			return nil, nil, err
		}
		recKeys = append(recKeys, recKey)
	}

	metadata := statement.MetaData(tx)
	return recKeys, metadata, nil
}

// readByRecKeys fetches full row data for specific RecKeys (for pagination).
// After getting the page's RecKeys from the cache, this method fetches the
// complete row data for just those records.
func (this *Postgres) readByRecKeys(query ifs.IQuery, recKeys []string, metadata *l8api.L8MetaData, resources ifs.IResources) ifs.IElements {
	if len(recKeys) == 0 {
		return object.NewQueryResult(nil, metadata)
	}

	this.mtx.Lock()
	defer this.mtx.Unlock()

	data, err := convert.NewRelationsDataForQuery(query)
	if err != nil {
		return object.NewError(err.Error())
	}

	tx, er := this.db.Begin()
	if er != nil {
		return object.NewError(er.Error())
	}

	defer func() {
		if er != nil {
			er = tx.Rollback()
		} else {
			er = tx.Commit()
		}
	}()

	// Create a map of recKey to order for preserving order
	recKeyOrder := make(map[string]int)
	for i, key := range recKeys {
		recKeyOrder[key] = i
	}

	for tableName, table := range data.Tables {
		node, ok := this.res.Introspector().NodeByTypeName(tableName)
		if !ok {
			return object.NewError("table not found " + tableName)
		}

		statement := stmt.NewStatement(node, table.Columns, query, this.res.Registry())

		var sqlStr string
		if strings.ToLower(tableName) == strings.ToLower(query.RootType().TypeName) {
			// Root table: fetch by RecKeys
			sqlStr = statement.Query2SqlByRecKeys(tableName, recKeys)
		} else {
			// Child tables: fetch by ParentKey matching any recKey
			// ParentKey of child tables contains the parent's RecKey
			st, err := statement.SelectStatement(tx)
			if err != nil {
				return object.NewError(err.Error())
			}
			if st == nil {
				continue
			}
			rows, err := st.Query()
			if err != nil {
				return object.NewError(err.Error())
			}
			dataRow, err := this.readRows(rows, statement)
			if err != nil {
				return object.NewError(err.Error())
			}
			// Filter child rows by checking if ParentKey matches any root RecKey
			for _, row := range dataRow {
				if this.parentKeyMatchesRecKeys(row.ParentKey, recKeys) {
					this.addRowToTable(table, row)
				}
			}
			continue
		}

		rows, err := tx.Query(sqlStr)
		if err != nil {
			return object.NewError(err.Error())
		}

		dataRow, err := this.readRows(rows, statement)
		if err != nil {
			return object.NewError(err.Error())
		}

		// Sort dataRow according to recKeyOrder to preserve sort order from cache
		sortedRows := make([]*l8orms.L8OrmRow, len(dataRow))
		for _, row := range dataRow {
			if idx, ok := recKeyOrder[row.RecKey]; ok {
				sortedRows[idx] = row
			}
		}
		// Add sorted rows to table (skip any nil entries from mismatched keys)
		for _, row := range sortedRows {
			if row != nil {
				this.addRowToTable(table, row)
			}
		}
	}

	return this.populateTsFields(convert.ConvertFrom(object.New(nil, data), metadata, resources), resources)
}

// parentKeyMatchesRecKeys checks if a child's ParentKey contains one of the root RecKeys.
// This is used to filter child table rows when fetching paginated data.
func (this *Postgres) parentKeyMatchesRecKeys(parentKey string, recKeys []string) bool {
	for _, recKey := range recKeys {
		if strings.Contains(parentKey, recKey) {
			return true
		}
	}
	return false
}

// addRowToTable adds a row to the table's nested structure.
// It initializes any missing intermediate structures (InstanceRows, AttributeRows).
func (this *Postgres) addRowToTable(table *l8orms.L8OrmTable, row *l8orms.L8OrmRow) {
	fldName := nameOfField(row.RecKey)
	if table.InstanceRows == nil {
		table.InstanceRows = make(map[string]*l8orms.L8OrmInstanceRows)
	}
	if table.InstanceRows[row.ParentKey] == nil {
		table.InstanceRows[row.ParentKey] = &l8orms.L8OrmInstanceRows{}
	}
	if table.InstanceRows[row.ParentKey].AttributeRows == nil {
		table.InstanceRows[row.ParentKey].AttributeRows = make(map[string]*l8orms.L8OrmAttributeRows)
	}
	if table.InstanceRows[row.ParentKey].AttributeRows[fldName] == nil {
		table.InstanceRows[row.ParentKey].AttributeRows[fldName] = &l8orms.L8OrmAttributeRows{}
	}
	if table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows == nil {
		table.InstanceRows[row.ParentKey].AttributeRows[fldName].Rows = make([]*l8orms.L8OrmRow, 0)
	}
	attrRows := table.InstanceRows[row.ParentKey].AttributeRows[fldName]
	attrRows.Rows = append(attrRows.Rows, row)
}
