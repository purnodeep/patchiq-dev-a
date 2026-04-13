package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// ComplianceCheckHandler is an M2 stub that always returns pass.
// Full compliance check integration requires M3 work.
type ComplianceCheckHandler struct{}

// NewComplianceCheckHandler creates a new ComplianceCheckHandler.
func NewComplianceCheckHandler() *ComplianceCheckHandler {
	return &ComplianceCheckHandler{}
}

// Execute always returns pass. Logs a warning that M3 is required for real compliance checks.
func (h *ComplianceCheckHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.ComplianceCheckConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("compliance check handler: unmarshal config: %w", err)
	}

	slog.WarnContext(ctx, "compliance check handler: M2 stub, always returns pass — M3 required for real compliance evaluation",
		"node_id", exec.Node.ID, "framework", cfg.Framework)

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"framework": cfg.Framework,
			"pass":      true,
			"stub":      true,
		},
	}, nil
}
