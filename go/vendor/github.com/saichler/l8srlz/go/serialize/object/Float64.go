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
package object

import (
	"encoding/binary"
	"math"
)

// addFloat64 serializes a 64-bit floating point number as 8 bytes.
// Uses IEEE 754 binary representation via math.Float64bits.
func addFloat64(f float64, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 8)
	i := math.Float64bits(f)
	loc := *location
	binary.BigEndian.PutUint64((*data)[loc:loc+8], i)
	*location += 8
}

// getFloat64 deserializes 8 bytes as a 64-bit floating point number.
// Uses IEEE 754 binary representation via math.Float64frombits.
func getFloat64(data *[]byte, location *int) float64 {
	loc := *location
	result := binary.BigEndian.Uint64((*data)[loc : loc+8])
	*location += 8
	return math.Float64frombits(result)
}
