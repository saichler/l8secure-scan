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

package logserver

import (
	"fmt"
	"os"
	"sync"

	"github.com/saichler/l8logfusion/go/agent/common"
	"github.com/saichler/l8logfusion/go/types/l8logf"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/web"
)

// LogService is the central service handler for receiving, persisting, and querying logs.
// It implements the ifs.IServiceHandler interface to integrate with the Layer 8 service framework.
//
// The service manages:
//   - A cache of open file handles for efficient log writing
//   - Thread-safe access to shared resources via mutex
//   - Persistent storage organized by source IP and filename
type LogService struct {
	// files is a cache of open file handles keyed by full file path
	files map[string]*os.File
	// mtx protects concurrent access to the files map
	mtx *sync.Mutex
	// dbLocation is the root directory for log storage (default: /data/logdb)
	dbLocation string
}

// NewLogService creates a new uninitialized LogService instance.
// The service must be activated via Activate() before use.
func NewLogService() *LogService {
	return &LogService{}
}

// Activate initializes the log service with the provided configuration.
// It sets up the file handle cache, creates the database directory, and
// registers the necessary protocol buffer types with the registry.
//
// Parameters:
//   - sla: Service level agreement containing the database location path
//   - vnic: Virtual network interface for registration
//
// Returns:
//   - error: Any error encountered during directory creation
func (this *LogService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8logf.L8LogF{})
	this.files = make(map[string]*os.File)
	this.dbLocation = sla.Args()[0].(string)
	this.mtx = &sync.Mutex{}
	err := os.MkdirAll(this.dbLocation, 0755)
	if err != nil {
		panic(err)
	}
	vnic.Resources().Introspector().Decorators().AddPrimaryKeyDecorator(&l8logf.L8File{}, "Path", "Name")
	return err
}

// DeActivate gracefully shuts down the log service.
// Currently performs no cleanup as file handles are managed by the OS.
func (this *LogService) DeActivate() error {
	return nil
}

// Post handles incoming log entries from collector agents.
// It writes each log entry to the appropriate file based on source IP and filename.
//
// Parameters:
//   - elements: Collection of L8LogF entries to persist
//   - vnic: Virtual network interface for logging errors
//
// Returns:
//   - ifs.IElements: Always returns nil (fire-and-forget pattern)
func (this *LogService) Post(elements ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	for _, elem := range elements.Elements() {
		l := elem.(*l8logf.L8LogF)
		f, e := this.fetchFile(l)
		if e != nil {
			vnic.Resources().Logger().Error(e.Error())
			continue
		}
		bts := make([]byte, 0)
		for _, msg := range l.Logs {
			bts = append(bts, msg...)
		}
		n, e := f.Write(bts)
		if e != nil {
			vnic.Resources().Logger().Error(e.Error())
			continue
		}
		if n != len(bts) {
			vnic.Resources().Logger().Error("Written bytes size mismatch ", n, "!=", len(bts))
		}
	}
	return nil
}

// Put handles update requests (not implemented for log service).
func (this *LogService) Put(elements ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Patch handles partial update requests (not implemented for log service).
func (this *LogService) Patch(elements ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Delete handles deletion requests (not implemented for log service).
func (this *LogService) Delete(elements ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Merge combines query results from multiple service instances.
// Used when querying logs across distributed log servers.
//
// Parameters:
//   - results: Map of service instance ID to query results
//
// Returns:
//   - ifs.IElements: Combined L8File containing all matching files and data
func (this *LogService) Merge(results map[string]ifs.IElements) ifs.IElements {
	fmt.Println("Merge Log files called with ", len(results))
	result := &l8logf.L8File{}
	result.Files = make([]*l8logf.L8File, 0)
	for _, elems := range results {
		for _, elem := range elems.Elements() {
			l := elem.(*l8logf.L8File)
			if l.Files != nil {
				for _, file := range l.Files {
					result.Files = append(result.Files, file)
				}
			}
			if l.Data != nil && l.Data.Content != "" {
				result.Data = l.Data
			}
		}
	}
	return object.New(nil, result)
}

// Get handles log query requests from the web UI or API clients.
// It supports two modes:
//   - Directory listing: Returns the file tree structure when path is "*" or no query
//   - File content: Returns paginated file content when path and name are specified
//
// Parameters:
//   - elements: Query elements containing search parameters
//   - vnic: Virtual network interface for accessing resources
//
// Returns:
//   - ifs.IElements: L8File containing directory listing or file content
func (this *LogService) Get(elements ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	fmt.Println("Get Log files called with ", len(elements.Elements()))
	q, err := elements.Query(vnic.Resources())
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return object.NewError(err.Error())
	}
	if q == nil {
		fmt.Println("Query is nill ")
		l8file := common.FileOf(this.dbLocation)
		return object.New(nil, l8file)
	}
	if q.ValueForParameter("path") == "*" {
		fmt.Println("Path is *")
		l8file := common.FileOf(this.dbLocation)
		return object.New(nil, l8file)
	}
	fmt.Println("Load Data case ", q.ValueForParameter("path"))
	resp, err := LoadData(q)
	return object.New(err, resp)

}

// Failed handles failed message delivery notifications (not implemented).
func (this *LogService) Failed(elements ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns the transaction configuration (not used for log service).
func (this *LogService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

// WebService returns the REST API configuration for the log service.
// It exposes a GET endpoint for querying logs via the web interface.
//
// Returns:
//   - ifs.IWebService: Web service configuration with log query endpoints
func (this *LogService) WebService() ifs.IWebService {
	ws := web.New(common.LogServiceName, common.LogServiceArea, 0)
	ws.AddEndpoint(&l8api.L8Query{}, ifs.GET, &l8logf.L8File{})
	return ws
}
