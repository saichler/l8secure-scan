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
)

// addInt32 serializes a 32-bit signed integer as 4 bytes using big-endian encoding.
// This is the most commonly used integer serializer, also used internally for
// type prefixes and length fields.
func addInt32(i int32, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 4)
	binary.BigEndian.PutUint32((*data)[*location:], uint32(i))
	*location += 4
}

// getInt32 deserializes 4 bytes as a 32-bit signed integer using big-endian decoding.
func getInt32(data *[]byte, location *int) int32 {
	result := int32(binary.BigEndian.Uint32((*data)[*location:]))
	*location += 4
	return result
}
