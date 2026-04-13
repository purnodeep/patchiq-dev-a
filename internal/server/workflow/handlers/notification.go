package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// NotificationSender sends notifications via the notification engine.
type NotificationSender interface {
	SendNotification(ctx context.Context, tenantID string, cfg workflow.NotificationConfig) error
}

// NotificationHandler sends a notification and completes (fire-and-forget).
type NotificationHandler struct {
	sender NotificationSender
}

// NewNotificationHandler creates a new NotificationHandler.
func NewNotificationHandler(sender NotificationSender) *NotificationHandler {
	return &NotificationHandler{sender: sender}
}

// Execute sends the notification. Failures are logged but do not fail the node.
func (h *NotificationHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.NotificationConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("notification handler: unmarshal config: %w", err)
	}

	if err := h.sender.SendNotification(ctx, exec.TenantID, cfg); err != nil {
		slog.ErrorContext(ctx, "notification handler: send failed",
			"node_id", exec.Node.ID, "channel", cfg.Channel, "error", err)
		return &workflow.NodeResult{
			Status: workflow.NodeExecCompleted,
			Output: map[string]any{
				"channel":    cfg.Channel,
				"target":     cfg.Target,
				"send_error": err.Error(),
			},
		}, nil
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"channel": cfg.Channel,
			"target":  cfg.Target,
			"sent":    true,
		},
	}, nil
}
