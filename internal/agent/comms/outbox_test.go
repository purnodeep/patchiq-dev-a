package comms_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func openTestDB(t *testing.T) *comms.Outbox {
	t.Helper()
	dir := t.TempDir()
	db, err := comms.OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return comms.NewOutbox(db)
}

func openTestDBRaw(t *testing.T) (*comms.Outbox, *comms.Inbox, *comms.AgentState) {
	t.Helper()
	dir := t.TempDir()
	db, err := comms.OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return comms.NewOutbox(db), comms.NewInbox(db), comms.NewAgentState(db)
}

func TestOutbox_Add(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id, err := outbox.Add(ctx, "inventory", []byte("payload-1"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestOutbox_Pending_OrderedByCreatedAt(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	outbox.Add(ctx, "inventory", []byte("first"))  //nolint:errcheck
	outbox.Add(ctx, "heartbeat", []byte("second")) //nolint:errcheck
	outbox.Add(ctx, "event", []byte("third"))      //nolint:errcheck

	items, err := outbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if string(items[0].Payload) != "first" {
		t.Errorf("expected first payload 'first', got %q", string(items[0].Payload))
	}
}

func TestOutbox_MarkSent(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id, _ := outbox.Add(ctx, "inventory", []byte("data"))
	if err := outbox.MarkSent(ctx, id); err != nil {
		t.Fatalf("MarkSent: %v", err)
	}

	items, _ := outbox.Pending(ctx, 10)
	if len(items) != 0 {
		t.Errorf("expected 0 pending after MarkSent, got %d", len(items))
	}
}

func TestOutbox_MarkFailed(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id, _ := outbox.Add(ctx, "inventory", []byte("data"))
	if err := outbox.MarkFailed(ctx, id, "server error"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	items, _ := outbox.Pending(ctx, 10)
	if len(items) != 0 {
		t.Errorf("expected 0 pending after MarkFailed, got %d", len(items))
	}
}

func TestOutbox_IncrementAttempts(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id, _ := outbox.Add(ctx, "inventory", []byte("data"))
	if err := outbox.IncrementAttempts(ctx, id, "transient error"); err != nil {
		t.Fatalf("IncrementAttempts: %v", err)
	}

	items, _ := outbox.Pending(ctx, 10)
	if len(items) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(items))
	}
	if items[0].Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", items[0].Attempts)
	}
}

func TestOutbox_PendingCount(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	// Empty outbox should return 0.
	count, err := outbox.PendingCount(ctx)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	// Add 3 items, mark 1 sent.
	id1, _ := outbox.Add(ctx, "inventory", []byte("large-payload-1"))
	outbox.Add(ctx, "heartbeat", []byte("large-payload-2")) //nolint:errcheck
	outbox.Add(ctx, "event", []byte("large-payload-3"))     //nolint:errcheck
	outbox.MarkSent(ctx, id1)                               //nolint:errcheck

	count, err = outbox.PendingCount(ctx)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 pending, got %d", count)
	}
}

func TestOutbox_Pending_BatchLimit(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		outbox.Add(ctx, "inventory", []byte("data")) //nolint:errcheck
	}

	items, _ := outbox.Pending(ctx, 3)
	if len(items) != 3 {
		t.Errorf("expected 3 items with limit, got %d", len(items))
	}
}
