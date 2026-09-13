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

package protocol

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
)

// MessageLog enables or disables message type logging for debugging and monitoring.
var MessageLog bool = false

// MsgLog is the global message type logger instance used for tracking message statistics.
var MsgLog = newMessageTypeLog()
var started bool = false

// MessageTypeLog tracks message counts by service name, area, and action type.
// It provides statistics and CSV export capabilities for debugging message flow.
type MessageTypeLog struct {
	mtx   sync.Mutex
	msgs  map[string]int
	total int
}

// newMessageTypeLog creates a new MessageTypeLog instance with initialized maps.
func newMessageTypeLog() *MessageTypeLog {
	return &MessageTypeLog{msgs: make(map[string]int), mtx: sync.Mutex{}}
}

// AddLog records a message occurrence for the given service name, area, and action.
// If MessageLog is disabled, this function returns immediately without logging.
func (this *MessageTypeLog) AddLog(serviceName string, serviceArea byte, action ifs.Action) {
	if !MessageLog {
		return
	}
	key := strings.New(serviceName, serviceArea, action).String()
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.msgs[key]++
	if !started {
		started = true
		go this.log()
	}
	this.total++
}

// Print outputs all message type counts and the total to stdout.
func (this *MessageTypeLog) Print() {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	for k, v := range this.msgs {
		fmt.Println(k, " - ", v)
	}
	fmt.Println("Total - ", this.total)
}

// log continuously writes the current statistics to /tmp/log.csv every second.
func (this *MessageTypeLog) log() {
	for {
		os.WriteFile("/tmp/log.csv", this.CSV(), 0777)
		time.Sleep(time.Second)
	}
}

// CSV returns the current message statistics formatted as CSV bytes.
func (this *MessageTypeLog) CSV() []byte {
	str := strings.New()
	str.Add("\"Key\",\"Count\"\n")
	this.mtx.Lock()
	defer this.mtx.Unlock()
	for k, v := range this.msgs {
		str.Add("\"")
		str.Add(k)
		str.Add("\",")
		str.Add(strconv.Itoa(v))
		str.Add("\n")
	}
	str.Add("\"Total\",").Add(strconv.Itoa(this.total)).Add("\n")
	return str.Bytes()
}

// Total returns the total number of messages logged across all types.
func (this *MessageTypeLog) Total() int {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	return this.total
}
