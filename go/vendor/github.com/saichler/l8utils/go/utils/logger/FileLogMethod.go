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
	"github.com/saichler/l8types/go/ifs"
	"os"
)

type FileLogMethod struct {
	filename string
	file     *os.File
}

func NewFileLogMethod(filename string) *FileLogMethod {
	return &FileLogMethod{filename: filename}
}

func (this *FileLogMethod) Log(level ifs.LogLevel, msg string) {
	if this.file == nil {
		_, err := os.Stat(this.filename)
		if err != nil {
			os.Create(this.filename)
		}
		f, e := os.OpenFile(this.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if e != nil {
			panic(e)
		}
		this.file = f
	}
	this.file.WriteString(msg)
	this.file.WriteString("\n")
}
