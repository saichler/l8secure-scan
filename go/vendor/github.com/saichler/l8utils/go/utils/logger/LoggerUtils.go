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
	"bytes"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/saichler/l8types/go/ifs"
	strings2 "github.com/saichler/l8utils/go/utils/strings"
)

func FormatLog(level ifs.LogLevel, t int64, args ...interface{}) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = LogTimeFormat(t, level) + "<panic formatting log: corrupted argument>"
		}
	}()
	str := strings2.New()
	str.Add(LogTimeFormat(t, level))
	if args != nil {
		for _, arg := range args {
			str.Add(str.StringOf(arg))
		}
	}
	return str.String()
}

func LogTimeFormat(epochSeconds int64, level ifs.LogLevel) string {
	t := time.Unix(epochSeconds, 0)
	buff := bytes.Buffer{}
	buff.WriteString(strconv.Itoa(t.Year()))
	buff.WriteString("-")
	buff.WriteString(intToString(int(t.Month())))
	buff.WriteString("-")
	buff.WriteString(intToString(t.Day()))
	buff.WriteString(" ")
	buff.WriteString(intToString(t.Hour()))
	buff.WriteString(":")
	buff.WriteString(intToString(t.Minute()))
	buff.WriteString(":")
	buff.WriteString(intToString(t.Second()))
	buff.WriteString(" ")
	buff.WriteString(level.String())
	buff.WriteString(" - ")
	return buff.String()
}

func intToString(i int) string {
	s := strconv.Itoa(i)
	if i < 10 {
		buff := bytes.Buffer{}
		buff.WriteString("0")
		buff.WriteString(s)
		return buff.String()
	}
	return s
}

func FileAndLine(path string, trimPath bool) string {
	filename, line := findFileAndLine(path, trimPath)
	buff := bytes.Buffer{}
	buff.WriteString(" (")
	buff.WriteString(filename)
	buff.WriteString(".")
	buff.WriteString(strconv.Itoa(line))
	buff.WriteString(")")
	return buff.String()
}

func findFileAndLine(path string, trimPath bool) (string, int) {
	index := 2
	ok := true
	filename := "/Unknown"
	line := -1
	for ok == true {
		_, filename, line, ok = runtime.Caller(index)
		if strings.Contains(filename, path) &&
			!strings.Contains(filename, "logger") &&
			!strings.Contains(filename, "Logger") {
			break
		}
		index++
	}
	if trimPath {
		index = strings.LastIndex(filename, "/")
		return filename[index+1:], line
	}
	index = strings.LastIndex(filename, "github")
	if index != -1 {
		return filename[index:], line
	}
	return filename, line
}

func WaitForSignal(resources ifs.IResources) {
	resources.Logger().Info(resources.SysConfig().LocalAlias, ", Waiting for os signal...")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	resources.Logger().Info(resources.SysConfig().LocalAlias, ", End signal received! ", sig)
}
