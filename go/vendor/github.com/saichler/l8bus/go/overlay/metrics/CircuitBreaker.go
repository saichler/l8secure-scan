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
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/saichler/l8types/go/ifs"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	ClosedState CircuitBreakerState = iota
	OpenState
	HalfOpenState
)

func (s CircuitBreakerState) String() string {
	switch s {
	case ClosedState:
		return "closed"
	case OpenState:
		return "open"
	case HalfOpenState:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures     int           // Maximum failures before opening
	ResetTimeout    time.Duration // Time to wait before attempting reset
	Timeout         time.Duration // Timeout for operations
	MaxConcurrency  int           // Maximum concurrent operations
	SuccessThreshold int          // Successful calls needed to close from half-open
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		MaxFailures:      5,
		ResetTimeout:     30 * time.Second,
		Timeout:          5 * time.Second,
		MaxConcurrency:   100,
		SuccessThreshold: 3,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name         string
	config       *CircuitBreakerConfig
	state        int32 // CircuitBreakerState
	failures     int64
	successes    int64
	lastFailTime int64 // Unix timestamp
	requests     int64 // Current concurrent requests
	mutex        sync.RWMutex
	logger       ifs.ILogger
	registry     *MetricsRegistry
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config *CircuitBreakerConfig, registry *MetricsRegistry, logger ifs.ILogger) *CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}

	cb := &CircuitBreaker{
		name:     name,
		config:   config,
		state:    int32(ClosedState),
		failures: 0,
		registry: registry,
		logger:   logger,
	}

	// Register metrics
	registry.Counter("layer8_circuit_breaker_requests_total", map[string]string{"name": name, "state": "closed"})
	registry.Counter("layer8_circuit_breaker_requests_total", map[string]string{"name": name, "state": "open"})
	registry.Counter("layer8_circuit_breaker_requests_total", map[string]string{"name": name, "state": "half_open"})
	registry.Gauge("layer8_circuit_breaker_failures", map[string]string{"name": name})
	registry.Gauge("layer8_circuit_breaker_state", map[string]string{"name": name})

	return cb
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	// Check if we can execute
	if err := cb.canExecute(); err != nil {
		return nil, err
	}

	// Increment concurrent requests
	concurrent := atomic.AddInt64(&cb.requests, 1)
	defer atomic.AddInt64(&cb.requests, -1)

	// Check concurrency limit
	if concurrent > int64(cb.config.MaxConcurrency) {
		cb.recordFailure()
		return nil, errors.New("circuit breaker: too many concurrent requests")
	}

	// Record request metric
	state := cb.GetState()
	requestCounter := cb.registry.Counter("layer8_circuit_breaker_requests_total", 
		map[string]string{"name": cb.name, "state": state.String()})
	requestCounter.Inc()

	// Execute with timeout using context for proper cancellation
	ctx, cancel := context.WithTimeout(context.Background(), cb.config.Timeout)
	defer cancel()

	resultChan := make(chan interface{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := fn()
		// Check if context was cancelled before sending to channels
		// This prevents goroutine from blocking on channel send after timeout
		select {
		case <-ctx.Done():
			return // Timeout already fired, exit without sending
		default:
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}
	}()

	select {
	case result := <-resultChan:
		cb.recordSuccess()
		return result, nil
	case err := <-errorChan:
		cb.recordFailure()
		return nil, err
	case <-ctx.Done():
		cb.recordFailure()
		return nil, errors.New("circuit breaker: operation timeout")
	}
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() error {
	state := cb.GetState()
	
	switch state {
	case ClosedState:
		return nil
	case OpenState:
		// Check if enough time has passed to try half-open
		lastFail := atomic.LoadInt64(&cb.lastFailTime)
		if time.Now().Unix()-lastFail >= int64(cb.config.ResetTimeout.Seconds()) {
			cb.transitionToHalfOpen()
			return nil
		}
		return errors.New("circuit breaker is open")
	case HalfOpenState:
		return nil
	default:
		return errors.New("circuit breaker in unknown state")
	}
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess() {
	successes := atomic.AddInt64(&cb.successes, 1)
	state := cb.GetState()

	switch state {
	case HalfOpenState:
		// If we have enough successes in half-open state, close the circuit
		if successes >= int64(cb.config.SuccessThreshold) {
			cb.transitionToClosed()
		}
	case ClosedState:
		// Reset failure count on success
		atomic.StoreInt64(&cb.failures, 0)
	}

	cb.updateMetrics()
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure() {
	failures := atomic.AddInt64(&cb.failures, 1)
	atomic.StoreInt64(&cb.lastFailTime, time.Now().Unix())
	
	state := cb.GetState()
	
	switch state {
	case ClosedState:
		if failures >= int64(cb.config.MaxFailures) {
			cb.transitionToOpen()
		}
	case HalfOpenState:
		// Any failure in half-open state returns to open
		cb.transitionToOpen()
	}

	cb.updateMetrics()
	cb.logger.Warning("Circuit breaker", cb.name, "recorded failure, total failures:", failures)
}

// transitionToClosed transitions the circuit breaker to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	atomic.StoreInt32(&cb.state, int32(ClosedState))
	atomic.StoreInt64(&cb.failures, 0)
	atomic.StoreInt64(&cb.successes, 0)
	
	cb.logger.Debug("Circuit breaker", cb.name, "transitioned to CLOSED state")
	cb.updateMetrics()
}

// transitionToOpen transitions the circuit breaker to open state
func (cb *CircuitBreaker) transitionToOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	atomic.StoreInt32(&cb.state, int32(OpenState))
	atomic.StoreInt64(&cb.successes, 0)
	
	cb.logger.Warning("Circuit breaker", cb.name, "transitioned to OPEN state")
	cb.updateMetrics()
}

// transitionToHalfOpen transitions the circuit breaker to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	atomic.StoreInt32(&cb.state, int32(HalfOpenState))
	atomic.StoreInt64(&cb.successes, 0)
	
	cb.logger.Debug("Circuit breaker", cb.name, "transitioned to HALF-OPEN state")
	cb.updateMetrics()
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}

// GetFailures returns the current failure count
func (cb *CircuitBreaker) GetFailures() int64 {
	return atomic.LoadInt64(&cb.failures)
}

// GetConcurrentRequests returns the current number of concurrent requests
func (cb *CircuitBreaker) GetConcurrentRequests() int64 {
	return atomic.LoadInt64(&cb.requests)
}

// updateMetrics updates the circuit breaker metrics
func (cb *CircuitBreaker) updateMetrics() {
	failures := atomic.LoadInt64(&cb.failures)
	state := cb.GetState()

	failureGauge := cb.registry.Gauge("layer8_circuit_breaker_failures", map[string]string{"name": cb.name})
	failureGauge.Set(failures)

	stateGauge := cb.registry.Gauge("layer8_circuit_breaker_state", map[string]string{"name": cb.name})
	stateGauge.Set(int64(state))
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	return CircuitBreakerStats{
		Name:               cb.name,
		State:              cb.GetState(),
		Failures:           atomic.LoadInt64(&cb.failures),
		Successes:          atomic.LoadInt64(&cb.successes),
		ConcurrentRequests: atomic.LoadInt64(&cb.requests),
		LastFailTime:       time.Unix(atomic.LoadInt64(&cb.lastFailTime), 0),
	}
}

// CircuitBreakerStats represents circuit breaker statistics
type CircuitBreakerStats struct {
	Name               string
	State              CircuitBreakerState
	Failures           int64
	Successes          int64
	ConcurrentRequests int64
	LastFailTime       time.Time
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	registry *MetricsRegistry
	logger   ifs.ILogger
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(registry *MetricsRegistry, logger ifs.ILogger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		registry: registry,
		logger:   logger,
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (m *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}

	breaker := NewCircuitBreaker(name, config, m.registry, m.logger)
	m.breakers[name] = breaker
	return breaker
}

// Get returns an existing circuit breaker
func (m *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	breaker, exists := m.breakers[name]
	return breaker, exists
}

// GetAll returns all circuit breakers
func (m *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	breakers := make(map[string]*CircuitBreaker)
	for name, breaker := range m.breakers {
		breakers[name] = breaker
	}
	return breakers
}

// Remove removes a circuit breaker from the manager
// Note: This does not clean up metrics in the registry - that remains a known limitation
func (m *CircuitBreakerManager) Remove(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.breakers[name]; exists {
		delete(m.breakers, name)
		m.logger.Debug("Removed circuit breaker:", name)
	}
}