package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// PreferenceResolver resolves which users and channels should receive a notification.
type PreferenceResolver interface {
	ResolveTargets(ctx context.Context, tenantID, triggerType string) ([]notify.ResolvedTarget, error)
}

// JobEnqueuer enqueues notification send jobs.
type JobEnqueuer interface {
	EnqueueNotification(ctx context.Context, args notify.SendJobArgs) error
}

// NotificationHandler handles domain events and dispatches notifications.
type NotificationHandler struct {
	resolver PreferenceResolver
	enqueuer JobEnqueuer
}

func NewNotificationHandler(resolver PreferenceResolver, enqueuer JobEnqueuer) *NotificationHandler {
	return &NotificationHandler{resolver: resolver, enqueuer: enqueuer}
}

// triggerTypeForEvent maps a domain event type to a notification trigger type.
func triggerTypeForEvent(evt domain.DomainEvent) string {
	switch evt.Type {
	case DeploymentStarted:
		return notify.TriggerDeploymentStarted
	case DeploymentCompleted:
		return notify.TriggerDeploymentCompleted
	case DeploymentFailed:
		return notify.TriggerDeploymentFailed
	case DeploymentRollbackTriggered:
		return notify.TriggerDeploymentRollback
	case ComplianceEvaluationCompleted:
		return notify.TriggerComplianceEvalComplete
	case CVERemediationAvailable:
		return notify.TriggerCVEPatchAvailable
	case CatalogSyncFailed:
		return notify.TriggerSystemHubSyncFailed
	case LicenseExpiring:
		return notify.TriggerSystemLicenseExpiring
	case InventoryScanCompleted:
		return notify.TriggerSystemScanCompleted
	case ComplianceThresholdBreach:
		return notify.TriggerComplianceThreshold
	case AgentDisconnected:
		return notify.TriggerAgentDisconnected
	case CVEDiscovered:
		if isCriticalCVE(evt) {
			return notify.TriggerCVECriticalDiscovered
		}
		return ""
	default:
		return ""
	}
}

func isCriticalCVE(evt domain.DomainEvent) bool {
	payload := payloadAsMap(evt)
	severity, _ := payload["severity"].(string)
	return severity == "critical"
}

func payloadAsMap(evt domain.DomainEvent) map[string]any {
	switch p := evt.Payload.(type) {
	case map[string]any:
		return p
	default:
		data, err := json.Marshal(p)
		if err != nil {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil
		}
		return m
	}
}

func (h *NotificationHandler) Handle(ctx context.Context, evt domain.DomainEvent) error {
	triggerType := triggerTypeForEvent(evt)
	if triggerType == "" {
		return nil
	}

	targets, err := h.resolver.ResolveTargets(ctx, evt.TenantID, triggerType)
	if err != nil {
		return fmt.Errorf("resolve notification targets for %s: %w", triggerType, err)
	}

	if len(targets) == 0 {
		slog.DebugContext(ctx, "no notification targets for trigger",
			"trigger_type", triggerType, "tenant_id", evt.TenantID)
		return nil
	}

	payload := payloadAsMap(evt)
	message := notify.FormatMessage(triggerType, payload)

	var enqueueFailures int
	for _, target := range targets {
		args := notify.SendJobArgs{
			TenantID:    evt.TenantID,
			UserID:      target.UserID,
			TriggerType: triggerType,
			ChannelID:   target.ChannelID,
			ChannelType: target.ChannelType,
			Recipient:   target.Recipient,
			Subject:     message,
			ShoutrrrURL: target.ShoutrrrURL,
			Message:     message,
		}
		if enqErr := h.enqueuer.EnqueueNotification(ctx, args); enqErr != nil {
			enqueueFailures++
			slog.ErrorContext(ctx, "enqueue notification failed",
				"trigger_type", triggerType, "user_id", target.UserID,
				"channel_id", target.ChannelID, "error", enqErr)
		}
	}

	if enqueueFailures == len(targets) {
		return fmt.Errorf("enqueue notifications for %s: all %d enqueues failed", triggerType, len(targets))
	}

	slog.InfoContext(ctx, "notifications dispatched",
		"trigger_type", triggerType, "target_count", len(targets),
		"enqueue_failures", enqueueFailures, "tenant_id", evt.TenantID)
	return nil
}
