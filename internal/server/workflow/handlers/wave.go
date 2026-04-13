package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// WaveDeploymentRequest contains the data needed to create a wave deployment.
type WaveDeploymentRequest struct {
	TenantID         string
	ExecutionID      string
	NodeID           string
	Endpoints        []string
	Percentage       int
	MaxParallel      int
	TimeoutMinutes   int
	SuccessThreshold int
}

// WaveDeployer creates deployment waves.
type WaveDeployer interface {
	CreateWorkflowDeployment(ctx context.Context, req WaveDeploymentRequest) (string, error)
}

// WaveHandler creates a deployment wave and pauses for completion.
type WaveHandler struct {
	deployer WaveDeployer
}

// NewWaveHandler creates a new WaveHandler.
func NewWaveHandler(deployer WaveDeployer) *WaveHandler {
	return &WaveHandler{deployer: deployer}
}

// Execute creates a deployment wave and pauses for results.
func (h *WaveHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.DeploymentWaveConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("wave handler: unmarshal config: %w", err)
	}

	endpoints, _ := exec.Context["filtered_endpoints"].([]string)
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints available for deployment wave",
			Output: map[string]any{},
		}, nil
	}

	deploymentID, err := h.deployer.CreateWorkflowDeployment(ctx, WaveDeploymentRequest{
		TenantID:         exec.TenantID,
		ExecutionID:      exec.ExecutionID,
		NodeID:           exec.Node.ID,
		Endpoints:        endpoints,
		Percentage:       cfg.Percentage,
		MaxParallel:      cfg.MaxParallel,
		TimeoutMinutes:   cfg.TimeoutMinutes,
		SuccessThreshold: cfg.SuccessThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("wave handler: create deployment: %w", err)
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecRunning,
		Pause:  true,
		Output: map[string]any{
			"deployment_id":     deploymentID,
			"endpoint_count":    len(endpoints),
			"percentage":        cfg.Percentage,
			"success_threshold": cfg.SuccessThreshold,
		},
	}, nil
}
