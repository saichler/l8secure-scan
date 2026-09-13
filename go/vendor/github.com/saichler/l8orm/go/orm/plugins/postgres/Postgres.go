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

// Package postgres provides a PostgreSQL implementation of the IORM interface.
// It handles database connections, table creation, query execution, and includes
// an in-memory query cache with TTL support for optimized pagination performance.
package postgres

import (
	"database/sql"
	"errors"
	strings2 "strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
	"github.com/saichler/l8types/go/types/l8reflect"
	"github.com/saichler/l8utils/go/utils/strings"
)

// cachedQuery represents a cached query result with its sorted RecKey array.
// This cache enables efficient pagination by storing the full result set's keys
// and serving page requests from memory rather than re-querying the database.
type cachedQuery struct {
	recKeys  []string            // Sorted array of record keys for the query
	stamp    int64               // Cache creation timestamp for invalidation
	lastUsed int64               // Last access time for TTL cleanup
	metadata *l8api.L8MetaData   // Query metadata (total count, etc.)
}

// touch updates the lastUsed timestamp to prevent TTL expiration.
func (cq *cachedQuery) touch() {
	atomic.StoreInt64(&cq.lastUsed, time.Now().Unix())
}

// pageKeys returns the subset of record keys for the requested page.
// If limit is 0 or negative, all keys are returned.
func (cq *cachedQuery) pageKeys(page, limit int32) []string {
	if limit <= 0 {
		return cq.recKeys
	}
	start := int(page * limit)
	if start >= len(cq.recKeys) {
		return []string{}
	}
	end := start + int(limit)
	if end > len(cq.recKeys) {
		end = len(cq.recKeys)
	}
	return cq.recKeys[start:end]
}

// Postgres implements the IORM interface for PostgreSQL databases.
// It provides connection pooling, automatic table creation, query caching,
// and batch write support for efficient database operations.
type Postgres struct {
	db        *sql.DB              // Database connection pool
	verifyed  map[string]bool      // Tracks verified/created tables
	mtx       *sync.Mutex          // Protects database operations
	res       ifs.IResources       // Layer 8 resources (introspector, registry, etc.)
	batchSize int                  // Maximum elements per write batch

	tsdb *Tsdb

	// Primary index for paging - caches query results for pagination
	indexMtx      *sync.RWMutex              // Protects index cache
	indexQueries  map[int64]*cachedQuery     // Query hash (+ AAA ID) -> cached results
	indexStamp    int64                      // Global invalidation stamp
	indexTTL      int64                      // Cache entry TTL in seconds
	indexStopCh   chan struct{}              // Signal to stop TTL cleaner
}

// NewPostgres creates a new PostgreSQL ORM instance with the given database connection.
// It initializes the query cache with a 30-second TTL and starts a background
// goroutine to clean up expired cache entries every 10 seconds.
func NewPostgres(db *sql.DB, resourcs ifs.IResources) *Postgres {
	p := &Postgres{
		db:           db,
		verifyed:     make(map[string]bool),
		mtx:          &sync.Mutex{},
		res:          resourcs,
		batchSize:    500,
		tsdb:         NewTsdb(db, false),
		indexMtx:     &sync.RWMutex{},
		indexQueries: make(map[int64]*cachedQuery),
		indexStamp:   time.Now().Unix(),
		indexTTL:     30,
		indexStopCh:  make(chan struct{}),
	}
	go p.indexTTLCleaner()
	return p
}

// indexTTLCleaner runs in a goroutine to periodically remove expired cache entries.
// It checks every 10 seconds and removes entries that haven't been accessed
// within the TTL window.
func (this *Postgres) indexTTLCleaner() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			this.cleanExpiredQueries()
		case <-this.indexStopCh:
			return
		}
	}
}

// cleanExpiredQueries removes cache entries that have exceeded their TTL.
func (this *Postgres) cleanExpiredQueries() {
	this.indexMtx.Lock()
	defer this.indexMtx.Unlock()
	now := time.Now().Unix()
	for hash, q := range this.indexQueries {
		if now-atomic.LoadInt64(&q.lastUsed) > this.indexTTL {
			delete(this.indexQueries, hash)
		}
	}
}

// invalidateIndex marks all cached queries as stale by updating the global stamp.
// Called after write or delete operations to ensure cache consistency.
func (this *Postgres) invalidateIndex() {
	this.indexMtx.Lock()
	defer this.indexMtx.Unlock()
	this.indexStamp = time.Now().Unix()
}

// collectTables recursively collects all table names needed for a type hierarchy.
// It traverses nested struct attributes to find all related table types.
func collectTables(node *l8reflect.L8Node, tables map[string]bool) {
	tables[node.TypeName] = true
	if node.Attributes != nil {
		for _, attr := range node.Attributes {
			if attr.IsStruct {
				if common.IsTimeSeriesType(attr.TypeName) {
					continue
				}
				_, ok := tables[attr.TypeName]
				if !ok {
					collectTables(attr, tables)
				}
			}
		}
	}
}

// verifyTables ensures all required tables exist in the database.
// It checks each table in the type hierarchy and creates missing tables.
func (this *Postgres) verifyTables(rootNode *l8reflect.L8Node) error {
	tables := make(map[string]bool)
	collectTables(rootNode, tables)
	for tableName, _ := range tables {
		_, ok := this.verifyed[tableName]
		if !ok {
			err := this.verifyTable(tableName)
			if err != nil {
				return err
			}
			this.verifyed[tableName] = true
		}
	}
	return nil
}

// verifyTable checks if a table exists and creates it if not.
// If the table already exists, it reconciles its columns with the current
// proto definition and adds any missing columns via ALTER TABLE.
// Uses a test query to detect non-existent tables.
func (this *Postgres) verifyTable(tableName string) error {
	q := strings.New("select * from ", tableName, " where false;")
	_, err := this.db.Exec(q.String())
	if err != nil {
		if strings2.Contains(err.Error(), "does not exist") {
			return this.createTable(tableName)
		}
		return err
	}
	// Table exists — reconcile its columns with the current proto definition.
	return this.migrateTable(tableName)
}

// migrateTable compares the live table columns against the current proto
// definition and adds any missing columns via ALTER TABLE ADD COLUMN.
// It is purely additive: columns that exist in the table but not in the
// proto are left alone, and type changes are not handled. Non-unique
// indexes are created for any newly added columns that are decorated as
// non-unique, matching the DDL pattern used by createTable.
func (this *Postgres) migrateTable(tableName string) error {
	node, ok := this.res.Introspector().NodeByTypeName(tableName)
	if !ok {
		return errors.New("Cannot find node for table " + tableName)
	}

	// Fetch the live column set. information_schema folds unquoted
	// identifiers to lowercase, so we compare case-insensitively.
	rows, err := this.db.Query(
		"SELECT column_name FROM information_schema.columns WHERE table_name = $1",
		strings2.ToLower(tableName))
	if err != nil {
		return err
	}
	liveColumns := make(map[string]bool)
	for rows.Next() {
		var colName string
		if scanErr := rows.Scan(&colName); scanErr != nil {
			rows.Close()
			return scanErr
		}
		liveColumns[strings2.ToLower(colName)] = true
	}
	rows.Close()

	// Walk the proto attributes with the same skip rules createTable uses,
	// collecting scalar attributes that are missing from the live table.
	missing := make([]string, 0)
	missingTypes := make(map[string]string)
	for attrName, attr := range node.Attributes {
		if attr.IsStruct {
			continue
		}
		if common.IsTimeSeriesType(attr.TypeName) {
			continue
		}
		if liveColumns[strings2.ToLower(attrName)] {
			continue
		}
		missing = append(missing, attrName)
		missingTypes[attrName] = postgresTypeOf(attr)
	}

	if len(missing) == 0 {
		return nil
	}

	this.res.Logger().Info("Migrating table ", tableName, ": adding columns ", missing)

	for _, attrName := range missing {
		alterQ := strings.New("ALTER TABLE ", tableName, " ADD COLUMN ", attrName, " ", missingTypes[attrName], ";")
		_, err = this.db.Exec(alterQ.String())
		if err != nil {
			return err
		}
	}

	// Recreate non-unique indexes for any newly added columns that are
	// decorated as non-unique. Use IF NOT EXISTS so a partially-applied
	// prior migration does not fail.
	nonUniqueFields, nonUniqueErr := this.res.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_NonUnique)
	if nonUniqueErr == nil && nonUniqueFields != nil {
		missingSet := make(map[string]bool, len(missing))
		for _, name := range missing {
			missingSet[name] = true
		}
		for _, fieldName := range nonUniqueFields {
			if !missingSet[fieldName] {
				continue
			}
			this.res.Logger().Info("Creating non-unique index ", tableName, "_", fieldName, "_idx")
			indexQ := strings.New("CREATE INDEX IF NOT EXISTS ", tableName, "_", fieldName, "_idx ON ", tableName, " (", fieldName, ");")
			_, err = this.db.Exec(indexQ.String())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// createTable generates and executes DDL to create a table for the given type.
// It creates columns for all non-struct attributes and adds a composite primary
// key (ParentKey, RecKey). Non-unique indexes are created for decorated fields.
func (this *Postgres) createTable(tableName string) error {
	q := strings.New("create table ", tableName, " (\n")
	q.Add("ParentKey text,\n")
	q.Add("RecKey text,\n")
	node, ok := this.res.Introspector().NodeByTypeName(tableName)
	nonUniqueFieldsIndex, nonUniqueErr := this.res.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_NonUnique)

	if !ok {
		return errors.New("Cannot find node for table " + tableName)
	}
	for attrName, attr := range node.Attributes {
		if attr.IsStruct {
			continue
		}
		q.Add(attrName)
		q.Add(" ")
		q.Add(postgresTypeOf(attr))
		q.Add(",\n")
	}
	q.Add("CONSTRAINT ", tableName, "_key PRIMARY KEY (ParentKey, RecKey)\n);")
	_, err := this.db.Exec(q.String())
	if err != nil {
		return err
	}

	// Create non-unique indexes if available
	if nonUniqueErr == nil && nonUniqueFieldsIndex != nil {
		for _, fieldName := range nonUniqueFieldsIndex {
			indexQ := strings.New("CREATE INDEX ", tableName, "_", fieldName, "_idx ON ", tableName, " (", fieldName, ");")
			_, err = this.db.Exec(indexQ.String())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// postgresTypeOf maps Go types to PostgreSQL column types.
// Maps and slices are stored as text (serialized), and enums default to integer.
func postgresTypeOf(node *l8reflect.L8Node) string {
	if node.IsMap || node.IsSlice {
		return "text"
	}
	switch node.TypeName {
	case "string":
		return "text"
	case "int32":
		return "integer"
	case "int64":
		return "bigint"
	case "float64":
		return "float8"
	case "float32":
		return "real"
	case "bool":
		return "boolean"
	}
	//default to enum for now - @TODO - reflect find what is the kind
	return "integer"
}

func (this *Postgres) AddTSDB(notifications []*l8notify.L8TSDBNotification) error {
	return this.tsdb.AddTSDB(notifications)
}

func (this *Postgres) GetTSDB(propertyId string, start, end int64) ([]*l8api.L8TimeSeriesPoint, error) {
	return this.tsdb.GetTSDB(propertyId, start, end)
}

func hashString(s string) int32 {
	var h int32
	for _, c := range s {
		h = 31*h + int32(c)
	}
	return h
}

// Close stops the TTL cleaner goroutine and closes the database connection.
func (this *Postgres) Close() error {
	close(this.indexStopCh)
	this.tsdb.Close()
	this.db.Close()
	return nil
}
