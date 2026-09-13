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
	"github.com/saichler/l8types/go/types/l8notify"
)

// IIntegration provides lookup of configured integration endpoints (SMTP, webhook, etc.).
type IIntegration interface {
	// GetIntegrationConfig retrieves a named integration configuration.
	GetIntegrationConfig(name string) (*l8notify.IntegrationConfig, error)
	// ListIntegrationConfigs retrieves all integrations of the given type
	// (INTEGRATION_TYPE_UNSPECIFIED returns all types).
	ListIntegrationConfigs(integrationType l8notify.IntegrationType) ([]*l8notify.IntegrationConfig, error)
	// Set the VNIC
	SetVNic(IVNic)
}
