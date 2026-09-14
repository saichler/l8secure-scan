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

package targets

import (
	"errors"
	"fmt"
	"github.com/saichler/l8orm/go/orm/common"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8types/go/ifs"
	"sync"
)

// TargetCallback implements lifecycle hooks for target operations.
// It provides Before and After callbacks that are invoked during
// target CRUD operations to perform validation, address tracking,
// and collector notification.
type TargetCallback struct {
	// addressValidation tracks all registered IP addresses to prevent duplicates
	addressValidation *sync.Map
	// iorm is the ORM interface for database operations
	iorm common.IORM
}

// newTargetCallback creates a new TargetCallback with the given ORM interface.
// It initializes the address validation map for tracking registered IPs.
func newTargetCallback(iorm common.IORM) *TargetCallback {
	return &TargetCallback{addressValidation: &sync.Map{}, iorm: iorm}
}

// Before is called before a target operation is persisted.
// For POST operations:
//   - Handles TargetAction requests to start/stop all targets of a type
//   - Validates IP addresses for L8PTargetList and L8PTarget to prevent duplicates
//
// For PATCH operations:
//   - Verifies the target exists before allowing updates
//
// Returns (elements, continue, error) where:
//   - elements: modified element(s) to process
//   - continue: whether to proceed with the operation
//   - error: any validation error
func (this *TargetCallback) Before(elem interface{}, action ifs.Action, notification bool, vnic ifs.IVNic) (interface{}, bool, error) {
	switch action {
	case ifs.POST:
		if !notification {
			targetAction, ok := elem.(*l8tpollaris.TargetAction)
			if ok {
				fmt.Println("Performing Action:", targetAction)
				this.startStopAll(targetAction.ActionState, targetAction.ActionType, vnic)
				return nil, false, nil
			}
			list, ok := elem.(*l8tpollaris.L8PTargetList)
			if ok {
				elems := make([]interface{}, 0)
				for _, item := range list.List {
					err := this.validateNewIP(item)
					if err != nil {
						return nil, true, err
					}
					elems = append(elems, item)
				}
				return elems, true, nil
			}
			item, ok := elem.(*l8tpollaris.L8PTarget)
			if ok {
				err := this.validateNewIP(item)
				if err != nil {
					return nil, true, err
				}
			}
		}
	case ifs.PATCH:
		if !notification {
			target, ok := elem.(*l8tpollaris.L8PTarget)
			if !ok {
				return nil, true, errors.New("invalid target")
			}
			_, err := Target(target.TargetId, vnic)
			if err != nil {
				return nil, true, err
			}
		}
	}

	return nil, true, nil
}

// After is called after a target operation is persisted.
// For POST operations with UP state: sends the target to a collector via round-robin
// For PATCH operations: notifies collectors of state changes
//   - DOWN state: multicasts to all collectors to stop polling
//   - UP state: uses round-robin to assign to a specific collector
//
// Returns (nil, true, error) - the first return value is unused for After callbacks.
func (this *TargetCallback) After(elem interface{}, action ifs.Action, notification bool, vnic ifs.IVNic) (interface{}, bool, error) {
	if action == ifs.POST && !notification {
		target, ok := elem.(*l8tpollaris.L8PTarget)
		if !ok {
			return nil, true, errors.New("invalid target")
		}
		if target.State == l8tpollaris.L8PTargetState_Up {
			collectorService, collectorArea := Links.Collector(target.LinksId)
			vnic.Resources().Logger().Info("Sending target to collector:", collectorService, " area ", collectorArea)
			err := vnic.RoundRobin(collectorService, collectorArea, ifs.POST, target)
			if err != nil {
				return nil, true, err
			}
		}
	}
	if action == ifs.PATCH && !notification {
		target, ok := elem.(*l8tpollaris.L8PTarget)
		if !ok {
			return nil, true, errors.New("invalid target")
		}

		currTarget, err := Target(target.TargetId, vnic)
		if err != nil {
			return nil, true, err
		}
		collectorService, collectorArea := Links.Collector(currTarget.LinksId)

		switch target.State {
		case l8tpollaris.L8PTargetState_Down:
			vnic.Resources().Logger().Info("Sending stop target to collector:", collectorService, " area ", collectorArea,
				" with hosts ", currTarget.Hosts)
			err = vnic.Multicast(collectorService, collectorArea, ifs.POST, currTarget)
			if err != nil {
				return nil, true, err
			}
		case l8tpollaris.L8PTargetState_Up:
			vnic.Resources().Logger().Info("Sending start target to collector:", collectorService, " area ", collectorArea,
				" with hosts ", currTarget.Hosts)
			err = vnic.RoundRobin(collectorService, collectorArea, ifs.POST, currTarget)
			if err != nil {
				return nil, true, err
			}
		}
	}
	return nil, true, nil
}

// validateNewIP validates that all IP addresses in the target are unique.
// It checks each host's protocol configurations for duplicate addresses.
// If validation passes, the addresses are registered in the addressValidation map.
// Returns an error if the target has no hosts, hosts have no configs, or
// any address is already registered.
func (this *TargetCallback) validateNewIP(target *l8tpollaris.L8PTarget) error {
	if target.Hosts == nil || len(target.Hosts) == 0 {
		return errors.New("invalid target, has no hosts")
	}
	for _, host := range target.Hosts {
		if host.Configs == nil || len(host.Configs) == 0 {
			return errors.New("invalid target, host has no configs")
		}
		for _, p := range host.Configs {
			_, ok := this.addressValidation.Load(p.Addr)
			if ok {
				return errors.New("invalid target, address " + p.Addr + " already exists")
			}
		}
		for _, p := range host.Configs {
			this.addressValidation.Store(p.Addr, true)
		}
	}
	return nil
}
