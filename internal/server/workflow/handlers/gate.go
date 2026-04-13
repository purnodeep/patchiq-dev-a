package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// GateHandler pauses the workflow for a configured wait period.
type GateHandler struct{}

// NewGateHandler creates a new GateHandler.
func NewGateHandler() *GateHandler { return &GateHandler{} }

// Execute records gate parameters and pauses for the configured wait.
func (h *GateHandler) Execute(_ context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.GateConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("gate handler: unmarshal config: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecRunning,
		Pause:  true,
		Output: map[string]any{
			"wait_minutes":      cfg.WaitMinutes,
			"failure_threshold": cfg.FailureThreshold,
			"health_check":      cfg.HealthCheck,
		},
	}, nil
}
