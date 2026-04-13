package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// RollbackRequest contains parameters for creating a rollback.
type RollbackRequest struct {
	TenantID       string
	ExecutionID    string
	NodeID         string
	DeploymentID   string
	Strategy       string
	RollbackScript string
}

// RollbackRequester creates rollback operations on deployments.
type RollbackRequester interface {
	CreateRollback(ctx context.Context, req RollbackRequest) (string, error)
}

// RollbackHandler rolls back a failed deployment wave.
type RollbackHandler struct {
	requester RollbackRequester
}

// NewRollbackHandler creates a new RollbackHandler.
func NewRollbackHandler(requester RollbackRequester) *RollbackHandler {
	return &RollbackHandler{requester: requester}
}

// Execute triggers a rollback for the deployment identified in the execution context.
func (h *RollbackHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.RollbackConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("rollback handler: unmarshal config: %w", err)
	}

	deploymentID, ok := exec.Context["deployment_id"].(string)
	if !ok || deploymentID == "" {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no deployment_id in execution context (add a DeploymentWave node before Rollback)",
		}, nil
	}

	rollbackID, err := h.requester.CreateRollback(ctx, RollbackRequest{
		TenantID:       exec.TenantID,
		ExecutionID:    exec.ExecutionID,
		NodeID:         exec.Node.ID,
		DeploymentID:   deploymentID,
		Strategy:       cfg.Strategy,
		RollbackScript: cfg.RollbackScript,
	})
	if err != nil {
		return nil, fmt.Errorf("rollback handler: create rollback: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"rollback_id": rollbackID,
			"strategy":    cfg.Strategy,
		},
	}, nil
}
