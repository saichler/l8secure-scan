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

// Package requests provides concurrent transaction request handling for
// distributed transactions. It manages parallel requests to multiple peer
// nodes and collects their responses synchronously.
package requests

import (
	"sync"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
)

// Requests manages concurrent requests to peer nodes during a transaction.
// It tracks pending requests and uses a condition variable to synchronize
// waiting for all responses to complete.
type Requests struct {
	cond    *sync.Cond
	pending map[string]string
	count   int
	vnic    ifs.IVNic
}

// NewRequest creates a new Requests instance for managing concurrent peer requests.
func NewRequest(vnic ifs.IVNic) *Requests {
	rq := &Requests{}
	rq.pending = make(map[string]string)
	rq.cond = sync.NewCond(&sync.Mutex{})
	rq.vnic = vnic
	return rq
}

// addOne registers a new pending request for the target node.
func (this *Requests) addOne(target string) {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	this.pending[target] = ""
	this.count++
}

// reportError records an error for a target and decrements the pending count.
// Broadcasts when all requests have completed.
func (this *Requests) reportError(target string, err error) {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	this.pending[target] = err.Error()
	this.count--
	if this.count == 0 {
		this.cond.Broadcast()
	}
}

// reportResult records a transaction result and decrements the pending count.
// If the transaction failed, stores the error message. Broadcasts when all complete.
func (this *Requests) reportResult(target string, tr *l8services.L8Transaction) {
	this.cond.L.Lock()
	defer this.cond.L.Unlock()
	if tr == nil {
		this.pending[target] = "Nil/Timeout Transaction"
	} else if tr.State == int32(ifs.Failed) {
		this.pending[target] = tr.ErrMsg
	}
	this.count--
	if this.count == 0 {
		this.cond.Broadcast()
	}
}

// requestFromPeer sends a transaction request to a single peer node.
// For replication, it clones the message and sets replica information.
func (this *Requests) requestFromPeer(msg *ifs.Message, target string, isReplicate bool, replicateNum byte) {
	this.addOne(target)

	if isReplicate {
		clone := msg.Clone()
		clone.SetTr_Replica(replicateNum)
		clone.SetTr_IsReplica(true)
		msg = clone
	}

	resp := this.vnic.Forward(msg, target)
	if resp != nil && resp.Error() != nil {
		this.vnic.Resources().Logger().Error(resp.Error())
		this.reportError(target, resp.Error())
		return
	}

	tr, _ := resp.Element().(*l8services.L8Transaction)
	this.reportResult(target, tr)
}

// RequestFromPeers sends transaction requests to multiple peers concurrently and waits
// for all responses. Returns success status and a map of peer UUIDs to error messages.
func RequestFromPeers(msg *ifs.Message, targets map[string]byte, vnic ifs.IVNic, isReplicate bool) (bool, map[string]string) {

	this := NewRequest(vnic)

	this.cond.L.Lock()
	defer this.cond.L.Unlock()

	for target, replica := range targets {
		go this.requestFromPeer(msg, target, isReplicate, replica)
	}

	if len(targets) > 0 {
		this.cond.Wait()
	}

	ok := true
	for _, e := range this.pending {
		if e != "" {
			ok = false
			break
		}
	}

	if !ok {
		msg.SetTr_State(ifs.Failed)
		return false, this.pending
	}

	return true, this.pending
}
