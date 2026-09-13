// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cache

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/saichler/l8types/go/types/l8api"
)

// PackAggregateResults writes aggregate results into metadata.KeyCount.Counts.
// Without GROUP BY: keys are aggregate aliases (e.g., "count", "sumMyInt32").
// With GROUP BY: keys are "alias:groupValue" (e.g., "count:GroupA", "sumMyInt32:GroupA").
// For multi-field GROUP BY: "alias:val1|val2".
func PackAggregateResults(groups []map[string]interface{}, aggregates []*l8api.L8AggregateFunction, groupByFields []string, metadata *l8api.L8MetaData) {
	if len(groupByFields) == 0 {
		// Single group — flat keys
		if len(groups) == 1 {
			for _, agg := range aggregates {
				if val, ok := ToFloat64(groups[0][agg.Alias]); ok {
					metadata.KeyCount.Counts[agg.Alias] = val
				}
			}
		}
		return
	}

	// Multiple groups — prefixed keys
	for _, group := range groups {
		groupKey := BuildGroupKeyString(group, groupByFields)
		for _, agg := range aggregates {
			key := agg.Alias + ":" + groupKey
			if val, ok := ToFloat64(group[agg.Alias]); ok {
				metadata.KeyCount.Counts[key] = val
			}
		}
	}
}

// BuildGroupKeyString creates a string key from group-by field values.
// Single field: "GroupA". Multiple fields: "Sales|West".
func BuildGroupKeyString(group map[string]interface{}, groupByFields []string) string {
	parts := make([]string, 0, len(groupByFields))
	for _, field := range groupByFields {
		val := group[field]
		if val != nil {
			parts = append(parts, fmt.Sprintf("%v", val))
		} else {
			parts = append(parts, "<nil>")
		}
	}
	return strings.Join(parts, "|")
}

// ToFloat64 converts any numeric type to float64.
func ToFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	// Handle protobuf enum types via reflection
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}
