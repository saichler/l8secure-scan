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

// Package states implements the transaction state machine for distributed
// ACID transactions. It manages transaction lifecycle through states:
// Created -> Queued -> Running -> Committed/Rollback -> Cleanup.
package states

import (
	"sync"

	"github.com/saichler/l8types/go/ifs"
)

// TransactionManager orchestrates distributed transactions across services.
// It maintains per-service transaction queues and coordinates the 2-phase
// commit protocol for ensuring data consistency.
type TransactionManager struct {
	serviceTransactions map[string]*ServiceTransactions
	services            ifs.IServices
	mtx                 *sync.Mutex
}

// NewTransactionManager creates a new TransactionManager linked to the service manager.
func NewTransactionManager(services ifs.IServices) *TransactionManager {
	tm := &TransactionManager{}
	tm.mtx = &sync.Mutex{}
	tm.services = services
	tm.serviceTransactions = make(map[string]*ServiceTransactions)
	return tm
}

// transactionsOf returns or creates the ServiceTransactions queue for a service.
func (this *TransactionManager) transactionsOf(msg *ifs.Message, nic ifs.IVNic) *ServiceTransactions {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	serviceKey := ServiceKey(msg.ServiceName(), msg.ServiceArea())
	st, ok := this.serviceTransactions[serviceKey]
	if !ok {
		this.serviceTransactions[serviceKey] = newServiceTransactions(nic)
		st = this.serviceTransactions[serviceKey]
	}
	return st
}

// Run processes a transaction based on its current state, routing to the
// appropriate handler (created, commit, rollback, or cleanup).
func (this *TransactionManager) Run(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	switch msg.Tr_State() {
	case ifs.Created:
		return this.created(msg, vnic)
	case ifs.Running:
		return this.commit(msg, vnic)
	case ifs.Rollback:
		return this.rollback(msg, vnic)
	case ifs.Cleanup:
		return this.cleanup(msg, vnic)
	default:
		panic("Unexpected transaction state " + msg.Tr_State().String() + ":" + msg.Tr_ErrMsg())
	}
}

// created handles newly created transactions by queuing them for processing.
func (this *TransactionManager) created(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	st := this.transactionsOf(msg, vnic)
	return st.queueTransaction(msg, vnic)
}

// commit processes the commit phase of a transaction.
func (this *TransactionManager) commit(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	st := this.transactionsOf(msg, vnic)
	return st.commitInternal(msg)
}

// rollback reverts a transaction that failed during commit.
func (this *TransactionManager) rollback(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	st := this.transactionsOf(msg, vnic)
	return st.rollbackInternal(msg)
}

// cleanup removes transaction state after successful commit.
func (this *TransactionManager) cleanup(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	st := this.transactionsOf(msg, vnic)
	return st.cleanupInternal(msg)
}
