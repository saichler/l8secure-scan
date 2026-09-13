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
	"github.com/saichler/l8utils/go/utils/strings"
	"strconv"
)

// InsertStatement returns a prepared INSERT statement with upsert capability.
// The statement is created lazily and cached for subsequent calls.
func (this *Statement) InsertStatement(tx *sql.Tx) (*sql.Stmt, error) {
	if this.insertStmt == nil {
		err := this.createInsertStatement(tx)
		if err != nil {
			return nil, err
		}
	}
	return this.insertStmt, nil
}

// createInsertStatement generates and prepares an INSERT SQL statement with ON CONFLICT handling.
// When a record with the same (ParentKey, RecKey) exists, it updates all other columns.
func (this *Statement) createInsertStatement(tx *sql.Tx) error {
	insertInto := strings.New("insert into ", this.node.TypeName)
	if this.fields == nil {
		this.fields, this.values = fieldsOf(this.node)
	}
	fields := strings.New(" (")
	values := strings.New(" values (")
	conflict := strings.New("ON CONFLICT (ParentKey,RecKey) DO UPDATE SET ")
	first := true
	firstConflict := true
	for _, field := range this.fields {
		if !first {
			fields.Add(",")
			values.Add(",")
		}
		first = false
		fields.Add(field)
		values.Add("$")
		values.Add(strconv.Itoa(this.values[field]))
		if field != "ParentKey" && field != "RecKey" {
			if !firstConflict {
				conflict.Add(",")
			}
			firstConflict = false
			conflict.Add(field)
			conflict.Add("=")
			conflict.Add("$")
			conflict.Add(strconv.Itoa(this.values[field]))
			conflict.Add(" ")
		}
	}
	fields.Add(") ")
	values.Add(") ")
	insertInto.Add(fields.String())
	insertInto.Add(values.String())
	insertInto.Add(conflict.String())
	insertInto.Add(";")

	st, err := tx.Prepare(insertInto.String())
	if err != nil {
		return err
	}
	this.insertStmt = st
	return nil
}
