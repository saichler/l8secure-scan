/*
 * © 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package logs provides the log collection agent functionality for L8LogFusion.
// It implements real-time log file monitoring, tailing, and transmission to
// the central log server over the Layer 8 virtual network overlay.
//
// The package supports:
//   - Single file monitoring with specific filename
//   - Wildcard monitoring of all .log and .err files in a directory tree
//   - Automatic file rotation detection and handling
//   - Intelligent buffering and batch transmission
//   - Crash recovery with resume from last read position
package logs

import (
	"fmt"
	common2 "github.com/saichler/probler/go/prob/common"
	"os"
	"time"

	"github.com/saichler/l8logfusion/go/agent/common"
	"github.com/saichler/l8logfusion/go/types/l8logf"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	"github.com/saichler/l8utils/go/utils/strings"
)

// LogCollector is the main agent component responsible for monitoring and collecting
// log files from a configured path. It tails log files in real-time and transmits
// new log entries to the central log server via the Layer 8 virtual network interface.
type LogCollector struct {
	// logConfig specifies the path and filename pattern to monitor
	logConfig *l8logf.L8LogConfig
	// vnic is the virtual network interface for communicating with the log server
	vnic ifs.IVNic
	// tracker keeps track of which files are already being tailed
	tracker *FileTracker
}

// NewLogCollector creates a new LogCollector instance with the specified configuration.
//
// Parameters:
//   - logConfig: Configuration specifying the log path and filename (or "*" for wildcard)
//   - vnic: Virtual network interface for transmitting logs to the server
//
// Returns:
//   - *LogCollector: A configured collector ready to start monitoring
func NewLogCollector(logConfig *l8logf.L8LogConfig, vnic ifs.IVNic) *LogCollector {
	return &LogCollector{logConfig: logConfig, vnic: vnic, tracker: NewFileTracker()}
}

// collect scans a directory tree for log files and starts a TailFile goroutine
// for each .log or .err file found. Each file is tracked to avoid duplicate tailing.
// The location parameter controls where tailing starts (0 = beginning, -1 = end).
func (this *LogCollector) collect(path, name string, location int64) {
	files := ScanDir(path)
	for _, fullPath := range files {
		if this.tracker.IsTailed(fullPath) {
			continue
		}
		this.tracker.MarkTailed(fullPath)
		go func(fp string) {
			err := TailFile(fp, this.vnic, location)
			if err != nil {
				SendLogs(name, this.vnic, err.Error())
			}
		}(fullPath)
	}
}

// Collect starts the log collection process based on the configured path and filename.
// If the filename is "*", it recursively collects all .log and .err files in the directory.
// Otherwise, it monitors the specific file, waiting for it to exist if necessary.
//
// The method blocks until interrupted by a system signal when using wildcard collection,
// or until an error occurs when monitoring a specific file.
func (this *LogCollector) Collect() {
	name := strings.New("agent-", this.logConfig.Path).String()
	if this.logConfig.Name == "*" {
		this.collect(this.logConfig.Path, name, 0)
		go this.rescanLoop(name)
		common2.WaitForSignal(this.vnic.Resources())
		return
	}

	fullPath := strings.New(this.logConfig.Path, "/", this.logConfig.Name).String()
	_, err := os.Stat(fullPath)
	for err != nil {
		fmt.Println("File '", fullPath, " does not exist ")
		time.Sleep(time.Second)
		_, err = os.Stat(fullPath)
	}
	time.Sleep(time.Second)
	err = TailFile(fullPath, this.vnic, 0)
	if err != nil {
		SendLogs(name, this.vnic, err.Error())
	}
}

// rescanLoop periodically checks for new active log files in the monitored directory.
// New files discovered are tailed from the end (location -1) to avoid replaying old data.
func (this *LogCollector) rescanLoop(name string) {
	for {
		time.Sleep(30 * time.Second)
		newFiles := this.tracker.NewActiveFiles(this.logConfig.Path)
		for _, fp := range newFiles {
			this.tracker.MarkTailed(fp)
			go func(path string) {
				err := TailFile(path, this.vnic, -1)
				if err != nil {
					SendLogs(name, this.vnic, err.Error())
				}
			}(fp)
		}
	}
}

// SendLogs transmits a batch of log entries to the central log server via unicast.
// It packages the logs with source identification (IP address and filename) and
// sends them to the configured remote log service.
//
// Parameters:
//   - filename: The name of the source log file
//   - nic: The virtual network interface for transmission
//   - logs: Variable number of log line strings to send
//
// The source IP is determined from the NODE_IP environment variable, falling back
// to the machine's detected IP address if not set.
func SendLogs(filename string, nic ifs.IVNic, logs ...string) {
	logF := &l8logf.L8LogF{}
	logF.SourceIp = os.Getenv("NODE_IP")
	if logF.SourceIp == "" {
		logF.SourceIp = ipsegment.MachineIP
	}
	logF.Filename = filename
	logF.Logs = logs
	nic.Unicast(nic.Resources().SysConfig().RemoteUuid, common.LogServiceName, common.LogServiceArea, ifs.POST, logF)
}
