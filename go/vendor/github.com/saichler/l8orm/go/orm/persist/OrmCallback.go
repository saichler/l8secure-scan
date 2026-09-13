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
package persist

import (
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// Before is called before a database operation is executed.
// It invokes the service callback's Before method for each element, allowing
// custom validation, transformation, or cancellation of the operation.
// Returns the potentially modified elements and a boolean indicating whether to proceed.
func (this *OrmService) Before(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) (ifs.IElements, bool) {
	if this.sla.Callback() != nil {
		elems := make([]interface{}, 0)
		for _, elem := range pb.Elements() {
			before, cont, err := this.sla.Callback().Before(elem, action, pb.Notification(), vnic)
			if err != nil {
				return object.NewError(err.Error()), true
			}
			if !cont {
				return nil, false
			}
			if before != nil {
				arr, ok := before.([]interface{})
				if ok {
					for _, item := range arr {
						elems = append(elems, item)
					}
				} else {
					elems = append(elems, before)
				}
			} else {
				elems = append(elems, elem)
			}
		}
		if pb.Notification() {
			return object.NewNotify(elems), true
		}
		return object.New(nil, elems), true
	}
	return pb, true
}

// After is called after a database operation has completed.
// It invokes the service callback's After method for each element, allowing
// post-processing, logging, or notification of the completed operation.
// Returns the potentially modified elements and a boolean indicating success.
func (this *OrmService) After(action ifs.Action, pb ifs.IElements, vnic ifs.IVNic) (ifs.IElements, bool) {
	if this.sla.Callback() != nil {
		elems := make([]interface{}, len(pb.Elements()))
		for i, elem := range pb.Elements() {
			after, cont, err := this.sla.Callback().After(elem, action, pb.Notification(), vnic)
			if err != nil {
				return object.NewError(err.Error()), true
			}
			if !cont {
				return nil, false
			}
			if after != nil {
				elems[i] = after
			} else {
				elems[i] = elem
			}
		}
		if pb.Notification() {
			return object.NewNotify(elems), true
		}
		return object.New(nil, elems), true
	}
	return pb, true
}
