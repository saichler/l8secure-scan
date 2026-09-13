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

package vnic

import (
	"errors"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/nets"
	"github.com/saichler/l8utils/go/utils/queues"
	"github.com/saichler/l8utils/go/utils/strings"
)

// TX handles outgoing message transmission for a VNic.
// It queues messages and writes them to the socket connection.
type TX struct {
	vnic         *VirtualNetworkInterface
	shuttingDown bool
	// The incoming data queue
	tx *queues.ByteQueue
}

func newTX(vnic *VirtualNetworkInterface) *TX {
	tx := &TX{}
	tx.vnic = vnic
	tx.tx = queues.NewByteQueue("TX", int(vnic.resources.SysConfig().TxQueueSize))
	return tx
}

func (this *TX) start() {
	go this.writeToSocket()
}

func (this *TX) shutdown() {
	this.shuttingDown = true
	if this.vnic.conn != nil {
		this.vnic.conn.Close()
	}
	this.tx.Shutdown()
}

func (this *TX) name() string {
	return "TX"
}

// loop of Writing data to socket
func (this *TX) writeToSocket() {
	// As long ad the port is active
	for this.vnic.running {
		// Get next data to write to the socket from the TX queue, if no data, this is a blocking call
		data := this.tx.Next()
		// if the data is not nil
		if data != nil && this.vnic.running {
			//Write the data to the socket
			err := nets.Write(data, this.vnic.conn, this.vnic.resources.SysConfig())
			// If there is an error
			if err != nil {
				if this.vnic.IsVNet {
					break
				}
				// If this is not a port on the switch, then try to reconnect.
				if !this.shuttingDown && this.vnic.running {
					this.vnic.reconnect()
					err = nets.Write(data, this.vnic.conn, this.vnic.resources.SysConfig())
				} else {
					break
				}
			}
			this.vnic.healthStatistics.Stamp()
			this.vnic.healthStatistics.IncrementTX(data)
		} else {
			// if the data is nil, break and cleanup
			break
		}
	}
	this.vnic.resources.Logger().Debug("TX for ", this.vnic.name, " ended.")
	this.vnic.Shutdown()
}

// Send Add the raw data to the tx queue to be written to the socket
func (this *TX) SendMessage(data []byte) error {
	// if the port is still active
	if this.vnic.running {
		// Add the data to the TX queue
		this.tx.Add(data)
	} else {
		return errors.New("Port is not active")
	}
	return nil
}

// Unicast is wrapping a protobuf with a secure message and send it to the vnet
func (this *TX) Unicast(destination, serviceName string, serviceArea byte, action ifs.Action, any ifs.IElements,
	p ifs.Priority, m ifs.MulticastMode, isRequest, isReply bool, msgNum uint32,
	tr_state ifs.TransactionState, tr_id, tr_errMsg string,
	tr_created, tr_queued, tr_running, tr_completed, tr_timeout int64, tr_replica byte, tr_isReplica bool, token string) error {
	if len(destination) != 36 {
		return errors.New(strings.New("Invalid destination address ", destination, " size ", len(destination)).String())
	}
	return this.Multicast(destination, serviceName, serviceArea, action, any, p, m,
		isRequest, isReply, msgNum, tr_state, tr_id, tr_errMsg, tr_created, tr_queued, tr_running, tr_completed, tr_timeout, tr_replica, tr_isReplica, token)
}

// Multicast is wrapping a protobuf with a secure message and send it to the vnet topic
func (this *TX) Multicast(destination, serviceName string, serviceArea byte, action ifs.Action, any ifs.IElements,
	p ifs.Priority, m ifs.MulticastMode, isRequest, isReply bool, msgNum uint32,
	tr_state ifs.TransactionState, tr_id, tr_errMsg string,
	tr_created, tr_queued, tr_running, tr_completed, tr_timeout int64, tr_replica byte, tr_isReplica bool,
	token string) error {
	// Create message payload
	data, err := this.vnic.protocol.CreateMessageFor(destination, serviceName, serviceArea, p, m, action,
		this.vnic.resources.SysConfig().LocalUuid, this.vnic.resources.SysConfig().RemoteUuid, any, isRequest, isReply, msgNum,
		tr_state, tr_id, tr_errMsg,
		tr_created, tr_queued, tr_running, tr_completed,
		tr_timeout, tr_replica, tr_isReplica, token)
	if err != nil {
		this.vnic.resources.Logger().Error("Failed to create message:", err)
		return err
	}
	//Send the secure message to the vnet
	return this.SendMessage(data)
}
