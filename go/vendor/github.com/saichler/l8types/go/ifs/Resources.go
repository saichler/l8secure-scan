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
	"github.com/google/uuid"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"sync"
)

var serviceMtx = &sync.Mutex{}

// IResources provides access to all shared resources in the Layer 8 system.
// It acts as a dependency injection container for core services.
type IResources interface {
	// Registry returns the type registry.
	Registry() IRegistry
	// Services returns the service manager.
	Services() IServices
	// Security returns the security provider.
	Security() ISecurityProvider
	// DataListener returns the data listener for incoming messages.
	DataListener() IDatatListener
	// Serializer returns a serializer for the given mode.
	Serializer(SerializerMode) ISerializer
	// Logger returns the logger instance.
	Logger() ILogger
	// SysConfig returns the system configuration.
	SysConfig() *l8sysconfig.L8SysConfig
	// Introspector returns the type introspector.
	Introspector() IIntrospector
	// AddService registers a service in the system config.
	AddService(string, int32)
	// Set sets a custom resource value.
	Set(interface{})
	// Copy copies resources from another IResources instance.
	Copy(IResources)
	// Return the Web Prefix
	WebPrefix() string
	// Data Directory
	DataDirectory() string
	// Certificate
	Certificate() (string, string, string)
	// Events Service
	Events() IEvents
	// Notify Service
	Notify() INotify
	// Integration Config lookup
	Integration() IIntegration
}

// AddService registers a service in the system configuration.
// Creates the service areas map if it doesn't exist.
// If modelName is provided, it is stored in the Models map for that area.
func AddService(sysConfig *l8sysconfig.L8SysConfig, serviceName string, serviceArea int32, modelName ...string) {
	serviceMtx.Lock()
	defer serviceMtx.Unlock()
	if sysConfig == nil {
		return
	}
	if sysConfig.LocalUuid == "" {
		sysConfig.LocalUuid = NewUuid()
	}
	if sysConfig.Services == nil {
		sysConfig.Services = &l8services.L8Services{}
	}
	if sysConfig.Services.ServiceToAreas == nil {
		sysConfig.Services.ServiceToAreas = make(map[string]*l8services.L8ServiceAreas)
	}
	_, ok := sysConfig.Services.ServiceToAreas[serviceName]
	if !ok {
		sysConfig.Services.ServiceToAreas[serviceName] = &l8services.L8ServiceAreas{}
		sysConfig.Services.ServiceToAreas[serviceName].Areas = make(map[int32]bool)
	}
	sysConfig.Services.ServiceToAreas[serviceName].Areas[serviceArea] = true
	if len(modelName) > 0 && modelName[0] != "" {
		if sysConfig.Services.ServiceToAreas[serviceName].Models == nil {
			sysConfig.Services.ServiceToAreas[serviceName].Models = make(map[int32]string)
		}
		sysConfig.Services.ServiceToAreas[serviceName].Models[serviceArea] = modelName[0]
	}
}

// RemoveService removes a service from the services registry.
func RemoveService(services *l8services.L8Services, serviceName string, serviceArea int32) {
	serviceMtx.Lock()
	defer serviceMtx.Unlock()
	if services == nil {
		return
	}
	if services.ServiceToAreas == nil {
		return
	}
	_, ok := services.ServiceToAreas[serviceName]
	if !ok {
		return
	}
	delete(services.ServiceToAreas[serviceName].Areas, serviceArea)
	if len(services.ServiceToAreas[serviceName].Areas) == 0 {
		delete(services.ServiceToAreas, serviceName)
	}
}

// MergeServices merges services from src into the health record's service map.
// It acquires serviceMtx to prevent concurrent map iteration/write panics
// when multiple services are being activated in parallel.
func MergeServices(hp *l8health.L8Health, services *l8services.L8Services) {
	serviceMtx.Lock()
	defer serviceMtx.Unlock()
	if hp.Services == nil {
		hp.Services = services
		return
	}
	for serviceName, serviceAreas := range services.ServiceToAreas {
		_, ok := hp.Services.ServiceToAreas[serviceName]
		if !ok {
			hp.Services.ServiceToAreas[serviceName] = serviceAreas
			continue
		}
		existing := hp.Services.ServiceToAreas[serviceName]
		if existing.Areas == nil {
			existing.Areas = serviceAreas.Areas
		} else {
			for svArea, score := range serviceAreas.Areas {
				existing.Areas[svArea] = score
			}
		}
		if serviceAreas.Models != nil {
			if existing.Models == nil {
				existing.Models = make(map[int32]string)
			}
			for svArea, model := range serviceAreas.Models {
				existing.Models[svArea] = model
			}
		}
	}
}

// NewUuid generates a new random UUID string.
func NewUuid() string {
	return uuid.New().String()
}
