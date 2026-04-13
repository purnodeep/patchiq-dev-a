package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// CommandSender sends commands to target endpoints via the gRPC outbox.
type CommandSender interface {
	SendCommand(ctx context.Context, tenantID string, endpointID string, commandType string, payload json.RawMessage) error
}

// RebootHandler sends reboot commands to target endpoints.
type RebootHandler struct {
	sender CommandSender
}

// NewRebootHandler creates a new RebootHandler.
func NewRebootHandler(sender CommandSender) *RebootHandler {
	return &RebootHandler{sender: sender}
}

// Execute sends reboot commands to all filtered endpoints.
func (h *RebootHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.RebootConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("reboot handler: unmarshal config: %w", err)
	}

	endpoints, _ := exec.Context["filtered_endpoints"].([]string)
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints available for reboot",
			Output: map[string]any{},
		}, nil
	}

	payload, _ := json.Marshal(map[string]any{
		"reboot_mode":          cfg.RebootMode,
		"grace_period_seconds": cfg.GracePeriodSeconds,
	})

	var failedCount int
	for _, epID := range endpoints {
		if err := h.sender.SendCommand(ctx, exec.TenantID, epID, "reboot", payload); err != nil {
			failedCount++
			slog.ErrorContext(ctx, "reboot handler: send command failed",
				"node_id", exec.Node.ID, "endpoint_id", epID, "error", err)
		}
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"reboot_mode":    cfg.RebootMode,
			"grace_period":   cfg.GracePeriodSeconds,
			"endpoint_count": len(endpoints),
			"failed_count":   failedCount,
			"commands_sent":  len(endpoints) - failedCount,
		},
	}, nil
}
