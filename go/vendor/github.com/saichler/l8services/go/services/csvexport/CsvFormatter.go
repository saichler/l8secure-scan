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

package csvexport

import (
	"fmt"
	"github.com/saichler/l8types/go/types/l8reflect"
	"reflect"
	"sort"
	"strings"
	"time"
)

func buildHeaders(node *l8reflect.L8Node) []string {
	var headers []string
	for name, attr := range node.Attributes {
		if attr.IsStruct || attr.IsSlice || attr.IsMap {
			continue
		}
		headers = append(headers, name)
	}
	sort.Strings(headers)
	return headers
}

func extractRow(item interface{}, headers []string, node *l8reflect.L8Node) []string {
	row := make([]string, len(headers))
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return row
	}

	for i, header := range headers {
		attr := node.Attributes[header]
		if attr == nil {
			continue
		}
		fieldName := attr.FieldName
		if fieldName == "" {
			fieldName = header
		}
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}
		row[i] = formatValue(field.Interface(), header)
	}
	return row
}

func formatValue(val interface{}, fieldName string) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case int32:
		if v == 0 {
			return ""
		}
		if isTimestampField(fieldName) {
			return time.Unix(int64(v), 0).Format("2006-01-02")
		}
		return fmt.Sprintf("%d", v)
	case int64:
		if v == 0 {
			return ""
		}
		if isTimestampField(fieldName) {
			return time.Unix(v, 0).Format("2006-01-02")
		}
		return fmt.Sprintf("%d", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case uint32:
		if v == 0 {
			return ""
		}
		return fmt.Sprintf("%d", v)
	case uint64:
		if v == 0 {
			return ""
		}
		return fmt.Sprintf("%d", v)
	default:
		s := fmt.Sprintf("%v", v)
		if s == "<nil>" {
			return ""
		}
		return s
	}
}

func isTimestampField(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, "date") ||
		strings.HasSuffix(lower, "time") ||
		strings.HasSuffix(lower, "at") ||
		strings.Contains(lower, "timestamp")
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

func buildCSV(headers []string, rows [][]string) string {
	var sb strings.Builder
	for i, h := range headers {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(escapeCSV(h))
	}
	sb.WriteByte('\n')

	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(escapeCSV(cell))
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}
