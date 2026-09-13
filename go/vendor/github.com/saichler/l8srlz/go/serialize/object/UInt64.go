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

import "encoding/binary"

// addUInt64 serializes a 64-bit unsigned integer as 8 bytes using big-endian encoding.
func addUInt64(i uint64, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 8)
	binary.BigEndian.PutUint64((*data)[*location:], i)
	*location += 8
}

// getUInt64 deserializes 8 bytes as a 64-bit unsigned integer using big-endian decoding.
func getUInt64(data *[]byte, location *int) uint64 {
	result := binary.BigEndian.Uint64((*data)[*location:])
	*location += 8
	return result
}
