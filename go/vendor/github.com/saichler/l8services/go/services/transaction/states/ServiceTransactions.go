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
	"bytes"
	"strconv"
	"sync"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
)

// ServiceTransactions manages the transaction queue for a single service.
// It processes transactions sequentially using a condition variable for
// blocking queue operations and maintains pre-commit state for rollback.
type ServiceTransactions struct {
	mtx     *sync.Mutex
	cond    *sync.Cond
	queue   []*ifs.Message
	running bool
	nic     ifs.IVNic

	preCommit    map[string]interface{}
	preCommitMtx *sync.Mutex
}

// newServiceTransactions creates a new transaction queue and starts its processor.
func newServiceTransactions(nic ifs.IVNic) *ServiceTransactions {
	serviceTransactions := &ServiceTransactions{}
	serviceTransactions.mtx = &sync.Mutex{}
	serviceTransactions.cond = sync.NewCond(serviceTransactions.mtx)
	serviceTransactions.queue = make([]*ifs.Message, 0)
	serviceTransactions.running = true
	serviceTransactions.nic = nic
	serviceTransactions.preCommitMtx = &sync.Mutex{}
	serviceTransactions.preCommit = map[string]interface{}{}

	go serviceTransactions.processTransactions()
	return serviceTransactions
}

// Next blocks until a transaction is available and returns it.
// Returns nil if the service is shutting down.
func (this *ServiceTransactions) Next() *ifs.Message {
	this.mtx.Lock()
	defer this.mtx.Unlock()
	for this.running && len(this.queue) == 0 {
		this.cond.Wait()
	}

	if !this.running {
		return nil
	}

	msg := this.queue[0]
	this.queue = this.queue[1:]
	return msg
}

// processTransactions is the background goroutine that dequeues and runs transactions.
func (this *ServiceTransactions) processTransactions() {
	for this.running {
		tr := this.Next()
		if tr == nil {
			continue
		}
		this.run(tr)
	}
}

// ServiceKey creates a unique key from service name and area for map indexing.
func ServiceKey(serviceName string, serviceArea byte) string {
	buff := bytes.Buffer{}
	buff.WriteString(serviceName)
	buff.WriteString(strconv.Itoa(int(serviceArea)))
	return buff.String()
}

// L8TransactionOf creates an L8Transaction protobuf from message transaction state.
func L8TransactionOf(msg *ifs.Message) *l8services.L8Transaction {
	return &l8services.L8Transaction{State: int32(msg.Tr_State()),
		Id:      msg.Tr_Id(),
		ErrMsg:  msg.Tr_ErrMsg(),
		Created: msg.Tr_Created(),
		Queued:  msg.Tr_Queued(),
		Running: msg.Tr_Running(),
		End:     msg.Tr_End(),
	}
}

// L8TransactionFor wraps L8TransactionOf result as IElements for responses.
func L8TransactionFor(msg *ifs.Message) ifs.IElements {
	return object.New(nil, L8TransactionOf(msg))
}
