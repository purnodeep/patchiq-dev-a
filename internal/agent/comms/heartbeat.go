package comms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Sentinel errors returned by RunHeartbeat.
var (
	ErrReEnrollRequired  = errors.New("server requested re-enrollment")
	ErrShutdownRequested = errors.New("server requested shutdown")
)

// HeartbeatConfig controls heartbeat stream behavior.
type HeartbeatConfig struct {
	Interval        time.Duration
	AgentID         string
	ProtocolVersion uint32
	StartTime       time.Time
	Logger          *slog.Logger
	// OnCommandsPending is called when the server indicates pending commands.
	// Used to trigger an immediate outbox/inbox sync cycle.
	OnCommandsPending func()
	// IntervalFunc, if non-nil, is called each tick to get the current
	// heartbeat interval. This allows the interval to change at runtime
	// (e.g. via the settings watcher). When nil, Interval is used as a
	// fixed value.
	IntervalFunc func() time.Duration
	// OfflineCheck, if non-nil, is called before each heartbeat send.
	// When it returns true the send is skipped (agent is in offline mode).
	OfflineCheck func() bool
	// OnHeartbeatSent, if non-nil, is called after each successful heartbeat
	// send with the timestamp of the send. Used to update the local status
	// provider so the agent's own /api/v1/status reflects the last heartbeat.
	OnHeartbeatSent func(time.Time)
}

// HeartbeatStream abstracts the bidirectional gRPC stream.
type HeartbeatStream interface {
	Send(req *pb.HeartbeatRequest) error
	Recv() (*pb.HeartbeatResponse, error)
	CloseSend() error
}

// HeartbeatStreamer opens a new heartbeat stream.
type HeartbeatStreamer interface {
	OpenHeartbeat(ctx context.Context) (HeartbeatStream, error)
}

// RunHeartbeat opens a heartbeat stream and runs send/receive loops.
// Returns ErrReEnrollRequired, ErrShutdownRequested, context error, or stream error.
func RunHeartbeat(ctx context.Context, streamer HeartbeatStreamer, cfg HeartbeatConfig, outbox *Outbox) error {
	if cfg.Interval <= 0 {
		return fmt.Errorf("heartbeat validate: interval must be > 0, got %v", cfg.Interval)
	}
	if cfg.AgentID == "" {
		return fmt.Errorf("heartbeat validate: agent_id is required")
	}
	if cfg.StartTime.IsZero() {
		return fmt.Errorf("heartbeat validate: start_time must be set")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	stream, err := streamer.OpenHeartbeat(ctx)
	if err != nil {
		return fmt.Errorf("open heartbeat stream: %w", err)
	}

	// Child context so cancellation propagates to both goroutines when the first one fails.
	innerCtx, innerCancel := context.WithCancel(ctx)
	defer innerCancel()

	errCh := make(chan error, 2)

	// Sender goroutine.
	go func() {
		errCh <- runHeartbeatSender(innerCtx, stream, cfg, outbox, logger)
	}()

	// Receiver goroutine.
	go func() {
		errCh <- runHeartbeatReceiver(innerCtx, stream, cfg, logger)
	}()

	// Wait for first error from either goroutine, then cancel the other.
	select {
	case e := <-errCh:
		innerCancel()
		if closeErr := stream.CloseSend(); closeErr != nil {
			logger.Debug("close heartbeat stream", "error", closeErr)
		}
		<-errCh // wait for second goroutine
		return e
	case <-ctx.Done():
		innerCancel()
		if closeErr := stream.CloseSend(); closeErr != nil {
			logger.Debug("close heartbeat stream", "error", closeErr)
		}
		<-errCh // wait for first goroutine
		<-errCh // wait for second goroutine
		return ctx.Err()
	}
}

func runHeartbeatSender(ctx context.Context, stream HeartbeatStream, cfg HeartbeatConfig, outbox *Outbox, logger *slog.Logger) error {
	// Send first beat immediately (unless offline).
	if cfg.OfflineCheck == nil || !cfg.OfflineCheck() {
		if err := sendHeartbeat(ctx, stream, cfg, outbox, logger); err != nil {
			return fmt.Errorf("heartbeat send: %w", err)
		}
	} else {
		logger.InfoContext(ctx, "heartbeat: skipping initial send, agent is offline")
	}

	currentInterval := cfg.Interval
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check offline mode before sending.
			if cfg.OfflineCheck != nil && cfg.OfflineCheck() {
				logger.DebugContext(ctx, "heartbeat: skipping send, agent is offline")
				continue
			}
			if err := sendHeartbeat(ctx, stream, cfg, outbox, logger); err != nil {
				return fmt.Errorf("heartbeat send: %w", err)
			}
			// Check if interval has changed dynamically.
			if cfg.IntervalFunc != nil {
				newInterval := cfg.IntervalFunc()
				if newInterval > 0 && newInterval != currentInterval {
					logger.InfoContext(ctx, "heartbeat: interval changed", "old", currentInterval, "new", newInterval)
					currentInterval = newInterval
					ticker.Reset(currentInterval)
				}
			}
		}
	}
}

func sendHeartbeat(ctx context.Context, stream HeartbeatStream, cfg HeartbeatConfig, outbox *Outbox, logger *slog.Logger) error {
	cpuPct, memUsed, diskUsed := systemResourceUsage(ctx)

	var queueDepth uint32
	if outbox != nil {
		count, err := outbox.PendingCount(ctx)
		if err != nil {
			logger.WarnContext(ctx, "heartbeat: failed to read outbox pending count, reporting max", "error", err)
			queueDepth = ^uint32(0) // sentinel: server should treat max as "unknown"
		} else {
			queueDepth = uint32(count)
		}
	}

	req := &pb.HeartbeatRequest{
		AgentId:         cfg.AgentID,
		ProtocolVersion: cfg.ProtocolVersion,
		Timestamp:       timestamppb.Now(),
		Status:          pb.AgentStatus_AGENT_STATUS_IDLE,
		ResourceUsage: &pb.ResourceUsage{
			CpuPercent:  cpuPct,
			MemoryBytes: memUsed,
			DiskBytes:   diskUsed,
		},
		UptimeSeconds:     int64(time.Since(cfg.StartTime).Seconds()),
		OfflineQueueDepth: queueDepth,
	}

	if err := stream.Send(req); err != nil {
		return err
	}
	if cfg.OnHeartbeatSent != nil {
		cfg.OnHeartbeatSent(time.Now())
	}
	return nil
}

func runHeartbeatReceiver(ctx context.Context, stream HeartbeatStream, cfg HeartbeatConfig, logger *slog.Logger) error {
	for {
		resp, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("heartbeat recv: %w", err)
		}

		switch resp.Directive {
		case pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_RE_ENROLL:
			logger.WarnContext(ctx, "server requested re-enrollment", "message", resp.DirectiveMessage)
			return ErrReEnrollRequired
		case pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_SHUTDOWN:
			logger.WarnContext(ctx, "server requested shutdown", "message", resp.DirectiveMessage)
			return ErrShutdownRequested
		case pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_UPDATE_REQUIRED:
			logger.WarnContext(ctx, "server indicated update required", "message", resp.DirectiveMessage)
		case pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_PROTOCOL_UNSUPPORTED:
			logger.WarnContext(ctx, "server indicated protocol unsupported", "message", resp.DirectiveMessage)
			return ErrReEnrollRequired
		default:
			if resp.Directive != pb.HeartbeatDirective_HEARTBEAT_DIRECTIVE_UNSPECIFIED {
				logger.WarnContext(ctx, "received unrecognized heartbeat directive",
					"directive", int32(resp.Directive),
					"message", resp.DirectiveMessage,
				)
			}
		}

		if resp.CommandsPending > 0 {
			logger.InfoContext(ctx, "server has pending commands", "count", resp.CommandsPending)
			if cfg.OnCommandsPending != nil {
				cfg.OnCommandsPending()
			}
		}
	}
}
