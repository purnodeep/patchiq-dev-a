package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestComplianceCheckHandler_AlwaysPasses(t *testing.T) {
	h := NewComplianceCheckHandler()
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "compliance-1",
			Config: json.RawMessage(`{"framework":"CIS"}`),
		},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecCompleted)
	}
	if result.Output["stub"] != true {
		t.Error("expected stub=true in output")
	}
}

func TestComplianceCheckHandler_EmptyConfig(t *testing.T) {
	h := NewComplianceCheckHandler()
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "compliance-1",
			Config: json.RawMessage(`{}`),
		},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want %q (M2 stub always passes)", result.Status, workflow.NodeExecCompleted)
	}
}

func TestComplianceCheckHandler_InvalidConfig(t *testing.T) {
	h := NewComplianceCheckHandler()
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "compliance-1",
			Config: json.RawMessage(`{invalid`),
		},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}
