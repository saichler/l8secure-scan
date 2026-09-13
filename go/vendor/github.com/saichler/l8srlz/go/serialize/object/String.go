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

// addString serializes a string by first writing its length as int32,
// then copying the string bytes. This length-prefixed format allows
// efficient deserialization without null terminators.
func addString(str string, data *[]byte, location *int) {
	addInt32(int32(len(str)), data, location)
	checkAndEnlarge(data, location, len(str))
	copy((*data)[*location:*location+len(str)], str)
	*location += len(str)
}

// getString deserializes a string by first reading its length,
// then extracting the string bytes from the buffer.
func getString(data *[]byte, location *int) string {
	l := getInt32(data, location)
	size := int(l)
	s := string((*data)[*location : *location+size])
	*location += size
	return s
}
