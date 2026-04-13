package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestRollbackHandler_Success(t *testing.T) {
	requester := &mockRollbackRequester{rollbackID: "rb-456"}
	h := NewRollbackHandler(requester)
	exec := &workflow.ExecutionContext{
		TenantID:    "tenant-1",
		ExecutionID: "exec-1",
		Node: workflow.Node{
			ID:     "rollback-1",
			Config: json.RawMessage(`{"strategy":"snapshot_restore","failure_threshold":5}`),
		},
		Context: map[string]any{"deployment_id": "deploy-123"},
	}

	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecCompleted)
	}
	if result.Output["rollback_id"] != "rb-456" {
		t.Errorf("rollback_id = %v, want rb-456", result.Output["rollback_id"])
	}
	if result.Output["strategy"] != "snapshot_restore" {
		t.Errorf("strategy = %v, want snapshot_restore", result.Output["strategy"])
	}

	// Verify the request was passed correctly.
	if requester.lastReq.DeploymentID != "deploy-123" {
		t.Errorf("DeploymentID = %q, want deploy-123", requester.lastReq.DeploymentID)
	}
	if requester.lastReq.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want tenant-1", requester.lastReq.TenantID)
	}
	if requester.lastReq.ExecutionID != "exec-1" {
		t.Errorf("ExecutionID = %q, want exec-1", requester.lastReq.ExecutionID)
	}
	if requester.lastReq.NodeID != "rollback-1" {
		t.Errorf("NodeID = %q, want rollback-1", requester.lastReq.NodeID)
	}
}

func TestRollbackHandler_NoDeploymentID(t *testing.T) {
	requester := &mockRollbackRequester{}
	h := NewRollbackHandler(requester)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "rollback-1",
			Config: json.RawMessage(`{"strategy":"snapshot_restore","failure_threshold":5}`),
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
	if result.Error == "" {
		t.Error("expected non-empty error message")
	}
}

func TestRollbackHandler_CreateRollbackError(t *testing.T) {
	requester := &mockRollbackRequester{err: errors.New("service unavailable")}
	h := NewRollbackHandler(requester)
	exec := &workflow.ExecutionContext{
		TenantID:    "tenant-1",
		ExecutionID: "exec-1",
		Node: workflow.Node{
			ID:     "rollback-1",
			Config: json.RawMessage(`{"strategy":"snapshot_restore","failure_threshold":5}`),
		},
		Context: map[string]any{"deployment_id": "deploy-123"},
	}

	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type mockRollbackRequester struct {
	rollbackID string
	err        error
	lastReq    RollbackRequest
}

func (m *mockRollbackRequester) CreateRollback(_ context.Context, req RollbackRequest) (string, error) {
	m.lastReq = req
	if m.err != nil {
		return "", m.err
	}
	return m.rollbackID, nil
}
