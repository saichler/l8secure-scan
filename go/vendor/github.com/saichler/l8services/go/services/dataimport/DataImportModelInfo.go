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

package dataimport

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
)

func (h *ModelInfoHandler) Post(elems ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	req, ok := elems.Element().(*l8api.L8ImportModelInfoRequest)
	if !ok {
		return object.NewError("invalid model info request type")
	}

	node, found := vnic.Resources().Introspector().NodeByTypeName(req.ModelType)
	if !found {
		return object.NewError("unknown model type: " + req.ModelType)
	}

	fields := buildFieldInfoList(node, vnic.Resources().Introspector())

	return object.New(nil, &l8api.L8ImportModelInfoResponse{
		ModelType: req.ModelType,
		Fields:    fields,
	})
}
