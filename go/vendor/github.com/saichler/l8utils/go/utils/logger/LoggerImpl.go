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

// Package logger provides an asynchronous logging framework with configurable log levels
// and multiple output methods. It supports file-based logging, console output, and
// custom log method implementations through the ILogMethod interface.
//
// The logger uses an internal queue to process log entries asynchronously, preventing
// logging operations from blocking application code. It supports multiple log levels:
// Trace, Debug, Info, Warning, and Error.
//
// Key features:
//   - Asynchronous log processing via internal queue (max 50,000 entries)
//   - Configurable log levels for filtering output
//   - Multiple simultaneous log outputs (file, console, custom)
//   - Automatic source file and line number capture for errors
//   - Test failure integration via Fail method
package logger

import (
	"errors"
	"testing"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/queues"
)

// ILogMethod defines the interface for custom log output implementations.
// Implementations can write log messages to files, console, network, or any other destination.
type ILogMethod interface {
	Log(level ifs.LogLevel, msg string)
}

// LoggerImpl is the main logger implementation providing asynchronous logging
// with configurable levels and multiple output methods.
type LoggerImpl struct {
	queue      *queues.Queue
	logMethods []ILogMethod
	logLevel   ifs.LogLevel
}

// LoggerEntry represents a single log message with timestamp and level metadata.
type LoggerEntry struct {
	anys []interface{}
	t    int64
	l    ifs.LogLevel
}

// NewLoggerImpl creates a new asynchronous logger with the specified output methods.
// Starts a background goroutine to process log entries from the queue.
func NewLoggerImpl(logMethods ...ILogMethod) *LoggerImpl {
	logImpl := &LoggerImpl{}
	logImpl.logMethods = logMethods
	logImpl.queue = queues.NewQueue("Logger Queue", 50000)
	go logImpl.processQueue()
	return logImpl
}

func (loggerImpl *LoggerImpl) processQueue() {
	for {
		entry := loggerImpl.queue.Next().(*LoggerEntry)
		str := FormatLog(entry.l, entry.t, entry.anys...)
		if len(loggerImpl.logMethods) == 1 {
			loggerImpl.logMethods[0].Log(entry.l, str)
		} else if len(loggerImpl.logMethods) == 2 {
			loggerImpl.logMethods[0].Log(entry.l, str)
			loggerImpl.logMethods[1].Log(entry.l, str)
		} else if len(loggerImpl.logMethods) == 3 {
			loggerImpl.logMethods[0].Log(entry.l, str)
			loggerImpl.logMethods[1].Log(entry.l, str)
			loggerImpl.logMethods[2].Log(entry.l, str)
		}
	}
}

func newEntry(l ifs.LogLevel, anys ...interface{}) *LoggerEntry {
	return &LoggerEntry{
		t:    time.Now().Unix(),
		l:    l,
		anys: anys,
	}
}

// Empty returns true if there are no pending log entries in the queue.
func (loggerImpl *LoggerImpl) Empty() bool {
	return loggerImpl.queue.Size() == 0
}

// Trace logs a message at Trace level (most verbose). Filtered if log level is higher.
func (loggerImpl *LoggerImpl) Trace(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Trace_Level {
		return
	}
	loggerImpl.queue.Add(newEntry(ifs.Trace_Level, anys...))
}

// Debug logs a message at Debug level. Useful for development troubleshooting.
func (loggerImpl *LoggerImpl) Debug(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Debug_Level {
		return
	}
	loggerImpl.queue.Add(newEntry(ifs.Debug_Level, anys...))
}

// Info logs a message at Info level. Standard operational messages.
func (loggerImpl *LoggerImpl) Info(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Info_Level {
		return
	}
	loggerImpl.queue.Add(newEntry(ifs.Info_Level, anys...))
}

// Warning logs a message at Warning level. Potential issues that don't prevent operation.
func (loggerImpl *LoggerImpl) Warning(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Warning_Level {
		return
	}
	loggerImpl.queue.Add(newEntry(ifs.Warning_Level, anys...))
}

// Error logs a message at Error level with automatic file/line capture.
// Returns an error containing the formatted message for convenient error propagation.
func (loggerImpl *LoggerImpl) Error(anys ...interface{}) error {
	anys = append(anys, FileAndLine(".go", false))
	loggerImpl.queue.Add(newEntry(ifs.Error_Level, anys...))
	err := FormatLog(ifs.Error_Level, time.Now().Unix(), anys...)
	return errors.New(err)
}

// Fail logs an error and marks the test as failed. Designed for use in test code.
func (loggerImpl *LoggerImpl) Fail(t interface{}, args ...interface{}) {
	args = append(args, FileAndLine("tests", true))
	loggerImpl.Error(args...)
	ts, ok := t.(*testing.T)
	if ok {
		ts.Fail()
	}
}

// SetLogLevel sets the minimum log level. Messages below this level are filtered.
func (loggerImpl *LoggerImpl) SetLogLevel(level ifs.LogLevel) {
	loggerImpl.logLevel = level
}
