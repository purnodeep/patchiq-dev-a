package comms_test

import (
	"context"
	"strconv"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

// mockOutboxSyncer implements comms.OutboxSyncer for testing.
type mockOutboxSyncer struct {
	sent []*pb.OutboxMessage
	acks []*pb.OutboxAck
	idx  int
}

func (m *mockOutboxSyncer) Send(msg *pb.OutboxMessage) error {
	m.sent = append(m.sent, msg)
	return nil
}

func (m *mockOutboxSyncer) Recv() (*pb.OutboxAck, error) {
	if m.idx >= len(m.acks) {
		// Default: accept with no rejection.
		return &pb.OutboxAck{MessageId: m.sent[len(m.sent)-1].GetMessageId()}, nil
	}
	ack := m.acks[m.idx]
	m.idx++
	return ack, nil
}

func (m *mockOutboxSyncer) CloseSend() error { return nil }

func TestSyncRunner_SendsAndAcks(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id1, err := outbox.Add(ctx, "inventory", []byte("payload-1"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	id2, err := outbox.Add(ctx, "heartbeat", []byte("payload-2"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	mock := &mockOutboxSyncer{}
	runner := comms.NewSyncRunner(mock, outbox, comms.SyncConfig{
		BatchSize:   100,
		MaxAttempts: 5,
	}, nil)

	if err := runner.SyncOnce(ctx); err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}

	if len(mock.sent) != 2 {
		t.Fatalf("expected 2 sent messages, got %d", len(mock.sent))
	}

	if mock.sent[0].GetMessageId() != strconv.FormatInt(id1, 10) {
		t.Errorf("expected message_id %q, got %q", strconv.FormatInt(id1, 10), mock.sent[0].GetMessageId())
	}
	if mock.sent[1].GetMessageId() != strconv.FormatInt(id2, 10) {
		t.Errorf("expected message_id %q, got %q", strconv.FormatInt(id2, 10), mock.sent[1].GetMessageId())
	}

	pending, err := outbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after sync, got %d", len(pending))
	}
}

func TestSyncRunner_TransientRejection_IncrementsAttempts(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id1, err := outbox.Add(ctx, "inventory", []byte("payload-1"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	mock := &mockOutboxSyncer{
		acks: []*pb.OutboxAck{
			{
				MessageId:       strconv.FormatInt(id1, 10),
				RejectionCode:   pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED,
				RejectionDetail: "server busy",
			},
		},
	}

	runner := comms.NewSyncRunner(mock, outbox, comms.SyncConfig{
		BatchSize:   100,
		MaxAttempts: 5,
	}, nil)

	if err := runner.SyncOnce(ctx); err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}

	pending, err := outbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending (transient rejection keeps item), got %d", len(pending))
	}
	if pending[0].Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", pending[0].Attempts)
	}
}

func TestSyncRunner_PermanentRejection_MarksFailed(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id1, err := outbox.Add(ctx, "inventory", []byte("bad-payload"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	mock := &mockOutboxSyncer{
		acks: []*pb.OutboxAck{
			{
				MessageId:       strconv.FormatInt(id1, 10),
				RejectionCode:   pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_PAYLOAD_INVALID,
				RejectionDetail: "invalid payload",
			},
		},
	}

	runner := comms.NewSyncRunner(mock, outbox, comms.SyncConfig{
		BatchSize:   100,
		MaxAttempts: 5,
	}, nil)

	if err := runner.SyncOnce(ctx); err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}

	pending, err := outbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after permanent rejection, got %d", len(pending))
	}
}

func TestSyncRunner_DeadLetter_ExceedsMaxAttempts(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	id1, err := outbox.Add(ctx, "inventory", []byte("data"))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Manually increment attempts to 4 (exceeds maxAttempts=4).
	for i := 0; i < 4; i++ {
		if err := outbox.IncrementAttempts(ctx, id1, "transient"); err != nil {
			t.Fatalf("IncrementAttempts: %v", err)
		}
	}

	mock := &mockOutboxSyncer{}
	runner := comms.NewSyncRunner(mock, outbox, comms.SyncConfig{
		BatchSize:   100,
		MaxAttempts: 4,
	}, nil)

	if err := runner.SyncOnce(ctx); err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}

	// Should NOT have sent anything — item was dead-lettered before send.
	if len(mock.sent) != 0 {
		t.Errorf("expected 0 sent messages for dead-lettered item, got %d", len(mock.sent))
	}

	pending, err := outbox.Pending(ctx, 10)
	if err != nil {
		t.Fatalf("Pending: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after dead-letter, got %d", len(pending))
	}
}

func TestSyncRunner_EmptyOutbox_NoSend(t *testing.T) {
	outbox := openTestDB(t)
	ctx := context.Background()

	mock := &mockOutboxSyncer{}
	runner := comms.NewSyncRunner(mock, outbox, comms.SyncConfig{
		BatchSize:   100,
		MaxAttempts: 5,
	}, nil)

	if err := runner.SyncOnce(ctx); err != nil {
		t.Fatalf("SyncOnce: %v", err)
	}

	if len(mock.sent) != 0 {
		t.Errorf("expected 0 sent messages for empty outbox, got %d", len(mock.sent))
	}
}
