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

package manager

import (
	"sync"
	"time"

	"github.com/saichler/l8types/go/ifs"
)

const electionDebounceWindow = 500 * time.Millisecond

// pendingElection holds the data needed to start a deferred election.
type pendingElection struct {
	serviceName string
	serviceArea byte
	vnic        ifs.IVNic
}

// ElectionDebouncer collects election requests during a debounce window
// and fires them once after the window expires with no new requests.
// This prevents concurrent election storms during service activation.
type ElectionDebouncer struct {
	pending map[string]*pendingElection
	timer   *time.Timer
	le      *LeaderElection
	mtx     sync.Mutex
}

// NewElectionDebouncer creates a new debouncer linked to the given LeaderElection.
func NewElectionDebouncer(le *LeaderElection) *ElectionDebouncer {
	return &ElectionDebouncer{
		pending: make(map[string]*pendingElection),
		le:      le,
	}
}

// RequestElection queues an election for the given service.
// If a debounce window is already active, the timer resets.
// Elections fire once the window expires with no new requests.
func (d *ElectionDebouncer) RequestElection(serviceName string, serviceArea byte, vnic ifs.IVNic) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	key := makeServiceKey(serviceName, serviceArea)
	d.pending[key] = &pendingElection{
		serviceName: serviceName,
		serviceArea: serviceArea,
		vnic:        vnic,
	}

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(electionDebounceWindow, d.fireElections)
}

// fireElections starts all pending elections and clears the queue.
func (d *ElectionDebouncer) fireElections() {
	d.mtx.Lock()
	elections := make([]*pendingElection, 0, len(d.pending))
	for _, pe := range d.pending {
		elections = append(elections, pe)
	}
	d.pending = make(map[string]*pendingElection)
	d.timer = nil
	d.mtx.Unlock()

	for _, pe := range elections {
		d.le.StartElectionForService(pe.serviceName, pe.serviceArea, pe.vnic)
	}
}
