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
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// rollbackInternal reverts a committed transaction using the saved pre-commit state.
// Converts the action type to its inverse operation (POST->DELETE, etc.).
func (this *ServiceTransactions) rollbackInternal(msg *ifs.Message) ifs.IElements {

	if msg.Action() == ifs.Notify {
		return nil
	}

	this.setRollbackAction(msg)

	this.preCommitMtx.Lock()
	defer this.preCommitMtx.Unlock()

	elem := this.preCommitObject(msg)
	resp := this.nic.Resources().Services().TransactionHandle(elem, msg.Action(), msg, this.nic)
	if resp != nil && resp.Error() != nil {
		msg.SetTr_State(ifs.Failed)
		msg.SetTr_ErrMsg("T05_Rollback.rollbackInternal: Handle Error: " + msg.Tr_Id() + " " + resp.Error().Error())
		delete(this.preCommit, msg.Tr_Id())
		return L8TransactionFor(msg)
	}
	delete(this.preCommit, msg.Tr_Id())
	return L8TransactionFor(msg)
}

// setRollbackAction converts the original action to its inverse for rollback.
// POST becomes DELETE, DELETE becomes POST, PUT/PATCH become PUT.
func (this *ServiceTransactions) setRollbackAction(msg *ifs.Message) {
	switch msg.Action() {
	case ifs.POST:
		msg.SetAction(ifs.DELETE)
	case ifs.DELETE:
		msg.SetAction(ifs.POST)
	case ifs.PUT:
		msg.SetAction(ifs.PUT)
	case ifs.PATCH:
		msg.SetAction(ifs.PUT)
	}
}

// preCommitObject retrieves the saved pre-commit state for rollback.
func (this *ServiceTransactions) preCommitObject(msg *ifs.Message) ifs.IElements {
	item := this.preCommit[msg.Tr_Id()]
	elem, ok := item.(ifs.IElements)
	if ok {
		return elem
	}
	return object.New(nil, item)
}
