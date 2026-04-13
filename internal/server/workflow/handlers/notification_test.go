package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestNotificationHandler_Success(t *testing.T) {
	sender := &mockNotificationSender{}
	h := NewNotificationHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "notify-1",
			Config: json.RawMessage(`{"channel":"slack","target":"#ops","message_template":"Deploy started"}`),
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
	if sender.sentCount != 1 {
		t.Errorf("sent count = %d, want 1", sender.sentCount)
	}
}

func TestNotificationHandler_SendError(t *testing.T) {
	sender := &mockNotificationSender{err: fmt.Errorf("connection refused")}
	h := NewNotificationHandler(sender)
	exec := &workflow.ExecutionContext{
		TenantID: "tenant-1",
		Node: workflow.Node{
			ID:     "notify-1",
			Config: json.RawMessage(`{"channel":"slack","target":"#ops"}`),
		},
		Context: map[string]any{},
	}
	result, err := h.Execute(context.Background(), exec)
	if err != nil {
		t.Fatalf("Execute should not return error (fire-and-forget): %v", err)
	}
	if result.Status != workflow.NodeExecCompleted {
		t.Errorf("status = %q, want completed (fire-and-forget)", result.Status)
	}
	if result.Output["send_error"] == nil {
		t.Error("expected send_error in output")
	}
}

type mockNotificationSender struct {
	sentCount int
	err       error
}

func (m *mockNotificationSender) SendNotification(_ context.Context, _ string, _ workflow.NotificationConfig) error {
	if m.err != nil {
		return m.err
	}
	m.sentCount++
	return nil
}
