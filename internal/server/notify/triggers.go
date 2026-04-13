package notify

import "fmt"

const (
	TriggerDeploymentStarted        = "deployment.started"
	TriggerDeploymentCompleted      = "deployment.completed"
	TriggerDeploymentFailed         = "deployment.failed"
	TriggerDeploymentRollback       = "deployment.rollback_initiated"
	TriggerComplianceThreshold      = "compliance.threshold_breach"
	TriggerComplianceEvalComplete   = "compliance.evaluation_complete"
	TriggerComplianceControlFailed  = "compliance.control_failed"
	TriggerComplianceSLAApproaching = "compliance.sla_approaching"
	TriggerComplianceSLAOverdue     = "compliance.sla_overdue"
	TriggerCVECriticalDiscovered    = "cve.critical_discovered"
	TriggerCVEExploitDetected       = "cve.exploit_detected"
	TriggerCVEKEVAdded              = "cve.kev_added"
	TriggerCVEPatchAvailable        = "cve.patch_available"
	TriggerAgentDisconnected        = "agent.disconnected"
	TriggerAgentOffline             = "agent.offline"
	TriggerSystemHubSyncFailed      = "system.hub_sync_failed"
	TriggerSystemLicenseExpiring    = "system.license_expiring"
	TriggerSystemScanCompleted      = "system.scan_completed"
)

// TriggerCategories maps the 4 UI category names to their 4 trigger types each (16 total).
// TriggerAgentDisconnected and TriggerComplianceThreshold are kept as constants for backwards
// compat with existing DB rows and event handlers but are not in the UI category map.
var TriggerCategories = map[string][]string{
	"deployments": {
		TriggerDeploymentStarted,
		TriggerDeploymentCompleted,
		TriggerDeploymentFailed,
		TriggerDeploymentRollback,
	},
	"compliance": {
		TriggerComplianceEvalComplete,
		TriggerComplianceControlFailed,
		TriggerComplianceSLAApproaching,
		TriggerComplianceSLAOverdue,
	},
	"security": {
		TriggerCVECriticalDiscovered,
		TriggerCVEExploitDetected,
		TriggerCVEKEVAdded,
		TriggerCVEPatchAvailable,
	},
	"system": {
		TriggerAgentOffline,
		TriggerSystemHubSyncFailed,
		TriggerSystemLicenseExpiring,
		TriggerSystemScanCompleted,
	},
}

// DefaultUrgency maps trigger types to their default urgency level.
var DefaultUrgency = map[string]string{
	TriggerDeploymentStarted:        "digest",
	TriggerDeploymentCompleted:      "digest",
	TriggerDeploymentFailed:         "immediate",
	TriggerDeploymentRollback:       "immediate",
	TriggerComplianceThreshold:      "immediate",
	TriggerComplianceEvalComplete:   "digest",
	TriggerComplianceControlFailed:  "immediate",
	TriggerComplianceSLAApproaching: "immediate",
	TriggerComplianceSLAOverdue:     "immediate",
	TriggerCVECriticalDiscovered:    "immediate",
	TriggerCVEExploitDetected:       "immediate",
	TriggerCVEKEVAdded:              "immediate",
	TriggerCVEPatchAvailable:        "digest",
	TriggerAgentDisconnected:        "immediate",
	TriggerAgentOffline:             "immediate",
	TriggerSystemHubSyncFailed:      "immediate",
	TriggerSystemLicenseExpiring:    "immediate",
	TriggerSystemScanCompleted:      "digest",
}

// CategoryForTrigger returns the UI category name for a trigger type, or "" if not found.
func CategoryForTrigger(triggerType string) string {
	for cat, triggers := range TriggerCategories {
		for _, t := range triggers {
			if t == triggerType {
				return cat
			}
		}
	}
	return ""
}

var validTriggers = func() map[string]bool {
	m := make(map[string]bool)
	for _, triggers := range TriggerCategories {
		for _, t := range triggers {
			m[t] = true
		}
	}
	// Legacy triggers: valid in DB and events, but not in UI category map.
	m[TriggerAgentDisconnected] = true
	m[TriggerComplianceThreshold] = true
	return m
}()

// AllTriggers returns the 16 UI-visible trigger types (from TriggerCategories only).
func AllTriggers() []string {
	all := make([]string, 0, 16)
	for _, triggers := range TriggerCategories {
		all = append(all, triggers...)
	}
	return all
}

// IsValidTrigger reports whether triggerType is a known trigger (including legacy).
func IsValidTrigger(triggerType string) bool {
	return validTriggers[triggerType]
}

// FormatMessage returns a human-readable notification message for the given trigger type.
func FormatMessage(triggerType string, payload map[string]any) string {
	getString := func(key string) string {
		if payload == nil {
			return "unknown"
		}
		if v, ok := payload[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return "unknown"
	}

	switch triggerType {
	case TriggerDeploymentStarted:
		return fmt.Sprintf("[PatchIQ] Deployment %s started", getString("deployment_id"))
	case TriggerDeploymentCompleted:
		return fmt.Sprintf("[PatchIQ] Deployment %s completed successfully", getString("deployment_id"))
	case TriggerDeploymentFailed:
		return fmt.Sprintf("[PatchIQ] Deployment %s failed: %s", getString("deployment_id"), getString("error"))
	case TriggerDeploymentRollback:
		return fmt.Sprintf("[PatchIQ] Deployment %s rollback initiated", getString("deployment_id"))
	case TriggerComplianceThreshold:
		return fmt.Sprintf("[PatchIQ] Compliance threshold breach: %s at %s%%", getString("policy"), getString("percentage"))
	case TriggerComplianceEvalComplete:
		return fmt.Sprintf("[PatchIQ] Compliance evaluation complete for framework %s", getString("framework"))
	case TriggerComplianceControlFailed:
		return fmt.Sprintf("[PatchIQ] Compliance control failed: %s", getString("control"))
	case TriggerComplianceSLAApproaching:
		return fmt.Sprintf("[PatchIQ] Compliance SLA approaching for %s", getString("policy"))
	case TriggerComplianceSLAOverdue:
		return fmt.Sprintf("[PatchIQ] Compliance SLA overdue for %s", getString("policy"))
	case TriggerCVECriticalDiscovered:
		return fmt.Sprintf("[PatchIQ] Critical CVE discovered: %s", getString("cve_id"))
	case TriggerCVEExploitDetected:
		return fmt.Sprintf("[PatchIQ] Exploit detected in wild for %s", getString("cve_id"))
	case TriggerCVEKEVAdded:
		return fmt.Sprintf("[PatchIQ] %s added to CISA KEV catalog", getString("cve_id"))
	case TriggerCVEPatchAvailable:
		return fmt.Sprintf("[PatchIQ] Patch available for %s", getString("cve_id"))
	case TriggerAgentDisconnected:
		return fmt.Sprintf("[PatchIQ] Agent %s disconnected", getString("hostname"))
	case TriggerAgentOffline:
		return fmt.Sprintf("[PatchIQ] Agent %s offline (>30 min)", getString("hostname"))
	case TriggerSystemHubSyncFailed:
		return "[PatchIQ] Hub sync failed"
	case TriggerSystemLicenseExpiring:
		return fmt.Sprintf("[PatchIQ] License expiring in %s days", getString("days"))
	case TriggerSystemScanCompleted:
		return fmt.Sprintf("[PatchIQ] Scan completed for %s endpoints", getString("count"))
	default:
		return fmt.Sprintf("[PatchIQ] notification: %s", triggerType)
	}
}
