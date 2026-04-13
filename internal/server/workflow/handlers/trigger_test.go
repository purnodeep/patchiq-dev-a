package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestTriggerHandler_Manual(t *testing.T) {
	h := NewTriggerHandler()
	exec := &workflow.ExecutionContext{
		Node: workflow.Node{
			ID:       "trigger-1",
			NodeType: workflow.NodeTrigger,
			Config:   json.RawMessage(`{"trigger_type":"manual"}`),
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
	if result.Output["trigger_type"] != "manual" {
		t.Errorf("trigger_type = %v, want manual", result.Output["trigger_type"])
	}
}

func TestTriggerHandler_InvalidConfig(t *testing.T) {
	h := NewTriggerHandler()
	exec := &workflow.ExecutionContext{
		Node: workflow.Node{
			ID:     "trigger-1",
			Config: json.RawMessage(`{invalid`),
		},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}
