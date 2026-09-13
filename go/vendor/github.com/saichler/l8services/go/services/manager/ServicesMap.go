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
	"strconv"
	"strings"
	"sync"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8services"
)

// ServicesMap is a thread-safe registry for service handlers,
// indexed by a combination of service name and area.
type ServicesMap struct {
	services *sync.Map
}

// NewServicesMap creates an empty ServicesMap with initialized sync.Map.
func NewServicesMap() *ServicesMap {
	newMap := &ServicesMap{}
	newMap.services = &sync.Map{}
	return newMap
}

// put stores a service handler indexed by service name and area.
func (mp *ServicesMap) put(serviceName string, serviceArea byte, handler ifs.IServiceHandler) {
	key := serviceKey(serviceName, serviceArea)
	mp.services.Store(key, handler)
}

// get retrieves a service handler by name and area. Returns (handler, true) if found.
func (mp *ServicesMap) get(serviceName string, serviceArea byte) (ifs.IServiceHandler, bool) {
	key := serviceKey(serviceName, serviceArea)
	value, ok := mp.services.Load(key)
	if value != nil {
		return value.(ifs.IServiceHandler), ok
	}
	return nil, ok
}

// del removes and returns a service handler by name and area.
func (mp *ServicesMap) del(serviceName string, serviceArea byte) (ifs.IServiceHandler, bool) {
	key := serviceKey(serviceName, serviceArea)
	value, ok := mp.services.LoadAndDelete(key)
	if value != nil {
		return value.(ifs.IServiceHandler), ok
	}
	return nil, ok
}

// contains checks if a service handler exists for the given name and area.
func (mp *ServicesMap) contains(serviceName string, serviceArea byte) bool {
	key := serviceKey(serviceName, serviceArea)
	_, ok := mp.services.Load(key)
	return ok
}

// webServices collects all web service interfaces from registered handlers.
func (mp *ServicesMap) webServices() []ifs.IWebService {
	result := make([]ifs.IWebService, 0)
	mp.services.Range(func(key, value interface{}) bool {
		svc := value.(ifs.IServiceHandler)
		if svc.WebService() != nil {
			result = append(result, svc.WebService())
		}
		return true
	})
	return result
}

// serviceKey generates a unique key by combining service name and area.
func serviceKey(serviceName string, serviceArea byte) string {
	buff := bytes.Buffer{}
	buff.WriteString(serviceName)
	buff.WriteString("--")
	buff.WriteString(strconv.Itoa(int(serviceArea)))
	return buff.String()
}

// serviceList converts the services map to an L8Services protobuf structure
// organized by service name and their associated areas.
func (mp *ServicesMap) serviceList() *l8services.L8Services {
	s := &l8services.L8Services{}
	s.ServiceToAreas = make(map[string]*l8services.L8ServiceAreas)
	mp.services.Range(func(key, value interface{}) bool {
		str := key.(string)
		index := strings.Index(str, "--")
		name := str[0:index]
		areaStr := str[index+2:]
		areaInt, _ := strconv.Atoi(areaStr)
		sv, ok := s.ServiceToAreas[name]
		if !ok {
			sv = &l8services.L8ServiceAreas{}
			sv.Areas = make(map[int32]bool)
			s.ServiceToAreas[name] = sv
		}
		sv.Areas[int32(areaInt)] = true
		return true
	})
	return s
}
