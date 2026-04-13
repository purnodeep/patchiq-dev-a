package comms_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

// mockHeartbeatStream implements comms.HeartbeatStream for testing.
type mockHeartbeatStream struct {
	mu            sync.Mutex
	sent          []*pb.HeartbeatRequest
	recvResponses []*pb.HeartbeatResponse
	recvIdx       int
	closed        bool
	closeCh       chan struct{}
}

func (m *mockHeartbeatStream) Send(req *pb.HeartbeatRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, req)
	return nil
}

func (m *mockHeartbeatStream) Recv() (*pb.HeartbeatResponse, error) {
	m.mu.Lock()
	if m.recvIdx >= len(m.recvResponses) {
		m.mu.Unlock()
		// Block until stream is closed — simulates idle stream waiting for server messages.
		<-m.closeCh
		return nil, errors.New("stream closed")
	}
	resp := m.recvResponses[m.recvIdx]
	m.recvIdx++
	m.mu.Unlock()
	return resp, nil
}

func (m *mockHeartbeatStream) CloseSend() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closeCh)
	}
	return nil
}

// Sent returns a copy of sent messages.
func (m *mockHeartbeatStream) Sent() []*pb.HeartbeatRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]*pb.HeartbeatRequest, len(m.sent))
	copy(cp, m.sent)
	return cp
}

// mockStreamer implements comms.HeartbeatStreamer for testing.
type mockStreamer struct {
	stream *mockHeartbeatStream
	err    error
}

func (m *mockStreamer) OpenHeartbeat(ctx context.Context) (comms.HeartbeatStream, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.stream, nil
}

func TestRunHeartbeat_RejectsZeroInterval(t *testing.T) {
	stream := &mockHeartbeatStream{closeCh: make(chan struct{})}
	streamer := &mockStreamer{stream: stream}

	cfg := comms.HeartbeatConfig{
		Interval:        0,
		AgentID:         "agent-1",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(context.Background(), streamer, cfg, nil)
	if err == nil {
		t.Fatal("expected error for zero interval")
	}
	if got := err.Error(); !strings.Contains(got, "interval must be > 0") {
		t.Fatalf("expected error containing %q, got: %v", "interval must be > 0", got)
	}
}

func TestRunHeartbeat_RejectsEmptyAgentID(t *testing.T) {
	stream := &mockHeartbeatStream{closeCh: make(chan struct{})}
	streamer := &mockStreamer{stream: stream}

	cfg := comms.HeartbeatConfig{
		Interval:        time.Second,
		AgentID:         "",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(context.Background(), streamer, cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty agent_id")
	}
	if got := err.Error(); !strings.Contains(got, "agent_id is required") {
		t.Fatalf("expected error containing %q, got: %v", "agent_id is required", got)
	}
}

func TestRunHeartbeat_RejectsZeroStartTime(t *testing.T) {
	stream := &mockHeartbeatStream{closeCh: make(chan struct{})}
	streamer := &mockStreamer{stream: stream}

	cfg := comms.HeartbeatConfig{
		Interval:        time.Second,
		AgentID:         "agent-1",
		ProtocolVersion: 1,
		// StartTime intentionally zero
	}

	err := comms.RunHeartbeat(context.Background(), streamer, cfg, nil)
	if err == nil {
		t.Fatal("expected error for zero start_time")
	}
	if got := err.Error(); !strings.Contains(got, "start_time must be set") {
		t.Fatalf("expected error containing %q, got: %v", "start_time must be set", got)
	}
}

func TestRunHeartbeat_HandlesProtocolUnsupported(t *testing.T) {
	stream := &mockHeartbeatStream{
		closeCh: make(chan struct{}),
		recvResponses: []*pb.HeartbeatResponse{
			{Directive: pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_PROTOCOL_UNSUPPORTED},
		},
	}
	streamer := &mockStreamer{stream: stream}
	outbox, _, _ := openTestDBRaw(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := comms.HeartbeatConfig{
		Interval:        100 * time.Millisecond,
		AgentID:         "agent-proto",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(ctx, streamer, cfg, outbox)
	if !errors.Is(err, comms.ErrReEnrollRequired) {
		t.Fatalf("expected ErrReEnrollRequired, got: %v", err)
	}
}

func TestRunHeartbeat_OpenHeartbeatFailure(t *testing.T) {
	streamer := &mockStreamer{err: errors.New("connection refused")}

	cfg := comms.HeartbeatConfig{
		Interval:        time.Second,
		AgentID:         "agent-fail",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(context.Background(), streamer, cfg, nil)
	if err == nil {
		t.Fatal("expected error for open heartbeat failure")
	}
	if got := err.Error(); !strings.Contains(got, "open heartbeat stream") {
		t.Fatalf("expected error containing %q, got: %v", "open heartbeat stream", got)
	}
}

func TestRunHeartbeat_NilOutbox(t *testing.T) {
	stream := &mockHeartbeatStream{closeCh: make(chan struct{})}
	streamer := &mockStreamer{stream: stream}

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	cfg := comms.HeartbeatConfig{
		Interval:        100 * time.Millisecond,
		AgentID:         "agent-nil-outbox",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(ctx, streamer, cfg, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	sent := stream.Sent()
	if len(sent) == 0 {
		t.Fatal("expected at least 1 heartbeat message")
	}
	if sent[0].OfflineQueueDepth != 0 {
		t.Errorf("OfflineQueueDepth = %d, want 0", sent[0].OfflineQueueDepth)
	}
}

func TestRunHeartbeat_SendsPeriodicBeats(t *testing.T) {
	stream := &mockHeartbeatStream{closeCh: make(chan struct{})}
	streamer := &mockStreamer{stream: stream}
	outbox, _, _ := openTestDBRaw(t)

	// macOS top command takes ~1-2s per invocation, so use generous timeouts.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := comms.HeartbeatConfig{
		Interval:        100 * time.Millisecond,
		AgentID:         "agent-123",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(ctx, streamer, cfg, outbox)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	sent := stream.Sent()
	// With 100ms interval and 5s timeout, expect at least 2 beats even with slow macOS top.
	if len(sent) < 2 {
		t.Fatalf("expected at least 2 heartbeat messages, got %d", len(sent))
	}
	for i, msg := range sent {
		if msg.AgentId != "agent-123" {
			t.Errorf("sent[%d].AgentId = %q, want %q", i, msg.AgentId, "agent-123")
		}
		if msg.ProtocolVersion != 1 {
			t.Errorf("sent[%d].ProtocolVersion = %d, want 1", i, msg.ProtocolVersion)
		}
	}
}

func TestRunHeartbeat_HandlesREEnrollDirective(t *testing.T) {
	stream := &mockHeartbeatStream{
		closeCh: make(chan struct{}),
		recvResponses: []*pb.HeartbeatResponse{
			{Directive: pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_RE_ENROLL},
		},
	}
	streamer := &mockStreamer{stream: stream}
	outbox, _, _ := openTestDBRaw(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := comms.HeartbeatConfig{
		Interval:        100 * time.Millisecond,
		AgentID:         "agent-456",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(ctx, streamer, cfg, outbox)
	if !errors.Is(err, comms.ErrReEnrollRequired) {
		t.Fatalf("expected ErrReEnrollRequired, got: %v", err)
	}
}

func TestRunHeartbeat_HandlesShutdownDirective(t *testing.T) {
	stream := &mockHeartbeatStream{
		closeCh: make(chan struct{}),
		recvResponses: []*pb.HeartbeatResponse{
			{Directive: pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_SHUTDOWN},
		},
	}
	streamer := &mockStreamer{stream: stream}
	outbox, _, _ := openTestDBRaw(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg := comms.HeartbeatConfig{
		Interval:        100 * time.Millisecond,
		AgentID:         "agent-789",
		ProtocolVersion: 1,
		StartTime:       time.Now(),
	}

	err := comms.RunHeartbeat(ctx, streamer, cfg, outbox)
	if !errors.Is(err, comms.ErrShutdownRequested) {
		t.Fatalf("expected ErrShutdownRequested, got: %v", err)
	}
}

func TestRunHeartbeat_IncludesUptimeAndQueueDepth(t *testing.T) {
	stream := &mockHeartbeatStream{closeCh: make(chan struct{})}
	streamer := &mockStreamer{stream: stream}
	outbox, _, _ := openTestDBRaw(t)

	// Add some items to the outbox so queue depth > 0.
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := outbox.Add(ctx, "test", []byte("data")); err != nil {
			t.Fatalf("outbox.Add: %v", err)
		}
	}

	// macOS top command takes ~1-2s per invocation, so use generous timeout.
	runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cfg := comms.HeartbeatConfig{
		Interval:        100 * time.Millisecond,
		AgentID:         "agent-uptime",
		ProtocolVersion: 1,
		StartTime:       time.Now().Add(-5 * time.Minute),
	}

	err := comms.RunHeartbeat(runCtx, streamer, cfg, outbox)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	sent := stream.Sent()
	if len(sent) == 0 {
		t.Fatal("expected at least 1 heartbeat message")
	}

	first := sent[0]
	// Uptime should be approximately 300 seconds (5 minutes).
	if first.UptimeSeconds < 295 || first.UptimeSeconds > 315 {
		t.Errorf("UptimeSeconds = %d, want ~300", first.UptimeSeconds)
	}
	if first.OfflineQueueDepth != 3 {
		t.Errorf("OfflineQueueDepth = %d, want 3", first.OfflineQueueDepth)
	}
}
