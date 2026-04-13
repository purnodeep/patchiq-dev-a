package comms_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestInbox_Store(t *testing.T) {
	_, inbox, _ := openTestDBRaw(t)
	ctx := context.Background()

	err := inbox.Store(ctx, comms.InboxItem{
		ID: "cmd-1", CommandType: "run_scan", Payload: []byte("payload"), Priority: 1,
	})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
}

func TestInbox_Pending_OrderedByPriority(t *testing.T) {
	_, inbox, _ := openTestDBRaw(t)
	ctx := context.Background()

	inbox.Store(ctx, comms.InboxItem{ID: "low", CommandType: "run_scan", Payload: []byte("a"), Priority: 0})        //nolint:errcheck
	inbox.Store(ctx, comms.InboxItem{ID: "high", CommandType: "install_patch", Payload: []byte("b"), Priority: 10}) //nolint:errcheck
	inbox.Store(ctx, comms.InboxItem{ID: "med", CommandType: "update_config", Payload: []byte("c"), Priority: 5})   //nolint:errcheck

	items, err := inbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].ID != "high" {
		t.Errorf("expected highest priority first, got %q", items[0].ID)
	}
}

func TestInbox_MarkCompleted(t *testing.T) {
	_, inbox, _ := openTestDBRaw(t)
	ctx := context.Background()

	inbox.Store(ctx, comms.InboxItem{ID: "cmd-1", CommandType: "run_scan", Payload: []byte("a")}) //nolint:errcheck
	if err := inbox.MarkCompleted(ctx, "cmd-1", []byte("result")); err != nil {
		t.Fatalf("MarkCompleted: %v", err)
	}

	items, _ := inbox.Pending(ctx, 10)
	if len(items) != 0 {
		t.Errorf("expected 0 pending after completion, got %d", len(items))
	}
}

func TestInbox_Store_DuplicateID_Idempotent(t *testing.T) {
	_, inbox, _ := openTestDBRaw(t)
	ctx := context.Background()

	item := comms.InboxItem{ID: "cmd-1", CommandType: "run_scan", Payload: []byte("a")}
	inbox.Store(ctx, item) //nolint:errcheck
	err := inbox.Store(ctx, item)
	if err != nil {
		t.Fatalf("duplicate Store should be idempotent: %v", err)
	}

	items, _ := inbox.Pending(ctx, 10)
	if len(items) != 1 {
		t.Errorf("expected 1 item after duplicate store, got %d", len(items))
	}
}
