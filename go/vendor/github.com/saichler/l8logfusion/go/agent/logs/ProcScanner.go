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

package logs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ActiveLogFiles walks /proc/[pid]/fd/ for all PIDs (except our own),
// resolves symlinks, and returns the set of .log/.err file paths under
// monitorDir that are currently open for writing by other processes.
func ActiveLogFiles(monitorDir string) map[string]bool {
	result := make(map[string]bool)
	selfPid := os.Getpid()

	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return result
	}

	for _, entry := range procEntries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == selfPid {
			continue
		}

		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			linkPath := filepath.Join(fdDir, fd.Name())
			target, err := os.Readlink(linkPath)
			if err != nil {
				continue
			}

			if !strings.HasPrefix(target, monitorDir) {
				continue
			}

			if !isLogFile(target) {
				continue
			}

			if isWriteMode(entry.Name(), fd.Name()) {
				result[target] = true
			}
		}
	}

	return result
}

// isWriteMode reads /proc/[pid]/fdinfo/[fd], parses the octal flags line,
// and checks the lowest 2 bits for O_WRONLY (1) or O_RDWR (2).
func isWriteMode(pid, fd string) bool {
	path := fmt.Sprintf("/proc/%s/fdinfo/%s", pid, fd)
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "flags:") {
			continue
		}

		flagStr := strings.TrimSpace(strings.TrimPrefix(line, "flags:"))
		flags, err := strconv.ParseInt(flagStr, 8, 64)
		if err != nil {
			return false
		}

		accessMode := flags & 0x3 // lowest 2 bits
		return accessMode == 1 || accessMode == 2 // O_WRONLY or O_RDWR
	}

	return false
}

// isLogFile returns true if the filename has a .log or .err suffix.
func isLogFile(filename string) bool {
	return strings.HasSuffix(filename, ".log") || strings.HasSuffix(filename, ".err")
}
