/*
 * Copyright (c) 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// ServiceHandler.go implements HTTP request handling for Layer 8 web services.
// It bridges HTTP requests to the Layer 8 network overlay, handling authentication,
// request routing, and response serialization.

package server

import (
	"fmt"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8services"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/saichler/l8bus/go/overlay/health"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ServiceHandler handles HTTP requests for a specific web service, routing them
// through the Layer 8 VNic to the appropriate service implementation. It manages
// authentication validation, request parsing, and response serialization.
type ServiceHandler struct {
	serviceName string          // Name of the service being handled
	serviceArea byte            // Service area identifier for routing
	vnic        ifs.IVNic       // Layer 8 Virtual Network Interface for communication
	webService  ifs.IWebService // The web service implementation
	authEnabled bool            // Whether authentication is required for this handler
}

// ServiceAction encapsulates request and response Protocol Buffer messages
// for a service operation.
type ServiceAction struct {
	body proto.Message // Request body message
	resp proto.Message // Response message
}

// Timeout specifies the default request timeout in seconds for VNic operations.
var Timeout = 30

// Target specifies a specific service instance UUID to route requests to.
// If empty, requests are routed based on the Method setting.
var Target = ""

// Method specifies the routing method for requests: M_Leader (leader-based),
// M_Local (local service), or M_Proximity (proximity-based routing).
var Method = ifs.M_Leader

// ServiceName returns the name of the service this handler manages.
func (this *ServiceHandler) ServiceName() string {
	return this.serviceName
}

// ServiceArea returns the service area identifier used for request routing.
func (this *ServiceHandler) ServiceArea() byte {
	return this.serviceArea
}

// serveHttp is the main HTTP handler function that processes incoming requests.
// It performs the following steps:
// 1. Validates bearer token authentication if enabled
// 2. Reads and parses the request body (supports query parameter for GET requests)
// 3. Routes the request through the Layer 8 VNic based on routing method
// 4. Serializes and returns the response as JSON
//
// Authentication tokens are checked in the following order:
// - Authorization header (Bearer token)
// - Adjacent token mapping (for cross-VNet requests)
//
// Returns HTTP 401 Unauthorized if authentication fails, HTTP 400 Bad Request
// for parsing errors, or HTTP 200 OK with JSON response on success.
func (this *ServiceHandler) serveHttp(w http.ResponseWriter, r *http.Request) {
	aaaid := ""
	if this.authEnabled {
		bearer := r.Header.Get("Authorization")
		if bearer == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		id, ok := this.vnic.Resources().Security().ValidateToken(bearer, this.vnic)
		if !ok && (id == "Token Setup TFA" || id == "Token Need TFA Verification") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(id))
			return
		}
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		aaaid = id
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Failed to read body for method " + r.Method + "\n"))
		w.Write([]byte(err.Error()))
		fmt.Println("Failed to read body for method " + r.Method + "\n")
		return
	}

	if strings.ToLower(r.Method) == "get" && (data == nil || len(data) == 0) {
		qData := r.URL.Query().Get("body")
		data = []byte(qData)
	}

	action := methodToAction(r.Method, nil)
	body, _, err := this.webService.Protos(string(data), action)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Cannot find pb for method " + r.Method + "\n"))
		w.Write([]byte(err.Error()))
		fmt.Println("Cannot find pb for method " + r.Method + "\n")
		return
	}

	action = methodToAction(r.Method, body)
	if q, ok := body.(*l8api.L8Query); ok && aaaid != "" {
		q.AaaId = aaaid
	}
	var elems ifs.IElements

	dest := this.vnic.Resources().SysConfig().RemoteUuid
	if this.serviceName == health.ServiceName {
		h, ok := body.(*l8health.L8Health)
		if ok {
			this.vnic.Resources().Logger().Info("Sending to destination ", h.Alias, " - ", h.AUuid)
			elems = this.vnic.Request(h.AUuid, this.serviceName, this.serviceArea, action, body, Timeout)
		} else {
			this.vnic.Resources().Logger().Info("Sending to vnet")
			elems = this.vnic.Request(dest, this.serviceName, this.serviceArea, action, body, Timeout)
		}
	} else {
		if Target != "" {
			elems = this.vnic.Request(Target, this.serviceName, this.serviceArea, action, body, Timeout, aaaid)
		} else {
			if Method == ifs.M_Leader {
				elems = this.vnic.LeaderRequest(this.serviceName, this.serviceArea, action, body, Timeout, aaaid)
			} else if Method == ifs.M_Local {
				elems = this.vnic.LocalRequest(this.serviceName, this.serviceArea, action, body, Timeout, aaaid)
			} else {
				elems = this.vnic.ProximityRequest(this.serviceName, this.serviceArea, action, body, Timeout, aaaid)
			}
		}
	}

	if elems.Error() != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error from single request:"))
		w.Write([]byte(elems.Error().Error()))
		fmt.Println("Error from single request:")
		fmt.Println(elems.Error().Error())
		return
	}

	trans, ok := elems.Element().(*l8services.L8Transaction)
	if ok && trans.ErrMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Validation Error"))
		w.Write([]byte(trans.ErrMsg))
		fmt.Println("Validation Error")
		fmt.Println(trans.ErrMsg)
		return
	}

	w.WriteHeader(http.StatusOK)
	response, e := elems.AsList(this.vnic.Resources().Registry())
	if e != nil {
		w.Write([]byte("{}"))
		/*
			w.Write([]byte("Erorr as list:"))
			w.Write([]byte(e.Error()))
		*/
		return
	}

	marshalOptions := protojson.MarshalOptions{
		UseEnumNumbers: true,
	}
	j, e := marshalOptions.Marshal(response.(proto.Message))
	if e != nil {
		w.Write([]byte("Erorr marshaling:" + reflect.ValueOf(response).Elem().Type().Name()))
		w.Write([]byte(e.Error()))
		fmt.Println("Erorr marshaling:" + reflect.ValueOf(response).Elem().Type().Name())
	} else {
		w.Write(j)
	}
}

// methodToAction converts an HTTP method string to a Layer 8 Action constant.
// If the request body contains an L8Query with "mapreduce" in the text, it returns
// the MapReduce variant of the action for distributed query execution.
//
// Supported mappings:
//   - POST   -> ifs.POST or ifs.MapR_POST
//   - GET    -> ifs.GET or ifs.MapR_GET
//   - DELETE -> ifs.DELETE or ifs.MapR_DELETE
//   - PUT    -> ifs.PUT or ifs.MapR_PUT
//   - PATCH  -> ifs.PATCH or ifs.MapR_PATCH
//
// Defaults to ifs.GET for unknown methods.
func methodToAction(method string, body proto.Message) ifs.Action {
	isMapReduce := false
	q, ok := body.(*l8api.L8Query)
	if ok {
		if strings.Contains(strings.ToLower(q.Text), "mapreduce") {
			isMapReduce = true
		}
	}
	switch method {
	case http.MethodPost:
		if isMapReduce {
			return ifs.MapR_POST
		}
		return ifs.POST
	case http.MethodGet:
		if isMapReduce {
			return ifs.MapR_GET
		}
		return ifs.GET
	case http.MethodDelete:
		if isMapReduce {
			return ifs.MapR_DELETE
		}
		return ifs.DELETE
	case http.MethodPut:
		if isMapReduce {
			return ifs.MapR_PUT
		}
		return ifs.PUT
	case http.MethodPatch:
		if isMapReduce {
			return ifs.MapR_PATCH
		}
		return ifs.PATCH
	}
	return ifs.GET
}
