// © 2025 Sharon Aicler (saichler@gmail.com)
//
// Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package strings

// escapeString escapes backslashes first, then each character in specialChars with a backslash prefix.
func escapeString(s string, specialChars string) string {
	result := New("")
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '\\' {
			result.Add("\\\\")
			continue
		}
		escaped := false
		for j := 0; j < len(specialChars); j++ {
			if ch == specialChars[j] {
				result.Add("\\")
				result.Add(string(ch))
				escaped = true
				break
			}
		}
		if !escaped {
			result.Add(string(ch))
		}
	}
	return result.String()
}

// unescapeString reverses all \X escape sequences back to X.
func unescapeString(s string) string {
	result := New("")
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			result.Add(string(s[i]))
		} else {
			result.Add(string(s[i]))
		}
	}
	return result.String()
}

// splitOnUnescaped splits s on unescaped occurrences of delim.
func splitOnUnescaped(s string, delim byte) []string {
	var parts []string
	current := New("")
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			current.Add(string(s[i]))
			i++
			current.Add(string(s[i]))
		} else if s[i] == delim {
			parts = append(parts, current.String())
			current = New("")
		} else {
			current.Add(string(s[i]))
		}
	}
	parts = append(parts, current.String())
	return parts
}

// indexOfUnescaped returns the index of the first unescaped occurrence of ch, or -1.
func indexOfUnescaped(s string, ch byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			continue
		}
		if s[i] == ch {
			return i
		}
	}
	return -1
}
