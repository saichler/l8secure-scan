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
	"fmt"
	"github.com/saichler/l8types/go/types/l8api"
	"strconv"
	"strings"
	"time"
)

// TransformValue applies all transforms for a given target field to a raw string value.
func TransformValue(raw string, transforms []*l8api.L8ImportValueTransform, fieldName string) (string, error) {
	result := raw
	for _, t := range transforms {
		if t.TargetField != fieldName {
			continue
		}
		var err error
		result, err = applyTransform(result, t)
		if err != nil {
			return "", fmt.Errorf("transform %s on field %s: %w",
				t.TransformType.String(), fieldName, err)
		}
	}
	return result, nil
}

func applyTransform(raw string, t *l8api.L8ImportValueTransform) (string, error) {
	switch t.TransformType {
	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_TRIM:
		return strings.TrimSpace(raw), nil

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_UPPERCASE:
		return strings.ToUpper(raw), nil

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_LOWERCASE:
		return strings.ToLower(raw), nil

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_DEFAULT:
		if strings.TrimSpace(raw) == "" {
			return t.DefaultValue, nil
		}
		return raw, nil

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_DATE_FORMAT:
		return transformDate(raw, t.FormatPattern)

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_ENUM_MAP:
		return transformEnum(raw, t.ValueMap)

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_UNIT_CONVERT:
		return transformUnit(raw, t.FormatPattern)

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_CONCATENATE:
		// Concatenation is handled at the row level, not per-value
		return raw, nil

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_SPLIT:
		return transformSplit(raw, t.ConcatenateSeparator, t.FormatPattern)

	case l8api.L8ImportTransformType_IMPORT_TRANSFORM_MONEY:
		return transformMoney(raw)

	default:
		return raw, nil
	}
}

// transformDate converts a date string to a Unix timestamp string.
func transformDate(raw string, pattern string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "0", nil
	}
	goFormat := datePatternToGo(pattern)
	t, err := time.Parse(goFormat, raw)
	if err != nil {
		return "", fmt.Errorf("cannot parse date '%s' with format '%s': %w", raw, pattern, err)
	}
	return fmt.Sprintf("%d", t.Unix()), nil
}

// datePatternToGo converts common date patterns to Go time format.
func datePatternToGo(pattern string) string {
	r := strings.NewReplacer(
		"YYYY", "2006", "yyyy", "2006",
		"YY", "06", "yy", "06",
		"MM", "01", "mm", "01",
		"DD", "02", "dd", "02",
		"HH", "15", "hh", "03",
		"mm", "04", // minute (context-dependent, after hour)
		"ss", "05",
	)
	return r.Replace(pattern)
}

// transformEnum maps a text label to its numeric enum value.
func transformEnum(raw string, valueMap map[string]string) (string, error) {
	if mapped, ok := valueMap[raw]; ok {
		return mapped, nil
	}
	// Try case-insensitive match
	lower := strings.ToLower(strings.TrimSpace(raw))
	for k, v := range valueMap {
		if strings.ToLower(strings.TrimSpace(k)) == lower {
			return v, nil
		}
	}
	return "", fmt.Errorf("no enum mapping for value '%s'", raw)
}

// transformUnit multiplies a numeric value by a factor (e.g., "100" for dollars→cents).
func transformUnit(raw string, factorStr string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "0", nil
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return "", fmt.Errorf("cannot parse number '%s': %w", raw, err)
	}
	factor, err := strconv.ParseFloat(factorStr, 64)
	if err != nil {
		return "", fmt.Errorf("invalid conversion factor '%s': %w", factorStr, err)
	}
	result := value * factor
	if result == float64(int64(result)) {
		return fmt.Sprintf("%d", int64(result)), nil
	}
	return fmt.Sprintf("%g", result), nil
}

// transformSplit splits a value by separator and returns the element at the specified index.
func transformSplit(raw string, separator string, indexStr string) (string, error) {
	if separator == "" {
		separator = ","
	}
	parts := strings.Split(raw, separator)
	idx, err := strconv.Atoi(indexStr)
	if err != nil {
		idx = 0
	}
	if idx >= 0 && idx < len(parts) {
		return strings.TrimSpace(parts[idx]), nil
	}
	return "", nil
}

// transformMoney parses a currency string like "$1,234.56" into cents as a string.
func transformMoney(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "0", nil
	}
	// Strip currency symbols and grouping separators
	cleaned := raw
	for _, ch := range []string{"$", "€", "£", "¥", ",", " "} {
		cleaned = strings.ReplaceAll(cleaned, ch, "")
	}
	cleaned = strings.TrimSpace(cleaned)
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return "", fmt.Errorf("cannot parse money '%s': %w", raw, err)
	}
	cents := int64(value * 100)
	return fmt.Sprintf("%d", cents), nil
}

// ApplyConcatenation handles CONCATENATE transforms at the row level.
func ApplyConcatenation(row map[string]string, transforms []*l8api.L8ImportValueTransform) {
	for _, t := range transforms {
		if t.TransformType != l8api.L8ImportTransformType_IMPORT_TRANSFORM_CONCATENATE {
			continue
		}
		sep := t.ConcatenateSeparator
		if sep == "" {
			sep = " "
		}
		var parts []string
		for _, field := range t.ConcatenateFields {
			if v, ok := row[field]; ok && v != "" {
				parts = append(parts, v)
			}
		}
		row[t.TargetField] = strings.Join(parts, sep)
	}
}
