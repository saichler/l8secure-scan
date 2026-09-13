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

// addFloat32 serializes a 32-bit floating point number as 4 bytes.
// Uses IEEE 754 binary representation via math.Float32bits.
func addFloat32(f float32, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 4)
	i := math.Float32bits(f)
	loc := *location
	binary.BigEndian.PutUint32((*data)[loc:loc+4], i)
	*location += 4
}

// getFloat32 deserializes 4 bytes as a 32-bit floating point number.
// Uses IEEE 754 binary representation via math.Float32frombits.
func getFloat32(data *[]byte, location *int) float32 {
	loc := *location
	result := binary.BigEndian.Uint32((*data)[loc : loc+4])
	*location += 4
	return math.Float32frombits(result)
}
