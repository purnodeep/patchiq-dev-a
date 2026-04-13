package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestScriptHandler_Pauses(t *testing.T) {
	dispatcher := &mockCommandDispatcher{commandID: "cmd-456"}
	h := NewScriptHandler(dispatcher)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "script-1",
			Config: json.RawMessage(`{"script_body":"echo hello","script_type":"shell","timeout_minutes":5,"failure_behavior":"halt"}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1", "ep-2"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !result.Pause {
		t.Error("expected Pause=true")
	}
	if result.Output["command_id"] != "cmd-456" {
		t.Errorf("command_id = %v, want cmd-456", result.Output["command_id"])
	}
}

func TestScriptHandler_NoEndpoints(t *testing.T) {
	dispatcher := &mockCommandDispatcher{}
	h := NewScriptHandler(dispatcher)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "script-1",
			Config: json.RawMessage(`{"script_body":"echo hello","script_type":"shell","timeout_minutes":5,"failure_behavior":"halt"}`),
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

type mockCommandDispatcher struct {
	commandID string
}

func (m *mockCommandDispatcher) DispatchScript(_ context.Context, _ ScriptDispatchRequest) (string, error) {
	return m.commandID, nil
}
