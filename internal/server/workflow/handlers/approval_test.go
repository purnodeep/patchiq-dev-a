package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestApprovalHandler_Pauses(t *testing.T) {
	store := &mockApprovalStore{}
	h := NewApprovalHandler(store)
	exec := &workflow.ExecutionContext{
		TenantID:    "tenant-1",
		ExecutionID: "exec-1",
		Node: workflow.Node{
			ID:     "approval-1",
			Config: json.RawMessage(`{"approver_roles":["admin","security"],"timeout_hours":24,"escalation_role":"director"}`),
		},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Pause {
		t.Error("expected Pause=true")
	}
	if result.Status != workflow.NodeExecRunning {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecRunning)
	}
	if store.createdCount != 1 {
		t.Errorf("created count = %d, want 1", store.createdCount)
	}
}

func TestApprovalHandler_InvalidConfig(t *testing.T) {
	store := &mockApprovalStore{}
	h := NewApprovalHandler(store)
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{invalid`)},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

func TestApprovalHandler_StoreError(t *testing.T) {
	store := &mockApprovalStore{err: fmt.Errorf("db error")}
	h := NewApprovalHandler(store)
	exec := &workflow.ExecutionContext{
		TenantID:    "tenant-1",
		ExecutionID: "exec-1",
		Node: workflow.Node{
			ID:     "approval-1",
			Config: json.RawMessage(`{"approver_roles":["admin"],"timeout_hours":24}`),
		},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error when store fails")
	}
}

type mockApprovalStore struct {
	createdCount int
	err          error
}

func (m *mockApprovalStore) CreateApproval(_ context.Context, _ ApprovalCreateRequest) error {
	if m.err != nil {
		return m.err
	}
	m.createdCount++
	return nil
}
