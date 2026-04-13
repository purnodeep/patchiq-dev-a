package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestWaveHandler_CreatesDeploymentAndPauses(t *testing.T) {
	deployer := &mockDeployer{deploymentID: "deploy-123"}
	h := NewWaveHandler(deployer)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "wave-1",
			Config: json.RawMessage(`{"percentage":25,"max_parallel":10,"timeout_minutes":60,"success_threshold":95}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1", "ep-2", "ep-3", "ep-4"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Pause {
		t.Error("expected Pause=true")
	}
	if result.Output["deployment_id"] != "deploy-123" {
		t.Errorf("deployment_id = %v, want deploy-123", result.Output["deployment_id"])
	}
}

func TestWaveHandler_NoEndpoints(t *testing.T) {
	deployer := &mockDeployer{}
	h := NewWaveHandler(deployer)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "wave-1",
			Config: json.RawMessage(`{"percentage":25}`),
		},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != workflow.NodeExecFailed {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecFailed)
	}
}

type mockDeployer struct {
	deploymentID string
	err          error
}

func (m *mockDeployer) CreateWorkflowDeployment(_ context.Context, _ WaveDeploymentRequest) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.deploymentID, nil
}
