package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestRebootHandler_Success(t *testing.T) {
	sender := &mockCommandSender{}
	h := NewRebootHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "reboot-1",
			Config: json.RawMessage(`{"reboot_mode":"graceful","grace_period_seconds":300}`),
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
	if sender.sentCount != 2 {
		t.Errorf("sent count = %d, want 2", sender.sentCount)
	}
	if sender.lastCommandType != "reboot" {
		t.Errorf("command type = %q, want reboot", sender.lastCommandType)
	}
	if result.Output["reboot_mode"] != "graceful" {
		t.Errorf("output reboot_mode = %v, want graceful", result.Output["reboot_mode"])
	}
	if result.Output["endpoint_count"] != 2 {
		t.Errorf("output endpoint_count = %v, want 2", result.Output["endpoint_count"])
	}
}

func TestRebootHandler_NoEndpoints(t *testing.T) {
	sender := &mockCommandSender{}
	h := NewRebootHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "reboot-1",
			Config: json.RawMessage(`{"reboot_mode":"immediate","grace_period_seconds":0}`),
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

func TestRebootHandler_SendError(t *testing.T) {
	sender := &mockCommandSender{err: fmt.Errorf("connection refused")}
	h := NewRebootHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "reboot-1",
			Config: json.RawMessage(`{"reboot_mode":"graceful","grace_period_seconds":60}`),
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

func TestRebootHandler_InvalidConfig(t *testing.T) {
	sender := &mockCommandSender{}
	h := NewRebootHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "reboot-1",
			Config: json.RawMessage(`{invalid`),
		},
		Context: map[string]any{},
	}
	_, err := h.Execute(context.Background(), exec)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}

// mockCommandSender is used by reboot and scan handler tests.
type mockCommandSender struct {
	sentCount       int
	lastCommandType string
	err             error
}

func (m *mockCommandSender) SendCommand(_ context.Context, _ string, _ string, commandType string, _ json.RawMessage) error {
	if m.err != nil {
		return m.err
	}
	m.sentCount++
	m.lastCommandType = commandType
	return nil
}
