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
	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
)

// Unicast sends a message to a specific destination VNic by UUID.
func (this *VirtualNetworkInterface) Unicast(destination, serviceName string, serviceArea byte,
	action ifs.Action, any interface{}) error {
	return this.unicast(destination, serviceName, serviceArea, action, any, ifs.P8, ifs.M_All)
}

// unicast is the internal implementation for sending a message to a specific destination.
func (this *VirtualNetworkInterface) unicast(destination, serviceName string, serviceArea byte,
	action ifs.Action, any interface{}, priority ifs.Priority, multicastMode ifs.MulticastMode) error {

	if destination == "" {
		destination = ifs.DESTINATION_Single
	}

	elems, err := createElements(any, this.resources)
	if err != nil {
		return err
	}
	return this.components.TX().Unicast(destination, serviceName, serviceArea, action, elems, priority, multicastMode,
		false, false, this.protocol.NextMessageNumber(), ifs.NotATransaction, "", "",
		-1, -1, -1, -1, -1, 0, false, "")
}

// Request sends a request to a destination and waits for a response with timeout.
func (this *VirtualNetworkInterface) Request(destination, serviceName string, serviceArea byte,
	action ifs.Action, any interface{}, timeoutSeconds int, tokens ...string) ifs.IElements {
	return this.request(destination, serviceName, serviceArea, action, any, ifs.P8, ifs.M_All, timeoutSeconds, tokens...)
}

// request is the internal implementation for sending requests and waiting for responses.
func (this *VirtualNetworkInterface) request(destination, serviceName string, serviceArea byte,
	action ifs.Action, any interface{}, priority ifs.Priority, multicastMode ifs.MulticastMode, timeoutInSeconds int, tokens ...string) ifs.IElements {

	if destination == "" {
		destination = ifs.DESTINATION_Single
	}

	request, err := this.requests.NewRequest(this.protocol.NextMessageNumber(), this.resources.SysConfig().LocalUuid, timeoutInSeconds, this.resources.Logger())
	if err != nil {
		return object.NewError(err.Error())
	}
	defer this.requests.DelRequest(request.MsgNum(), request.MsgSource())

	elements, err := createElements(any, this.resources)
	if err != nil {
		return object.NewError(err.Error())
	}
	token := ""
	if tokens != nil && len(tokens) > 0 {
		token = tokens[0]
	}
	e := this.components.TX().Unicast(destination, serviceName, serviceArea, action, elements, priority, multicastMode,
		true, false, request.MsgNum(), ifs.NotATransaction, "", "",
		-1, -1, -1, -1, int64(timeoutInSeconds), 0, false, token)
	if e != nil {
		return object.NewError(e.Error())
	}
	request.Wait()
	return request.Response()
}

// Reply sends a response back to the originator of a request message.
func (this *VirtualNetworkInterface) Reply(msg *ifs.Message, response ifs.IElements) error {
	reply := msg.CloneReply(this.resources.SysConfig().LocalUuid, this.resources.SysConfig().RemoteUuid)
	data, e := this.protocol.CreateMessageForm(reply, response)
	if e != nil {
		this.resources.Logger().Error(e)
		return e
	}
	hp := health.HealthOf(msg.Source(), this.resources)
	alias := " No Alias Yet"
	if hp != nil {
		alias = hp.Alias
	}
	this.resources.Logger().Debug("Replying to ", msg.Source(), " ", alias)
	return this.SendMessage(data)
}
