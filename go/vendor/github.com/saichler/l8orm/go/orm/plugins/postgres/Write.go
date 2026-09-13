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
)

// WriteRelational persists relational data to the database.
// It verifies all required tables exist, then writes all rows within a transaction.
// For POST/PUT actions, uses INSERT with ON CONFLICT UPDATE (upsert).
// For PATCH actions, uses UPDATE with COALESCE to preserve existing values.
func (this *Postgres) WriteRelational(action ifs.Action, data *l8orms.L8OrmRData) error {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	rootNode, ok := this.res.Introspector().NodeByTypeName(data.RootTypeName)
	if !ok {
		return errors.New("Cannot find node for root type name " + data.RootTypeName)
	}
	err := this.verifyTables(rootNode)
	if err != nil {
		return err
	}
	err = this.writeData(action, data)
	if err != nil {
		return err
	}
	return nil
}

// writeData writes all table data within a single database transaction.
// It iterates through all tables and rows, executing the appropriate
// insert or update statements based on the action.
func (this *Postgres) writeData(action ifs.Action, data *l8orms.L8OrmRData) error {
	tx, err := this.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	for tableName, table := range data.Tables {
		node, ok := this.res.Introspector().NodeByTypeName(tableName)
		if !ok {
			err = errors.New("No node was found for " + tableName)
			return err
		}
		statement := stmt.NewStatement(node, table.Columns, nil, this.res.Registry())

		var sqlStmt *sql.Stmt
		if action == ifs.PATCH {
			sqlStmt, err = statement.UpdateStatement(tx)
		} else {
			sqlStmt, err = statement.InsertStatement(tx)
		}
		if err != nil {
			return err
		}

		for _, instRows := range table.InstanceRows {
			for _, attrRows := range instRows.AttributeRows {
				for _, row := range attrRows.Rows {
					args, e := statement.RowValues(action, row)
					if e != nil {
						err = e
						return err
					}
					_, e = sqlStmt.Exec(args...)
					if e != nil {
						err = e
						return err
					}
				}
			}
		}
	}
	return nil
}

// Write converts Go objects to relational data and persists them to the database.
// It invalidates the query cache after writing, and processes large element sets
// in batches (default 500 elements per batch) to avoid memory issues.
func (this *Postgres) Write(action ifs.Action, elems ifs.IElements, resources ifs.IResources) error {
	// Invalidate the index cache on write
	defer this.invalidateIndex()

	elements := elems.Elements()

	// If within batch size, process directly (original behavior)
	if len(elements) <= this.batchSize {
		relData := convert.ConvertTo(action, elems, resources)
		if relData.Error() != nil {
			return relData.Error()
		}
		data := relData.Element().(*l8orms.L8OrmRData)
		if err := this.WriteRelational(action, data); err != nil {
			return err
		}
		return this.writeTsData(data)
	}

	// Process in batches of batchSize
	for start := 0; start < len(elements); start += this.batchSize {
		end := start + this.batchSize
		if end > len(elements) {
			end = len(elements)
		}

		// Create batch slice
		batchSlice := make([]interface{}, end-start)
		for i := start; i < end; i++ {
			batchSlice[i-start] = elements[i]
		}

		// Convert and write this batch
		batchElems := object.New(nil, batchSlice)
		relData := convert.ConvertTo(action, batchElems, resources)
		if relData.Error() != nil {
			return relData.Error()
		}

		data := relData.Element().(*l8orms.L8OrmRData)
		if err := this.WriteRelational(action, data); err != nil {
			return err
		}
		if err := this.writeTsData(data); err != nil {
			return err
		}
	}
	return nil
}

func (this *Postgres) writeTsData(data *l8orms.L8OrmRData) error {
	if len(data.TsData) == 0 {
		return nil
	}
	return this.tsdb.AddTSDB(data.TsData)
}
