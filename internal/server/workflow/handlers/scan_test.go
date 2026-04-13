package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestScanHandler_Success(t *testing.T) {
	sender := &mockCommandSender{}
	h := NewScanHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "scan-1",
			Config: json.RawMessage(`{"scan_type":"inventory"}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1", "ep-2", "ep-3"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, workflow.NodeExecCompleted)
	}
	if sender.sentCount != 3 {
		t.Errorf("sent count = %d, want 3", sender.sentCount)
	}
	if sender.lastCommandType != "run_scan" {
		t.Errorf("command type = %q, want run_scan", sender.lastCommandType)
	}
	if result.Output["scan_type"] != "inventory" {
		t.Errorf("output scan_type = %v, want inventory", result.Output["scan_type"])
	}
}

func TestScanHandler_NoEndpoints(t *testing.T) {
	sender := &mockCommandSender{}
	h := NewScanHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "scan-1",
			Config: json.RawMessage(`{"scan_type":"inventory"}`),
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

func TestScanHandler_SendError(t *testing.T) {
	sender := &mockCommandSender{err: fmt.Errorf("timeout")}
	h := NewScanHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "scan-1",
			Config: json.RawMessage(`{"scan_type":"vulnerability"}`),
		},
		Context: map[string]any{
			"filtered_endpoints": []string{"ep-1"},
		},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute should not return error: %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want completed (best-effort)", result.Status)
	}
	if result.Output["failed_count"] != 1 {
		t.Errorf("failed_count = %v, want 1", result.Output["failed_count"])
	}
}

func TestScanHandler_InvalidConfig(t *testing.T) {
	sender := &mockCommandSender{}
	h := NewScanHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "scan-1",
			Config: json.RawMessage(`not-json`),
		},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}
