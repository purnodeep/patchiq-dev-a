package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestDecisionHandler_Equals(t *testing.T) {
	h := NewDecisionHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{"field":"trigger_type","operator":"equals","value":"manual"}`)},
		Context: map[string]any{"trigger_type": "manual"},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output["branch"] != "yes" {
		t.Errorf("branch = %v, want yes", result.Output["branch"])
	}
}

func TestDecisionHandler_GreaterThan(t *testing.T) {
	h := NewDecisionHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{"field":"filtered_endpoint_count","operator":"gt","value":"5"}`)},
		Context: map[string]any{"filtered_endpoint_count": 10},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output["branch"] != "yes" {
		t.Errorf("branch = %v, want yes", result.Output["branch"])
	}
}

func TestDecisionHandler_LessThan_No(t *testing.T) {
	h := NewDecisionHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{"field":"filtered_endpoint_count","operator":"lt","value":"5"}`)},
		Context: map[string]any{"filtered_endpoint_count": 10},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output["branch"] != "no" {
		t.Errorf("branch = %v, want no", result.Output["branch"])
	}
}

func TestDecisionHandler_FieldNotFound(t *testing.T) {
	h := NewDecisionHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{"field":"missing","operator":"equals","value":"x"}`)},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output["branch"] != "no" {
		t.Errorf("branch = %v, want no", result.Output["branch"])
	}
}

func TestDecisionHandler_NotEquals(t *testing.T) {
	h := NewDecisionHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{"field":"status","operator":"not_equals","value":"failed"}`)},
		Context: map[string]any{"status": "running"},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output["branch"] != "yes" {
		t.Errorf("branch = %v, want yes", result.Output["branch"])
	}
}

func TestDecisionHandler_In(t *testing.T) {
	h := NewDecisionHandler()
	exec := &workflow.ExecutionContext{
		Node:    workflow.Node{Config: json.RawMessage(`{"field":"os","operator":"in","value":"linux,windows"}`)},
		Context: map[string]any{"os": "linux"},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Output["branch"] != "yes" {
		t.Errorf("branch = %v, want yes", result.Output["branch"])
	}
}
