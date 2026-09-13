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

package ifs

import "bytes"

// LogLevel defines the severity of log messages.
type LogLevel int

const (
	Trace_Level   LogLevel = 1 // Most verbose, for detailed debugging
	Debug_Level   LogLevel = 2 // Debug information
	Info_Level    LogLevel = 3 // General information
	Warning_Level LogLevel = 4 // Warning conditions
	Error_Level   LogLevel = 5 // Error conditions
)

// String returns the string representation of a log level.
func (l LogLevel) String() string {
	switch l {
	case Trace_Level:
		return "(Trace)"
	case Debug_Level:
		return "(Debug)"
	case Info_Level:
		return "(Info) "
	case Warning_Level:
		return "(Warn )"
	case Error_Level:
		return "(Error)"
	}
	return ""
}

// ILogger defines the logging interface for the Layer 8 system.
type ILogger interface {
	// Trace logs at trace level (most verbose).
	Trace(...interface{})
	// Debug logs at debug level.
	Debug(...interface{})
	// Info logs at info level.
	Info(...interface{})
	// Warning logs at warning level.
	Warning(...interface{})
	// Error logs at error level and returns an error.
	Error(...interface{}) error
	// Empty returns true if no messages have been logged.
	Empty() bool
	// Fail logs a failure with context and panics.
	Fail(interface{}, ...interface{})
	// SetLogLevel sets the minimum log level to output.
	SetLogLevel(LogLevel)
}

func App(msg ...string) string {
	if msg != nil {
		buff := bytes.Buffer{}
		for _, l := range msg {
			buff.WriteString(l)
		}
		return buff.String()
	}
	return ""
}
