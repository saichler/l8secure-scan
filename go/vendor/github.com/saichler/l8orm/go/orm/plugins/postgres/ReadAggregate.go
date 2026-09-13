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

	"github.com/saichler/l8orm/go/orm/stmt"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/cache"
)

// readAggregate executes an aggregate query against the database and returns
// results packed into L8MetaData.KeyCount.Counts. Elements slice is empty.
func (this *Postgres) readAggregate(q ifs.IQuery) ifs.IElements {
	this.mtx.Lock()
	defer this.mtx.Unlock()

	rootNode, ok := this.res.Introspector().NodeByTypeName(q.RootType().TypeName)
	if !ok {
		return object.NewError("table not found " + q.RootType().TypeName)
	}

	err := this.verifyTables(rootNode)
	if err != nil {
		return object.NewError(err.Error())
	}

	tx, err := this.db.Begin()
	if err != nil {
		return object.NewError(err.Error())
	}
	defer tx.Commit()

	statement := stmt.NewStatement(rootNode, nil, q, this.res.Registry())
	sqlStr, ok := statement.AggregateSql(q)
	if !ok {
		return object.NewError("failed to generate aggregate SQL")
	}

	rows, err := tx.Query(sqlStr)
	if err != nil {
		return object.NewError(err.Error())
	}
	defer rows.Close()

	groups, err := scanAggregateRows(rows, q)
	if err != nil {
		return object.NewError(err.Error())
	}

	metadata := &l8api.L8MetaData{
		KeyCount: &l8api.L8Count{
			Counts: make(map[string]float64),
		},
	}
	cache.PackAggregateResults(groups, q.Aggregates(), q.GroupBy(), metadata)

	return object.NewQueryResult([]interface{}{}, metadata)
}

// scanAggregateRows scans SQL result rows into []map[string]interface{}.
// Column order matches the SELECT clause: group-by fields first, then aggregates.
func scanAggregateRows(rows *sql.Rows, q ifs.IQuery) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, errors.New("aggregate query returned no columns")
	}

	groups := make([]map[string]interface{}, 0)

	// Build column name list: group-by fields, then aggregate aliases
	colNames := make([]string, 0, len(cols))
	for _, gb := range q.GroupBy() {
		colNames = append(colNames, gb)
	}
	for _, agg := range q.Aggregates() {
		colNames = append(colNames, agg.Alias)
	}

	for rows.Next() {
		// Create scan targets
		scanVals := make([]interface{}, len(cols))
		scanPtrs := make([]interface{}, len(cols))
		for i := range scanVals {
			scanPtrs[i] = &scanVals[i]
		}

		if err := rows.Scan(scanPtrs...); err != nil {
			return nil, err
		}

		group := make(map[string]interface{})
		for i, name := range colNames {
			if i < len(scanVals) {
				group[name] = scanVals[i]
			}
		}
		groups = append(groups, group)
	}

	return groups, nil
}
