package events

import (
	"fmt"
	"github.com/saichler/l8types/go/types/l8events"
	"time"
)

var categoryNames = map[l8events.EventCategory]string{
	l8events.EventCategory_EVENT_CATEGORY_AUDIT:        "Audit Event",
	l8events.EventCategory_EVENT_CATEGORY_SECURITY:     "Security Event",
	l8events.EventCategory_EVENT_CATEGORY_SYSTEM:       "System Event",
	l8events.EventCategory_EVENT_CATEGORY_MONITORING:   "Monitoring Event",
	l8events.EventCategory_EVENT_CATEGORY_INTEGRATION:  "Integration Event",
	l8events.EventCategory_EVENT_CATEGORY_NETWORK:      "Network Event",
	l8events.EventCategory_EVENT_CATEGORY_KUBERNETES:   "Kubernetes Event",
	l8events.EventCategory_EVENT_CATEGORY_PERFORMANCE:  "Performance Event",
	l8events.EventCategory_EVENT_CATEGORY_SYSLOG:       "Syslog Event",
	l8events.EventCategory_EVENT_CATEGORY_TRAP:         "Trap Event",
	l8events.EventCategory_EVENT_CATEGORY_COMPUTE:      "Compute Event",
	l8events.EventCategory_EVENT_CATEGORY_STORAGE:      "Storage Event",
	l8events.EventCategory_EVENT_CATEGORY_POWER:        "Power Event",
	l8events.EventCategory_EVENT_CATEGORY_GPU:          "GPU Event",
	l8events.EventCategory_EVENT_CATEGORY_TOPOLOGY:     "Topology Event",
	l8events.EventCategory_EVENT_CATEGORY_AUTOMATION:   "Automation Event",
}

func toRecord(category l8events.EventCategory, subCategory fmt.Stringer,
	sourceId, sourceType, sourceName, message string) *l8events.EventRecord {
	eventType := categoryNames[category]
	if eventType == "" {
		eventType = "Event"
	}
	return &l8events.EventRecord{
		Category:   category,
		EventType:  eventType,
		Severity:   l8events.Severity_SEVERITY_INFO,
		SourceId:   sourceId,
		SourceType: sourceType,
		SourceName: sourceName,
		Message:    message,
		OccurredAt: time.Now().Unix(),
	}
}

func auditToRecord(evt *l8events.AuditEvent) *l8events.EventRecord {
	r := toRecord(l8events.EventCategory_EVENT_CATEGORY_AUDIT, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
	r.Attributes = map[string]string{
		"subCategory": evt.SubCategory.String(),
		"userId":      evt.UserId,
		"userName":    evt.UserName,
		"userIp":      evt.UserIp,
		"action":      evt.Action,
		"entityName":  evt.EntityName,
	}
	return r
}

func securityToRecord(evt *l8events.SecurityEvent) *l8events.EventRecord {
	r := toRecord(l8events.EventCategory_EVENT_CATEGORY_SECURITY, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
	r.Severity = l8events.Severity_SEVERITY_WARNING
	r.Attributes = map[string]string{
		"subCategory":    evt.SubCategory.String(),
		"userId":         evt.UserId,
		"userName":       evt.UserName,
		"userIp":         evt.UserIp,
		"targetResource": evt.TargetResource,
		"authMethod":     evt.AuthMethod,
		"failureReason":  evt.FailureReason,
	}
	return r
}

func systemToRecord(evt *l8events.SystemEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_SYSTEM, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func monitoringToRecord(evt *l8events.MonitoringEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_MONITORING, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func integrationToRecord(evt *l8events.IntegrationEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_INTEGRATION, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func networkToRecord(evt *l8events.NetworkEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_NETWORK, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func kubernetesToRecord(evt *l8events.KubernetesEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_KUBERNETES, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func performanceToRecord(evt *l8events.PerformanceEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_PERFORMANCE, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func syslogToRecord(evt *l8events.SyslogEvent) *l8events.EventRecord {
	msg := evt.ParsedMessage
	if msg == "" {
		msg = evt.RawMessage
	}
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_SYSLOG, nil,
		evt.SourceId, evt.SourceType, evt.DeviceName, msg)
}

func trapToRecord(evt *l8events.TrapEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_TRAP, nil,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func computeToRecord(evt *l8events.ComputeEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_COMPUTE, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func storageToRecord(evt *l8events.StorageEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_STORAGE, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func powerToRecord(evt *l8events.PowerEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_POWER, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func gpuToRecord(evt *l8events.GpuEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_GPU, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func topologyToRecord(evt *l8events.TopologyEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_TOPOLOGY, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}

func automationToRecord(evt *l8events.AutomationEvent) *l8events.EventRecord {
	return toRecord(l8events.EventCategory_EVENT_CATEGORY_AUTOMATION, evt.SubCategory,
		evt.SourceId, evt.SourceType, "", evt.Message)
}
