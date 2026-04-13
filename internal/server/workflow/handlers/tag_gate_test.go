package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestTagGateHandler_MatchPass(t *testing.T) {
	resolver := &mockTagResolver{
		tags: map[string][]string{
			"ep-1": {"production", "linux"},
			"ep-2": {"production", "windows"},
		},
	}
	h := NewTagGateHandler(resolver)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "tag-gate-1",
			Config: json.RawMessage(`{"tag_expression":"production"}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1", "ep-2"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecCompleted)
	}
	if result.Output["matched_count"] != 2 {
		t.Errorf("matched_count = %v, want 2", result.Output["matched_count"])
	}
}

func TestTagGateHandler_PartialMatch(t *testing.T) {
	resolver := &mockTagResolver{
		tags: map[string][]string{
			"ep-1": {"production", "linux"},
			"ep-2": {"staging", "windows"},
		},
	}
	h := NewTagGateHandler(resolver)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "tag-gate-1",
			Config: json.RawMessage(`{"tag_expression":"production"}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1", "ep-2"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecCompleted)
	}
	if result.Output["matched_count"] != 1 {
		t.Errorf("matched_count = %v, want 1", result.Output["matched_count"])
	}
}

func TestTagGateHandler_NoMatch(t *testing.T) {
	resolver := &mockTagResolver{
		tags: map[string][]string{
			"ep-1": {"staging"},
			"ep-2": {"staging"},
		},
	}
	h := NewTagGateHandler(resolver)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "tag-gate-1",
			Config: json.RawMessage(`{"tag_expression":"production"}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1", "ep-2"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecFailed {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecFailed)
	}
}

func TestTagGateHandler_NoEndpoints(t *testing.T) {
	resolver := &mockTagResolver{tags: map[string][]string{}}
	h := NewTagGateHandler(resolver)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "tag-gate-1",
			Config: json.RawMessage(`{"tag_expression":"production"}`),
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

func TestTagGateHandler_InvalidConfig(t *testing.T) {
	resolver := &mockTagResolver{tags: map[string][]string{}}
	h := NewTagGateHandler(resolver)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "tag-gate-1",
			Config: json.RawMessage(`{bad}`),
		},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

type mockTagResolver struct {
	tags map[string][]string
}

func (m *mockTagResolver) ResolveTags(_ context.Context, _ string, endpointIDs []string) (map[string][]string, error) {
	result := make(map[string][]string)
	for _, id := range endpointIDs {
		if tags, ok := m.tags[id]; ok {
			result[id] = tags
		}
	}
	return result, nil
}
