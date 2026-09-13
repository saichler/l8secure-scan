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
package common

import (
	l8common "github.com/saichler/l8common/go/types/l8common"
)

// SumLineMoney sums a Money field across a slice of line items.
// The getter extracts a *Money from each item (caller must type-assert inside).
func SumLineMoney(lines []interface{}, getter func(interface{}) *l8common.Money) *l8common.Money {
	total := int64(0)
	currencyId := ""
	for _, line := range lines {
		if m := getter(line); m != nil {
			total += m.Amount
			if currencyId == "" {
				currencyId = m.CurrencyId
			}
		}
	}
	if currencyId == "" {
		return nil
	}
	return &l8common.Money{Amount: total, CurrencyId: currencyId}
}

// MoneyAdd returns a + b (same currency). Returns nil if both are nil.
func MoneyAdd(a, b *l8common.Money) *l8common.Money {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return &l8common.Money{Amount: a.Amount + b.Amount, CurrencyId: a.CurrencyId}
}

// MoneySubtract returns a - b (same currency). Returns nil if both are nil.
func MoneySubtract(a, b *l8common.Money) *l8common.Money {
	if a == nil && b == nil {
		return nil
	}
	if a == nil {
		return &l8common.Money{Amount: -b.Amount, CurrencyId: b.CurrencyId}
	}
	if b == nil {
		return a
	}
	return &l8common.Money{Amount: a.Amount - b.Amount, CurrencyId: a.CurrencyId}
}

// SumLineFloat64 sums a float64 field across a slice of line items.
func SumLineFloat64(lines []interface{}, getter func(interface{}) float64) float64 {
	total := float64(0)
	for _, line := range lines {
		total += getter(line)
	}
	return total
}

// SumLineInt64 sums an int64 field across a slice of line items.
func SumLineInt64(lines []interface{}, getter func(interface{}) int64) int64 {
	total := int64(0)
	for _, line := range lines {
		total += getter(line)
	}
	return total
}

// MoneyAmount returns the amount of a Money value, or 0 if nil.
func MoneyAmount(m *l8common.Money) int64 {
	if m == nil {
		return 0
	}
	return m.Amount
}

// MoneyIsZero returns true if the Money value is nil or has a zero amount.
func MoneyIsZero(m *l8common.Money) bool {
	return m == nil || m.Amount == 0
}
