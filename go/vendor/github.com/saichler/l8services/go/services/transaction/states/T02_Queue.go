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

package states

import (
	"github.com/saichler/l8types/go/ifs"
)

// addTransaction adds a transaction to the queue after verifying this node is the leader.
// Sets the transaction state to Queued and broadcasts to wake up the processor.
func (this *ServiceTransactions) addTransaction(msg *ifs.Message, vnic ifs.IVNic) error {
	msg.SetTr_State(ifs.Queued)
	if vnic.Resources().SysConfig().LocalUuid != vnic.Resources().Services().GetLeader(msg.ServiceName(), msg.ServiceArea()) {
		return vnic.Resources().Logger().Error("A non leader has got the message")
	}
	this.mtx.Lock()
	defer this.mtx.Unlock()
	this.queue = append(this.queue, msg)
	this.cond.Broadcast()
	return nil
}

// queueTransaction queues a transaction for processing. GET operations are handled
// immediately without queueing; other operations are added to the queue.
func (this *ServiceTransactions) queueTransaction(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	if msg.Action() == ifs.GET {
		return this.internalGet(msg)
	}

	err := this.addTransaction(msg, vnic)
	if err != nil {
		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg(err.Error())
	}
	return L8TransactionFor(msg)
}
