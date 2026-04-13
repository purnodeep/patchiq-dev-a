package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestCompleteHandler_Basic(t *testing.T) {
	h := NewCompleteHandler()
	exec := &workflow.ExecutionContext{
		Node: workflow.Node{
			ID:     "complete-1",
			Config: json.RawMessage(`{}`),
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
	if result.Output["generate_report"] != false {
		t.Errorf("generate_report = %v, want false", result.Output["generate_report"])
	}
	if result.Output["notify_on_complete"] != false {
		t.Errorf("notify_on_complete = %v, want false", result.Output["notify_on_complete"])
	}
}

func TestCompleteHandler_WithReportFlag(t *testing.T) {
	h := NewCompleteHandler()
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "complete-1",
			Config: json.RawMessage(`{"generate_report":true,"notify_on_complete":true}`),
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
	if result.Output["generate_report"] != true {
		t.Errorf("generate_report = %v, want true", result.Output["generate_report"])
	}
	if result.Output["notify_on_complete"] != true {
		t.Errorf("notify_on_complete = %v, want true", result.Output["notify_on_complete"])
	}
}
