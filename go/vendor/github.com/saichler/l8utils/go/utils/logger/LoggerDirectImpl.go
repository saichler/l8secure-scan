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

package logger

import (
	"errors"
	"testing"
	"time"

	"github.com/saichler/l8types/go/ifs"
)

type LoggerDirectImpl struct {
	logMethods []ILogMethod
	logLevel   ifs.LogLevel
}

func NewLoggerDirectImpl(logMethods ...ILogMethod) *LoggerDirectImpl {
	logImpl := &LoggerDirectImpl{}
	logImpl.logMethods = logMethods
	return logImpl
}

func (loggerImpl *LoggerDirectImpl) processEntry(entry *LoggerEntry) {
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

func (loggerImpl *LoggerDirectImpl) Empty() bool {
	return true
}

func (loggerImpl *LoggerDirectImpl) Trace(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Trace_Level {
		return
	}
	loggerImpl.processEntry(newEntry(ifs.Trace_Level, anys...))
}

func (loggerImpl *LoggerDirectImpl) Debug(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Debug_Level {
		return
	}
	loggerImpl.processEntry(newEntry(ifs.Debug_Level, anys...))
}

func (loggerImpl *LoggerDirectImpl) Info(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Info_Level {
		return
	}
	loggerImpl.processEntry(newEntry(ifs.Info_Level, anys...))
}

func (loggerImpl *LoggerDirectImpl) Warning(anys ...interface{}) {
	if loggerImpl.logLevel > ifs.Warning_Level {
		return
	}
	loggerImpl.processEntry(newEntry(ifs.Warning_Level, anys...))
}

func (loggerImpl *LoggerDirectImpl) Error(anys ...interface{}) error {
	anys = append(anys, FileAndLine(".go", false))
	loggerImpl.processEntry(newEntry(ifs.Error_Level, anys...))
	err := FormatLog(ifs.Error_Level, time.Now().Unix(), anys...)
	return errors.New(err)
}

func (loggerImpl *LoggerDirectImpl) Fail(t interface{}, args ...interface{}) {
	args = append(args, FileAndLine("tests", true))
	loggerImpl.Error(args...)
	ts, ok := t.(*testing.T)
	if ok {
		ts.Fail()
	}
}

func (loggerImpl *LoggerDirectImpl) SetLogLevel(level ifs.LogLevel) {
	loggerImpl.logLevel = level
}
