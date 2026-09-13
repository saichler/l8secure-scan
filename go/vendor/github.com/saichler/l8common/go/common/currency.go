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

// ConvertMoney applies a pre-resolved exchange rate to a Money value.
// This is pure math — no service lookups. Callers resolve the rate first.
func ConvertMoney(amount *l8common.Money, rate float64, toCurrency string) *l8common.Money {
	if amount == nil || amount.Amount == 0 {
		return &l8common.Money{CurrencyId: toCurrency}
	}
	return &l8common.Money{
		Amount:     int64(float64(amount.Amount) * rate),
		CurrencyId: toCurrency,
	}
}
