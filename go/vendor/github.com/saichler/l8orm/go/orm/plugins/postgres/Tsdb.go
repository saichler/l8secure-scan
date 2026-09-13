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
	"sync"

	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
)

// Tsdb implements the ITSDB interface using TimescaleDB (PostgreSQL extension).
// It stores time series data in a single hypertable with a narrow schema
// (stamp, prop_id, value) for efficient compression and querying.
type Tsdb struct {
	db       *sql.DB
	mtx      *sync.Mutex
	verified bool
	ownsDb   bool
}

// NewTsdb creates a new TSDB instance. If ownsDb is true, Close() will close
// the database connection. Set ownsDb to false when sharing the connection
// with the relational ORM.
func NewTsdb(db *sql.DB, ownsDb bool) *Tsdb {
	return &Tsdb{
		db:     db,
		mtx:    &sync.Mutex{},
		ownsDb: ownsDb,
	}
}

// verifyTable creates the l8tsdb hypertable and index if they don't exist.
func (this *Tsdb) verifyTable() error {
	_, err := this.db.Exec(`CREATE TABLE IF NOT EXISTS l8tsdb (
		stamp    TIMESTAMPTZ NOT NULL,
		prop_id  TEXT        NOT NULL,
		value    FLOAT8      NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = this.db.Exec(
		`SELECT create_hypertable('l8tsdb', 'stamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE)`)
	if err != nil {
		return err
	}

	_, err = this.db.Exec(
		`CREATE INDEX IF NOT EXISTS idx_l8tsdb_prop_stamp ON l8tsdb (prop_id, stamp DESC)`)
	if err != nil {
		return err
	}

	return nil
}

// AddTSDB writes time series notifications to the database in a single transaction.
func (this *Tsdb) AddTSDB(notifications []*l8notify.L8TSDBNotification) error {
	this.mtx.Lock()
	defer this.mtx.Unlock()

	if !this.verified {
		if err := this.verifyTable(); err != nil {
			return err
		}
		this.verified = true
	}

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

	stmt, err := tx.Prepare(
		"INSERT INTO l8tsdb (stamp, prop_id, value) VALUES (to_timestamp($1), $2, $3)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, n := range notifications {
		if n == nil || n.Point == nil {
			continue
		}
		_, err = stmt.Exec(n.Point.Stamp, n.PropertyId, n.Point.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTSDB retrieves time series data points for a property within a time range.
func (this *Tsdb) GetTSDB(propertyId string, start, end int64) ([]*l8api.L8TimeSeriesPoint, error) {
	rows, err := this.db.Query(
		"SELECT extract(epoch from stamp)::bigint, value FROM l8tsdb "+
			"WHERE prop_id = $1 AND stamp BETWEEN to_timestamp($2) AND to_timestamp($3) "+
			"ORDER BY stamp",
		propertyId, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []*l8api.L8TimeSeriesPoint
	for rows.Next() {
		p := &l8api.L8TimeSeriesPoint{}
		if err := rows.Scan(&p.Stamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// GetTSDBLatest retrieves the most recent N data points for a property, ordered chronologically.
func (this *Tsdb) GetTSDBLatest(propertyId string, limit int) ([]*l8api.L8TimeSeriesPoint, error) {
	rows, err := this.db.Query(
		"SELECT stamp_epoch, value FROM ("+
			"SELECT extract(epoch from stamp)::bigint AS stamp_epoch, value FROM l8tsdb "+
			"WHERE prop_id = $1 ORDER BY stamp DESC LIMIT $2"+
			") sub ORDER BY stamp_epoch",
		propertyId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []*l8api.L8TimeSeriesPoint
	for rows.Next() {
		p := &l8api.L8TimeSeriesPoint{}
		if err := rows.Scan(&p.Stamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// SetRetention configures automatic data expiry for the TSDB hypertable.
func (this *Tsdb) SetRetention(interval string) error {
	_, err := this.db.Exec(
		"SELECT add_retention_policy('l8tsdb', INTERVAL '" + interval + "', if_not_exists => true)")
	return err
}

// Close releases the database connection if this instance owns it.
func (this *Tsdb) Close() error {
	if this.ownsDb {
		return this.db.Close()
	}
	return nil
}
