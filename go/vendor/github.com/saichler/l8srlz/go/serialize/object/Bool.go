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

// addBool serializes a boolean value to a single byte (1 for true, 0 for false).
// It ensures the buffer has capacity for the byte before writing.
func addBool(b bool, data *[]byte, location *int) {
	checkAndEnlarge(data, location, 1)
	if b {
		(*data)[*location] = 1
	}
	*location++
}

// getBool deserializes a boolean value from a single byte.
// Returns true if the byte is 1, false otherwise.
func getBool(data *[]byte, location *int) bool {
	b := (*data)[*location]
	*location++
	if b == 1 {
		return true
	}
	return false
}
