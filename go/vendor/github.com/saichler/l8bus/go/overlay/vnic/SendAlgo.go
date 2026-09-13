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
	"github.com/saichler/l8types/go/ifs"
)

// Proximity sends a message to a service instance on the nearest (same VNet) network segment.
func (this *VirtualNetworkInterface) Proximity(serviceName string, serviceArea byte, action ifs.Action, any interface{}) error {
	return this.multicast(ifs.P8, ifs.M_Proximity, serviceName, serviceArea, action, any)
}

// ProximityRequest sends a request to the nearest service instance and waits for a response.
func (this *VirtualNetworkInterface) ProximityRequest(serviceName string, serviceArea byte, action ifs.Action, any interface{}, timeout int, tokens ...string) ifs.IElements {
	return this.request("", serviceName, serviceArea, action, any, ifs.P8, ifs.M_Proximity, timeout, tokens...)
}

// RoundRobin sends a message to service instances in rotation for load balancing.
func (this *VirtualNetworkInterface) RoundRobin(serviceName string, serviceArea byte, action ifs.Action, any interface{}) error {
	return this.multicast(ifs.P8, ifs.M_RoundRobin, serviceName, serviceArea, action, any)
}

// RoundRobinRequest sends a request using round-robin selection and waits for a response.
func (this *VirtualNetworkInterface) RoundRobinRequest(serviceName string, serviceArea byte, action ifs.Action, any interface{}, timeout int, tokens ...string) ifs.IElements {
	return this.request("", serviceName, serviceArea, action, any, ifs.P8, ifs.M_RoundRobin, timeout, tokens...)
}

// Local sends a message to a service instance on the local VNic.
func (this *VirtualNetworkInterface) Local(serviceName string, serviceArea byte, action ifs.Action, any interface{}) error {
	return this.multicast(ifs.P8, ifs.M_Local, serviceName, serviceArea, action, any)
}

// LocalRequest sends a request to a local service instance and waits for a response.
func (this *VirtualNetworkInterface) LocalRequest(serviceName string, serviceArea byte, action ifs.Action, any interface{}, timeout int, tokens ...string) ifs.IElements {
	return this.request("", serviceName, serviceArea, action, any, ifs.P8, ifs.M_Local, timeout, tokens...)
}

// Leader sends a message to the leader service instance (earliest registered).
func (this *VirtualNetworkInterface) Leader(serviceName string, serviceArea byte, action ifs.Action, any interface{}) error {
	return this.multicast(ifs.P8, ifs.M_Leader, serviceName, serviceArea, action, any)
}

// LeaderRequest sends a request to the leader service instance and waits for a response.
func (this *VirtualNetworkInterface) LeaderRequest(serviceName string, serviceArea byte, action ifs.Action, any interface{}, timeout int, tokens ...string) ifs.IElements {
	return this.request("", serviceName, serviceArea, action, any, ifs.P8, ifs.M_Leader, timeout, tokens...)
}
