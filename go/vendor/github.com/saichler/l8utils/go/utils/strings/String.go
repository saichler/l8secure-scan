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

// Package strings provides efficient string building and manipulation utilities.
// It wraps bytes.Buffer to provide a fluent API for string concatenation with
// automatic type conversion from various primitive types.
//
// Key features:
//   - Efficient string concatenation using bytes.Buffer
//   - Automatic conversion of primitives to strings (int, float, bool, etc.)
//   - Fluent API with method chaining support
//   - Optional space insertion between concatenated values
package strings

import (
	"bytes"
)

// String is a wrapper over buff.bytes to make it seamless to concatenate strings
type String struct {
	buff               *bytes.Buffer
	TypesPrefix        bool
	AddSpaceWhenAdding bool
}

// New construct a new String instance and initialize the buff with the input string
func New(anys ...interface{}) *String {
	s := &String{}
	s.init()
	if anys != nil {
		for _, any := range anys {
			s.Add(s.StringOf(any))
		}
	}
	return s
}

// init initialize the buff if needed
func (s *String) init() {
	if s.buff == nil {
		s.buff = &bytes.Buffer{}
	}
}

// Add concatenate a string to this String instance
func (s *String) Add(strs ...string) *String {
	s.init()
	if s.AddSpaceWhenAdding && len(s.Bytes()) > 0 {
		s.buff.WriteString(" ")
	}
	if strs != nil {
		for _, str := range strs {
			s.buff.WriteString(str)
		}
	}
	return s
}

// Join concatenate a String instance to this String instance
func (s *String) Join(other *String) *String {
	s.init()
	s.buff.Write(other.buff.Bytes())
	return s
}

// String convert the String instance buffer to primitive string
func (s *String) String() string {
	s.init()
	return s.buff.String()
}

// IsBlank return is this String instance is blank
func (s *String) IsBlank() bool {
	s.init()
	return s.buff.Len() == 0
}

// Len return the length of the current string
func (s *String) Len() int {
	s.init()
	return s.buff.Len()
}

// Bytes return the bytes of the string
func (s *String) Bytes() []byte {
	s.init()
	return s.buff.Bytes()
}

// AddBytes appends raw bytes to the string buffer.
func (s *String) AddBytes(bytes []byte) {
	s.init()
	s.buff.Write(bytes)
}
