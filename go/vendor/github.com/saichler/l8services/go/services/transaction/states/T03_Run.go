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
	"github.com/saichler/l8services/go/services/replication"
	"github.com/saichler/l8services/go/services/transaction/requests"
	"github.com/saichler/l8types/go/ifs"
)

// run executes the transaction by distributing it to all participants (or replicas).
// Implements 2-phase commit: sends to all targets, handles failures with rollback,
// and cleans up after successful commit.
func (this *ServiceTransactions) run(msg *ifs.Message) {
	this.nic.Resources().Logger().Debug("T02_Run.run: ", msg.Tr_Id(), " for ServiceName ", msg.ServiceName(), " area ", msg.ServiceArea())
	//Check if this is the leader, again, just to make sure
	if this.nic.Resources().Services().GetLeader(msg.ServiceName(), msg.ServiceArea()) != this.nic.Resources().SysConfig().LocalUuid {
		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg("A non leader has got the message")
		this.nic.Reply(msg, L8TransactionFor(msg))
		return
	}

	//notify the originator that the transaction is running
	msg.SetTr_State(ifs.Running)
	this.nic.Reply(msg, L8TransactionFor(msg))

	var targets map[string]byte
	var err error
	isReplicate := false
	service, _ := this.nic.Resources().Services().ServiceHandler(msg.ServiceName(), msg.ServiceArea())
	if service.TransactionConfig().Replication() {
		//First see if there are already replication for this item
		targets, err = replication.ReplicationFor(msg, this.nic.Resources(), service)
		if err != nil {
			msg.SetTr_State(ifs.Failed)
			msg.SetTr_ErrMsg("T04_Commit.commitInternal: Protocol Error: " + msg.Tr_Id() + " " + err.Error())
			this.nic.Resources().Logger().Debug(msg.Tr_Id() + " " + err.Error())
			this.nic.Reply(msg, L8TransactionFor(msg))
			return
		}
		//If there are no replications, take from the roundrobin.
		if len(targets) == 0 {
			targets = this.nic.Resources().Services().RoundRobinParticipants(msg.ServiceName(), msg.ServiceArea(),
				service.TransactionConfig().ReplicationCount())
		}
		isReplicate = true
	} else {
		targets = this.nic.Resources().Services().GetParticipants(msg.ServiceName(), msg.ServiceArea())
	}

	this.nic.Resources().Logger().Debug("T02_Run.run: Sending to targets", msg.Tr_Id())
	ok, peers := requests.RequestFromPeers(msg, targets, this.nic, isReplicate)
	if !ok {
		commitedTargets := make(map[string]byte)
		errMsg := ""
		for k, v := range peers {
			if v == "" {
				commitedTargets[k] = targets[k]
			} else {
				errMsg = v
			}
		}

		// Send Rollback only to those peers that commited successfully
		msg.SetTr_State(ifs.Rollback)
		requests.RequestFromPeers(msg, commitedTargets, this.nic, isReplicate)

		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg("T02_Run.run: Failed to commit:" + errMsg)

		this.nic.Reply(msg, L8TransactionFor(msg))
		return
	}
	this.nic.Resources().Logger().Debug("T02_Run.run: Transaction committed: ", msg.Tr_Id())
	msg.SetTr_State(ifs.Committed)
	this.nic.Reply(msg, L8TransactionFor(msg))

	//cleanup
	msg.SetTr_State(ifs.Cleanup)
	requests.RequestFromPeers(msg, targets, this.nic, isReplicate)
}
