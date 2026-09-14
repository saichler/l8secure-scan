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
	"os"
	"path/filepath"
	"sync"
)

// FileTracker provides thread-safe tracking of which log files are already
// being tailed, and discovery of newly active files.
type FileTracker struct {
	tailed map[string]bool
	mu     sync.Mutex
}

// NewFileTracker creates a new FileTracker instance.
func NewFileTracker() *FileTracker {
	return &FileTracker{tailed: make(map[string]bool)}
}

// MarkTailed records a file path as currently being tailed.
func (ft *FileTracker) MarkTailed(path string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.tailed[path] = true
}

// IsTailed returns true if the file is already being tailed.
func (ft *FileTracker) IsTailed(path string) bool {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return ft.tailed[path]
}

// NewActiveFiles calls ActiveLogFiles for the given directory and returns
// only the paths that are not already being tailed.
func (ft *FileTracker) NewActiveFiles(monitorDir string) []string {
	active := ActiveLogFiles(monitorDir)

	ft.mu.Lock()
	defer ft.mu.Unlock()

	var newFiles []string
	for path := range active {
		if !ft.tailed[path] {
			newFiles = append(newFiles, path)
		}
	}
	return newFiles
}

// ScanDir recursively walks root and returns all .log/.err file paths found.
func ScanDir(root string) []string {
	var results []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		full := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			results = append(results, ScanDir(full)...)
		} else if isLogFile(entry.Name()) {
			results = append(results, full)
		}
	}
	return results
}
