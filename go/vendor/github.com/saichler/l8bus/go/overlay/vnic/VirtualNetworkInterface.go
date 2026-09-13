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

package vnic

import (
	"errors"
	"net"
	"os"
	"sync"
	"time"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8bus/go/overlay/metrics"
	"github.com/saichler/l8bus/go/overlay/plugins"
	"github.com/saichler/l8bus/go/overlay/protocol"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8utils/go/utils/ipsegment"
	requests2 "github.com/saichler/l8utils/go/utils/requests"
	"github.com/saichler/l8utils/go/utils/strings"
)

// VirtualNetworkInterface represents a network interface that connects to a VNet switch.
// It provides bidirectional communication, health monitoring, metrics collection,
// and circuit breaker patterns for resilient distributed communication.
type VirtualNetworkInterface struct {
	// Resources for this VNic such as registry, security & config
	resources ifs.IResources
	// The socket connection
	conn net.Conn
	// The socket connection mutex
	connMtx *sync.Mutex
	// is running
	running bool
	// Sub components/go routines
	components *SubComponents
	// The Protocol
	protocol *protocol.Protocol
	// Name for this VNic expressing the connection path in aside -->> zside
	name string
	// Indicates if this vnic in on the switch internal, hence need no keep alive
	IsVNet bool
	// Last reconnect attempt
	last_reconnect_attempt int64

	requests *requests2.Requests

	healthStatistics      *HealthStatistics
	connectionMetrics     *metrics.ConnectionMetrics
	circuitBreaker        *metrics.CircuitBreaker
	circuitBreakerManager *metrics.CircuitBreakerManager
	circuitBreakerName    string
	metricsRegistry       *metrics.MetricsRegistry
	connected             bool
}

// NewVirtualNetworkInterface creates a new VNic instance. If conn is nil, the VNic
// will actively connect to a VNet switch. If conn is provided, the VNic operates
// in server mode, receiving connections from the VNet.
func NewVirtualNetworkInterface(resources ifs.IResources, conn net.Conn) *VirtualNetworkInterface {
	vnic := &VirtualNetworkInterface{}
	vnic.conn = conn
	vnic.resources = resources
	vnic.connMtx = &sync.Mutex{}
	vnic.protocol = protocol.New(vnic)
	vnic.components = newSubomponents()
	vnic.components.addComponent(newRX(vnic))
	vnic.components.addComponent(newTX(vnic))
	vnic.components.addComponent(newKeepAlive(vnic))
	vnic.requests = requests2.NewRequests()
	vnic.healthStatistics = &HealthStatistics{}
	services := vnic.resources.SysConfig().Services
	if services == nil {
		services = &l8services.L8Services{}
		vnic.resources.SysConfig().Services = services
	}
	if services.ServiceToAreas == nil {
		services.ServiceToAreas = make(map[string]*l8services.L8ServiceAreas)
	}
	services.ServiceToAreas[health.ServiceName] = &l8services.L8ServiceAreas{}
	if services.ServiceToAreas[health.ServiceName].Areas == nil {
		services.ServiceToAreas[health.ServiceName].Areas = make(map[int32]bool)
	}
	services.ServiceToAreas[health.ServiceName].Areas[0] = true

	// Initialize metrics system
	vnic.metricsRegistry = metrics.GetGlobalRegistry(resources.Logger())

	// Initialize connection metrics if we have a connection
	if conn != nil {
		remoteAddr := conn.RemoteAddr().String()
		connectionID := ifs.NewUuid()
		vnic.connectionMetrics = metrics.NewConnectionMetrics(connectionID, remoteAddr)

		// Initialize circuit breaker for this connection
		vnic.circuitBreakerManager = metrics.NewCircuitBreakerManager(vnic.metricsRegistry, resources.Logger())
		cbConfig := metrics.DefaultCircuitBreakerConfig()
		vnic.circuitBreakerName = strings.New("vnic_", connectionID).String()
		vnic.circuitBreaker = vnic.circuitBreakerManager.GetOrCreate(vnic.circuitBreakerName, cbConfig)
	}

	if vnic.resources.SysConfig().LocalUuid == "" {
		vnic.resources.SysConfig().LocalUuid = ifs.NewUuid()
	}

	if conn == nil {
		health.Activate(vnic, false)
		if resources.SysConfig().RemoteVnet == "" {
			sla := ifs.NewServiceLevelAgreement(&plugins.PluginService{}, plugins.ServiceName, 0, false, nil)
			vnic.resources.Services().Activate(sla, vnic)
		}
	}
	vnic.resources.Events().SetVNic(vnic)
	return vnic
}

// IsVnet returns false as this is a VNic, not a VNet switch.
func (this *VirtualNetworkInterface) IsVnet() bool {
	return false
}

// Start initiates the VNic, either connecting to a VNet switch or accepting
// an incoming connection. It starts all sub-components (TX, RX, KeepAlive).
func (this *VirtualNetworkInterface) Start() {
	this.running = true
	if this.conn == nil {
		this.connectToSwitch()
	} else {
		this.receiveConnection()
	}
	this.name = strings.New(this.resources.SysConfig().LocalAlias, " -->> ", this.resources.SysConfig().RemoteAlias).String()
}

func (this *VirtualNetworkInterface) connectToSwitch() {
	for this.running {
		err := this.connect()
		if err == nil {
			break
		}
		this.resources.Logger().Error("Failed to connect to vnet: ", this.resources.SysConfig().LocalAlias,
			err.Error(), ", retrying in 5 seconds...")
		time.Sleep(time.Second * 5)
	}
	if !this.running {
		return
	}
	this.components.start()
	this.connected = true
}

func (this *VirtualNetworkInterface) connect() error {
	// Dial the destination and validate the secret and key
	destination := this.resources.SysConfig().RemoteVnet
	if destination == "" {
		destination = ipsegment.MachineIP
		if ifs.NetworkMode_K8s() {
			destination = os.Getenv("NODE_IP")
		} else if ifs.NetworkMode_DOCKER() {
			// inside a containet the switch ip will be the external subnet + ".1"
			// for example if the address of the container is 172.1.1.112, the switch will be accessible
			// via 172.1.1.1
			subnet := ipsegment.IpSegment.ExternalSubnet()
			destination = strings.New(subnet, ".1").String()
		}
	} else {
	}

	this.resources.Logger().Debug("Trying to connect to vnet at IP - ", destination)
	// Try to dial to the switch
	conn, err := this.resources.Security().CanDial(destination, this.resources.SysConfig().VnetPort)
	if err != nil {
		return errors.New(strings.New("Error connecting to the vnet: ", err.Error()).String())
	}
	// Verify that the switch accepts this connection
	if this.resources.SysConfig().LocalUuid == "" {
		return errors.New("local UUID is empty, cannot validate connection")
	}
	this.syncServicesWithConfig()
	// Save local services before the handshake because ExecuteProtocol
	// overwrites SysConfig().Services with the remote side's service list.
	// Restoring afterward prevents contamination on reconnect.
	localServices := this.resources.SysConfig().Services
	err = this.resources.Security().ValidateConnection(conn, this.resources.SysConfig())
	this.resources.SysConfig().Services = localServices
	if err != nil {
		return errors.New(strings.New("Error validating connection: ", err.Error()).String())
	}
	this.conn = conn
	this.resources.SysConfig().Address = conn.LocalAddr().String()
	this.resources.Logger().Debug("Connected!")
	return nil
}

func (this *VirtualNetworkInterface) syncServicesWithConfig() {
	s1 := this.resources.Services().Services()
	s2 := this.resources.SysConfig().Services
	if s2 == nil {
		this.resources.SysConfig().Services = s1
		return
	}
	for k, v := range s1.ServiceToAreas {
		for k1, _ := range v.Areas {
			_, ok := s2.ServiceToAreas[k]
			if !ok {
				s2.ServiceToAreas[k] = &l8services.L8ServiceAreas{}
				s2.ServiceToAreas[k].Areas = make(map[int32]bool)
			}
			s2.ServiceToAreas[k].Areas[k1] = true
		}
	}
}

func (this *VirtualNetworkInterface) receiveConnection() {
	this.IsVNet = true
	this.resources.SysConfig().Address = this.conn.RemoteAddr().String()
	this.components.start()
}

// Shutdown gracefully stops the VNic, closing the connection and cleaning up
// all resources including circuit breakers and sub-components.
func (this *VirtualNetworkInterface) Shutdown() {
	this.resources.Logger().Debug("Shutdown was called on ", this.resources.SysConfig().LocalAlias)
	this.running = false
	if this.conn != nil {
		this.conn.Close()
	}
	this.components.shutdown()

	// Clean up circuit breaker to prevent memory leak
	if this.circuitBreakerManager != nil && this.circuitBreakerName != "" {
		this.circuitBreakerManager.Remove(this.circuitBreakerName)
	}

	if this.resources.DataListener() != nil {
		go this.resources.DataListener().ShutdownVNic(this)
	}
}

// Name returns the connection path name in the format "local -->> remote".
func (this *VirtualNetworkInterface) Name() string {
	if this.name == "" {
		this.name = strings.New(this.resources.SysConfig().LocalUuid,
			" -->> ",
			this.resources.SysConfig().RemoteUuid).String()
	}
	return this.name
}

// SendMessage sends a message through the TX component to the connected VNet.
func (this *VirtualNetworkInterface) SendMessage(data []byte) error {
	return this.components.TX().SendMessage(data)
}

// ServiceAPI returns an API interface for communicating with a specific service.
// It supports GET, POST, PUT, DELETE operations with request/reply semantics.
func (this *VirtualNetworkInterface) ServiceAPI(serviceName string, serviceArea byte) ifs.ServiceAPI {
	return NewAPI(serviceName, serviceArea, this, false, false)
}

// Resources returns the IResources instance for this VNic.
func (this *VirtualNetworkInterface) Resources() ifs.IResources {
	return this.resources
}

func (this *VirtualNetworkInterface) reconnect() {
	this.connMtx.Lock()
	defer this.connMtx.Unlock()
	if !this.running {
		return
	}
	if time.Now().Unix()-this.last_reconnect_attempt < 5 {
		return
	}
	this.last_reconnect_attempt = time.Now().Unix()

	this.resources.Logger().Debug("***** Trying to reconnect to ", this.resources.SysConfig().RemoteAlias, " *****")

	if this.conn != nil {
		this.conn.Close()
	}

	err := this.connect()
	if err != nil {
		this.resources.Logger().Error("***** Failed to reconnect to ", this.resources.SysConfig().RemoteAlias, " *****")
	} else {
		this.resources.Logger().Debug("***** Reconnected to ", this.resources.SysConfig().RemoteAlias, " *****")
	}
}

// WaitForConnection blocks until the VNic is connected to the VNet and
// health information is available. It also activates the security provider.
func (this *VirtualNetworkInterface) WaitForConnection() {
	for !this.connected {
		time.Sleep(time.Millisecond * 100)
	}
	hp := health.HealthOf(this.resources.SysConfig().LocalUuid, this.resources)
	for hp == nil {
		time.Sleep(time.Millisecond * 100)
		hp = health.HealthOf(this.resources.SysConfig().LocalUuid, this.resources)
	}
	secService, ok := this.resources.Security().(ifs.ISecurityProviderActivate)
	if ok {
		secService.Activate(this)
	} else {
	}
}

// Running returns true if the VNic is currently active.
func (this *VirtualNetworkInterface) Running() bool {
	return this.running
}

// RecordMessageSent records metrics for an outgoing message
func (this *VirtualNetworkInterface) RecordMessageSent(bytes int64) {
	if this.connectionMetrics != nil {
		this.connectionMetrics.RecordMessageSent(bytes)
	}

	// Update global metrics
	if this.metricsRegistry != nil {
		sentCounter := this.metricsRegistry.Counter("layer8_messages_sent_total",
			map[string]string{"vnic_id": this.resources.SysConfig().LocalUuid})
		sentCounter.Inc()

		bytesCounter := this.metricsRegistry.Counter("layer8_bytes_sent_total",
			map[string]string{"vnic_id": this.resources.SysConfig().LocalUuid})
		bytesCounter.Add(bytes)
	}
}

// RecordMessageReceived records metrics for an incoming message
func (this *VirtualNetworkInterface) RecordMessageReceived(bytes int64) {
	if this.connectionMetrics != nil {
		this.connectionMetrics.RecordMessageReceived(bytes)
	}

	// Update global metrics
	if this.metricsRegistry != nil {
		receivedCounter := this.metricsRegistry.Counter("layer8_messages_received_total",
			map[string]string{"vnic_id": this.resources.SysConfig().LocalUuid})
		receivedCounter.Inc()

		bytesCounter := this.metricsRegistry.Counter("layer8_bytes_received_total",
			map[string]string{"vnic_id": this.resources.SysConfig().LocalUuid})
		bytesCounter.Add(bytes)
	}
}

// RecordError records a connection error
func (this *VirtualNetworkInterface) RecordError() {
	if this.connectionMetrics != nil {
		this.connectionMetrics.RecordError()
	}

	if this.metricsRegistry != nil {
		errorCounter := this.metricsRegistry.Counter("layer8_connection_errors_total",
			map[string]string{"vnic_id": this.resources.SysConfig().LocalUuid})
		errorCounter.Inc()
	}
}

// RecordLatency records a latency measurement
func (this *VirtualNetworkInterface) RecordLatency(latencyMs int64) {
	if this.connectionMetrics != nil {
		this.connectionMetrics.RecordLatency(latencyMs)
	}

	if this.metricsRegistry != nil {
		latencyHistogram := this.metricsRegistry.Histogram("layer8_message_latency_ms",
			map[string]string{"vnic_id": this.resources.SysConfig().LocalUuid})
		latencyHistogram.Observe(latencyMs)
	}
}

// GetConnectionHealth returns the current connection health score
func (this *VirtualNetworkInterface) GetConnectionHealth() int64 {
	if this.connectionMetrics != nil {
		return this.connectionMetrics.GetHealthScore()
	}
	return 100 // Default to healthy if no metrics
}

// GetCircuitBreaker returns the circuit breaker for this connection
func (this *VirtualNetworkInterface) GetCircuitBreaker() *metrics.CircuitBreaker {
	return this.circuitBreaker
}

// ExecuteWithCircuitBreaker executes a function with circuit breaker protection
func (this *VirtualNetworkInterface) ExecuteWithCircuitBreaker(fn func() (interface{}, error)) (interface{}, error) {
	if this.circuitBreaker != nil {
		return this.circuitBreaker.Execute(fn)
	}
	// Fallback to direct execution if no circuit breaker
	return fn()
}

// SetResponse sets the response for a pending request identified by the message sequence.
func (this *VirtualNetworkInterface) SetResponse(msg *ifs.Message, pb ifs.IElements) {
	request := this.requests.GetRequest(msg.Sequence(), this.resources.SysConfig().LocalUuid)
	request.SetResponse(pb)
}
