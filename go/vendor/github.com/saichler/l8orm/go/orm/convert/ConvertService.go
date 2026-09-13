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

// Package convert provides bidirectional conversion between Go objects and relational data structures.
// It implements a Layer 8 service that can be deployed as part of the service mesh to handle
// object-to-relational and relational-to-object transformations remotely.
package convert

import (
	"github.com/saichler/l8orm/go/types/l8orms"
	"github.com/saichler/l8types/go/ifs"
)

const (
	// ServiceName is the identifier used to register this service in the Layer 8 service mesh.
	ServiceName = "Convert"
)

// ConvertService implements the IServicePointHandler interface to provide
// object-relational conversion as a distributed service. It handles POST and PATCH
// requests by converting objects to relational data, and GET requests by converting
// relational data back to objects.
type ConvertService struct {
}

// Activate initializes the ConvertService when it's registered with the service mesh.
// It registers the L8OrmRData type with the type registry to enable serialization.
func (this *ConvertService) Activate(sla *ifs.ServiceLevelAgreement, vnic ifs.IVNic) error {
	vnic.Resources().Registry().Register(&l8orms.L8OrmRData{})
	return nil
}

// DeActivate performs cleanup when the service is unregistered.
// Currently no cleanup is needed for this service.
func (this *ConvertService) DeActivate() error {
	return nil
}

// Post handles POST requests by converting Go objects to relational data format.
// The input elements are transformed into L8OrmRData for database persistence.
func (this *ConvertService) Post(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertTo(ifs.POST, pb, vnic.Resources())
}

// Put handles PUT requests. Currently not implemented as full replacement
// is handled through POST with conflict resolution.
func (this *ConvertService) Put(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Patch handles PATCH requests by converting objects for partial updates.
// Unlike POST, PATCH preserves zero-valued fields as null to avoid overwriting existing data.
func (this *ConvertService) Patch(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertTo(ifs.PATCH, pb, vnic.Resources())
}

// Delete handles DELETE requests. Currently not implemented as deletion
// doesn't require object-relational conversion.
func (this *ConvertService) Delete(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Get handles GET requests by converting relational data back to Go objects.
// The input L8OrmRData is transformed into the original object structure.
func (this *ConvertService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return ConvertFrom(pb, nil, vnic.Resources())
}

// GetCopy handles copy requests. Currently not implemented.
func (this *ConvertService) GetCopy(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
	return nil
}

// Failed handles failure notifications from the service mesh.
// Currently no special failure handling is implemented.
func (this *ConvertService) Failed(pb ifs.IElements, vnic ifs.IVNic, msg *ifs.Message) ifs.IElements {
	return nil
}

// TransactionConfig returns the transaction configuration for this service.
// Returns nil as the convert service doesn't require transaction management.
func (this *ConvertService) TransactionConfig() ifs.ITransactionConfig {
	return nil
}

// WebService returns the web service configuration for REST API exposure.
// Returns nil as the convert service is not exposed via REST.
func (this *ConvertService) WebService() ifs.IWebService {
	return nil
}
