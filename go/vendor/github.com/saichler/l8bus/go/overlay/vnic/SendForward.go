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
	"reflect"

	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8utils/go/utils/strings"
	"google.golang.org/protobuf/proto"
)

// Forward sends a message to a destination and returns the response.
// It preserves transaction context from the original message.
func (this *VirtualNetworkInterface) Forward(msg *ifs.Message, destination string) ifs.IElements {
	pb, err := this.protocol.ElementsOf(msg)
	if err != nil {
		return object.NewError(err.Error())
	}

	timeout := 15
	if msg.Tr_Timeout() > 0 {
		timeout = int(msg.Tr_Timeout())
	}

	request, err := this.requests.NewRequest(this.protocol.NextMessageNumber(), this.resources.SysConfig().LocalUuid, timeout, this.resources.Logger())
	if err != nil {
		return object.NewError(err.Error())
	}

	defer this.requests.DelRequest(request.MsgNum(), request.MsgSource())

	e := this.components.TX().Unicast(destination, msg.ServiceName(), msg.ServiceArea(), msg.Action(),
		pb, ifs.P8, ifs.M_All, true, false, request.MsgNum(),
		msg.Tr_State(), msg.Tr_Id(), msg.Tr_ErrMsg(),
		msg.Tr_Created(), msg.Tr_Queued(), msg.Tr_Running(), msg.Tr_End(), msg.Tr_Timeout(), msg.Tr_Replica(), msg.Tr_IsReplica(), msg.AAAId())
	if e != nil {
		return object.NewError(e.Error())
	}
	request.Wait()
	return request.Response()
}

// createElements converts various input types to IElements for message transmission.
// Supports queries, proto messages, slices, and existing IElements.
func createElements(any interface{}, resources ifs.IResources) (ifs.IElements, error) {
	if any == nil {
		return object.New(nil, nil), nil
	}
	pq, ok := any.(*l8api.L8Query)
	if ok {
		return object.NewQuery(pq.Text, resources)
	}

	gsql, ok := any.(string)
	if ok {
		return object.NewQuery(gsql, resources)
	}

	elems, ok := any.(ifs.IElements)
	if ok {
		return elems, nil
	}

	pb, ok := any.(proto.Message)
	if ok {
		return object.New(nil, pb), nil
	}

	v := reflect.ValueOf(any)

	if v.Kind() == reflect.Slice {
		pbs := make([]proto.Message, v.Len())
		for i := 0; i < v.Len(); i++ {
			elm := v.Index(i)
			elements, ok := elm.Interface().(ifs.IElements)
			if ok {
				for _, epb := range elements.Elements() {
					pb, ok = epb.(proto.Message)
					if ok {
						pbs[i] = pb
					} else {
						panic(strings.New("Uknown input type ", reflect.ValueOf(pb).String()).String())
					}
				}
			} else {
				pb, ok = elm.Interface().(proto.Message)
				if ok {
					pbs[i] = pb
				} else {
					panic(strings.New("Uknown input type ", reflect.ValueOf(pb).String()).String())
				}
			}
		}
		return object.New(nil, pbs), nil
	}
	panic(strings.New("Uknown input type ", reflect.ValueOf(any).String()).String())
}
