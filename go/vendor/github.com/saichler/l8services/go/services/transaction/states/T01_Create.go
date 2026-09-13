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
	"context"
	"runtime"
	"time"

	"github.com/saichler/l8types/go/ifs"
)

// createTransaction initializes a new transaction in the message if not already set.
// Generates a unique ID and sets the state to Created.
func createTransaction(msg *ifs.Message) {
	if msg.Tr_State() == ifs.NotATransaction {
		msg.SetTr_Id(ifs.NewUuid())
		msg.SetTr_State(ifs.Created)
	}
}

// Create initiates a new transaction by creating its ID and forwarding to the leader.
// Returns immediately with Created state while the actual work continues asynchronously.
func (this *TransactionManager) Create(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	//Create the new transaction inside the message
	createTransaction(msg)

	//To Keep the same flow, we are going to forward the transaction to the leader
	//even if this is the leader
	go func() {
		runtime.Gosched()
		leader := vnic.Resources().Services().GetLeader(msg.ServiceName(), msg.ServiceArea())
		leaderResponse := vnic.Forward(msg, leader)
		//Send the final resulth to the initiator.
		vnic.Reply(msg, leaderResponse)
	}()

	//Return the temporary response as the transaction state created
	return L8TransactionFor(msg)
}

// Create2 is an alternative Create implementation with timeout support.
// Uses context with 30-second timeout for the forwarding operation.
func (this *TransactionManager) Create2(msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	createTransaction(msg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	go func() {
		defer cancel()

		select {
		case <-ctx.Done():
			vnic.Resources().Logger().Warning("Transaction forwarding cancelled")
			return
		default:
			leader := vnic.Resources().Services().GetLeader(msg.ServiceName(), msg.ServiceArea())
			if leader == "" {
				vnic.Resources().Logger().Error("No leader found for service")
				return
			}

			leaderResponse := vnic.Forward(msg, leader)
			if err := vnic.Reply(msg, leaderResponse); err != nil {
				vnic.Resources().Logger().Error("Failed to reply:", err)
			}
		}
	}()

	return L8TransactionFor(msg)
}
