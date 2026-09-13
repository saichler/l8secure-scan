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

package health

import (
	"github.com/saichler/l8services/go/services/base"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8api"
	"github.com/saichler/l8types/go/types/l8health"
	"github.com/saichler/l8types/go/types/l8services"
	"github.com/saichler/l8types/go/types/l8sysconfig"
	"github.com/saichler/l8utils/go/utils/web"
)

// ServiceName is the identifier used to register and lookup the health service.
const (
	ServiceName = "Health"
)

// Activate registers the health service with the given VNic. If voter is true,
// the service participates in leader election. The service is initialized with
// the local node's health information and exposes a web API for health queries.
func Activate(vnic ifs.IVNic, voter bool) {
	serviceArea := ServiceArea(vnic.Resources())
	serviceConfig := ifs.NewServiceLevelAgreement(&base.BaseService{}, ServiceName, serviceArea, true, &HealthServiceCallback{})

	services := &l8services.L8Services{}
	services.ServiceToAreas = make(map[string]*l8services.L8ServiceAreas)
	services.ServiceToAreas[ServiceName] = &l8services.L8ServiceAreas{}
	services.ServiceToAreas[ServiceName].Areas = make(map[int32]bool)
	services.ServiceToAreas[ServiceName].Areas[int32(serviceArea)] = true

	serviceConfig.SetServiceItem(&l8health.L8Health{AUuid: vnic.Resources().SysConfig().LocalUuid, Services: services})
	serviceConfig.SetServiceItemList(&l8health.L8HealthList{})
	serviceConfig.SetInitItems([]interface{}{serviceConfig.ServiceItem()})

	serviceConfig.SetVoter(voter)
	serviceConfig.SetTransactional(false)
	serviceConfig.SetPrimaryKeys("AUuid")
	serviceConfig.SetAlwaysOverwrite("l8health.services.servicetoareas")

	webService := web.New(ServiceName, serviceArea, vnic.Resources().SysConfig().VnetPort)
	webService.AddEndpoint(&l8api.L8Query{}, ifs.GET, &l8health.L8HealthList{})
	webService.AddEndpoint(&l8health.L8Health{}, ifs.GET, &l8health.L8Health{})
	serviceConfig.SetWebService(webService)

	serviceConfig.SetServiceGroup(ifs.SystemServiceGroup)
	base.Activate(serviceConfig, vnic)
}

// HealthOf retrieves the health record for a node identified by its UUID.
// Returns nil if the node is not found in the health registry.
func HealthOf(uuid string, r ifs.IResources) *l8health.L8Health {
	sh, ok := HealthService(r)
	if ok {
		filter := &l8health.L8Health{}
		filter.AUuid = uuid
		h := sh.Get(object.New(nil, filter), nil)
		result, _ := h.Element().(*l8health.L8Health)
		return result
	}
	return nil
}

// HealthService returns the health service handler for the given resources.
// The second return value indicates if the service was found.
func HealthService(r ifs.IResources) (ifs.IServiceHandler, bool) {
	return r.Services().ServiceHandler(ServiceName, ServiceArea(r))
}

// HealthServiceCache returns the health service as a cache interface for direct
// access to all cached health records.
func HealthServiceCache(r ifs.IResources) (ifs.IServiceHandlerCache, bool) {
	hs, _ := HealthService(r)
	hc, ok := hs.(ifs.IServiceHandlerCache)
	return hc, ok
}

// All returns a map of all known health records indexed by node UUID.
func All(r ifs.IResources) map[string]*l8health.L8Health {
	hc, _ := HealthServiceCache(r)
	all := hc.All()
	result := make(map[string]*l8health.L8Health)
	for _, h := range all {
		hp := h.(*l8health.L8Health)
		result[hp.AUuid] = hp
	}
	return result
}

// ServiceCatalog returns a unified map of all services across all nodes in the cluster.
// The result maps serviceName -> L8ServiceAreas (with Areas and Models populated).
// This aggregates data from all health records, merging areas and models.
func ServiceCatalog(r ifs.IResources) map[string]*l8services.L8ServiceAreas {
	result := make(map[string]*l8services.L8ServiceAreas)
	allHealth := All(r)
	for _, hp := range allHealth {
		if hp.Services == nil || hp.Services.ServiceToAreas == nil {
			continue
		}
		for svcName, svcAreas := range hp.Services.ServiceToAreas {
			existing, ok := result[svcName]
			if !ok {
				existing = &l8services.L8ServiceAreas{
					Areas:  make(map[int32]bool),
					Models: make(map[int32]string),
				}
				result[svcName] = existing
			}
			for area, available := range svcAreas.Areas {
				existing.Areas[area] = available
			}
			for area, model := range svcAreas.Models {
				existing.Models[area] = model
			}
		}
	}
	return result
}

// ServiceArea returns the service area for health based on the resources configuration.
// Area 0 is for primary/local VNet, Area 1 is for remote/secondary VNet.
func ServiceArea(r ifs.IResources) byte {
	return ServiceAreaByConfig(r.SysConfig())
}

// ServiceAreaByConfig determines the service area based on the system configuration.
// Returns 1 if RemoteVnet is configured, otherwise returns 0.
func ServiceAreaByConfig(config *l8sysconfig.L8SysConfig) byte {
	if config.RemoteVnet != "" {
		return byte(1)
	}
	return byte(0)
}
