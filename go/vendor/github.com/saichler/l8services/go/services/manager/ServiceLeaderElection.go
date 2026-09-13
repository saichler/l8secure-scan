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
	"bytes"
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/saichler/l8types/go/ifs"
)

// Election timing constants
const (
	electionTimeout  = 3 * time.Second // Time to wait for election responses
	heartbeatTimeout = 5 * time.Second // Time before leader is considered dead
	heartbeatPeriod  = 2 * time.Second // Interval between leader heartbeats
)

// electionState represents the current state of a node in the election process.
type electionState byte

const (
	idle      electionState = 0 // No election activity
	electing  electionState = 1 // Currently running an election
	isLeader  electionState = 2 // This node is the leader
	hasLeader electionState = 3 // Another node is the leader
)

// leaderInfo holds the election state and timing information for a service.
type leaderInfo struct {
	leaderUuid        string
	lastHeartbeat     time.Time
	state             electionState
	electionTimer     *time.Timer
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	electionRunning   bool // Prevents concurrent elections
	heartbeatRunning  bool // Prevents concurrent heartbeat goroutines
	monitorRunning    bool // Prevents concurrent monitor goroutines
	mtx               sync.RWMutex
}

// LeaderElection manages distributed leader election using a bully algorithm.
// Each service can have its own leader, tracked separately by service key.
// Higher UUID values have higher priority in elections.
type LeaderElection struct {
	leaders        sync.Map // key: serviceKey string -> *leaderInfo
	serviceManager *ServiceManager
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewLeaderElection creates a new LeaderElection instance with a cancellable context.
func NewLeaderElection(sm *ServiceManager) *LeaderElection {
	ctx, cancel := context.WithCancel(context.Background())
	return &LeaderElection{
		serviceManager: sm,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// handleElection routes election-related messages to the appropriate handler
// based on the action type (request, response, announcement, heartbeat, etc.).
func (le *LeaderElection) handleElection(action ifs.Action, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	switch action {
	case ifs.ElectionRequest:
		return le.handleElectionRequest(vnic, msg)
	case ifs.ElectionResponse:
		return le.handleElectionResponse(vnic, msg)
	case ifs.LeaderAnnouncement:
		return le.handleLeaderAnnouncement(vnic, msg)
	case ifs.LeaderHeartbeat:
		return le.handleLeaderHeartbeat(vnic, msg)
	case ifs.LeaderQuery:
		return le.handleLeaderQuery(vnic, msg)
	case ifs.LeaderResign:
		return le.handleLeaderResign(vnic, msg)
	case ifs.LeaderChallenge:
		return le.handleLeaderChallenge(vnic, msg)
	}
	return nil
}

// handleElectionRequest processes an incoming election request from another node.
// If this node has higher priority (higher UUID), it responds and starts its own election.
func (le *LeaderElection) handleElectionRequest(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	localUuid := vnic.Resources().SysConfig().LocalUuid
	senderUuid := msg.Source()

	// Skip self-multicast
	if senderUuid == localUuid {
		return nil
	}

	vnic.Resources().Logger().Debug("Election request from", senderUuid, "for", msg.ServiceName(), "area", msg.ServiceArea(), "local:", localUuid)

	// Check if this node can participate in voting
	if !le.canVote(msg.ServiceName(), msg.ServiceArea()) {
		vnic.Resources().Logger().Debug("This node cannot vote for", msg.ServiceName(), "area", msg.ServiceArea())
		return nil
	}

	// If sender has lower priority (higher UUID), respond that we're still alive
	if senderUuid < localUuid {
		vnic.Resources().Logger().Debug("Responding to election request and starting own election")
		vnic.Unicast(senderUuid, msg.ServiceName(), msg.ServiceArea(), ifs.ElectionResponse, nil)
		// Start our own election since we have higher priority
		go le.startElection(msg.ServiceName(), msg.ServiceArea(), vnic)
	}

	return nil
}

// handleElectionResponse handles responses from higher-priority nodes,
// indicating this node should abort its election attempt.
func (le *LeaderElection) handleElectionResponse(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	key := makeServiceKey(msg.ServiceName(), msg.ServiceArea())
	info := le.getLeaderInfo(key)
	if info == nil {
		vnic.Resources().Logger().Debug("Election response received but no leader info for", msg.ServiceName(), "area", msg.ServiceArea())
		return nil
	}

	vnic.Resources().Logger().Debug("Election response from", msg.Source(), "for", msg.ServiceName(), "area", msg.ServiceArea())

	info.mtx.Lock()

	// If we're in an election and received a response, a higher-priority node exists
	if info.state == electing {
		vnic.Resources().Logger().Debug("Higher priority node exists, aborting election")
		info.state = hasLeader
		// Don't call timer.Stop() - just change state and the election goroutine will handle it
	}
	info.mtx.Unlock()

	return nil
}

// handleLeaderAnnouncement processes a leader announcement, updating local state
// and starting either heartbeat sending (if leader) or monitoring (if follower).
func (le *LeaderElection) handleLeaderAnnouncement(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	key := makeServiceKey(msg.ServiceName(), msg.ServiceArea())
	info := le.getOrCreateLeaderInfo(key)

	localUuid := vnic.Resources().SysConfig().LocalUuid
	senderUuid := msg.Source()

	vnic.Resources().Logger().Debug("Leader announcement from", senderUuid, "for", msg.ServiceName(), "area", msg.ServiceArea())

	info.mtx.Lock()
	// Accept the leader announcement
	info.leaderUuid = senderUuid
	info.lastHeartbeat = time.Now()

	if senderUuid == localUuid {
		vnic.Resources().Logger().Debug("I am the leader")
		info.state = isLeader
		info.mtx.Unlock()
		// Don't stop timers here - let startHeartbeat handle it
		le.startHeartbeat(key, msg.ServiceName(), msg.ServiceArea(), vnic)
	} else {
		vnic.Resources().Logger().Debug("Following leader:", senderUuid)
		info.state = hasLeader
		info.mtx.Unlock()
		// Don't stop timers here - let startHeartbeatMonitor handle it
		le.startHeartbeatMonitor(key, msg.ServiceName(), msg.ServiceArea(), vnic)
	}

	vnic.Resources().Logger().Debug("Leader for", msg.ServiceName(), "area", msg.ServiceArea(), "is", senderUuid)
	return nil
}

// handleLeaderHeartbeat updates the last heartbeat timestamp from the leader,
// keeping the leader-follower relationship alive.
func (le *LeaderElection) handleLeaderHeartbeat(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	key := makeServiceKey(msg.ServiceName(), msg.ServiceArea())
	info := le.getLeaderInfo(key)
	if info == nil {
		return nil
	}

	info.mtx.Lock()
	info.leaderUuid = msg.Source()
	info.lastHeartbeat = time.Now()
	info.mtx.Unlock()

	return nil
}

// handleLeaderQuery responds to leader queries by announcing if this node is the leader.
func (le *LeaderElection) handleLeaderQuery(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	// Check if this node can participate in voting
	if !le.canVote(msg.ServiceName(), msg.ServiceArea()) {
		return nil
	}

	key := makeServiceKey(msg.ServiceName(), msg.ServiceArea())
	info := le.getLeaderInfo(key)

	localUuid := vnic.Resources().SysConfig().LocalUuid
	if info != nil {
		info.mtx.RLock()
		leaderUuid := info.leaderUuid
		state := info.state
		info.mtx.RUnlock()

		if state == isLeader && leaderUuid == localUuid {
			// Respond that we are the leader
			vnic.Unicast(msg.Source(), msg.ServiceName(), msg.ServiceArea(), ifs.LeaderAnnouncement, nil)
		}
	}

	return nil
}

// handleLeaderResign handles a leader resignation, clearing the leader state
// and triggering a new election.
func (le *LeaderElection) handleLeaderResign(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	key := makeServiceKey(msg.ServiceName(), msg.ServiceArea())
	info := le.getLeaderInfo(key)

	if info != nil {
		shouldStopTimers := false
		info.mtx.Lock()
		if info.leaderUuid == msg.Source() {
			info.leaderUuid = ""
			info.state = idle
			shouldStopTimers = true
		}
		info.mtx.Unlock()

		if shouldStopTimers {
			le.stopTimers(info)
		}

		// Start new election
		go le.startElection(msg.ServiceName(), msg.ServiceArea(), vnic)
	}

	return nil
}

// handleLeaderChallenge responds to challenges by sending a heartbeat if leader,
// or starting an election if no leader info exists.
func (le *LeaderElection) handleLeaderChallenge(vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	// Check if this node can participate in voting
	if !le.canVote(msg.ServiceName(), msg.ServiceArea()) {
		return nil
	}

	key := makeServiceKey(msg.ServiceName(), msg.ServiceArea())
	info := le.getLeaderInfo(key)

	localUuid := vnic.Resources().SysConfig().LocalUuid
	if info != nil {
		info.mtx.RLock()
		state := info.state
		info.mtx.RUnlock()

		if state == isLeader {
			// Respond with heartbeat to prove we're still the leader
			vnic.Multicast(msg.ServiceName(), msg.ServiceArea(), ifs.LeaderHeartbeat, nil)
		}
	} else if vnic.Resources().SysConfig().LocalUuid == localUuid {
		// No leader info, start election
		go le.startElection(msg.ServiceName(), msg.ServiceArea(), vnic)
	}

	return nil
}

// startElection initiates the bully election algorithm by multicasting an election
// request and waiting for responses from higher-priority nodes.
func (le *LeaderElection) startElection(serviceName string, serviceArea byte, vnic ifs.IVNic) {
	// Check if this node can participate in voting
	if !le.canVote(serviceName, serviceArea) {
		vnic.Resources().Logger().Debug("This node cannot vote for", serviceName, "area", serviceArea, "- skipping election")
		return
	}

	// Check global context
	if le.ctx.Err() != nil {
		return
	}

	key := makeServiceKey(serviceName, serviceArea)
	info := le.getOrCreateLeaderInfo(key)

	info.mtx.Lock()
	if info.state == electing || info.state == isLeader {
		vnic.Resources().Logger().Debug("Election already in progress or already leader for", serviceName, "area", serviceArea)
		info.mtx.Unlock()
		return
	}
	// Prevent concurrent elections
	if info.electionRunning {
		vnic.Resources().Logger().Debug("Election goroutine already running for", serviceName, "area", serviceArea)
		info.mtx.Unlock()
		return
	}
	info.state = electing
	info.electionRunning = true
	ctx := info.ctx
	info.mtx.Unlock()

	// Track goroutine
	le.wg.Add(1)
	info.wg.Add(1)
	defer func() {
		le.wg.Done()
		info.wg.Done()
		info.mtx.Lock()
		info.electionRunning = false
		info.mtx.Unlock()
	}()

	vnic.Resources().Logger().Debug("Starting election for", serviceName, "area", serviceArea)

	// Send election request to all nodes
	vnic.Multicast(serviceName, serviceArea, ifs.ElectionRequest, nil)

	vnic.Resources().Logger().Debug("Waiting", electionTimeout, "for election responses")

	// Wait for responses or context cancellation
	timer := time.NewTimer(electionTimeout)
	defer timer.Stop()

	info.mtx.Lock()
	info.electionTimer = timer
	info.mtx.Unlock()

	select {
	case <-timer.C:
		// Timeout - check if we should become leader
	case <-ctx.Done():
		vnic.Resources().Logger().Debug("Election cancelled for", serviceName, "area", serviceArea)
		return
	}

	info.mtx.Lock()
	currentState := info.state
	vnic.Resources().Logger().Debug("Election timeout elapsed, state:", currentState)

	if currentState == electing {
		// No higher-priority node responded, we are the leader
		localUuid := vnic.Resources().SysConfig().LocalUuid
		info.state = isLeader
		info.leaderUuid = localUuid
		info.mtx.Unlock()

		vnic.Resources().Logger().Debug("Elected as leader for", serviceName, "area", serviceArea)
		vnic.Multicast(serviceName, serviceArea, ifs.LeaderAnnouncement, nil)
		le.startHeartbeat(key, serviceName, serviceArea, vnic)
	} else {
		vnic.Resources().Logger().Debug("Not becoming leader, state changed to:", currentState)
		info.mtx.Unlock()
	}
}

// startHeartbeat begins periodic heartbeat multicasts to maintain leader status.
func (le *LeaderElection) startHeartbeat(key string, serviceName string, serviceArea byte, vnic ifs.IVNic) {
	info := le.getLeaderInfo(key)
	if info == nil {
		return
	}

	info.mtx.Lock()
	// Check if heartbeat is already running
	if info.heartbeatRunning {
		info.mtx.Unlock()
		return
	}
	info.heartbeatRunning = true
	ctx := info.ctx
	info.mtx.Unlock()

	// Check if context is still valid
	if ctx.Err() != nil {
		info.mtx.Lock()
		info.heartbeatRunning = false
		info.mtx.Unlock()
		return
	}

	// Track goroutine
	le.wg.Add(1)
	info.wg.Add(1)

	go func() {
		defer le.wg.Done()
		defer info.wg.Done()
		defer func() {
			info.mtx.Lock()
			info.heartbeatRunning = false
			info.mtx.Unlock()
		}()

		ticker := time.NewTicker(heartbeatPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info.mtx.RLock()
				state := info.state
				info.mtx.RUnlock()

				if state != isLeader {
					return
				}

				vnic.Multicast(serviceName, serviceArea, ifs.LeaderHeartbeat, nil)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// startHeartbeatMonitor watches for leader heartbeats and triggers election on timeout.
func (le *LeaderElection) startHeartbeatMonitor(key string, serviceName string, serviceArea byte, vnic ifs.IVNic) {
	info := le.getLeaderInfo(key)
	if info == nil {
		return
	}

	info.mtx.Lock()
	// Check if monitor is already running
	if info.monitorRunning {
		info.mtx.Unlock()
		return
	}
	info.monitorRunning = true
	ctx := info.ctx
	info.mtx.Unlock()

	// Check if context is still valid
	if ctx.Err() != nil {
		info.mtx.Lock()
		info.monitorRunning = false
		info.mtx.Unlock()
		return
	}

	// Track goroutine
	le.wg.Add(1)
	info.wg.Add(1)

	go func() {
		defer le.wg.Done()
		defer info.wg.Done()
		defer func() {
			info.mtx.Lock()
			info.monitorRunning = false
			info.mtx.Unlock()
		}()

		ticker := time.NewTicker(heartbeatTimeout)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info.mtx.RLock()
				lastHb := info.lastHeartbeat
				state := info.state
				info.mtx.RUnlock()

				if state != hasLeader {
					return
				}

				if time.Since(lastHb) > heartbeatTimeout {
					vnic.Resources().Logger().Debug("Leader heartbeat timeout for", serviceName, "area", serviceArea)
					info.mtx.Lock()
					info.state = idle
					info.leaderUuid = ""
					info.mtx.Unlock()
					go le.startElection(serviceName, serviceArea, vnic)
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// stopTimers stops election timers and resets running flags to allow new goroutines.
func (le *LeaderElection) stopTimers(info *leaderInfo) {
	// Simplified version - only stop the election timer
	// Don't cancel contexts or wait for goroutines to avoid disrupting services
	info.mtx.Lock()
	defer info.mtx.Unlock()

	if info.electionTimer != nil {
		info.electionTimer.Stop()
		info.electionTimer = nil
	}

	// Reset running flags to allow new goroutines to start
	info.electionRunning = false
	info.heartbeatRunning = false
	info.monitorRunning = false
}

// getLeaderInfo retrieves the leaderInfo for a service key, returning nil if not found.
func (le *LeaderElection) getLeaderInfo(key string) *leaderInfo {
	info, ok := le.leaders.Load(key)
	if !ok {
		return nil
	}
	return info.(*leaderInfo)
}

// getOrCreateLeaderInfo retrieves or creates a leaderInfo for a service key.
// Uses LoadOrStore for thread-safe concurrent access.
func (le *LeaderElection) getOrCreateLeaderInfo(key string) *leaderInfo {
	// Try to load existing info first
	if info, ok := le.leaders.Load(key); ok {
		return info.(*leaderInfo)
	}

	// Create new info
	ctx, cancel := context.WithCancel(le.ctx)
	newInfo := &leaderInfo{
		state:         idle,
		lastHeartbeat: time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Use LoadOrStore to handle concurrent creation attempts
	actual, _ := le.leaders.LoadOrStore(key, newInfo)
	actualInfo := actual.(*leaderInfo)

	// If we didn't win the race, cancel our context
	if actualInfo != newInfo {
		cancel()
	}

	return actualInfo
}

// StartElectionForService starts an election for a specific service asynchronously.
func (le *LeaderElection) StartElectionForService(serviceName string, serviceArea byte, vnic ifs.IVNic) {
	go le.startElection(serviceName, serviceArea, vnic)
}

// GetLeader returns the UUID of the current leader for a service, or empty string if none.
func (le *LeaderElection) GetLeader(serviceName string, serviceArea byte) string {
	key := makeServiceKey(serviceName, serviceArea)
	info := le.getLeaderInfo(key)
	if info == nil {
		return ""
	}

	info.mtx.RLock()
	defer info.mtx.RUnlock()

	if info.leaderUuid == "" {
		return ""
	}

	return info.leaderUuid
}

// IsLeader checks if the given UUID is the current leader for a service.
func (le *LeaderElection) IsLeader(serviceName string, serviceArea byte, uuid string) bool {
	key := makeServiceKey(serviceName, serviceArea)
	info := le.getLeaderInfo(key)
	if info == nil {
		return false
	}

	info.mtx.RLock()
	defer info.mtx.RUnlock()
	return info.leaderUuid == uuid && info.state == isLeader
}

// makeServiceKey creates a unique key by combining service name and area.
func makeServiceKey(serviceName string, serviceArea byte) string {
	buff := bytes.Buffer{}
	buff.WriteString(serviceName)
	buff.WriteString("-")
	buff.WriteString(strconv.Itoa(int(serviceArea)))
	return buff.String()
}

// canVote checks if this node can participate in voting for the given service or group.
// Service groups are always voters.
// For non-transactional stateful services, all peers are voters.
// For transactional services, the Voter() flag on TransactionConfig controls participation.
func (le *LeaderElection) canVote(serviceName string, serviceArea byte) bool {
	if le.serviceManager == nil {
		return false
	}

	handler, ok := le.serviceManager.services.get(serviceName, serviceArea)
	if !ok {
		// Check if serviceName is a group name — groups are always voters
		return le.isGroup(serviceName)
	}

	txConfig := handler.TransactionConfig()
	if txConfig == nil {
		return true
	}

	return txConfig.Voter()
}

// isGroup checks if the given name is a service group that any service maps to.
func (le *LeaderElection) isGroup(groupName string) bool {
	found := false
	le.serviceManager.serviceToGroup.Range(func(key, value interface{}) bool {
		if value.(string) == groupName {
			found = true
			return false
		}
		return true
	})
	return found
}

// Shutdown gracefully stops all leader election activities and waits for goroutines to exit
func (le *LeaderElection) Shutdown() {
	// Cancel global context to signal all goroutines to stop
	if le.cancel != nil {
		le.cancel()
	}

	// Cancel all per-service contexts and stop timers
	le.leaders.Range(func(key, value interface{}) bool {
		info := value.(*leaderInfo)
		info.mtx.Lock()
		if info.cancel != nil {
			info.cancel()
		}
		if info.electionTimer != nil {
			info.electionTimer.Stop()
		}
		info.mtx.Unlock()

		// Wait for this service's goroutines to finish
		info.wg.Wait()
		return true
	})

	// Wait for all global goroutines to complete
	le.wg.Wait()
}
