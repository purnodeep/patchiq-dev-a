package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// TriggerHandler validates the trigger condition and writes metadata to context.
type TriggerHandler struct{}

func NewTriggerHandler() *TriggerHandler { return &TriggerHandler{} }

func (h *TriggerHandler) Execute(_ context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.TriggerConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("trigger handler: unmarshal config: %w", err)
	}
	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"trigger_type": cfg.TriggerType,
			"triggered_at": time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}
