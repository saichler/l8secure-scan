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

package health

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
)

// RoundRobin implements round-robin load balancing across service participants.
// It distributes requests evenly across all available instances of a service.
type RoundRobin struct {
	participants []string
	index        int
}

// NewRoundRobin creates a new RoundRobin balancer for the specified service.
// It populates the participant list from the health service registry.
func NewRoundRobin(serviceName string, serviceArea byte, r ifs.IResources) *RoundRobin {
	rr := &RoundRobin{}
	rr.participants = make([]string, 0)
	pMap := Participants(serviceName, serviceArea, r)
	for uuid, _ := range pMap {
		rr.participants = append(rr.participants, uuid)
	}
	return rr
}

// Next returns the UUID of the next participant in the round-robin sequence.
func (this *RoundRobin) Next() string {
	if this.index >= len(this.participants) {
		this.index = 0
	}
	next := this.participants[this.index]
	this.index++
	return next
}

// Participants returns a map of UUIDs for all nodes providing the specified service.
func Participants(serviceName string, serviceArea byte, r ifs.IResources) map[string]bool {
	hc, _ := HealthServiceCache(r)
	all := hc.All()
	result := make(map[string]bool)
	for _, h := range all {
		hp := h.(*l8health.L8Health)
		if hp.Services != nil && hp.Services.ServiceToAreas != nil {
			areas, ok := hp.Services.ServiceToAreas[serviceName]
			if ok && areas.Areas != nil {
				_, ok2 := areas.Areas[int32(serviceArea)]
				if ok2 {
					result[hp.AUuid] = true
				}
			}
		}
	}
	return result
}
