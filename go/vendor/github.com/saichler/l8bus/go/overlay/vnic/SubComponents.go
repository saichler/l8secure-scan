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

package vnic

// SubComponents manages the lifecycle of VNic sub-components (TX, RX, etc.).
// It provides ordered startup and shutdown to ensure proper dependency handling.
type SubComponents struct {
	components map[string]SubComponent
}

// SubComponent defines the interface for VNic sub-components that can be started and stopped.
type SubComponent interface {
	name() string
	start()
	shutdown()
}

// newSubomponents creates a new SubComponents container.
func newSubomponents() *SubComponents {
	egComponents := &SubComponents{}
	egComponents.components = make(map[string]SubComponent)
	return egComponents
}

// start initializes all registered sub-components.
func (egComponents *SubComponents) start() {
	for _, component := range egComponents.components {
		component.start()
	}
}

// shutdown stops all sub-components in order: TX first, then RX, then others.
func (egComponents *SubComponents) shutdown() {
	// Shutdown in specific order: TX first, then RX, then others
	if tx := egComponents.components["TX"]; tx != nil {
		tx.shutdown()
	}
	if rx := egComponents.components["RX"]; rx != nil {
		rx.shutdown()
	}
	// Then shutdown remaining components
	for name, component := range egComponents.components {
		if name != "TX" && name != "RX" {
			component.shutdown()
		}
	}
}

// addComponent registers a new sub-component by its name.
func (egComponents *SubComponents) addComponent(component SubComponent) {
	egComponents.components[component.name()] = component
}

// TX returns the TX (transmit) sub-component.
func (egComponents *SubComponents) TX() *TX {
	return egComponents.components["TX"].(*TX)
}
