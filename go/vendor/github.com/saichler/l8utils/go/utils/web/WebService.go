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

// Package web provides RESTful web service definition and request handling utilities.
// It enables defining service endpoints with protocol buffer request/response types
// and supports multiple body types per action for flexible API design.
//
// Key features:
//   - Define HTTP endpoints with protobuf body/response types
//   - Automatic JSON-to-protobuf unmarshaling for request bodies
//   - Support for multiple request body types per endpoint
//   - VNet (Virtual Network) integration for service-to-service communication
//   - Plugin registry loading for service extensions
package web

import (
	"encoding/base64"
	"errors"
	"github.com/saichler/l8utils/go/utils/strings"
	"os"
	"reflect"
	"strconv"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8web"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// WebService defines a RESTful service with typed endpoints using protocol buffers.
// Each endpoint maps an action to request/response protobuf types.
type WebService struct {
	webService *l8web.L8WebService
	plugin     string
	protos     map[string]proto.Message
}

// New creates a new WebService with the specified name, area, and VNet port.
// Automatically loads the registry plugin if available.
func New(serviceName string, serviceArea byte, vnet uint32) ifs.IWebService {
	webService := &WebService{}
	webService.webService = &l8web.L8WebService{}
	webService.webService.ServiceName = serviceName
	webService.webService.ServiceArea = int32(serviceArea)
	webService.webService.Endpoints = make(map[int32]*l8web.L8WebServiceEndpoint)
	webService.protos = make(map[string]proto.Message)
	webService.webService.Vnet = vnet

	filename := serviceName + "-" + strconv.Itoa(int(serviceArea)) + "-registry.so"
	_, err := os.Stat(filename)
	if err == nil {
		data, err := os.ReadFile(filename)
		if err == nil {
			webService.plugin = base64.StdEncoding.EncodeToString(data)
		}
	}

	return webService
}

// Protos parses the JSON request body and returns the appropriate request/response proto types.
// Tries the primary body type first, then falls back to alternative types if parsing fails.
func (this *WebService) Protos(bodyData string, action ifs.Action) (proto.Message, proto.Message, error) {
	endpoint, ok := this.webService.Endpoints[int32(action)]
	if !ok {
		msg := strings.New("endpoint not found for action ", this.ServiceName(), " area ", this.ServiceArea(), " ", strconv.Itoa(int(action)))
		return nil, nil, errors.New(msg.String())
	}

	bodyError := make(map[string]string)

	bodyType := endpoint.PrimaryBody

	body := this.protos[bodyType]
	body = reflect.New(reflect.ValueOf(body).Elem().Type()).Interface().(proto.Message)
	err := protojson.Unmarshal([]byte(bodyData), body)
	var resp proto.Message

	if err != nil {
		bodyError[bodyType] = err.Error()
		bodyType = ""
		for k, v := range endpoint.Body2Response {
			if k != endpoint.PrimaryBody {
				body = this.protos[k]
				body = reflect.New(reflect.ValueOf(body).Elem().Type()).Interface().(proto.Message)
				if k != endpoint.PrimaryBody {
					err = protojson.Unmarshal([]byte(bodyData), body)
					if err == nil {
						resp = this.protos[v]
						resp = reflect.New(reflect.ValueOf(resp).Elem().Type()).Interface().(proto.Message)
						bodyType = k
						break
					}
					bodyError[bodyType] = err.Error()
				}
			}
		}
	}
	if bodyType == "" {
		msg := strings.New("cannot find any matching body type ", this.ServiceName(), " area ", this.ServiceArea(), " ", strconv.Itoa(int(action)))
		msg.Add(" ")
		for k, v := range bodyError {
			msg.Add(k)
			msg.Add("-")
			msg.Add(v)
			msg.Add(" ")
		}
		return nil, nil, errors.New(msg.String())
	}
	return body, resp, nil
}

// AddEndpoint registers a new endpoint mapping an action to body and response types.
// Multiple body types can be registered for the same action.
func (this *WebService) AddEndpoint(body proto.Message, action ifs.Action, resp proto.Message) {
	if body == nil {
		body = &l8web.L8Empty{}
	}
	if resp == nil {
		resp = &l8web.L8Empty{}
	}
	bodyType := this.typeOf(body)

	actionEndpoint, ok := this.webService.Endpoints[int32(action)]
	if !ok {
		actionEndpoint = &l8web.L8WebServiceEndpoint{}
		actionEndpoint.Body2Response = make(map[string]string)
		actionEndpoint.PrimaryBody = bodyType
		this.webService.Endpoints[int32(action)] = actionEndpoint
	}
	respType := this.typeOf(resp)
	actionEndpoint.Body2Response[bodyType] = respType

	this.protos[bodyType] = body
	this.protos[respType] = resp
}

func (this *WebService) typeOf(pb proto.Message) string {
	if pb == nil {
		return ""
	}
	v := reflect.ValueOf(pb).Elem()
	return v.Type().Name()
}

// Serialize returns the underlying L8WebService protobuf for transmission.
func (this *WebService) Serialize() *l8web.L8WebService {
	return this.webService
}

// DeSerialize reconstructs the WebService from a serialized L8WebService using the registry.
func (this *WebService) DeSerialize(ws *l8web.L8WebService, r ifs.IRegistry) error {
	for k, v := range ws.Endpoints {
		for bodyType, respType := range v.Body2Response {
			info, e := r.Info(bodyType)
			if e != nil {
				return e
			}
			body, e := info.NewInstance()
			if e != nil {
				return e
			}

			info, e = r.Info(respType)
			if e != nil {
				return e
			}
			resp, e := info.NewInstance()
			if e != nil {
				return e
			}
			this.AddEndpoint(body.(proto.Message), ifs.Action(k), resp.(proto.Message))
		}
	}
	return nil
}

// ServiceName returns the name of this web service.
func (this *WebService) ServiceName() string {
	return this.webService.ServiceName
}

// ServiceArea returns the service area identifier.
func (this *WebService) ServiceArea() byte {
	return byte(this.webService.ServiceArea)
}

// Vnet returns the VNet port for service-to-service communication.
func (this *WebService) Vnet() uint32 {
	return this.webService.Vnet
}

// Plugin returns the base64-encoded plugin binary if available.
func (this *WebService) Plugin() string {
	return this.plugin
}
