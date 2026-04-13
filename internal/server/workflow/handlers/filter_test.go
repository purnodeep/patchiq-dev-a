package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestFilterHandler_Success(t *testing.T) {
	ds := &mockFilterDataSource{endpoints: []string{"ep-1", "ep-2", "ep-3", "ep-4", "ep-5"}}
	h := NewFilterHandler(ds)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "filter-1",
			Config: json.RawMessage(`{"os_types":["linux"]}`),
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
	count, ok := result.Output["filtered_endpoint_count"].(int)
	if !ok || count != 5 {
		t.Errorf("filtered_endpoint_count = %v, want 5", result.Output["filtered_endpoint_count"])
	}
}

func TestFilterHandler_ZeroEndpoints(t *testing.T) {
	ds := &mockFilterDataSource{endpoints: []string{}}
	h := NewFilterHandler(ds)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "filter-1",
			Config: json.RawMessage(`{"os_types":["linux"]}`),
		},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecFailed {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecFailed)
	}
}

type mockFilterDataSource struct {
	endpoints []string
}

func (m *mockFilterDataSource) FilterEndpoints(_ context.Context, _ string, _ workflow.FilterConfig) ([]string, error) {
	return m.endpoints, nil
}
