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

package dataimport

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// ParsedRow represents one row of imported data with column→value mapping.
type ParsedRow struct {
	RowNumber int
	Values    map[string]string
}

// ParsedFile represents a parsed import file with headers and rows.
type ParsedFile struct {
	Headers []string
	Rows    []ParsedRow
}

// ParseFile parses file data into a uniform ParsedFile structure.
// Format is detected from the format parameter, then file extension, then defaults to CSV.
func ParseFile(data []byte, format string, fileName string) (*ParsedFile, error) {
	if format == "" {
		format = detectFormat(fileName)
	}
	switch strings.ToLower(format) {
	case "csv", "tsv":
		return parseCSV(data, format)
	case "json":
		return parseJSON(data)
	case "xml":
		return parseXML(data)
	default:
		return parseCSV(data, "csv")
	}
}

func detectFormat(fileName string) string {
	lower := strings.ToLower(fileName)
	switch {
	case strings.HasSuffix(lower, ".json"):
		return "json"
	case strings.HasSuffix(lower, ".xml"):
		return "xml"
	case strings.HasSuffix(lower, ".tsv"):
		return "tsv"
	default:
		return "csv"
	}
}

func parseCSV(data []byte, format string) (*ParsedFile, error) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	if format == "tsv" {
		reader.Comma = '\t'
	}
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, errors.New("CSV parse error: " + err.Error())
	}
	if len(records) < 1 {
		return nil, errors.New("CSV file is empty")
	}

	headers := records[0]
	rows := make([]ParsedRow, 0, len(records)-1)
	for i := 1; i < len(records); i++ {
		values := make(map[string]string, len(headers))
		for j, header := range headers {
			if j < len(records[i]) {
				values[header] = records[i][j]
			}
		}
		rows = append(rows, ParsedRow{RowNumber: i, Values: values})
	}
	return &ParsedFile{Headers: headers, Rows: rows}, nil
}

func parseJSON(data []byte) (*ParsedFile, error) {
	var records []map[string]interface{}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, errors.New("JSON parse error: " + err.Error())
	}
	if len(records) == 0 {
		return nil, errors.New("JSON array is empty")
	}

	// Extract headers from first record
	headerSet := make(map[string]bool)
	var headers []string
	for key := range records[0] {
		if !headerSet[key] {
			headers = append(headers, key)
			headerSet[key] = true
		}
	}

	rows := make([]ParsedRow, 0, len(records))
	for i, record := range records {
		values := make(map[string]string, len(headers))
		for key, val := range record {
			if val != nil {
				values[key] = jsonValueToString(val)
			}
		}
		rows = append(rows, ParsedRow{RowNumber: i + 1, Values: values})
	}
	return &ParsedFile{Headers: headers, Rows: rows}, nil
}

func jsonValueToString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

type xmlRows struct {
	Rows []xmlRow `xml:"row"`
}

type xmlRow struct {
	Fields []xmlField `xml:",any"`
}

type xmlField struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

func parseXML(data []byte) (*ParsedFile, error) {
	var doc xmlRows
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, errors.New("XML parse error: " + err.Error())
	}
	if len(doc.Rows) == 0 {
		return nil, errors.New("XML has no rows")
	}

	// Extract headers from first row
	headerSet := make(map[string]bool)
	var headers []string
	for _, field := range doc.Rows[0].Fields {
		name := field.XMLName.Local
		if !headerSet[name] {
			headers = append(headers, name)
			headerSet[name] = true
		}
	}

	rows := make([]ParsedRow, 0, len(doc.Rows))
	for i, row := range doc.Rows {
		values := make(map[string]string, len(headers))
		for _, field := range row.Fields {
			values[field.XMLName.Local] = field.Value
		}
		rows = append(rows, ParsedRow{RowNumber: i + 1, Values: values})
	}
	return &ParsedFile{Headers: headers, Rows: rows}, nil
}
