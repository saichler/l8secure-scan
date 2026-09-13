/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ifs

const (
	SystemServiceGroup = "System"
)

// ServiceLevelAgreement (SLA) defines the configuration and behavior for a service.
// It specifies how a service handles data, transactions, replication, and web endpoints.
type ServiceLevelAgreement struct {
	// Core service configuration
	serviceHandlerInstance IServiceHandler  // The handler instance for this service
	serviceName            string           // Unique name identifying this service
	serviceArea            byte             // Service area/partition number
	stateful               bool             // True if service maintains state
	serviceGroup           string           // Group name for shared elections; blank = use serviceName
	callback               IServiceCallback // Lifecycle callbacks for before/after operations

	// Data configuration
	serviceItem     interface{}   // Prototype for single items
	serviceItemList interface{}   // Prototype for item lists
	initItems       []interface{} // Initial items to populate on activation
	primaryKeys     []string      // Primary key field names for unique identification
	uniqueKeys      []string      // Unique index field names
	nonuniqueKeys   []string      // Non-unique index field names for lookups
	alwaysOverwrite []string      // Fields that always overwrite (no merge)
	store           IStorage      // Persistent storage backend

	// Transaction and replication configuration
	voter            bool // True if this service participates in voting
	transactional    bool // True if operations are transactional
	replication      bool // True if data is replicated
	replicationCount int  // Number of replicas to maintain

	// Web service configuration
	webService IWebService   // Web service endpoints
	args       []interface{} // Additional arguments

	// Metadata functions for aggregation
	metadataFunc map[string]func(interface{}) (bool, string)
}

// NewServiceLevelAgreement creates a new SLA with the required configuration.
// Parameters:
//   - serviceHandlerInstance: The handler that processes requests for this service
//   - serviceName: Unique name for the service
//   - serviceArea: Partition/area number for the service
//   - stateful: Whether the service maintains state (caches data)
//   - callback: Optional lifecycle callbacks for before/after operations
func NewServiceLevelAgreement(serviceHandlerInstance IServiceHandler, serviceName string, serviceArea byte, stateful bool, callback IServiceCallback) *ServiceLevelAgreement {
	return &ServiceLevelAgreement{serviceHandlerInstance: serviceHandlerInstance, serviceName: serviceName, serviceArea: serviceArea, callback: callback, stateful: stateful}
}

// Getters and Setters
func (this *ServiceLevelAgreement) ServiceName() string {
	return this.serviceName
}

func (this *ServiceLevelAgreement) ServiceArea() byte {
	return this.serviceArea
}

func (this *ServiceLevelAgreement) Stateful() bool {
	return this.stateful
}

func (this *ServiceLevelAgreement) ServiceGroup() string {
	if this.serviceGroup == "" {
		return this.serviceName
	}
	return this.serviceGroup
}

func (this *ServiceLevelAgreement) SetServiceGroup(serviceGroup string) {
	this.serviceGroup = serviceGroup
}

func (this *ServiceLevelAgreement) ServiceHandlerInstance() IServiceHandler {
	return this.serviceHandlerInstance
}

func (this *ServiceLevelAgreement) Callback() IServiceCallback {
	return this.callback
}

func (this *ServiceLevelAgreement) ServiceItem() interface{} {
	return this.serviceItem
}

func (this *ServiceLevelAgreement) SetServiceItem(serviceItem interface{}) {
	this.serviceItem = serviceItem
}

func (this *ServiceLevelAgreement) ServiceItemList() interface{} {
	return this.serviceItemList
}

func (this *ServiceLevelAgreement) SetServiceItemList(serviceItemList interface{}) {
	this.serviceItemList = serviceItemList
}

func (this *ServiceLevelAgreement) InitItems() []interface{} {
	return this.initItems
}

func (this *ServiceLevelAgreement) SetInitItems(initItems []interface{}) {
	this.initItems = initItems
}

func (this *ServiceLevelAgreement) PrimaryKeys() []string {
	return this.primaryKeys
}

func (this *ServiceLevelAgreement) UniqueKeys() []string {
	return this.uniqueKeys
}

func (this *ServiceLevelAgreement) NonUniqueKeys() []string {
	return this.nonuniqueKeys
}

func (this *ServiceLevelAgreement) AlwaysOverwrite() []string {
	return this.alwaysOverwrite
}

func (this *ServiceLevelAgreement) SetPrimaryKeys(primaryKeys ...string) {
	this.primaryKeys = primaryKeys
}

func (this *ServiceLevelAgreement) SetUniqueKeys(uniqueKeys ...string) {
	this.uniqueKeys = uniqueKeys
}

func (this *ServiceLevelAgreement) SetNonUniqueKeys(nonuniqueKeys ...string) {
	this.nonuniqueKeys = nonuniqueKeys
}

func (this *ServiceLevelAgreement) SetAlwaysOverwrite(alwaysOverwrite ...string) {
	this.alwaysOverwrite = alwaysOverwrite
}

func (this *ServiceLevelAgreement) Store() IStorage {
	return this.store
}

func (this *ServiceLevelAgreement) SetStore(store IStorage) {
	this.store = store
}

func (this *ServiceLevelAgreement) Voter() bool {
	return this.voter
}

func (this *ServiceLevelAgreement) SetVoter(voter bool) {
	this.voter = voter
}

func (this *ServiceLevelAgreement) Transactional() bool {
	return this.transactional
}

func (this *ServiceLevelAgreement) SetTransactional(transactional bool) {
	this.transactional = transactional
}

func (this *ServiceLevelAgreement) Replication() bool {
	return this.replication
}

func (this *ServiceLevelAgreement) SetReplication(replication bool) {
	this.replication = replication
}

func (this *ServiceLevelAgreement) ReplicationCount() int {
	return this.replicationCount
}

func (this *ServiceLevelAgreement) SetReplicationCount(replicationCount int) {
	this.replicationCount = replicationCount
}

func (this *ServiceLevelAgreement) WebService() IWebService {
	return this.webService
}

func (this *ServiceLevelAgreement) SetWebService(webService IWebService) {
	this.webService = webService
}

func (this *ServiceLevelAgreement) Args() []interface{} {
	return this.args
}

func (this *ServiceLevelAgreement) SetArgs(args ...interface{}) {
	this.args = args
}

func (this *ServiceLevelAgreement) AddMetadataFunc(name string, f func(interface{}) (bool, string)) {
	if this.metadataFunc == nil {
		this.metadataFunc = make(map[string]func(interface{}) (bool, string))
	}
	this.metadataFunc[name] = f
}

func (this *ServiceLevelAgreement) MetadataFunc() map[string]func(interface{}) (bool, string) {
	return this.metadataFunc
}

// IServiceCallback provides lifecycle hooks for service operations.
// Callbacks can modify data, abort operations, or perform side effects.
type IServiceCallback interface {
	// Before is called before an operation is executed.
	// Returns: modified data, continue flag, error
	// If continue is false, the operation is aborted.
	Before(interface{}, Action, bool, IVNic) (interface{}, bool, error)
	// After is called after an operation completes successfully.
	// Returns: modified data, continue flag, error
	After(interface{}, Action, bool, IVNic) (interface{}, bool, error)
}
