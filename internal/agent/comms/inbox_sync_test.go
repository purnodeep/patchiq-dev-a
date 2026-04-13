package comms_test

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

// mockInboxFetcher implements comms.InboxFetcher for testing.
type mockInboxFetcher struct {
	stream comms.InboxStream
	err    error
}

func (m *mockInboxFetcher) FetchCommands(ctx context.Context) (comms.InboxStream, error) {
	return m.stream, m.err
}

// mockInboxStream implements comms.InboxStream for testing.
type mockInboxStream struct {
	commands []*pb.CommandRequest
	idx      int
}

func (m *mockInboxStream) Recv() (*pb.CommandRequest, error) {
	if m.idx >= len(m.commands) {
		return nil, io.EOF
	}
	cmd := m.commands[m.idx]
	m.idx++
	return cmd, nil
}

// errorAfterStream sends all commands then returns a non-EOF error.
type errorAfterStream struct {
	commands []*pb.CommandRequest
	idx      int
	err      error
}

func (m *errorAfterStream) Recv() (*pb.CommandRequest, error) {
	if m.idx >= len(m.commands) {
		return nil, m.err
	}
	cmd := m.commands[m.idx]
	m.idx++
	return cmd, nil
}

// partialFailStream sends commands and calls a hook before delivering the second command.
// This allows the first command to be stored successfully, then the hook can break the DB.
type partialFailStream struct {
	commands     []*pb.CommandRequest
	idx          int
	beforeSecond func()
}

func (m *partialFailStream) Recv() (*pb.CommandRequest, error) {
	if m.idx >= len(m.commands) {
		return nil, io.EOF
	}
	if m.idx == 1 && m.beforeSecond != nil {
		m.beforeSecond()
	}
	cmd := m.commands[m.idx]
	m.idx++
	return cmd, nil
}

func newTestInbox(t *testing.T) *comms.Inbox {
	t.Helper()
	dir := t.TempDir()
	db, err := comms.OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return comms.NewInbox(db)
}

func newBrokenInbox(t *testing.T) *comms.Inbox {
	t.Helper()
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "broken.db"))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	// Inbox backed by a DB without the inbox table — all stores will fail.
	t.Cleanup(func() { db.Close() })
	return comms.NewInbox(db)
}

func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestFetchInbox_HappyPath(t *testing.T) {
	inbox := newTestInbox(t)
	ctx := context.Background()

	stream := &mockInboxStream{
		commands: []*pb.CommandRequest{
			{CommandId: "cmd-1", Type: pb.CommandType_COMMAND_TYPE_RUN_SCAN, Payload: []byte("p1"), Priority: 5},
			{CommandId: "cmd-2", Type: pb.CommandType_COMMAND_TYPE_INSTALL_PATCH, Payload: []byte("p2"), Priority: 10},
		},
	}
	fetcher := &mockInboxFetcher{stream: stream}

	err := comms.FetchInbox(ctx, fetcher, inbox, testLogger())
	if err != nil {
		t.Fatalf("FetchInbox returned error: %v", err)
	}

	items, err := inbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 pending items, got %d", len(items))
	}
	// Pending returns highest priority first.
	if items[0].ID != "cmd-2" {
		t.Errorf("expected cmd-2 first (priority 10), got %q", items[0].ID)
	}
	if items[1].ID != "cmd-1" {
		t.Errorf("expected cmd-1 second (priority 5), got %q", items[1].ID)
	}
}

func TestFetchInbox_EmptyStream(t *testing.T) {
	inbox := newTestInbox(t)
	ctx := context.Background()

	stream := &mockInboxStream{commands: nil}
	fetcher := &mockInboxFetcher{stream: stream}

	err := comms.FetchInbox(ctx, fetcher, inbox, testLogger())
	if err != nil {
		t.Fatalf("FetchInbox returned error: %v", err)
	}

	items, err := inbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 pending items, got %d", len(items))
	}
}

func TestFetchInbox_StreamError(t *testing.T) {
	inbox := newTestInbox(t)
	ctx := context.Background()

	stream := &errorAfterStream{
		commands: []*pb.CommandRequest{
			{CommandId: "cmd-1", Type: pb.CommandType_COMMAND_TYPE_RUN_SCAN, Payload: []byte("p1")},
		},
		err: fmt.Errorf("network failure"),
	}
	fetcher := &mockInboxFetcher{stream: stream}

	err := comms.FetchInbox(ctx, fetcher, inbox, testLogger())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "fetch inbox recv: network failure" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestFetchInbox_FetcherError(t *testing.T) {
	inbox := newTestInbox(t)
	ctx := context.Background()

	fetcher := &mockInboxFetcher{err: fmt.Errorf("connection refused")}

	err := comms.FetchInbox(ctx, fetcher, inbox, testLogger())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "fetch inbox: connection refused" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestFetchInbox_AllStoresFail(t *testing.T) {
	inbox := newBrokenInbox(t)
	ctx := context.Background()

	stream := &mockInboxStream{
		commands: []*pb.CommandRequest{
			{CommandId: "cmd-1", Type: pb.CommandType_COMMAND_TYPE_RUN_SCAN, Payload: []byte("p1")},
			{CommandId: "cmd-2", Type: pb.CommandType_COMMAND_TYPE_RUN_SCAN, Payload: []byte("p2")},
		},
	}
	fetcher := &mockInboxFetcher{stream: stream}

	err := comms.FetchInbox(ctx, fetcher, inbox, testLogger())
	if err == nil {
		t.Fatal("expected error when all stores fail, got nil")
	}
	expected := "fetch inbox: all 2 received commands failed to store"
	if got := err.Error(); got != expected {
		t.Errorf("expected error %q, got %q", expected, got)
	}
}

func TestFetchInbox_PartialStoreFailure(t *testing.T) {
	ctx := context.Background()

	// Create a real DB, then drop the inbox table mid-stream after the first
	// command is stored successfully. The hook fires before delivering the
	// second command, so the first store succeeds and the second fails.
	dir := t.TempDir()
	db, err := comms.OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	inbox := comms.NewInbox(db)

	stream := &partialFailStream{
		commands: []*pb.CommandRequest{
			{CommandId: "cmd-ok", Type: pb.CommandType_COMMAND_TYPE_RUN_SCAN, Payload: []byte("p1"), Priority: 1},
			{CommandId: "cmd-fail", Type: pb.CommandType_COMMAND_TYPE_RUN_SCAN, Payload: []byte("p2"), Priority: 2},
		},
		beforeSecond: func() {
			// Drop the inbox table so the second store fails.
			db.Exec("DROP TABLE inbox") //nolint:errcheck
		},
	}
	fetcher := &mockInboxFetcher{stream: stream}

	err = comms.FetchInbox(ctx, fetcher, inbox, testLogger())
	// Partial failure: one stored, one failed. Should return nil.
	if err != nil {
		t.Fatalf("expected nil for partial failure, got: %v", err)
	}
}
