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

import (
	"github.com/saichler/l8types/go/types/l8notify"
)

// NetworkMode defines the deployment environment for network configuration.
type NetworkMode int

const (
	// NETWORK_NATIVE is the default mode for bare-metal or VM deployments.
	NETWORK_NATIVE NetworkMode = 1
	// NETWORK_DOCKER is for Docker container deployments.
	NETWORK_DOCKER NetworkMode = 2
	// NETWORK_K8s is for Kubernetes deployments.
	NETWORK_K8s NetworkMode = 3
)

const (
	// SysMsg is the service name for internal system messages.
	SysMsg = "sysMsg"
	// SysAreaPrimary is the primary service area for system messages.
	SysAreaPrimary = byte(99)
	// SysAreaSecondary is the secondary service area for system messages.
	SysAreaSecondary = byte(98)
)

var networkMode NetworkMode = NETWORK_NATIVE

// SetNetworkMode configures the global network deployment mode.
func SetNetworkMode(mode NetworkMode) {
	networkMode = mode
}

// NetworkMode_Native returns true if running in native/bare-metal mode.
func NetworkMode_Native() bool {
	return networkMode == NETWORK_NATIVE
}

// NetworkMode_DOCKER returns true if running in Docker mode.
func NetworkMode_DOCKER() bool {
	return networkMode == NETWORK_DOCKER
}

// NetworkMode_K8s returns true if running in Kubernetes mode.
func NetworkMode_K8s() bool {
	return networkMode == NETWORK_K8s
}

// IVNic (Virtual Network Interface Card) is the primary interface for network communication
// in the Layer 8 system. It provides multiple messaging patterns including unicast, multicast,
// round-robin, proximity-based, and leader-based communication.
type IVNic interface {
	// Start begins the VNic's network operations.
	Start()
	// Shutdown gracefully stops the VNic.
	Shutdown()
	// Name returns the VNic's identifier.
	Name() string
	// SendMessage sends raw bytes over the network.
	SendMessage([]byte) error

	// Unicast sends a message to a specific destination without expecting a response.
	// Parameters: destination UUID, service name, service area, action, payload
	Unicast(string, string, byte, Action, interface{}) error
	// Request sends a unicast message and waits for a response.
	// Parameters: destination UUID, service name, service area, action, payload, timeout (ms), optional AAA ID
	Request(string, string, byte, Action, interface{}, int, ...string) IElements
	// Reply sends a response to a previous request.
	Reply(*Message, IElements) error

	// Multicast sends a message to all providers of a service without expecting responses.
	// Parameters: service name, service area, action, payload
	Multicast(string, byte, Action, interface{}) error

	// RoundRobin sends a message to one provider in a round-robin fashion.
	// Parameters: service name, service area, action, payload
	RoundRobin(string, byte, Action, interface{}) error
	// RoundRobinRequest sends a request to one provider in round-robin fashion and waits for response.
	RoundRobinRequest(string, byte, Action, interface{}, int, ...string) IElements

	// Proximity sends a message to the nearest (lowest latency) service provider.
	Proximity(string, byte, Action, interface{}) error
	// ProximityRequest sends a request to the nearest provider and waits for response.
	ProximityRequest(string, byte, Action, interface{}, int, ...string) IElements

	// Leader sends a message to the elected leader of a service.
	Leader(string, byte, Action, interface{}) error
	// LeaderRequest sends a request to the leader and waits for response.
	LeaderRequest(string, byte, Action, interface{}, int, ...string) IElements

	// Local sends a message to a service provider on the same VNic (in-process).
	Local(string, byte, Action, interface{}) error
	// LocalRequest sends a local request and waits for response.
	LocalRequest(string, byte, Action, interface{}, int, ...string) IElements

	// Forward forwards a message to a specific destination.
	Forward(*Message, string) IElements
	// ServiceAPI returns a simplified API for CRUD operations on a service.
	ServiceAPI(string, byte) ServiceAPI
	// Resources returns the resources context for this VNic.
	Resources() IResources
	// NotifyServiceAdded broadcasts that new services are available.
	NotifyServiceAdded([]string, byte) error
	// NotifyServiceRemoved broadcasts that a service is no longer available.
	NotifyServiceRemoved(string, byte) error
	// PropertyChangeNotification handles property change notifications.
	PropertyChangeNotification(set *l8notify.L8NotificationSet)
	// WaitForConnection blocks until a connection is established.
	WaitForConnection()
	// Running returns true if the VNic is active.
	Running() bool
	// SetResponse stores a response for a pending request.
	SetResponse(*Message, IElements)
	// IsVnet returns true if this VNic is a VNet switch.
	IsVnet() bool
}

// ServiceAPI provides a simplified CRUD interface for interacting with a service.
type ServiceAPI interface {
	// Post creates a new resource.
	Post(interface{}) IElements
	// Put replaces an existing resource.
	Put(interface{}) IElements
	// Patch partially updates a resource.
	Patch(interface{}) IElements
	// Delete removes a resource.
	Delete(interface{}) IElements
	// Get retrieves a resource by key.
	Get(string) IElements
}

// IDatatListener handles incoming data events on a VNic.
type IDatatListener interface {
	// ShutdownVNic is called when a VNic is shutting down.
	ShutdownVNic(nic IVNic)
	// HandleData processes incoming data bytes.
	HandleData([]byte, IVNic)
	// Failed is called when message delivery fails.
	Failed([]byte, IVNic, string)
}
