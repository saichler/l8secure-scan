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

// Package common defines the core interfaces for the L8 ORM system.
// These interfaces abstract database operations, allowing different backend
// implementations (e.g., PostgreSQL, MySQL) to be used interchangeably.
package common

import (
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
)

// IORM defines the primary interface for Object-Relational Mapping operations.
// Implementations of this interface handle the conversion between Go objects
// and database records, providing CRUD operations at the object level.
type IORM interface {
	// Read executes a query and returns the matching elements as Go objects.
	// The query specifies the criteria, projections, and pagination parameters.
	Read(ifs.IQuery, ifs.IResources) ifs.IElements

	// Write persists elements to the database based on the specified action.
	// Action can be POST (insert), PUT (replace), or PATCH (partial update).
	Write(ifs.Action, ifs.IElements, ifs.IResources) error

	// Delete removes records matching the query criteria from the database.
	Delete(ifs.IQuery, ifs.IResources) error

	// Close releases database connections and cleans up resources.
	Close() error
}

// IsTimeSeriesType returns true for types that are handled by the TSDB service
// and should be skipped by the relational ORM.
func IsTimeSeriesType(typeName string) bool {
	return typeName == "L8TimeSeriesPoint"
}

// ITSDB defines the interface for Time Series Database operations.
// Implementations handle storage and retrieval of time series data points
// using a backend optimized for time-ordered data (e.g., TimescaleDB).
type ITSDB interface {
	// AddTSDB writes time series data points to the TSDB.
	AddTSDB(notifications []*l8notify.L8TSDBNotification) error

	// GetTSDB retrieves time series data points for a property within a time range.
	// start and end are Unix timestamps (seconds).
	GetTSDB(propertyId string, start, end int64) ([]*l8api.L8TimeSeriesPoint, error)

	// Close releases database connections and cleans up resources.
	Close() error
}

// IORMRelational extends IORM with methods for working directly with relational data.
// This interface is useful when you need more control over the relational representation
// of data, bypassing the automatic object conversion.
type IORMRelational interface {
	IORM

	// ReadRelational executes a query and returns raw relational data.
	// This is useful for advanced scenarios where the relational structure is needed.
	ReadRelational(ifs.IQuery) (*l8orms.L8OrmRData, error)

	// WriteRelational persists relational data directly to the database.
	// The action determines whether to insert, replace, or patch the data.
	WriteRelational(ifs.Action, *l8orms.L8OrmRData) error
}
