package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// CompleteHandler is the terminal node handler that marks a workflow as complete.
type CompleteHandler struct{}

// NewCompleteHandler creates a new CompleteHandler.
func NewCompleteHandler() *CompleteHandler {
	return &CompleteHandler{}
}

// Execute marks the workflow as complete and returns config flags in the output.
func (h *CompleteHandler) Execute(_ context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.CompleteConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("complete handler: unmarshal config: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"generate_report":    cfg.GenerateReport,
			"notify_on_complete": cfg.NotifyOnComplete,
		},
	}, nil
}
