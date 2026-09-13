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

package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/saichler/l8types/go/ifs"
)

// ConnectionHealthState represents the health state of a connection
type ConnectionHealthState int

const (
	HealthyState ConnectionHealthState = iota
	DegradedState
	UnhealthyState
	CriticalState
)

func (s ConnectionHealthState) String() string {
	switch s {
	case HealthyState:
		return "healthy"
	case DegradedState:
		return "degraded"
	case UnhealthyState:
		return "unhealthy"
	case CriticalState:
		return "critical"
	default:
		return "unknown"
	}
}

// ConnectionMetrics tracks metrics for a single connection
type ConnectionMetrics struct {
	ConnectionID     string
	RemoteAddr       string
	ConnectedAt      time.Time
	LastActivity     int64 // Unix timestamp
	
	// Traffic metrics
	MessagesSent     int64
	MessagesReceived int64
	BytesSent        int64
	BytesReceived    int64
	
	// Performance metrics
	LatencySum       int64 // Total latency in milliseconds
	LatencyCount     int64
	ErrorCount       int64
	TimeoutCount     int64
	
	// Health scoring
	HealthScore      int64 // 0-100, where 100 is perfect health
	State            int32 // ConnectionHealthState
	
	mutex           sync.RWMutex
}

// NewConnectionMetrics creates a new connection metrics tracker
func NewConnectionMetrics(connectionID, remoteAddr string) *ConnectionMetrics {
	return &ConnectionMetrics{
		ConnectionID: connectionID,
		RemoteAddr:   remoteAddr,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now().Unix(),
		HealthScore:  100, // Start with perfect health
		State:        int32(HealthyState),
	}
}

// RecordMessageSent increments sent message counters
func (c *ConnectionMetrics) RecordMessageSent(bytes int64) {
	atomic.AddInt64(&c.MessagesSent, 1)
	atomic.AddInt64(&c.BytesSent, bytes)
	atomic.StoreInt64(&c.LastActivity, time.Now().Unix())
	c.updateHealthScore()
}

// RecordMessageReceived increments received message counters
func (c *ConnectionMetrics) RecordMessageReceived(bytes int64) {
	atomic.AddInt64(&c.MessagesReceived, 1)
	atomic.AddInt64(&c.BytesReceived, bytes)
	atomic.StoreInt64(&c.LastActivity, time.Now().Unix())
	c.updateHealthScore()
}

// RecordLatency records a latency measurement
func (c *ConnectionMetrics) RecordLatency(latencyMs int64) {
	atomic.AddInt64(&c.LatencySum, latencyMs)
	atomic.AddInt64(&c.LatencyCount, 1)
	c.updateHealthScore()
}

// RecordError increments the error counter
func (c *ConnectionMetrics) RecordError() {
	atomic.AddInt64(&c.ErrorCount, 1)
	c.updateHealthScore()
}

// RecordTimeout increments the timeout counter
func (c *ConnectionMetrics) RecordTimeout() {
	atomic.AddInt64(&c.TimeoutCount, 1)
	c.updateHealthScore()
}

// GetAverageLatency returns the average latency in milliseconds
func (c *ConnectionMetrics) GetAverageLatency() float64 {
	count := atomic.LoadInt64(&c.LatencyCount)
	if count == 0 {
		return 0
	}
	sum := atomic.LoadInt64(&c.LatencySum)
	return float64(sum) / float64(count)
}

// GetHealthScore returns the current health score (0-100)
func (c *ConnectionMetrics) GetHealthScore() int64 {
	return atomic.LoadInt64(&c.HealthScore)
}

// GetState returns the current connection health state
func (c *ConnectionMetrics) GetState() ConnectionHealthState {
	return ConnectionHealthState(atomic.LoadInt32(&c.State))
}

// updateHealthScore recalculates the health score based on various factors
func (c *ConnectionMetrics) updateHealthScore() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	score := int64(100) // Start with perfect score

	// Factor 1: Error rate (errors per message)
	totalMessages := atomic.LoadInt64(&c.MessagesSent) + atomic.LoadInt64(&c.MessagesReceived)
	if totalMessages > 0 {
		errorRate := float64(atomic.LoadInt64(&c.ErrorCount)) / float64(totalMessages)
		score -= int64(errorRate * 50) // Errors can reduce score by up to 50 points
	}

	// Factor 2: Timeout rate
	if totalMessages > 0 {
		timeoutRate := float64(atomic.LoadInt64(&c.TimeoutCount)) / float64(totalMessages)
		score -= int64(timeoutRate * 30) // Timeouts can reduce score by up to 30 points
	}

	// Factor 3: Latency (penalize high latency)
	avgLatency := c.GetAverageLatency()
	if avgLatency > 100 { // More than 100ms is considered degraded
		latencyPenalty := (avgLatency - 100) / 10 // 1 point per 10ms over 100ms
		if latencyPenalty > 20 {
			latencyPenalty = 20 // Cap at 20 points
		}
		score -= int64(latencyPenalty)
	}

	// Factor 4: Inactivity (penalize connections with no recent activity)
	lastActivity := atomic.LoadInt64(&c.LastActivity)
	inactivitySeconds := time.Now().Unix() - lastActivity
	if inactivitySeconds > 300 { // More than 5 minutes of inactivity
		inactivityPenalty := (inactivitySeconds - 300) / 60 // 1 point per minute after 5 minutes
		if inactivityPenalty > 10 {
			inactivityPenalty = 10 // Cap at 10 points
		}
		score -= inactivityPenalty
	}

	// Ensure score stays within bounds
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	atomic.StoreInt64(&c.HealthScore, score)

	// Update state based on score
	var newState ConnectionHealthState
	switch {
	case score >= 80:
		newState = HealthyState
	case score >= 60:
		newState = DegradedState
	case score >= 20:
		newState = UnhealthyState
	default:
		newState = CriticalState
	}

	atomic.StoreInt32(&c.State, int32(newState))
}

// GetSnapshot returns a snapshot of current metrics
func (c *ConnectionMetrics) GetSnapshot() ConnectionMetricsSnapshot {
	return ConnectionMetricsSnapshot{
		ConnectionID:     c.ConnectionID,
		RemoteAddr:       c.RemoteAddr,
		ConnectedAt:      c.ConnectedAt,
		LastActivity:     time.Unix(atomic.LoadInt64(&c.LastActivity), 0),
		MessagesSent:     atomic.LoadInt64(&c.MessagesSent),
		MessagesReceived: atomic.LoadInt64(&c.MessagesReceived),
		BytesSent:        atomic.LoadInt64(&c.BytesSent),
		BytesReceived:    atomic.LoadInt64(&c.BytesReceived),
		AverageLatency:   c.GetAverageLatency(),
		ErrorCount:       atomic.LoadInt64(&c.ErrorCount),
		TimeoutCount:     atomic.LoadInt64(&c.TimeoutCount),
		HealthScore:      atomic.LoadInt64(&c.HealthScore),
		State:            ConnectionHealthState(atomic.LoadInt32(&c.State)),
	}
}

// ConnectionMetricsSnapshot represents a point-in-time snapshot of connection metrics
type ConnectionMetricsSnapshot struct {
	ConnectionID     string
	RemoteAddr       string
	ConnectedAt      time.Time
	LastActivity     time.Time
	MessagesSent     int64
	MessagesReceived int64
	BytesSent        int64
	BytesReceived    int64
	AverageLatency   float64
	ErrorCount       int64
	TimeoutCount     int64
	HealthScore      int64
	State            ConnectionHealthState
}

// ConnectionHealthManager manages health metrics for all connections
type ConnectionHealthManager struct {
	connections map[string]*ConnectionMetrics
	mutex       sync.RWMutex
	registry    *MetricsRegistry
	logger      ifs.ILogger
}

// NewConnectionHealthManager creates a new connection health manager
func NewConnectionHealthManager(registry *MetricsRegistry, logger ifs.ILogger) *ConnectionHealthManager {
	return &ConnectionHealthManager{
		connections: make(map[string]*ConnectionMetrics),
		registry:    registry,
		logger:      logger,
	}
}

// AddConnection registers a new connection for monitoring
func (m *ConnectionHealthManager) AddConnection(connectionID, remoteAddr string) *ConnectionMetrics {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metrics := NewConnectionMetrics(connectionID, remoteAddr)
	m.connections[connectionID] = metrics

	// Update global connection count metric
	connectionCount := m.registry.Gauge("layer8_connections_total", nil)
	connectionCount.Set(int64(len(m.connections)))

	m.logger.Debug("Added connection health monitoring for", connectionID, "to", remoteAddr)
	return metrics
}

// RemoveConnection unregisters a connection from monitoring
func (m *ConnectionHealthManager) RemoveConnection(connectionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.connections, connectionID)

	// Update global connection count metric
	connectionCount := m.registry.Gauge("layer8_connections_total", nil)
	connectionCount.Set(int64(len(m.connections)))

	m.logger.Debug("Removed connection health monitoring for", connectionID)
}

// GetConnection returns metrics for a specific connection
func (m *ConnectionHealthManager) GetConnection(connectionID string) (*ConnectionMetrics, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	metrics, exists := m.connections[connectionID]
	return metrics, exists
}

// GetAllConnections returns metrics for all connections
func (m *ConnectionHealthManager) GetAllConnections() []ConnectionMetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	snapshots := make([]ConnectionMetricsSnapshot, 0, len(m.connections))
	for _, metrics := range m.connections {
		snapshots = append(snapshots, metrics.GetSnapshot())
	}
	return snapshots
}

// GetHealthyConnections returns connections in healthy state
func (m *ConnectionHealthManager) GetHealthyConnections() []ConnectionMetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var healthy []ConnectionMetricsSnapshot
	for _, metrics := range m.connections {
		if metrics.GetState() == HealthyState {
			healthy = append(healthy, metrics.GetSnapshot())
		}
	}
	return healthy
}

// GetUnhealthyConnections returns connections in degraded, unhealthy, or critical state
func (m *ConnectionHealthManager) GetUnhealthyConnections() []ConnectionMetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var unhealthy []ConnectionMetricsSnapshot
	for _, metrics := range m.connections {
		state := metrics.GetState()
		if state != HealthyState {
			unhealthy = append(unhealthy, metrics.GetSnapshot())
		}
	}
	return unhealthy
}