package events_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPreferenceResolver struct {
	results []notify.ResolvedTarget
	err     error
}

func (m *mockPreferenceResolver) ResolveTargets(_ context.Context, _, _ string) ([]notify.ResolvedTarget, error) {
	return m.results, m.err
}

type mockJobEnqueuer struct {
	enqueued []notify.SendJobArgs
	err      error
}

func (m *mockJobEnqueuer) EnqueueNotification(_ context.Context, args notify.SendJobArgs) error {
	m.enqueued = append(m.enqueued, args)
	return m.err
}

func TestNotificationHandler_DispatchesToMatchingUsers(t *testing.T) {
	resolver := &mockPreferenceResolver{
		results: []notify.ResolvedTarget{
			{UserID: "user-1", ChannelID: "chan-1", ShoutrrrURL: "slack://hook1"},
			{UserID: "user-2", ChannelID: "chan-2", ShoutrrrURL: "smtp://mail"},
		},
	}
	enqueuer := &mockJobEnqueuer{}
	handler := events.NewNotificationHandler(resolver, enqueuer)

	evt := domain.DomainEvent{
		ID:         "evt-1",
		Type:       "deployment.failed",
		TenantID:   "tenant-1",
		Resource:   "deployment",
		ResourceID: "deploy-1",
		Action:     "failed",
		Payload:    map[string]any{"deployment_id": "deploy-1", "error": "timeout"},
	}

	err := handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, 2, len(enqueuer.enqueued))
	assert.Equal(t, "user-1", enqueuer.enqueued[0].UserID)
	assert.Equal(t, "user-2", enqueuer.enqueued[1].UserID)
}

func TestNotificationHandler_NoPreferences_NoJobs(t *testing.T) {
	resolver := &mockPreferenceResolver{results: nil}
	enqueuer := &mockJobEnqueuer{}
	handler := events.NewNotificationHandler(resolver, enqueuer)

	evt := domain.DomainEvent{
		Type:     "deployment.started",
		TenantID: "tenant-1",
	}

	err := handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, enqueuer.enqueued)
}

func TestNotificationHandler_CVEFiltersCritical(t *testing.T) {
	resolver := &mockPreferenceResolver{
		results: []notify.ResolvedTarget{
			{UserID: "user-1", ChannelID: "chan-1", ShoutrrrURL: "slack://hook"},
		},
	}
	enqueuer := &mockJobEnqueuer{}
	handler := events.NewNotificationHandler(resolver, enqueuer)

	// Non-critical CVE should not trigger
	evt := domain.DomainEvent{
		Type:     "cve.discovered",
		TenantID: "tenant-1",
		Payload:  map[string]any{"severity": "medium"},
	}
	err := handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, enqueuer.enqueued)

	// Critical CVE should trigger
	evt.Payload = map[string]any{"severity": "critical"}
	err = handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Equal(t, 1, len(enqueuer.enqueued))
}

func TestNotificationHandler_NewTriggerTypes(t *testing.T) {
	cases := []struct {
		name        string
		eventType   string
		payload     map[string]any
		wantTrigger string
	}{
		{
			name:        "deployment rollback triggered",
			eventType:   "deployment.rollback_triggered",
			payload:     map[string]any{"deployment_id": "d1"},
			wantTrigger: notify.TriggerDeploymentRollback,
		},
		{
			name:        "compliance evaluation completed",
			eventType:   "compliance.evaluation_completed",
			payload:     map[string]any{"framework": "CIS"},
			wantTrigger: notify.TriggerComplianceEvalComplete,
		},
		{
			name:        "cve remediation available",
			eventType:   "cve.remediation_available",
			payload:     map[string]any{"cve_id": "CVE-2024-1234"},
			wantTrigger: notify.TriggerCVEPatchAvailable,
		},
		{
			name:        "catalog sync failed triggers hub sync failed",
			eventType:   "catalog.sync_failed",
			payload:     nil,
			wantTrigger: notify.TriggerSystemHubSyncFailed,
		},
		{
			name:        "license expiring",
			eventType:   "license.expiring",
			payload:     map[string]any{"days": "14"},
			wantTrigger: notify.TriggerSystemLicenseExpiring,
		},
		{
			name:        "inventory scan completed",
			eventType:   "inventory.scan_completed",
			payload:     map[string]any{"count": "42"},
			wantTrigger: notify.TriggerSystemScanCompleted,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolver := &mockPreferenceResolver{
				results: []notify.ResolvedTarget{{UserID: "u1", ChannelID: "c1", ShoutrrrURL: "x"}},
			}
			// Capture the trigger type via a custom enqueuer
			var capturedTrigger string
			enqueuer := &capturingEnqueuer{capture: &capturedTrigger}
			handler := events.NewNotificationHandler(resolver, enqueuer)

			evt := domain.DomainEvent{
				Type:     tc.eventType,
				TenantID: "tenant-1",
				Payload:  tc.payload,
			}
			err := handler.Handle(context.Background(), evt)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTrigger, capturedTrigger,
				"event %s: want trigger %s, got %s", tc.eventType, tc.wantTrigger, capturedTrigger)
		})
	}
}

type capturingEnqueuer struct {
	capture *string
}

func (c *capturingEnqueuer) EnqueueNotification(_ context.Context, args notify.SendJobArgs) error {
	*c.capture = args.TriggerType
	return nil
}

func TestNotificationHandler_UnknownEvent_NoOp(t *testing.T) {
	resolver := &mockPreferenceResolver{
		results: []notify.ResolvedTarget{{UserID: "u1", ChannelID: "c1", ShoutrrrURL: "x"}},
	}
	enqueuer := &mockJobEnqueuer{}
	handler := events.NewNotificationHandler(resolver, enqueuer)

	evt := domain.DomainEvent{
		Type:     "endpoint.created",
		TenantID: "tenant-1",
	}
	err := handler.Handle(context.Background(), evt)
	require.NoError(t, err)
	assert.Empty(t, enqueuer.enqueued)
}
