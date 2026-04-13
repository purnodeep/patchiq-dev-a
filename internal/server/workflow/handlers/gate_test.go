package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestGateHandler_Pauses(t *testing.T) {
	h := NewGateHandler()
	exec := &workflow.ExecutionContext{
		Node: workflow.Node{
			ID:     "gate-1",
			Config: json.RawMessage(`{"wait_minutes":60,"failure_threshold":5,"health_check":true}`),
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
	if result.Output["wait_minutes"] != 60 {
		t.Errorf("wait_minutes = %v, want 60", result.Output["wait_minutes"])
	}
}

func TestGateHandler_InvalidConfig(t *testing.T) {
	h := NewGateHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{invalid`)},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}
