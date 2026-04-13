package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// ScanHandler sends run_scan commands to target endpoints.
type ScanHandler struct {
	sender CommandSender
}

// NewScanHandler creates a new ScanHandler.
func NewScanHandler(sender CommandSender) *ScanHandler {
	return &ScanHandler{sender: sender}
}

// Execute sends scan commands to all filtered endpoints.
func (h *ScanHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.ScanConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("scan handler: unmarshal config: %w", err)
	}

	endpoints, _ := exec.Context["filtered_endpoints"].([]string)
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints available for scan",
			Output: map[string]any{},
		}, nil
	}

	payload, _ := json.Marshal(map[string]any{
		"scan_type": cfg.ScanType,
	})

	var failedCount int
	for _, epID := range endpoints {
		if err := h.sender.SendCommand(ctx, exec.TenantID, epID, "run_scan", payload); err != nil {
			failedCount++
			slog.ErrorContext(ctx, "scan handler: send command failed",
				"node_id", exec.Node.ID, "endpoint_id", epID, "error", err)
		}
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"scan_type":      cfg.ScanType,
			"endpoint_count": len(endpoints),
			"failed_count":   failedCount,
			"commands_sent":  len(endpoints) - failedCount,
		},
	}, nil
}
