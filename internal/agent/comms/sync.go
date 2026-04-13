package comms

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AgentState provides key-value persistence using the agent_state table.
type AgentState struct {
	db *sql.DB
}

// NewAgentState creates an AgentState backed by the given SQLite database.
func NewAgentState(db *sql.DB) *AgentState {
	return &AgentState{db: db}
}

// Get retrieves a value by key. Returns empty string if not found.
func (s *AgentState) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM agent_state WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("agent state get %q: %w", key, err)
	}
	return value, nil
}

// Set upserts a key-value pair.
func (s *AgentState) Set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agent_state (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("agent state set %q: %w", key, err)
	}
	return nil
}

// SyncConfig configures the outbox sync runner.
type SyncConfig struct {
	BatchSize    int
	SyncInterval time.Duration
	MaxAttempts  int
	// OfflineCheck, if non-nil, is called before each sync cycle. When it
	// returns true the cycle is skipped (agent is in offline mode).
	OfflineCheck func() bool
	// BandwidthFunc, if non-nil, returns the current bandwidth limit in Kbps.
	// 0 means unlimited. Called before each sync cycle to pick up runtime changes.
	BandwidthFunc func() int
}

// OutboxSyncer abstracts the gRPC SyncOutbox bidi stream.
type OutboxSyncer interface {
	Send(msg *pb.OutboxMessage) error
	Recv() (*pb.OutboxAck, error)
	CloseSend() error
}

// OutboxStreamOpener opens a new SyncOutbox bidi stream.
type OutboxStreamOpener interface {
	OpenOutboxStream(ctx context.Context) (OutboxSyncer, error)
}

// SyncRunner drains the outbox over a gRPC SyncOutbox stream.
type SyncRunner struct {
	stream        OutboxSyncer
	opener        OutboxStreamOpener
	outbox        *Outbox
	config        SyncConfig
	logger        *slog.Logger
	triggerC      chan struct{}
	offlineCheck  func() bool
	bandwidthFunc func() int // returns current bandwidth limit in Kbps; 0 = unlimited
	throttler     *Throttler
}

// NewSyncRunner creates a SyncRunner that sends pending outbox items over the given stream.
// This constructor is used for testing where a pre-built stream is available.
func NewSyncRunner(stream OutboxSyncer, outbox *Outbox, config SyncConfig, logger *slog.Logger) *SyncRunner {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 5
	}
	return &SyncRunner{stream: stream, outbox: outbox, config: config, logger: logger, triggerC: make(chan struct{}, 1)}
}

// NewSyncRunnerWithOpener creates a SyncRunner that opens streams on demand via the opener.
// This is the production constructor used by Client.Run.
func NewSyncRunnerWithOpener(opener OutboxStreamOpener, outbox *Outbox, config SyncConfig, logger *slog.Logger) *SyncRunner {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 5
	}
	if config.SyncInterval <= 0 {
		config.SyncInterval = 30 * time.Second
	}
	r := &SyncRunner{opener: opener, outbox: outbox, config: config, logger: logger, triggerC: make(chan struct{}, 1), offlineCheck: config.OfflineCheck}
	if config.BandwidthFunc != nil {
		r.SetBandwidthFunc(config.BandwidthFunc)
	}
	return r
}

// SetOfflineCheck sets a function called before each sync cycle. When it
// returns true the cycle is skipped (agent is in offline mode). Must be
// called before Run.
func (r *SyncRunner) SetOfflineCheck(f func() bool) {
	r.offlineCheck = f
}

// SetBandwidthFunc sets a callback that returns the current bandwidth limit
// in Kbps. The throttler is updated before each sync cycle so runtime
// changes to the setting take effect promptly. Must be called before Run.
func (r *SyncRunner) SetBandwidthFunc(f func() int) {
	r.bandwidthFunc = f
	r.throttler = NewThrottler(f())
}

// Trigger requests an immediate sync cycle. Non-blocking; if a trigger is already
// pending it is coalesced.
func (r *SyncRunner) Trigger() {
	select {
	case r.triggerC <- struct{}{}:
	default:
		// Already triggered, coalesce.
	}
}

// Run executes the periodic sync loop. It syncs immediately on start (to drain
// items queued while offline), then every SyncInterval. Extra syncs can be
// requested via Trigger(). Run returns when ctx is cancelled or a fatal stream
// error occurs that cannot be recovered by reopening the stream.
func (r *SyncRunner) Run(ctx context.Context) error {
	r.logger.InfoContext(ctx, "sync runner started", "interval", r.config.SyncInterval)

	// Drain immediately on connect.
	r.runSyncCycle(ctx)

	ticker := time.NewTicker(r.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.InfoContext(ctx, "sync runner stopped")
			return ctx.Err()
		case <-ticker.C:
			r.runSyncCycle(ctx)
		case <-r.triggerC:
			r.runSyncCycle(ctx)
		}
	}
}

// runSyncCycle opens a stream (if using an opener) and calls SyncOnce. Errors
// are logged but do not stop the loop — the next tick will retry.
func (r *SyncRunner) runSyncCycle(ctx context.Context) {
	if r.offlineCheck != nil && r.offlineCheck() {
		r.logger.DebugContext(ctx, "sync: skipping cycle, agent is offline")
		return
	}

	// Refresh throttler limit from current settings before each cycle.
	if r.bandwidthFunc != nil && r.throttler != nil {
		r.throttler.SetLimit(r.bandwidthFunc())
	}

	if r.opener != nil {
		stream, err := r.opener.OpenOutboxStream(ctx)
		if err != nil {
			r.logger.WarnContext(ctx, "sync: open outbox stream failed, will retry", "error", err)
			return
		}
		r.stream = stream
	}
	if r.stream == nil {
		r.logger.WarnContext(ctx, "sync: no stream available, skipping cycle")
		return
	}
	if err := r.SyncOnce(ctx); err != nil {
		r.logger.WarnContext(ctx, "sync: cycle failed, will retry", "error", err)
	}
	// Close the stream after each cycle when using opener (stream-per-sync).
	if r.opener != nil {
		if closeErr := r.stream.CloseSend(); closeErr != nil {
			r.logger.Debug("sync: close stream after cycle", "error", closeErr)
		}
		r.stream = nil
	}
}

// SyncOnce drains up to BatchSize pending items from the outbox, sending each over the
// stream and processing the server's ack. Dead-lettered items (attempts >= MaxAttempts)
// are marked failed without being sent. Transient rejections stop the batch early so the
// next tick can retry.
func (r *SyncRunner) SyncOnce(ctx context.Context) error {
	items, err := r.outbox.Pending(ctx, r.config.BatchSize)
	if err != nil {
		return fmt.Errorf("sync: pending: %w", err)
	}
	if len(items) == 0 {
		return nil
	}

	for _, item := range items {
		// Dead-letter items that exceeded max attempts.
		if item.Attempts >= r.config.MaxAttempts {
			if markErr := r.outbox.MarkFailed(ctx, item.ID, fmt.Sprintf("exceeded max attempts (%d)", r.config.MaxAttempts)); markErr != nil {
				return fmt.Errorf("sync: dead-letter id=%d: %w", item.ID, markErr)
			}
			r.logger.WarnContext(ctx, "dead-lettered outbox item", "id", item.ID, "attempts", item.Attempts)
			continue
		}

		msgType := mapMessageType(item.MessageType)
		ts, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
		if err != nil {
			// CreatedAt is written by the agent itself; a parse failure means
			// the row is corrupt. Fall back to now so the message still sends.
			r.logger.WarnContext(ctx, "outbox item has unparseable created_at, using now",
				"id", item.ID, "created_at", item.CreatedAt, "error", err)
			ts = time.Now().UTC()
		}

		msg := &pb.OutboxMessage{
			MessageId:       strconv.FormatInt(item.ID, 10),
			ProtocolVersion: 1,
			Type:            msgType,
			Payload:         item.Payload,
			Timestamp:       timestamppb.New(ts),
		}

		// Apply bandwidth throttling before sending.
		if r.throttler != nil {
			msgSize := proto.Size(msg)
			r.throttler.Throttle(msgSize)
		}

		if sendErr := r.stream.Send(msg); sendErr != nil {
			return fmt.Errorf("sync: send id=%d: %w", item.ID, sendErr)
		}

		ack, recvErr := r.stream.Recv()
		if recvErr != nil {
			return fmt.Errorf("sync: recv: %w", recvErr)
		}

		switch {
		case ack.GetRejectionCode() == pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_UNSPECIFIED:
			if markErr := r.outbox.MarkSent(ctx, item.ID); markErr != nil {
				return fmt.Errorf("sync: mark sent id=%d: %w", item.ID, markErr)
			}
		case isTransient(ack.GetRejectionCode()):
			if incErr := r.outbox.IncrementAttempts(ctx, item.ID, ack.GetRejectionDetail()); incErr != nil {
				return fmt.Errorf("sync: increment attempts id=%d: %w", item.ID, incErr)
			}
			r.logger.WarnContext(ctx, "transient rejection", "id", item.ID, "detail", ack.GetRejectionDetail())
			return nil // stop batch, retry next tick
		default:
			if markErr := r.outbox.MarkFailed(ctx, item.ID, ack.GetRejectionDetail()); markErr != nil {
				return fmt.Errorf("sync: mark failed id=%d: %w", item.ID, markErr)
			}
			r.logger.ErrorContext(ctx, "permanent rejection", "id", item.ID, "code", ack.GetRejectionCode(), "detail", ack.GetRejectionDetail())
		}
	}
	return nil
}

func mapMessageType(s string) pb.OutboxMessageType {
	switch s {
	case "inventory":
		return pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_INVENTORY
	case "command_result":
		return pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_COMMAND_RESULT
	case "heartbeat":
		return pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_HEARTBEAT
	case "event":
		return pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_EVENT
	default:
		return pb.OutboxMessageType_OUTBOX_MESSAGE_TYPE_UNSPECIFIED
	}
}

func isTransient(code pb.OutboxRejectionCode) bool {
	return code == pb.OutboxRejectionCode_OUTBOX_REJECTION_CODE_SERVER_OVERLOADED
}
