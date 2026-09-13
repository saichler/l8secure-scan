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
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// handle dispatches requests to the appropriate handler method based on action type.
// For CRUD actions (POST, PUT, PATCH, DELETE, GET), it calls the corresponding handler method.
// For Map-Reduce actions, it delegates to the MapReduce method.
func (this *ServiceManager) handle(h ifs.IServiceHandler, pb ifs.IElements, action ifs.Action, msg *ifs.Message, vnic ifs.IVNic) ifs.IElements {
	if h == nil {
		return object.New(nil, pb)
	}
	_, isMapReduce := h.(ifs.IMapReduceService)
	switch action {
	case ifs.POST:
		return h.Post(pb, vnic)
	case ifs.PUT:
		return h.Put(pb, vnic)
	case ifs.PATCH:
		return h.Patch(pb, vnic)
	case ifs.DELETE:
		return h.Delete(pb, vnic)
	case ifs.GET:
		return h.Get(pb, vnic)
	default:
		if isMapReduce {
			return this.MapReduce(h, pb, action, msg, vnic)
		}
		return object.NewError("invalid action, ignoring")
	}
}
