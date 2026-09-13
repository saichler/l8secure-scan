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

import "time"

// Message represents a network message in the Layer 8 system.
// It contains routing information, payload data, and optional transaction metadata.
// Messages support various delivery modes (unicast, multicast, round-robin, etc.)
// and can be part of distributed transactions.
type Message struct {
	// Routing fields
	source        string        // UUID of the sending node
	vnet          string        // Virtual network identifier
	destination   string        // UUID of the target node (empty for multicast)
	serviceName   string        // Target service name
	serviceArea   byte          // Target service area/partition
	priority      Priority      // Message priority (P1-P8)
	multicastMode MulticastMode // Delivery mode (unicast, multicast, etc.)

	// Message metadata
	action      Action           // Action type (POST, PUT, PATCH, DELETE, GET, etc.)
	tr_state    TransactionState // Transaction state if transactional
	aaaId       string           // Authentication/Authorization/Audit ID
	sequence    uint32           // Message sequence number
	timeout     uint16           // Request timeout in milliseconds
	request     bool             // True if this is a request expecting response
	reply       bool             // True if this is a reply to a request
	failMessage string           // Error message if delivery failed
	data        []byte           // Payload data

	// Transaction fields (only used when tr_state != NotATransaction)
	tr_id        string // Transaction ID
	tr_errMsg    string // Transaction error message
	tr_created   int64  // Transaction creation timestamp (Unix seconds)
	tr_queued    int64  // Transaction queued timestamp
	tr_running   int64  // Transaction started running timestamp
	tr_end       int64  // Transaction completion timestamp
	tr_timeout   int64  // Transaction timeout value
	tr_replica   byte   // Replica number for replicated data
	tr_isReplica bool   // True if this is a replica operation
}

// Init initializes all fields of the message. Used for creating a fully-configured message.
func (this *Message) Init(destination, serviceName string, serviceArea byte,
	priority Priority, multicastMode MulticastMode, action Action, source, vnet string, data []byte,
	isRequest, isReply bool, msgNum uint32,
	tr_state TransactionState, tr_id, tr_errMsg string, tr_created, tr_queued, tr_running, tr_end, tr_timeout int64, tr_replica byte, tr_isreplica bool) {
	this.destination = destination
	this.serviceName = serviceName
	this.serviceArea = serviceArea
	this.priority = priority
	this.multicastMode = multicastMode
	this.action = action
	this.source = source
	this.vnet = vnet
	this.data = data
	this.request = isRequest
	this.reply = isReply
	this.sequence = msgNum
	this.tr_state = tr_state
	this.tr_id = tr_id
	this.tr_errMsg = tr_errMsg
	this.tr_created = tr_created
	this.tr_queued = tr_queued
	this.tr_running = tr_running
	this.tr_end = tr_end
	this.tr_timeout = tr_timeout
	this.tr_replica = tr_replica
	this.tr_isReplica = tr_isreplica
}

// Getters - Methods to retrieve message field values

// Source returns the UUID of the sending node.
func (this *Message) Source() string {
	return this.source
}

func (this *Message) Vnet() string {
	return this.vnet
}

func (this *Message) Destination() string {
	return this.destination
}

func (this *Message) ServiceName() string {
	return this.serviceName
}

func (this *Message) ServiceArea() byte {
	return this.serviceArea
}

func (this *Message) Sequence() uint32 {
	return this.sequence
}

func (this *Message) Priority() Priority {
	return this.priority
}

func (this *Message) MulticastMode() MulticastMode {
	return this.multicastMode
}

func (this *Message) Action() Action {
	return this.action
}

func (this *Message) Timeout() uint16 {
	return this.timeout
}

func (this *Message) Request() bool {
	return this.request
}

func (this *Message) Reply() bool {
	return this.reply
}

func (this *Message) FailMessage() string {
	return this.failMessage
}

func (this *Message) Data() []byte {
	return this.data
}

func (this *Message) AAAId() string {
	return this.aaaId
}

func (this *Message) Tr_State() TransactionState {
	return this.tr_state
}

func (this *Message) Tr_Id() string {
	return this.tr_id
}

func (this *Message) Tr_ErrMsg() string {
	return this.tr_errMsg
}

func (this *Message) Tr_Created() int64 {
	return this.tr_created
}

func (this *Message) Tr_Queued() int64 {
	return this.tr_queued
}

func (this *Message) Tr_Running() int64 {
	return this.tr_running
}

func (this *Message) Tr_End() int64 {
	return this.tr_end
}

func (this *Message) Tr_Timeout() int64 {
	return this.tr_timeout
}

func (this *Message) Tr_Replica() byte {
	return this.tr_replica
}

func (this *Message) Tr_IsReplica() bool {
	return this.tr_isReplica
}

// Setters - Methods to modify message field values

// SetSource sets the source UUID.
func (this *Message) SetSource(source string) {
	this.source = source
}

func (this *Message) SetVnet(vnet string) {
	this.vnet = vnet
}

func (this *Message) SetDestination(destination string) {
	this.destination = destination
}

func (this *Message) SetServiceName(serviceName string) {
	this.serviceName = serviceName
}

func (this *Message) SetServiceArea(serviceArea byte) {
	this.serviceArea = serviceArea
}

func (this *Message) SetSequence(sequence uint32) {
	this.sequence = sequence
}

func (this *Message) SetPriority(priority Priority) {
	this.priority = priority
}

func (this *Message) SetMulticastMode(multicastMode MulticastMode) {
	this.multicastMode = multicastMode
}

func (this *Message) SetAction(action Action) {
	this.action = action
}

func (this *Message) SetTimeout(timeout uint16) {
	this.timeout = timeout
}

func (this *Message) SetRequestReply(request, reply bool) {
	this.request = request
	this.reply = reply
}

func (this *Message) SetFailMessage(failMessage string) {
	this.failMessage = failMessage
}

func (this *Message) SetAAAId(aaaId string) {
	this.aaaId = aaaId
}

func (this *Message) SetData(data []byte) {
	this.data = data
}

func (this *Message) SetTr_State(trstate TransactionState) {
	this.tr_state = trstate
	switch trstate {
	case Created:
		this.tr_created = time.Now().Unix()
	case Queued:
		this.tr_queued = time.Now().Unix()
	case Running:
		this.tr_running = time.Now().Unix()
	case Failed:
		fallthrough
	case Committed:
		this.tr_end = time.Now().Unix()
	}
}

func (this *Message) SetTr_Id(trid string) {
	this.tr_id = trid
}

func (this *Message) SetTr_ErrMsg(errMsg string) {
	this.tr_errMsg = errMsg
}

func (this *Message) SetTr_Timeout(timeout int64) {
	this.tr_timeout = timeout
}

func (this *Message) SetTr_Replica(replica byte) {
	this.tr_replica = replica
}

func (this *Message) SetTr_IsReplica(isReplica bool) {
	this.tr_isReplica = isReplica
}
