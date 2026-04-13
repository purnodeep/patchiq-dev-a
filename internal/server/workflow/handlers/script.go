package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// ScriptDispatchRequest contains the data needed to dispatch a script command.
type ScriptDispatchRequest struct {
	TenantID    string
	ExecutionID string
	NodeID      string
	Endpoints   []string
	ScriptBody  string
	ScriptType  string
	Timeout     int
}

// CommandDispatcher dispatches script commands to endpoints.
type CommandDispatcher interface {
	DispatchScript(ctx context.Context, req ScriptDispatchRequest) (string, error)
}

// ScriptHandler dispatches a script to endpoints and pauses for results.
type ScriptHandler struct {
	dispatcher CommandDispatcher
}

// NewScriptHandler creates a new ScriptHandler.
func NewScriptHandler(dispatcher CommandDispatcher) *ScriptHandler {
	return &ScriptHandler{dispatcher: dispatcher}
}

// Execute dispatches the script and pauses for completion.
func (h *ScriptHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.ScriptConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("script handler: unmarshal config: %w", err)
	}

	endpoints, _ := exec.Context["filtered_endpoints"].([]string)
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints available for script execution",
			Output: map[string]any{},
		}, nil
	}

	commandID, err := h.dispatcher.DispatchScript(ctx, ScriptDispatchRequest{
		TenantID:    exec.TenantID,
		ExecutionID: exec.ExecutionID,
		NodeID:      exec.Node.ID,
		Endpoints:   endpoints,
		ScriptBody:  cfg.ScriptBody,
		ScriptType:  cfg.ScriptType,
		Timeout:     cfg.TimeoutMinutes,
	})
	if err != nil {
		return nil, fmt.Errorf("script handler: dispatch script: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecRunning,
		Pause:  true,
		Output: map[string]any{
			"command_id":       commandID,
			"endpoint_count":   len(endpoints),
			"failure_behavior": cfg.FailureBehavior,
		},
	}, nil
}
