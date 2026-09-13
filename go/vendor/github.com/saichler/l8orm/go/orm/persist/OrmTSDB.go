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
package persist

import (
	"errors"
	"strconv"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8notify"
)

const tsdbQueryType = "L8TSDBQuery"

// AddTSDB writes time series notifications to the TSDB.
func (this *OrmService) AddTSDB(notifications []*l8notify.L8TSDBNotification) {
	if this.tsdb == nil {
		return
	}
	this.tsdb.AddTSDB(notifications)
}

// GetTSDB retrieves time series data for a property within a time range.
func (this *OrmService) GetTSDB(propertyId string, start, end int64) []*l8api.L8TimeSeriesPoint {
	if this.tsdb == nil {
		return nil
	}
	points, err := this.tsdb.GetTSDB(propertyId, start, end)
	if err != nil {
		return nil
	}
	return points
}

// isTsdbQuery checks if a parsed query targets the TSDB.
func isTsdbQuery(query ifs.IQuery) bool {
	return query.RootType().TypeName == tsdbQueryType
}

// handleTsdbQuery extracts propertyId, start, end from the query's
// where clause and delegates to GetTSDB.
func (this *OrmService) handleTsdbQuery(query ifs.IQuery) ifs.IElements {
	if this.tsdb == nil {
		return object.NewError("TSDB is not configured")
	}
	propertyId, start, end, err := extractTsdbParams(query)
	if err != nil {
		return object.NewError(err.Error())
	}
	points := this.GetTSDB(propertyId, start, end)
	return object.New(nil, points)
}

// extractTsdbParams walks the query's criteria expression tree and extracts
// the propertyId, tsdbstart, and tsdbend parameters from comparators.
func extractTsdbParams(query ifs.IQuery) (string, int64, int64, error) {
	var propertyId string
	var start, end int64

	expr := query.Criteria()
	for expr != nil {
		cond := expr.Condition()
		for cond != nil {
			comp := cond.Comparator()
			if comp != nil {
				left := comp.Left()
				right := comp.Right()
				switch left {
				case "PropertyId":
					propertyId = right
				case "Tsdbstart":
					start = parseTimestamp(right)
				case "Tsdbend":
					end = parseTimestamp(right)
				}
			}
			cond = cond.Next()
		}
		expr = expr.Next()
	}

	if propertyId == "" {
		return "", 0, 0, errors.New("propertyId is required for TSDB query")
	}
	if start == 0 || end == 0 {
		return "", 0, 0, errors.New("tsdbstart and tsdbend are required for TSDB query")
	}

	return propertyId, start, end, nil
}

// parseTimestamp converts a string timestamp value to int64.
func parseTimestamp(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}
