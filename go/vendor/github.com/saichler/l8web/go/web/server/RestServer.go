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

// Package server provides a RESTful HTTPS server implementation for the Layer 8 framework.
// It supports TLS, bearer token authentication, and seamless integration with Layer 8's
// Virtual Network Interface (VNic) for distributed service communication.
//
// The server registers web services dynamically and routes HTTP requests through the Layer 8
// network overlay, enabling proximity-based routing and service discovery.
package server

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8utils/go/utils/maps"
)

// endPoints tracks registered endpoint paths to prevent duplicate registrations.
var endPoints = maps.NewSyncMap()

// RestServer implements the ifs.IWebServer interface and provides HTTPS
// server functionality with Layer 8 integration. It manages web service registration,
// TLS configuration, and request routing.
type RestServer struct {
	webServer        *http.Server // The underlying Go HTTP server
	RestServerConfig              // Embedded configuration
}

// RestServerConfig contains the configuration options for creating a REST server.
type RestServerConfig struct {
	Host           string // Host address to bind to (e.g., "localhost", "0.0.0.0")
	Port           int    // Port number to listen on
	Authentication bool   // Enable bearer token authentication for endpoints
	Prefix         string // URL prefix for all registered endpoints (e.g., "/api/v1/")
	CertDomain     string // TLS certificate PEM (required)
	CertPrivate    string // TLS private key PEM (required)
}

// NewRestServerNoIndex creates a REST server in proxy mode, which disables
// the default index.html serving. This is used when the server operates
// behind a reverse proxy that handles static file serving.
func NewRestServerNoIndex(config *RestServerConfig) (ifs.IWebServer, error) {
	proxyMode = true
	return NewRestServer(config)
}

// NewRestServer creates a new HTTPS REST server with the provided configuration.
// It initializes the HTTP multiplexer and loads any web UI files.
// CertDomain and CertPrivate are required — the server only supports HTTPS.
func NewRestServer(config *RestServerConfig) (ifs.IWebServer, error) {
	if config.CertDomain == "" || config.CertPrivate == "" {
		return nil, fmt.Errorf("CertDomain and CertPrivate are required: RestServer only supports HTTPS")
	}
	rs := &RestServer{}
	rs.Authentication = config.Authentication
	rs.Host = config.Host
	rs.Port = config.Port
	rs.Prefix = config.Prefix
	rs.CertDomain = config.CertDomain
	rs.CertPrivate = config.CertPrivate

	http.DefaultServeMux = http.NewServeMux()
	rs.LoadWebUI()
	return rs, nil
}

// patternOf constructs the URL pattern for a service handler.
// The pattern format is: {Prefix}{serviceArea}/{serviceName}
// For example: "/api/v1/100/UserService"
func (this *RestServer) patternOf(handler *ServiceHandler) string {
	buff := bytes.Buffer{}
	buff.WriteString(this.Prefix)
	buff.WriteString(strconv.Itoa(int(handler.serviceArea)))
	buff.WriteString("/")
	buff.WriteString(handler.serviceName)
	fmt.Println("Server Path=", buff.String())
	return buff.String()
}

// RegisterWebService registers a web service with the server, creating an HTTP handler
// that routes requests through the Layer 8 VNic. Each service is assigned a unique
// URL pattern based on its service area and name. Duplicate registrations are ignored.
func (this *RestServer) RegisterWebService(ws ifs.IWebService, vnic ifs.IVNic) {
	authEnabled = this.Authentication
	handler := &ServiceHandler{authEnabled: this.Authentication}
	handler.serviceName = ws.ServiceName()
	handler.serviceArea = ws.ServiceArea()
	handler.vnic = vnic
	handler.webService = ws

	path := this.patternOf(handler)
	_, ok := endPoints.Get(path)
	if !ok {
		endPoints.Put(path, true)
		fmt.Println("Registering path=", path)
		http.DefaultServeMux.HandleFunc(this.patternOf(handler), handler.serveHttp)
	}
}

// Start begins listening for HTTPS requests. This method blocks until
// the server is stopped.
func (this *RestServer) Start() error {
	this.webServer = &http.Server{
		Addr:    this.Host + ":" + strconv.Itoa(this.Port),
		Handler: http.DefaultServeMux,
	}

	cert, err := tls.X509KeyPair([]byte(this.CertDomain), []byte(this.CertPrivate))
	if err != nil {
		panic(fmt.Sprintf("failed to parse TLS certificate: %v", err))
	}
	this.webServer.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	return this.webServer.ListenAndServeTLS("", "")
}

// RegisterHandler registers a custom HTTP handler at the given path,
// prefixed with the server's URL prefix. Use this for webhook endpoints
// and other custom handlers that don't follow the service area/name pattern.
func (this *RestServer) RegisterHandler(path string, handler http.Handler) {
	fullPath := this.Prefix + path
	_, ok := endPoints.Get(fullPath)
	if !ok {
		endPoints.Put(fullPath, true)
		fmt.Println("Registering path=", fullPath)
		http.DefaultServeMux.Handle(fullPath, handler)
	}
}

// Stop gracefully shuts down the server and cleans up registered endpoints.
// It uses the RestServer itself as the context for shutdown coordination.
func (this *RestServer) Stop() {
	this.webServer.Shutdown(this)
	endPoints.Clean()
	fmt.Println("Cleaned!")
}

// Deadline implements context.Context interface for shutdown coordination.
// Returns the current time as the deadline.
func (this *RestServer) Deadline() (deadline time.Time, ok bool) {
	return time.Now(), true
}

// Done implements context.Context interface for shutdown coordination.
// Returns nil as this context doesn't support cancellation signaling.
func (this *RestServer) Done() <-chan struct{} {
	return nil
}

// Err implements context.Context interface for shutdown coordination.
// Returns nil as this context doesn't track cancellation errors.
func (this *RestServer) Err() error {
	return nil
}

// Value implements context.Context interface for shutdown coordination.
// Returns nil as this context doesn't store any values.
func (this *RestServer) Value(key interface{}) interface{} {
	return nil
}
