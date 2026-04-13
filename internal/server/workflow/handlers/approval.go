package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// ApprovalCreateRequest contains the data needed to create an approval record.
type ApprovalCreateRequest struct {
	TenantID      string
	ExecutionID   string
	NodeID        string
	ApproverRoles []string
	TimeoutHours  int
}

// ApprovalStore persists approval records.
type ApprovalStore interface {
	CreateApproval(ctx context.Context, req ApprovalCreateRequest) error
}

// ApprovalHandler creates an approval request and pauses the workflow.
type ApprovalHandler struct {
	store ApprovalStore
}

// NewApprovalHandler creates a new ApprovalHandler.
func NewApprovalHandler(store ApprovalStore) *ApprovalHandler {
	return &ApprovalHandler{store: store}
}

// Execute creates an approval record and pauses the execution.
func (h *ApprovalHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.ApprovalConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("approval handler: unmarshal config: %w", err)
	}

	req := ApprovalCreateRequest{
		TenantID:      exec.TenantID,
		ExecutionID:   exec.ExecutionID,
		NodeID:        exec.Node.ID,
		ApproverRoles: cfg.ApproverRoles,
		TimeoutHours:  cfg.TimeoutHours,
	}

	if err := h.store.CreateApproval(ctx, req); err != nil {
		return nil, fmt.Errorf("approval handler: create approval: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecRunning,
		Pause:  true,
		Output: map[string]any{
			"approver_roles":  cfg.ApproverRoles,
			"timeout_hours":   cfg.TimeoutHours,
			"escalation_role": cfg.EscalationRole,
		},
	}, nil
}
