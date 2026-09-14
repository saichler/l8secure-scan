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

package vnet

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8system"
)

// Services manages service registration and discovery for the VNet.
// It tracks services by name, area, and UUID, supporting various multicast modes
// for service selection (proximity, local, leader, round-robin, all).
type Services struct {
	services   *sync.Map
	routeTable *RouteTable
	roundrobin *sync.Map
}

// newServices creates a new Services manager with the given route table.
func newServices(routeTable *RouteTable) *Services {
	return &Services{services: &sync.Map{}, routeTable: routeTable, roundrobin: &sync.Map{}}
}

// addService registers a service with its name, area, and UUID for discovery.
func (this *Services) addService(data *l8system.L8ServiceData) {
	m1, ok := this.services.Load(data.ServiceName)
	if !ok {
		m1 = &sync.Map{}
		this.services.Store(data.ServiceName, m1)
	}
	area := byte(data.ServiceArea)
	m2, ok := m1.(*sync.Map).Load(area)
	if !ok {
		m2 = &sync.Map{}
		m1.(*sync.Map).Store(area, m2)
	}
	m2.(*sync.Map).Store(data.ServiceUuid, time.Now().UnixMilli())
}

// removeService unregisters services matching the UUIDs in the removed map.
func (this *Services) removeService(removed map[string]string) {
	for uuid, _ := range removed {
		this.services.Range(func(key, value interface{}) bool {
			m1 := value.(*sync.Map)
			m1.Range(func(key, value interface{}) bool {
				m2 := value.(*sync.Map)
				m2.Delete(uuid)
				return true
			})
			return true
		},
		)
	}
}

// serviceUuids returns all service UUIDs for a given service name and area, with their registration timestamps.
func (this *Services) serviceUuids(serviceName string, serviceArea byte) map[string]int64 {
	m1, ok := this.services.Load(serviceName)
	if !ok {
		return map[string]int64{}
	}
	m2, ok := m1.(*sync.Map).Load(serviceArea)
	if !ok {
		return map[string]int64{}
	}

	result := make(map[string]int64)
	m2.(*sync.Map).Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(int64)
		return true
	})

	return result
}

// serviceFor selects a service UUID based on the multicast mode:
// M_Proximity: prefer services on the same VNet as source
// M_Local: prefer service matching the source UUID
// M_Leader: select the earliest registered service (leader election)
// M_RoundRobin: distribute requests across services in rotation
// M_All: select any available service
func (this *Services) serviceFor(serviceName string, serviceArea byte, source string, mode ifs.MulticastMode) string {
	m1, ok := this.services.Load(serviceName)
	if !ok {
		return ""
	}
	m2, ok := m1.(*sync.Map).Load(serviceArea)
	if !ok {
		return ""
	}
	result := ""
	switch mode {
	case ifs.M_Proximity:
		sourceVnet, _ := this.routeTable.vnetOf(source)
		m2.(*sync.Map).Range(func(key, value interface{}) bool {
			k := key.(string)
			result = k // make sure if there is a service,use it anyway even if there is no proximity
			v, _ := this.routeTable.vnetOf(k)
			if v == sourceVnet {
				result = k
				return false
			}
			return true
		})
	case ifs.M_Local:
		m2.(*sync.Map).Range(func(key, value interface{}) bool {
			k := key.(string)
			result = k // make sure if there is a service,use it anyway
			if k == source {
				result = k
				return false
			}
			return true
		})
	case ifs.M_Leader:
		minTime := int64(math.MaxInt64)
		m2.(*sync.Map).Range(func(key, value interface{}) bool {
			k := key.(string)
			v := value.(int64)
			if v < minTime {
				result = k
				minTime = v
			} else if v == minTime {
				if strings.Compare(result, k) == -1 {
					result = k
				}
			}
			return true
		})
	case ifs.M_RoundRobin:
		svR, ok := this.roundrobin.Load(serviceName)
		if !ok {
			svR = &sync.Map{}
			this.roundrobin.Store(serviceName, svR)
		}
		rrS := svR.(*sync.Map)
		found := false
		m2.(*sync.Map).Range(func(key, value interface{}) bool {
			k := key.(string)
			result = k // make sure we have a result in anyway
			_, ok = rrS.Load(k)
			if !ok {
				rrS.Store(k, true)
				found = true
				return false
			}
			return true
		})
		if !found {
			rrS.Clear()
			rrS.Store(result, true)
		}
	case ifs.M_All:
		fallthrough
	default:
		m2.(*sync.Map).Range(func(key, value interface{}) bool {
			k := key.(string)
			result = k // make sure if there is a service,use it anyway
			if k != source {
				result = k
				return false
			}
			return true
		})
	}
	if result == "" {
	}
	return result
}
