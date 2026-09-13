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
	"bytes"
	"fmt"
	"github.com/saichler/l8types/go/ifs"
	"reflect"
	"strings"
)

// Query2CountSql generates a SELECT COUNT(*) SQL string from a query.
// Applies the query's criteria as a WHERE clause if present.
func (this *Statement) Query2CountSql(query ifs.IQuery, typeName string) string {
	buff := bytes.Buffer{}
	buff.WriteString("SELECT COUNT(*) FROM ")
	buff.WriteString(typeName)

	if query != nil && query.Criteria() != nil && typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" WHERE ")
			buff.WriteString(str)
		}
	}
	return buff.String()
}

// Query2Sql generates a SELECT SQL string from a query object.
// It handles column projections, aggregate functions, WHERE criteria,
// GROUP BY, HAVING, ORDER BY, LIMIT, and OFFSET clauses.
// Returns false if the query doesn't select any columns for this table.
func (this *Statement) Query2Sql(query ifs.IQuery, typeName string) (string, bool) {
	// Delegate to aggregate SQL builder when aggregate functions are present
	if len(query.Aggregates()) > 0 {
		return this.query2AggregateSql(query, typeName)
	}

	buff := bytes.Buffer{}
	if query.Properties() == nil || len(query.Properties()) == 0 {
		buff.WriteString("Select ")
		if this.fields == nil {
			this.fields, this.values = fieldsOf(this.node)
		}
		first := true
		for _, fieldName := range this.fields {
			if !first {
				buff.WriteString(",")
			}
			first = false
			buff.WriteString(fieldName)
		}
	} else {
		buff.WriteString("Select ParentKey,RecKey")
		this.fields = []string{"ParentKey", "RecKey"}
		for _, prop := range query.Properties() {
			buff.WriteString(",")
			if prop.Node().Parent.TypeName == typeName {
				this.fields = append(this.fields, prop.Node().FieldName)
				buff.WriteString(prop.Node().FieldName)
			}
		}
		if len(this.fields) == 2 {
			return "", false
		}
	}
	buff.WriteString(" from ")
	buff.WriteString(typeName)

	if query.Criteria() == nil {
		return buff.String(), true
	}

	if typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" where ")
			buff.WriteString(str)
		}

		// Add ORDER BY clause if SortBy is specified
		if query.SortBy() != "" {
			buff.WriteString(" ORDER BY ")
			buff.WriteString(query.SortBy())
			if query.Descending() {
				buff.WriteString(" DESC")
			} else {
				buff.WriteString(" ASC")
			}
		}

		// Add LIMIT clause if Limit is specified
		if query.Limit() > 0 {
			buff.WriteString(fmt.Sprintf(" LIMIT %d", query.Limit()))
		}

		// Add OFFSET clause for pagination (Page starts from 0)
		if query.Page() > 0 && query.Limit() > 0 {
			offset := query.Page() * query.Limit()
			buff.WriteString(fmt.Sprintf(" OFFSET %d", offset))
		}
	}
	return buff.String(), true
}

// AggregateSql generates an aggregate SQL string for the query's root type.
// This is the public entry point used by ReadAggregate.
func (this *Statement) AggregateSql(query ifs.IQuery) (string, bool) {
	return this.query2AggregateSql(query, query.RootType().TypeName)
}

// query2AggregateSql generates a SELECT SQL string for aggregate queries.
// It builds the SELECT clause with aggregate functions and group-by fields,
// then appends WHERE, GROUP BY, HAVING, ORDER BY, LIMIT, and OFFSET clauses.
func (this *Statement) query2AggregateSql(query ifs.IQuery, typeName string) (string, bool) {
	buff := bytes.Buffer{}
	buff.WriteString("Select ")

	first := true

	// Add group-by fields to SELECT
	for _, gb := range query.GroupBy() {
		if !first {
			buff.WriteString(",")
		}
		first = false
		buff.WriteString(gb)
	}

	// Add aggregate functions to SELECT
	for _, agg := range query.Aggregates() {
		if !first {
			buff.WriteString(",")
		}
		first = false
		buff.WriteString(strings.ToUpper(agg.Function))
		buff.WriteString("(")
		buff.WriteString(agg.Field)
		buff.WriteString(") AS ")
		buff.WriteString(agg.Alias)
	}

	buff.WriteString(" from ")
	buff.WriteString(typeName)

	// Add WHERE clause
	if query.Criteria() != nil && typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" where ")
			buff.WriteString(str)
		}
	}

	// Add GROUP BY clause
	if len(query.GroupBy()) > 0 {
		buff.WriteString(" GROUP BY ")
		for i, gb := range query.GroupBy() {
			if i > 0 {
				buff.WriteString(",")
			}
			buff.WriteString(gb)
		}
	}

	// Add HAVING clause
	if query.Having() != nil && typeName == query.RootType().TypeName {
		ok, str := expression(query.Having(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" HAVING ")
			buff.WriteString(str)
		}
	}

	// Add ORDER BY clause
	if query.SortBy() != "" {
		buff.WriteString(" ORDER BY ")
		buff.WriteString(query.SortBy())
		if query.Descending() {
			buff.WriteString(" DESC")
		} else {
			buff.WriteString(" ASC")
		}
	}

	// Add LIMIT clause
	if query.Limit() > 0 {
		buff.WriteString(fmt.Sprintf(" LIMIT %d", query.Limit()))
	}

	// Add OFFSET clause
	if query.Page() > 0 && query.Limit() > 0 {
		offset := query.Page() * query.Limit()
		buff.WriteString(fmt.Sprintf(" OFFSET %d", offset))
	}

	return buff.String(), true
}

// expression converts an IExpression to a SQL WHERE clause fragment.
// It recursively processes the expression tree, combining conditions with operators.
func expression(exp ifs.IExpression, typeName string) (bool, string) {
	if isNil(exp) {
		return false, ""
	}

	buff := bytes.Buffer{}
	condOK, condStr := condition(exp.Condition(), typeName)
	if condOK {
		buff.WriteString("(")
		buff.WriteString(condStr)
	}

	nextOK, nextStr := expression(exp.Next(), typeName)
	if nextOK {
		if !condOK {
			buff.WriteString("(")
		}
		buff.WriteString(exp.Operator())
		buff.WriteString(nextStr)
	}
	if nextOK || condOK {
		buff.WriteString(")")
	}
	return condOK || nextOK, buff.String()
}

// condition converts an ICondition to a SQL condition string.
// It combines comparators with logical operators (AND/OR).
func condition(cond ifs.ICondition, typeName string) (bool, string) {
	if isNil(cond) {
		return false, ""
	}
	result := bytes.Buffer{}
	okCond, exp1 := comparator(cond.Comparator(), typeName)
	if okCond {
		result.WriteString(exp1)
	}
	okNext, exp2 := condition(cond.Next(), typeName)
	if okNext {
		result.WriteString(cond.Operator())
		result.WriteString(exp2)
	}
	return okCond || okNext, result.String()
}

// comparator converts an IComparator to a SQL comparison expression.
// It handles string quoting based on property types.
func comparator(comp ifs.IComparator, typeName string) (bool, string) {
	if isNil(comp) {
		return false, ""
	}
	leftOK := false
	leftString := false
	rightOK := false
	rightString := false

	if !isNil(comp.LeftProperty()) {
		if comp.LeftProperty().Node().Parent.TypeName == typeName {
			leftOK = true
			if comp.LeftProperty().IsString() {
				leftString = true
			}
		}
	}

	if !isNil(comp.RightProperty()) {
		if comp.RightProperty().Node().Parent.TypeName == typeName {
			rightOK = true
			if comp.RightProperty().IsString() {
				rightString = true
			}
		}
	}

	buff := bytes.Buffer{}
	if leftString && !rightString {
		buff.WriteString(comp.Left())
		rightValue := stripQuotes(comp.Right())
		convertedValue, hasWildcard := convertWildcard(rightValue)
		if hasWildcard && comp.Operator() == "=" {
			buff.WriteString(" LIKE ")
		} else {
			buff.WriteString(comp.Operator())
		}
		buff.WriteString("'")
		buff.WriteString(convertedValue)
		buff.WriteString("'")
	} else if !leftString && rightString {
		leftValue := stripQuotes(comp.Left())
		convertedValue, hasWildcard := convertWildcard(leftValue)
		buff.WriteString("'")
		buff.WriteString(convertedValue)
		buff.WriteString("'")
		if hasWildcard && comp.Operator() == "=" {
			buff.WriteString(" LIKE ")
		} else {
			buff.WriteString(comp.Operator())
		}
		buff.WriteString(comp.Right())
	} else {
		buff.WriteString(comp.Left())
		buff.WriteString(comp.Operator())
		buff.WriteString(comp.Right())
	}
	return leftOK || rightOK, buff.String()
}

// isNil checks if an interface value is nil, including nil interface values.
func isNil(any interface{}) bool {
	if any == nil {
		return true
	}
	return reflect.ValueOf(any).IsNil()
}

// stripQuotes removes surrounding single or double quotes from a string.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '"' && s[len(s)-1] == '"') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// convertWildcard checks if a value contains a wildcard (*) and converts it to SQL LIKE syntax (%).
// Returns the converted value and true if a wildcard was found, otherwise returns the original value and false.
func convertWildcard(value string) (string, bool) {
	if strings.Contains(value, "*") {
		return strings.ReplaceAll(value, "*", "%"), true
	}
	return value, false
}

// Query2RecKeysSql generates SQL to fetch only RecKeys without LIMIT/OFFSET.
// Used by the primary index cache to fetch all matching RecKeys for pagination.
// The full result set is cached, and individual pages are served from cache.
func (this *Statement) Query2RecKeysSql(query ifs.IQuery, typeName string) string {
	buff := bytes.Buffer{}
	buff.WriteString("SELECT RecKey FROM ")
	buff.WriteString(typeName)

	if query.Criteria() != nil && typeName == query.RootType().TypeName {
		ok, str := expression(query.Criteria(), query.RootType().TypeName)
		if ok {
			buff.WriteString(" WHERE ")
			buff.WriteString(str)
		}
	}

	// Add ORDER BY (always include, no LIMIT/OFFSET)
	if query.SortBy() != "" {
		buff.WriteString(" ORDER BY ")
		buff.WriteString(query.SortBy())
		if query.Descending() {
			buff.WriteString(" DESC")
		} else {
			buff.WriteString(" ASC")
		}
	}

	return buff.String()
}

// Query2SqlByRecKeys generates SQL to fetch rows by specific RecKeys.
// Used by the primary index to fetch full data for a page of cached RecKeys.
func (this *Statement) Query2SqlByRecKeys(typeName string, recKeys []string) string {
	buff := bytes.Buffer{}
	buff.WriteString("SELECT ")

	if this.fields == nil {
		this.fields, this.values = fieldsOf(this.node)
	}
	first := true
	for _, fieldName := range this.fields {
		if !first {
			buff.WriteString(",")
		}
		first = false
		buff.WriteString(fieldName)
	}

	buff.WriteString(" FROM ")
	buff.WriteString(typeName)
	buff.WriteString(" WHERE RecKey IN (")

	first = true
	for _, key := range recKeys {
		if !first {
			buff.WriteString(",")
		}
		first = false
		buff.WriteString("'")
		buff.WriteString(escapeSQL(key))
		buff.WriteString("'")
	}
	buff.WriteString(")")

	return buff.String()
}

// escapeSQL escapes single quotes in SQL string values by doubling them.
func escapeSQL(s string) string {
	result := bytes.Buffer{}
	for _, c := range s {
		if c == '\'' {
			result.WriteRune('\'')
		}
		result.WriteRune(c)
	}
	return result.String()
}
