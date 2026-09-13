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
package common

import (
	"errors"
	"fmt"
	"github.com/saichler/l8types/go/ifs"
)

// StatusTransitionConfig defines a state machine for an entity's status field.
// StatusGetter/StatusSetter/FilterBuilder operate on interface{} — the caller
// type-asserts inside these functions.
type StatusTransitionConfig struct {
	StatusGetter  func(interface{}) int32
	StatusSetter  func(interface{}, int32)
	FilterBuilder func(interface{}) interface{}
	ServiceName   string
	ServiceArea   byte
	InitialStatus int32
	Transitions   map[int32][]int32
	StatusNames   map[int32]string
}

// BuildValidator returns an ActionValidateFunc that enforces status transitions.
func (cfg *StatusTransitionConfig) BuildValidator() ActionValidateFunc {
	return func(entity interface{}, action ifs.Action, vnic ifs.IVNic) error {
		if action == ifs.POST {
			if cfg.InitialStatus > 0 {
				cfg.StatusSetter(entity, cfg.InitialStatus)
			}
			return nil
		}
		if action != ifs.PUT && action != ifs.PATCH {
			return nil
		}

		newStatus := cfg.StatusGetter(entity)
		filter := cfg.FilterBuilder(entity)
		old, err := GetEntity(cfg.ServiceName, cfg.ServiceArea, filter, vnic)
		if err != nil {
			return fmt.Errorf("failed to fetch existing entity for status validation: %w", err)
		}
		if old == nil {
			return errors.New("entity not found for status transition validation")
		}

		oldStatus := cfg.StatusGetter(old)
		if oldStatus == newStatus {
			return nil
		}

		allowed, ok := cfg.Transitions[oldStatus]
		if !ok {
			return fmt.Errorf("status %s is terminal and cannot be changed",
				cfg.statusName(oldStatus))
		}
		for _, a := range allowed {
			if a == newStatus {
				return nil
			}
		}
		return fmt.Errorf("invalid status transition from %s to %s",
			cfg.statusName(oldStatus), cfg.statusName(newStatus))
	}
}

func (cfg *StatusTransitionConfig) statusName(val int32) string {
	if n, ok := cfg.StatusNames[val]; ok {
		return n
	}
	return fmt.Sprintf("UNKNOWN(%d)", val)
}
