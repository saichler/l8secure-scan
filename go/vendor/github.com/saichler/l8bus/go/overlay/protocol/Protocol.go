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
	"sync/atomic"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// Discovery_Enabled controls whether peer discovery via UDP broadcast is enabled.
var Discovery_Enabled = true

// Protocol handles message creation, serialization, and sequence numbering
// for the Layer8 overlay network communication.
type Protocol struct {
	sequence atomic.Uint32
	vnic     ifs.IVNic
}

// New creates a new Protocol instance with the given resources.
func New(vnic ifs.IVNic) *Protocol {
	p := &Protocol{}
	p.vnic = vnic
	return p
}

// MessageOf deserializes raw bytes into a Message struct.
func (this *Protocol) MessageOf(data []byte) (*ifs.Message, error) {
	msg := &ifs.Message{}
	_, err := msg.Unmarshal(data, this.vnic.Resources())
	return msg, err
}

// ElementsOf extracts the payload elements from a message.
func (this *Protocol) ElementsOf(msg *ifs.Message) (ifs.IElements, error) {
	return ElementsOf(msg, this.vnic.Resources())
}

// ElementsOf is a standalone function to extract payload elements from a message
// using the provided resources for deserialization.
func ElementsOf(msg *ifs.Message, resourcs ifs.IResources) (ifs.IElements, error) {
	result := &object.Elements{}
	err := result.Deserialize(msg.Data(), resourcs.Registry())
	if err != nil {
		return nil, err
	}
	return result, err
}

// NextMessageNumber returns the next unique message sequence number.
// This is used for message ordering and request/reply correlation.
func (this *Protocol) NextMessageNumber() uint32 {
	return this.sequence.Add(1)
}

// DataFor serializes elements into bytes for message transmission.
func DataFor(elems ifs.IElements, security ifs.ISecurityProvider) ([]byte, error) {
	var data []byte
	var err error

	data, err = elems.Serialize()
	return data, err
}

// CreateMessageFor creates a complete message with all routing and metadata.
// It supports unicast/multicast modes, request/reply patterns, transactions,
// and priority-based scheduling.
func (this *Protocol) CreateMessageFor(destination, serviceName string, serviceArea byte,
	priority ifs.Priority, multicastMode ifs.MulticastMode, action ifs.Action, source, vnet string, o ifs.IElements,
	isRequest, isReply bool, msgNum uint32,
	tr_state ifs.TransactionState, tr_id, tr_errMsg string,
	tr_created, tr_queued, tr_running, tr_complete, tr_timeout int64, tr_replica byte, tr_isReplica bool,
	aaaid string) ([]byte, error) {

	//Disable priority for now until i figure out what is causing starvation
	priority = ifs.P8

	var data []byte
	var err error

	data, err = o.Serialize()
	if err != nil {
		return nil, err
	}

	msg, err := this.vnic.Resources().Security().Message(aaaid, this.vnic)
	defer MsgLog.AddLog(serviceName, serviceArea, action)
	if err != nil {
		return nil, err
	}
	msg.Init(destination,
		serviceName,
		serviceArea,
		priority,
		multicastMode,
		action,
		source,
		vnet,
		data,
		isRequest,
		isReply,
		msgNum,
		tr_state,
		tr_id,
		tr_errMsg,
		tr_created,
		tr_queued,
		tr_running,
		tr_complete,
		tr_timeout,
		tr_replica,
		tr_isReplica)

	return msg.Marshal(nil, this.vnic.Resources())
}

// CreateMessageForm creates a message from an existing Message template with new payload elements.
func (this *Protocol) CreateMessageForm(msg *ifs.Message, o ifs.IElements) ([]byte, error) {
	var data []byte
	var err error
	if o == nil {
		o = object.New(nil, nil)
	}
	data, err = o.Serialize()
	if err != nil {
		return nil, err
	}

	//create the wrapping message for the destination
	msg.SetData(data)
	return msg.Marshal(nil, this.vnic.Resources())
}
