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

// addByte serializes a single byte value.
// It ensures the buffer has capacity for the byte before writing.
func addByte(b byte, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 1)
	(*data)[*location] = b
	*location++
}

// getByte deserializes a single byte value from the buffer.
func getByte(data *[]byte, location *int) byte {
	b := (*data)[*location]
	*location++
	return b
}
