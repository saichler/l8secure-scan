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

// Package logserver implements the central log aggregation server for L8LogFusion.
// It receives log entries from distributed collector agents, persists them to disk
// organized by source IP and filename, and provides query APIs for log retrieval.
//
// Key features:
//   - Receives logs via Layer 8 virtual network overlay
//   - Organizes logs by source IP and filename in /data/logdb
//   - Provides paginated log retrieval (5KB per page)
//   - File handle caching for optimized I/O performance
//   - REST API for web-based log browsing
package logserver

import (
	"io"
	"os"
	"path/filepath"
	strings2 "strings"

	"github.com/saichler/l8logfusion/go/agent/common"
	"github.com/saichler/l8logfusion/go/types/l8logf"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/strings"
)

// ActivateLogService registers and activates the log service with the virtual network.
// It creates a service level agreement with the configured database location
// and enables the service to receive log entries from collectors.
//
// Parameters:
//   - vnic: The virtual network interface to register the service with
//
// The service stores logs at /data/logdb by default.
func ActivateLogService(logDbPath string, vnic ifs.IVNic) {
	sla := ifs.NewServiceLevelAgreement(&LogService{}, common.LogServiceName, common.LogServiceArea, true, nil)
	sla.SetArgs(logDbPath)
	vnic.Resources().Services().Activate(sla, vnic)
}

// fetchFile retrieves or creates a file handle for writing log entries.
// It maintains a cache of open file handles to avoid repeated file operations.
// The file path is constructed from the database location, source IP, and filename.
//
// Parameters:
//   - lf: Log entry containing source IP and filename information
//
// Returns:
//   - *os.File: An open file handle for appending log entries
//   - error: Any error encountered during file creation or access
//
// Thread-safe: Uses mutex to protect the file handle cache.
func (this *LogService) fetchFile(lf *l8logf.L8LogF) (*os.File, error) {
	filename := strings.New(this.dbLocation, "/", lf.SourceIp, "/", lf.Filename).String()
	this.mtx.Lock()
	defer this.mtx.Unlock()
	f, ok := this.files[filename]
	if ok {
		return f, nil
	}
	index := strings2.LastIndex(filename, "/")
	path := filename[0:index]
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}
	f, err = os.Create(filename)
	if err != nil {
		return nil, err
	}
	this.files[filename] = f
	return f, nil
}

// Files returns the map of currently open file handles.
// This is primarily used for testing and monitoring purposes.
func (this *LogService) Files() map[string]*os.File {
	return this.files
}

// LogsService retrieves the active LogService instance from the resources.
// Returns nil if the service is not registered or activated.
//
// Parameters:
//   - r: The resources container holding registered services
//
// Returns:
//   - *LogService: The active log service instance, or nil if not found
func LogsService(r ifs.IResources) *LogService {
	sh, ok := r.Services().ServiceHandler(common.LogServiceName, common.LogServiceArea)
	if !ok {
		return nil
	}
	return sh.(*LogService)
}

// LoadData reads a page of log file content based on the query parameters.
// It implements KB-based pagination with 5KB per page for efficient retrieval
// of large log files.
//
// Parameters:
//   - q: Query containing path, name, and page parameters
//
// Returns:
//   - *l8logf.L8File: File metadata with the requested page content
//   - error: Any error encountered during file reading
//
// The returned L8File contains:
//   - Path and Name identifying the file
//   - Data.Page: Current page number
//   - Data.Size: Total file size in bytes
//   - Data.Content: Up to 5KB of file content for the requested page
func LoadData(q ifs.IQuery) (*l8logf.L8File, error) {
	path := q.ValueForParameter("path")
	name := q.ValueForParameter("name")
	l8file := &l8logf.L8File{}
	l8file.Path = path
	l8file.Name = name
	l8file.Data = &l8logf.L8FileData{}

	// KB-based paging: 5KB per page
	const bytesPerPage = 5120 // 5KB
	l8file.Data.Page = q.Page()

	// Get file info to determine total size
	filePath := filepath.Join(l8file.Path, l8file.Name)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return l8file, err
	}
	l8file.Data.Size = int32(fileInfo.Size())

	// Calculate byte offset based on page number
	offset := int64(l8file.Data.Page) * bytesPerPage

	// If offset is beyond file size, return empty content
	if offset >= fileInfo.Size() {
		l8file.Data.Content = ""
		return l8file, nil
	}

	// Open file and seek to the offset
	file, err := os.Open(filePath)
	if err != nil {
		return l8file, err
	}
	defer file.Close()

	_, err = file.Seek(offset, 0)
	if err != nil {
		return l8file, err
	}

	// Read up to bytesPerPage bytes
	buffer := make([]byte, bytesPerPage)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return l8file, err
	}

	// Set the actual content read
	l8file.Data.Content = string(buffer[:n])
	return l8file, nil
}
