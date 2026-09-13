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
	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// commitInternal performs the commit phase on a participant node.
// Saves pre-commit state for rollback, then executes the transaction handler.
func (this *ServiceTransactions) commitInternal(msg *ifs.Message) ifs.IElements {
	if msg.Action() == ifs.Notify {
		//_, err := services.Notify()
		return nil
	}

	pb, err := protocol.ElementsOf(msg, this.nic.Resources())
	if err != nil {
		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg("T04_Commit.commitInternal: Protocol Error: " + msg.Tr_Id() + " " + err.Error())
		this.nic.Resources().Logger().Debug(msg.Tr_Id() + " " + err.Error())
		return L8TransactionFor(msg)
	}

	if msg.Tr_IsReplica() {
		pb = object.NewReplicaRequest(pb, msg.Tr_Replica())
	}

	err = this.setPreCommitObject(pb, msg)
	if err != nil {
		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg(err.Error())
		this.nic.Resources().Logger().Debug(msg.Tr_Id() + " " + err.Error())
		return L8TransactionFor(msg)
	}

	this.nic.Resources().Logger().Debug("T04_Commit.commitInternal: Before Transaction Handle ", msg.Tr_Id())
	resp := this.nic.Resources().Services().TransactionHandle(pb, msg.Action(), msg, this.nic)
	if resp != nil && resp.Error() != nil {
		this.preCommitMtx.Lock()
		delete(this.preCommit, msg.Tr_Id())
		this.preCommitMtx.Unlock()
		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg("T04_Commit.commitInternal: Handle Error: " + msg.Tr_Id() + " " + resp.Error().Error())
		this.nic.Resources().Logger().Debug(msg.Tr_ErrMsg())

		return L8TransactionFor(msg)
	}
	this.nic.Resources().Logger().Debug("T04_Commit.commitInternal: Transaction commited on node ",
		this.nic.Resources().SysConfig().LocalUuid, " - ", msg.Tr_Id())
	msg.SetTr_State(ifs.Committed)
	return L8TransactionFor(msg)
}

// setPreCommitObject saves the current state before committing for potential rollback.
// For PUT/DELETE/PATCH, fetches the existing object; for POST, stores the new object.
func (this *ServiceTransactions) setPreCommitObject(pb ifs.IElements, msg *ifs.Message) error {

	if msg.Action() == ifs.PUT ||
		msg.Action() == ifs.DELETE ||
		msg.Action() == ifs.PATCH {
		//Get the object before performing the action so we could rollback
		//if necessary.
		resp := this.nic.Resources().Services().TransactionHandle(pb, ifs.GET, msg, this.nic)
		if resp != nil && resp.Error() != nil {
			return resp.Error()
		}
		this.preCommitMtx.Lock()
		defer this.preCommitMtx.Unlock()
		this.preCommit[msg.Tr_Id()] = resp
	} else {
		this.preCommitMtx.Lock()
		defer this.preCommitMtx.Unlock()
		this.preCommit[msg.Tr_Id()] = pb
	}
	return nil
}
