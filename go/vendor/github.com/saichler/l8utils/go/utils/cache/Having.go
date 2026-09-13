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
	"strconv"

	"github.com/saichler/l8types/go/ifs"
)

// filterByHaving filters grouped aggregate results by the HAVING clause.
// Returns only groups whose aggregate values satisfy the HAVING expression.
func filterByHaving(groups []map[string]interface{}, having ifs.IExpression) []map[string]interface{} {
	if having == nil {
		return groups
	}
	result := make([]map[string]interface{}, 0)
	for _, group := range groups {
		if matchHaving(group, having) {
			result = append(result, group)
		}
	}
	return result
}

// matchHaving evaluates a HAVING expression against a group's aggregate results.
// The expression tree is walked recursively, evaluating conditions against map values.
func matchHaving(group map[string]interface{}, expr ifs.IExpression) bool {
	if expr == nil {
		return true
	}

	condResult := true
	childResult := true
	nextResult := true

	isOr := expr.Operator() == "or"
	if isOr {
		condResult = false
		childResult = false
		nextResult = false
	}

	// Evaluate this node's condition
	if expr.Condition() != nil {
		condResult = matchHavingCondition(group, expr.Condition())
	}

	// Evaluate child (nested expression)
	if expr.Child() != nil {
		childResult = matchHaving(group, expr.Child())
	}

	// Evaluate next (chained expression)
	if expr.Next() != nil {
		nextResult = matchHaving(group, expr.Next())
	}

	if isOr {
		return condResult || childResult || nextResult
	}
	return condResult && childResult && nextResult
}

// matchHavingCondition evaluates a single HAVING condition chain against a group.
func matchHavingCondition(group map[string]interface{}, cond ifs.ICondition) bool {
	if cond == nil {
		return true
	}

	compResult := matchHavingComparator(group, cond.Comparator())

	if cond.Next() == nil {
		return compResult
	}

	nextResult := matchHavingCondition(group, cond.Next())

	if cond.Operator() == "or" {
		return compResult || nextResult
	}
	return compResult && nextResult
}

// matchHavingComparator evaluates a single comparison against a group's aggregate map.
// Left operand is looked up in the group map (aggregate alias), right is a literal value.
func matchHavingComparator(group map[string]interface{}, comp ifs.IComparator) bool {
	if comp == nil {
		return true
	}

	leftKey := comp.Left()
	leftVal, ok := group[leftKey]
	if !ok {
		return false
	}

	leftFloat, ok := ToFloat64(leftVal)
	if !ok {
		return false
	}

	rightFloat, err := strconv.ParseFloat(comp.Right(), 64)
	if err != nil {
		return false
	}

	switch comp.Operator() {
	case "=":
		return leftFloat == rightFloat
	case "!=":
		return leftFloat != rightFloat
	case ">":
		return leftFloat > rightFloat
	case "<":
		return leftFloat < rightFloat
	case ">=":
		return leftFloat >= rightFloat
	case "<=":
		return leftFloat <= rightFloat
	}
	return false
}
