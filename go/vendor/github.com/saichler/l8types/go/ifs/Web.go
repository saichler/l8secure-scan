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

package ifs

import (
	"google.golang.org/protobuf/proto"
	"net/http"

	"github.com/saichler/l8types/go/types/l8web"
)

const (
	// WebService is the service name for web service handling.
	WebService = "WebService"
)

// IWebServer manages HTTP server lifecycle and web service registration.
type IWebServer interface {
	// RegisterWebService registers a web service with the server.
	RegisterWebService(IWebService, IVNic)
	// Start begins accepting HTTP connections.
	Start() error
	// Stop gracefully shuts down the server.
	Stop()
}

// IWebService defines the configuration for exposing a Layer 8 service via HTTP.
type IWebService interface {
	// Vnet returns the virtual network identifier.
	Vnet() uint32
	// ServiceName returns the backing Layer 8 service name.
	ServiceName() string
	// ServiceArea returns the service area.
	ServiceArea() byte
	// Protos returns the request and response protobuf types for an action.
	Protos(string, Action) (proto.Message, proto.Message, error)
	// AddEndpoint registers a request/response type mapping for an action.
	AddEndpoint(proto.Message, Action, proto.Message)
	// Serialize converts the web service config to protobuf.
	Serialize() *l8web.L8WebService
	// DeSerialize populates the web service from protobuf.
	DeSerialize(*l8web.L8WebService, IRegistry) error
	// Plugin returns the plugin name if any.
	Plugin() string
}

// IPlugin defines the interface for web service plugins.
type IPlugin interface {
	// Install initializes and installs the plugin with a VNic.
	Install(IVNic) error
}

// IWebProxy provides HTTP reverse proxy functionality.
type IWebProxy interface {
	// RegisterHandlers registers HTTP handlers with the given mux.
	RegisterHandlers(mux *http.ServeMux)
	// ProxyRequest forwards an HTTP request to the appropriate backend.
	ProxyRequest(w http.ResponseWriter, r *http.Request) error
	// SetValidator sets the bearer token validator.
	SetValidator(BearerValidator)
}

// BearerValidator validates bearer tokens from HTTP requests.
type BearerValidator interface {
	// ValidateBearerToken checks the bearer token in the request.
	ValidateBearerToken(r *http.Request) error
}
