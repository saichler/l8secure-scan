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

package common

import (
	"os"
	"path/filepath"

	"github.com/saichler/l8logfusion/go/types/l8logf"
)

// FileOf recursively scans a directory and builds a hierarchical L8File structure
// representing the file system tree. It creates an L8File object for each file and
// subdirectory found, preserving the directory structure for log file navigation.
//
// Parameters:
//   - path: The absolute or relative path to the directory to scan
//
// Returns:
//   - *l8logf.L8File: A tree structure representing the directory contents,
//     or nil if the directory cannot be read
//
// The returned L8File has IsDirectory set to true and contains a slice of child
// L8File objects representing files and subdirectories. Each child file stores
// its name and parent path for later retrieval.
func FileOf(path string) *l8logf.L8File {
	f := &l8logf.L8File{}
	f.IsDirectory = true
	f.Files = make([]*l8logf.L8File, 0)
	files, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	for _, file := range files {
		if file.IsDir() {
			subDir := FileOf(filepath.Join(path, file.Name()))
			if subDir == nil {
				continue
			}
			subDir.Name = file.Name()
			subDir.Path = path
			f.Files = append(f.Files, subDir)
		} else {
			sub := &l8logf.L8File{}
			sub.Name = file.Name()
			sub.Path = path
			f.Files = append(f.Files, sub)
		}
	}
	return f
}
