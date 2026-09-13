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

// addInt serializes a platform-dependent int as 8 bytes (int64) using big-endian encoding.
// This ensures consistent serialization across 32-bit and 64-bit systems.
func addInt(i int, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 8)
	binary.BigEndian.PutUint64((*data)[*location:], uint64(i))
	*location += 8
}

// getInt deserializes 8 bytes as a platform-dependent int using big-endian decoding.
func getInt(data *[]byte, location *int) int {
	result := int(binary.BigEndian.Uint64((*data)[*location:]))
	*location += 8
	return result
}
