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

package common

import (
	"errors"
	"github.com/saichler/l8types/go/ifs"
)

// LocalServiceChecker checks if a local service is available.
// This matches the signature of functions like employees.Employees(vnic) or jobs.Jobs(vnic).
type LocalServiceChecker func(vnic ifs.IVNic) (ifs.IServiceHandler, bool)

// LocalEntityLookup looks up an entity by ID in the local service.
// Returns the entity (can be ignored) and an error if not found.
type LocalEntityLookup func(id string, vnic ifs.IVNic) (interface{}, error)

// ValidateReference validates that a referenced entity exists.
// It first tries local lookup if the service is available locally,
// otherwise falls back to a remote request.
//
// Parameters:
//   - id: the ID to validate (if empty, validation passes immediately)
//   - entityName: name of the entity for error messages (e.g., "Manager", "Job")
//   - serviceName: name of the service for remote requests
//   - serviceArea: service area byte for remote requests
//   - localChecker: function to check if local service exists
//   - localLookup: function to look up entity locally by ID
//   - filter: filter object for remote request (should have the ID field set)
//   - vnic: the virtual NIC for service communication
//
// Returns nil if the reference is valid, or an error if validation fails.
func ValidateReference(
	id string,
	entityName string,
	serviceName string,
	serviceArea byte,
	localChecker LocalServiceChecker,
	localLookup LocalEntityLookup,
	filter interface{},
	vnic ifs.IVNic,
) error {
	if id == "" {
		return nil
	}

	_, ok := localChecker(vnic)
	if ok {
		_, err := localLookup(id, vnic)
		return err
	}

	resp := vnic.Request("", serviceName, serviceArea, ifs.GET, filter, 15)
	if resp.Error() != nil {
		return resp.Error()
	}
	if resp.Element() == nil {
		return errors.New("No " + entityName + " was found for id " + id)
	}
	return nil
}
