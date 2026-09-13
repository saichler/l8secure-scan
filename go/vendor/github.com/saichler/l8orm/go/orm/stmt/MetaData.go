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
	"github.com/saichler/l8types/go/types/l8api"
)

// MetaData executes a COUNT query and returns metadata with total record count.
// This is used to provide pagination information (total records) in query results.
func (this *Statement) MetaData(tx *sql.Tx) *l8api.L8MetaData {
	stmt, err := this.metadataStatement(tx)
	if err != nil {
		return nil
	}
	metadata := &l8api.L8MetaData{}
	metadata.KeyCount = &l8api.L8Count{}
	metadata.KeyCount.Counts = make(map[string]float64)
	totalRecords := 0
	rows, err := stmt.Query()
	if err != nil {
		return nil
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&totalRecords)
		if err != nil {
			return nil
		}
	}
	metadata.KeyCount.Counts["Total"] = float64(totalRecords)
	return metadata
}

// metadataStatement returns a prepared COUNT statement for the table.
// The statement is created lazily and cached for subsequent calls.
func (this *Statement) metadataStatement(tx *sql.Tx) (*sql.Stmt, error) {
	if this.metaDataStmt == nil {
		err := this.createMetadataStatement(tx)
		if err != nil {
			return nil, err
		}
	}
	return this.metaDataStmt, nil
}

// createMetadataStatement generates and prepares a COUNT SQL statement.
func (this *Statement) createMetadataStatement(tx *sql.Tx) error {
	sql := this.Query2CountSql(this.query, this.node.TypeName)
	st, err := tx.Prepare(sql)
	if err != nil {
		return err
	}
	this.metaDataStmt = st
	return nil
}
