/*
© 2025 Sharon Aicler (saichler@gmail.com)

Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
You may obtain a copy of the License at:

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ifs

// Priority defines message priority levels (P1 highest, P8 lowest).
// Higher priority messages are processed before lower priority ones.
type Priority byte

const (
	P8 Priority = 0 // Lowest priority
	P7 Priority = 1
	P6 Priority = 2
	P5 Priority = 3
	P4 Priority = 4
	P3 Priority = 5
	P2 Priority = 6
	P1 Priority = 7 // Highest priority
)

// MulticastMode defines how messages are routed to service providers.
type VNicMethod byte
type MulticastMode byte

const (
	Invalid_VNICMethod VNicMethod = 0
	Unicast            VNicMethod = 1
	Request            VNicMethod = 2
	Multicast          VNicMethod = 3
	RoundRobin         VNicMethod = 4
	RoundRobinRequest  VNicMethod = 5
	Proximity          VNicMethod = 6
	ProximityRequest   VNicMethod = 7
	Leader             VNicMethod = 8
	LeaderRequest      VNicMethod = 9
	Local              VNicMethod = 10
	LocalRequest       VNicMethod = 11

	M_All        MulticastMode = 0   // Send to all providers
	M_RoundRobin MulticastMode = 1   // Send to one provider in round-robin order
	M_Proximity  MulticastMode = 2   // Send to nearest (lowest latency) provider
	M_Local      MulticastMode = 3   // Send to local provider only (same VNic)
	M_Leader     MulticastMode = 4   // Send to the elected leader
	M_Unicast    MulticastMode = 128 // Direct unicast to specific destination
)

// Action defines the type of operation to perform on a service.
type Action byte

const (
	// CRUD Actions
	POST   Action = 1 // Create a new resource
	PUT    Action = 2 // Replace an existing resource
	PATCH  Action = 3 // Partially update a resource
	DELETE Action = 4 // Delete a resource
	GET    Action = 5 // Retrieve a resource

	// Message Actions
	Reply     Action = 6 // Response to a request
	Notify    Action = 7 // Property change notification
	Handle    Action = 8 // Generic message handling
	EndPoints Action = 9 // Endpoint discovery

	// Leader Election Actions
	ElectionRequest    Action = 10 // Initiate election for a service
	ElectionResponse   Action = 11 // Response from higher-priority node (still alive)
	LeaderAnnouncement Action = 12 // Announce new leader for service
	LeaderHeartbeat    Action = 13 // Periodic heartbeat from current leader
	LeaderQuery        Action = 14 // Query who is the current leader
	LeaderResign       Action = 15 // Graceful leader resignation
	LeaderChallenge    Action = 16 // Challenge current leader validity

	// Participant Registry Actions
	ServiceRegister   Action = 17 // Announce: "I'm hosting this service"
	ServiceUnregister Action = 18 // Announce: "I'm no longer hosting this service"
	ServiceQuery      Action = 19 // Query: "Who is hosting this service?"

	// Map-Reduce Actions (distributed query processing)
	MapR_POST   Action = 21 // Map-reduce POST phase
	MapR_PUT    Action = 22 // Map-reduce PUT phase
	MapR_PATCH  Action = 23 // Map-reduce PATCH phase
	MapR_DELETE Action = 24 // Map-reduce DELETE phase
	MapR_GET    Action = 25 // Map-reduce GET phase
)

// TransactionState represents the lifecycle state of a distributed transaction.
type TransactionState uint8

const (
	NotATransaction TransactionState = 0 // This message is not part of a transaction
	Created         TransactionState = 1 // Transaction has been created
	Queued          TransactionState = 2 // Transaction is queued for execution
	Running         TransactionState = 3 // Transaction is currently executing
	Committed       TransactionState = 4 // Transaction completed successfully
	Rollback        TransactionState = 5 // Transaction is being rolled back
	Failed          TransactionState = 6 // Transaction failed
	Cleanup         TransactionState = 7 // Transaction cleanup in progress
)

// String returns the string representation of a TransactionState.
func (t TransactionState) String() string {
	switch t {
	case NotATransaction:
		return "NotATransaction"
	case Created:
		return "Created"
	case Queued:
		return "Queued"
	case Running:
		return "Running"
	case Committed:
		return "Committed"
	case Rollback:
		return "Rollback"
	case Failed:
		return "Failed"
	case Cleanup:
		return "Cleanup"
	}
	return "Unknown"
}

const (
	// DESTINATION_Single is a placeholder UUID for round-robin single destination.
	DESTINATION_Single = "signleXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
	// DESTINATION_Leader is a placeholder UUID for leader-based routing.
	DESTINATION_Leader = "leaderXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
)
