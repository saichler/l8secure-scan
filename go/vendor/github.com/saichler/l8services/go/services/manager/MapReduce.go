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
	"strconv"
	"sync"

	"github.com/saichler/l8types/go/ifs"
)

// MapReduce implements the Map-Reduce pattern for distributed processing.
// It converts Map-Reduce actions to standard CRUD actions, distributes requests
// to all participants, and merges the results using the service's Merge method.
func (this *ServiceManager) MapReduce(h ifs.IServiceHandler, pb ifs.IElements, action ifs.Action, msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	cMsg := msg.Clone()
	switch action {
	case ifs.MapR_POST:
		cMsg.SetAction(ifs.POST)
	case ifs.MapR_PUT:
		cMsg.SetAction(ifs.PUT)
	case ifs.MapR_DELETE:
		cMsg.SetAction(ifs.DELETE)
	case ifs.MapR_PATCH:
		cMsg.SetAction(ifs.PATCH)
	case ifs.MapR_GET:
		cMsg.SetAction(ifs.GET)
	default:
		panic("unknown action " + strconv.Itoa(int(action)))
	}
	results := this.PeerRequest(cMsg, vnic)
	mh := h.(ifs.IMapReduceService)
	return mh.Merge(results)
}

// PeerRequest sends a message to all participants of a service concurrently
// and collects their responses. Returns a map of UUID to response elements.
func (this *ServiceManager) PeerRequest(msg *ifs.Message, nic ifs.IVNic) map[string]ifs.IElements {
	edges := this.GetParticipants(msg.ServiceName(), msg.ServiceArea())
	this.resources.Logger().Debug("Edges for ", msg.ServiceName(), " area ", msg.ServiceArea(), " ", len(edges))
	wg := sync.WaitGroup{}
	mtx := sync.Mutex{}
	results := make(map[string]ifs.IElements)
	wg.Add(len(edges))
	for uuid, _ := range edges {
		go func(uuid string) {
			defer wg.Done()
			resp := nic.Forward(msg, uuid)
			mtx.Lock()
			results[uuid] = resp
			mtx.Unlock()
		}(uuid)
	}
	wg.Wait()
	return results
}
