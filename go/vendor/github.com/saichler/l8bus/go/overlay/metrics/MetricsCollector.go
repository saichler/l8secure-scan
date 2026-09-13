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
	"github.com/saichler/l8utils/go/utils/strings"
)

// MetricType represents different types of metrics
type MetricType int

const (
	CounterType MetricType = iota
	GaugeType
	HistogramType
)

// Metric represents a single metric with its value and metadata
type Metric struct {
	Name        string
	Type        MetricType
	Value       int64
	Labels      map[string]string
	LastUpdated time.Time
}

// MetricsRegistry manages all metrics for the system
type MetricsRegistry struct {
	metrics map[string]*Metric
	mutex   sync.RWMutex
	logger  ifs.ILogger
}

// NewMetricsRegistry creates a new metrics registry
func NewMetricsRegistry(logger ifs.ILogger) *MetricsRegistry {
	return &MetricsRegistry{
		metrics: make(map[string]*Metric),
		logger:  logger,
	}
}

// Counter creates or updates a counter metric
func (r *MetricsRegistry) Counter(name string, labels map[string]string) *CounterMetric {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.buildKey(name, labels)
	metric, exists := r.metrics[key]
	if !exists {
		metric = &Metric{
			Name:        name,
			Type:        CounterType,
			Value:       0,
			Labels:      labels,
			LastUpdated: time.Now(),
		}
		r.metrics[key] = metric
	}

	return &CounterMetric{metric: metric}
}

// Gauge creates or updates a gauge metric
func (r *MetricsRegistry) Gauge(name string, labels map[string]string) *GaugeMetric {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.buildKey(name, labels)
	metric, exists := r.metrics[key]
	if !exists {
		metric = &Metric{
			Name:        name,
			Type:        GaugeType,
			Value:       0,
			Labels:      labels,
			LastUpdated: time.Now(),
		}
		r.metrics[key] = metric
	}

	return &GaugeMetric{metric: metric}
}

// Histogram creates or updates a histogram metric
func (r *MetricsRegistry) Histogram(name string, labels map[string]string) *HistogramMetric {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := r.buildKey(name, labels)
	metric, exists := r.metrics[key]
	if !exists {
		metric = &Metric{
			Name:        name,
			Type:        HistogramType,
			Value:       0,
			Labels:      labels,
			LastUpdated: time.Now(),
		}
		r.metrics[key] = metric
	}

	return &HistogramMetric{
		metric:    metric,
		buckets:   make(map[int64]int64),
		sum:       0,
		count:     0,
		bucketsMu: sync.RWMutex{},
	}
}

// GetAllMetrics returns a snapshot of all current metrics
func (r *MetricsRegistry) GetAllMetrics() map[string]*Metric {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	snapshot := make(map[string]*Metric)
	for key, metric := range r.metrics {
		// Create a copy to avoid concurrent access issues
		snapshot[key] = &Metric{
			Name:        metric.Name,
			Type:        metric.Type,
			Value:       atomic.LoadInt64(&metric.Value),
			Labels:      metric.Labels,
			LastUpdated: metric.LastUpdated,
		}
	}
	return snapshot
}

// buildKey creates a unique key for a metric based on name and labels
func (r *MetricsRegistry) buildKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key = strings.New(key, "_", k, "_", v).String()
	}
	return key
}

// CounterMetric represents a counter that only increases
type CounterMetric struct {
	metric *Metric
}

// Inc increments the counter by 1
func (c *CounterMetric) Inc() {
	c.Add(1)
}

// Add increments the counter by the given value
func (c *CounterMetric) Add(value int64) {
	atomic.AddInt64(&c.metric.Value, value)
	c.metric.LastUpdated = time.Now()
}

// Get returns the current counter value
func (c *CounterMetric) Get() int64 {
	return atomic.LoadInt64(&c.metric.Value)
}

// GaugeMetric represents a gauge that can go up and down
type GaugeMetric struct {
	metric *Metric
}

// Set sets the gauge to a specific value
func (g *GaugeMetric) Set(value int64) {
	atomic.StoreInt64(&g.metric.Value, value)
	g.metric.LastUpdated = time.Now()
}

// Inc increments the gauge by 1
func (g *GaugeMetric) Inc() {
	g.Add(1)
}

// Dec decrements the gauge by 1
func (g *GaugeMetric) Dec() {
	g.Add(-1)
}

// Add adds the given value to the gauge
func (g *GaugeMetric) Add(value int64) {
	atomic.AddInt64(&g.metric.Value, value)
	g.metric.LastUpdated = time.Now()
}

// Get returns the current gauge value
func (g *GaugeMetric) Get() int64 {
	return atomic.LoadInt64(&g.metric.Value)
}

// HistogramMetric represents a histogram with configurable buckets
type HistogramMetric struct {
	metric    *Metric
	buckets   map[int64]int64 // bucket upper bound -> count
	sum       int64
	count     int64
	bucketsMu sync.RWMutex
}

// Observe records a new observation in the histogram
func (h *HistogramMetric) Observe(value int64) {
	h.bucketsMu.Lock()
	defer h.bucketsMu.Unlock()

	atomic.AddInt64(&h.sum, value)
	atomic.AddInt64(&h.count, 1)

	// Update buckets (simplified bucket logic)
	bucketBounds := []int64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	for _, bound := range bucketBounds {
		if value <= bound {
			h.buckets[bound]++
			break
		}
	}

	h.metric.LastUpdated = time.Now()
}

// GetSum returns the sum of all observed values
func (h *HistogramMetric) GetSum() int64 {
	return atomic.LoadInt64(&h.sum)
}

// GetCount returns the count of all observed values
func (h *HistogramMetric) GetCount() int64 {
	return atomic.LoadInt64(&h.count)
}

// GetBuckets returns a copy of the current bucket counts
func (h *HistogramMetric) GetBuckets() map[int64]int64 {
	h.bucketsMu.RLock()
	defer h.bucketsMu.RUnlock()

	buckets := make(map[int64]int64)
	for bound, count := range h.buckets {
		buckets[bound] = count
	}
	return buckets
}

// Global metrics registry instance
var globalRegistry *MetricsRegistry
var registryOnce sync.Once

// GetGlobalRegistry returns the global metrics registry
func GetGlobalRegistry(logger ifs.ILogger) *MetricsRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewMetricsRegistry(logger)
	})
	return globalRegistry
}