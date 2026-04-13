package notify_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/stretchr/testify/assert"
)

func TestTriggerTypes(t *testing.T) {
	tests := []struct {
		name        string
		triggerType string
		valid       bool
	}{
		{"deployment started", "deployment.started", true},
		{"deployment completed", "deployment.completed", true},
		{"deployment failed", "deployment.failed", true},
		{"compliance breach", "compliance.threshold_breach", true},
		{"agent disconnected", "agent.disconnected", true},
		{"cve critical", "cve.critical_discovered", true},
		{"unknown type", "unknown.event", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, notify.IsValidTrigger(tt.triggerType))
		})
	}
}

func TestAllTriggers_Count(t *testing.T) {
	all := notify.AllTriggers()
	if len(all) != 16 {
		t.Errorf("expected 16 triggers, got %d", len(all))
	}
}

func TestTriggerCategories(t *testing.T) {
	cats := notify.TriggerCategories
	if len(cats["deployments"]) != 4 {
		t.Errorf("deployments: expected 4 triggers, got %d", len(cats["deployments"]))
	}
	if len(cats["compliance"]) != 4 {
		t.Errorf("compliance: expected 4 triggers, got %d", len(cats["compliance"]))
	}
	if len(cats["security"]) != 4 {
		t.Errorf("security: expected 4 triggers, got %d", len(cats["security"]))
	}
	if len(cats["system"]) != 4 {
		t.Errorf("system: expected 4 triggers, got %d", len(cats["system"]))
	}
}

func TestDefaultUrgency_ImmediateTypes(t *testing.T) {
	immediate := []string{
		notify.TriggerDeploymentFailed,
		notify.TriggerDeploymentRollback,
		notify.TriggerComplianceControlFailed,
		notify.TriggerComplianceSLAApproaching,
		notify.TriggerComplianceSLAOverdue,
		notify.TriggerCVECriticalDiscovered,
		notify.TriggerCVEExploitDetected,
		notify.TriggerCVEKEVAdded,
		notify.TriggerAgentOffline,
		notify.TriggerSystemLicenseExpiring,
		notify.TriggerSystemHubSyncFailed,
	}
	for _, tr := range immediate {
		u, ok := notify.DefaultUrgency[tr]
		if !ok {
			t.Errorf("trigger %s missing from DefaultUrgency", tr)
		} else if u != "immediate" {
			t.Errorf("trigger %s: want urgency=immediate, got %s", tr, u)
		}
	}
}

func TestCategoryForTrigger(t *testing.T) {
	cases := []struct {
		trigger string
		want    string
	}{
		{notify.TriggerDeploymentFailed, "deployments"},
		{notify.TriggerCVECriticalDiscovered, "security"},
		{notify.TriggerComplianceControlFailed, "compliance"},
		{notify.TriggerAgentOffline, "system"},
		{"nonexistent.trigger", ""},
	}
	for _, tc := range cases {
		got := notify.CategoryForTrigger(tc.trigger)
		if got != tc.want {
			t.Errorf("CategoryForTrigger(%s) = %q, want %q", tc.trigger, got, tc.want)
		}
	}
}

func TestFormatMessage_NewTriggers(t *testing.T) {
	payload := map[string]any{"deployment_id": "dep-1", "hostname": "host-1", "cve_id": "CVE-2024-1234"}
	triggers := []string{
		notify.TriggerDeploymentRollback,
		notify.TriggerComplianceEvalComplete,
		notify.TriggerComplianceControlFailed,
		notify.TriggerComplianceSLAApproaching,
		notify.TriggerComplianceSLAOverdue,
		notify.TriggerCVEExploitDetected,
		notify.TriggerCVEKEVAdded,
		notify.TriggerCVEPatchAvailable,
		notify.TriggerSystemHubSyncFailed,
		notify.TriggerSystemLicenseExpiring,
		notify.TriggerSystemScanCompleted,
		notify.TriggerAgentOffline,
	}
	for _, tr := range triggers {
		msg := notify.FormatMessage(tr, payload)
		assert.NotEmpty(t, msg, "FormatMessage(%s) returned empty string", tr)
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name        string
		triggerType string
		payload     map[string]any
		wantContain string
	}{
		{
			name:        "deployment failed",
			triggerType: "deployment.failed",
			payload:     map[string]any{"deployment_id": "abc-123", "error": "timeout"},
			wantContain: "failed",
		},
		{
			name:        "agent disconnected",
			triggerType: "agent.disconnected",
			payload:     map[string]any{"hostname": "web-01"},
			wantContain: "disconnected",
		},
		{
			name:        "unknown trigger uses generic",
			triggerType: "unknown.type",
			payload:     map[string]any{},
			wantContain: "notification",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := notify.FormatMessage(tt.triggerType, tt.payload)
			assert.Contains(t, msg, tt.wantContain)
			assert.NotEmpty(t, msg)
		})
	}
}
