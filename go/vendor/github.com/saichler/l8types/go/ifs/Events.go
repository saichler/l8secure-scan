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
	"github.com/saichler/l8types/go/types/l8events"
)

// IEvents provides the API for raising events across the Layer 8 system.
// Implementations route events to the Events service via the VNic.
type IEvents interface {
	// PostEvent creates and posts a generic EventRecord.
	PostEvent(category l8events.EventCategory, eventType string,
		severity l8events.Severity, sourceId, sourceName, sourceType, message string,
		attributes map[string]string)

	// PostAuditEvent raises an audit trail event (category=AUDIT).
	PostAuditEvent(*l8events.AuditEvent)
	// PostSystemEvent raises a system/infrastructure event (category=SYSTEM).
	PostSystemEvent(*l8events.SystemEvent)
	// PostMonitoringEvent raises a polling/collection lifecycle event (category=MONITORING).
	PostMonitoringEvent(*l8events.MonitoringEvent)
	// PostSecurityEvent raises a security event (category=SECURITY).
	PostSecurityEvent(*l8events.SecurityEvent)
	// PostIntegrationEvent raises an external system integration event (category=INTEGRATION).
	PostIntegrationEvent(*l8events.IntegrationEvent)
	// PostNetworkEvent raises a network infrastructure event (category=NETWORK).
	PostNetworkEvent(*l8events.NetworkEvent)
	// PostKubernetesEvent raises a Kubernetes cluster event (category=KUBERNETES).
	PostKubernetesEvent(*l8events.KubernetesEvent)
	// PostPerformanceEvent raises a performance threshold violation event (category=PERFORMANCE).
	PostPerformanceEvent(*l8events.PerformanceEvent)
	// PostSyslogEvent raises a parsed syslog message event (category=SYSLOG).
	PostSyslogEvent(*l8events.SyslogEvent)
	// PostTrapEvent raises a parsed SNMP trap event (category=TRAP).
	PostTrapEvent(*l8events.TrapEvent)
	// PostComputeEvent raises a hypervisor/VM event (category=COMPUTE).
	PostComputeEvent(*l8events.ComputeEvent)
	// PostStorageEvent raises a storage system event (category=STORAGE).
	PostStorageEvent(*l8events.StorageEvent)
	// PostPowerEvent raises a power infrastructure event (category=POWER).
	PostPowerEvent(*l8events.PowerEvent)
	// PostGpuEvent raises a GPU device event (category=GPU).
	PostGpuEvent(*l8events.GpuEvent)
	// PostTopologyEvent raises a topology change event (category=TOPOLOGY).
	PostTopologyEvent(*l8events.TopologyEvent)
	// PostAutomationEvent raises an automation workflow event (category=AUTOMATION).
	PostAutomationEvent(*l8events.AutomationEvent)
	// Set the VNIC
	SetVNic(IVNic)
}
