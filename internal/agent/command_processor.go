package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/agent/comms"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	commandProcessorBatchSize = 10
	commandProcessorInterval  = 2 * time.Second
)

// HistoryRecord represents a completed command for local persistence.
type HistoryRecord struct {
	ID              string
	PatchName       string
	PatchVersion    string
	Action          string
	Result          string
	ErrorMessage    string
	CompletedAt     string
	DurationSeconds int
	RebootRequired  bool
	Stdout          string
	Stderr          string
	ExitCode        int
}

// HistoryWriter persists command execution results to local storage.
type HistoryWriter interface {
	InsertHistoryRecord(ctx context.Context, record HistoryRecord) error
}

// CommandProcessor polls the inbox and dispatches commands to the registry.
// On completion it writes a command_result outbox message so SyncRunner
// ships the result back to the server.
type CommandProcessor struct {
	inbox        *comms.Inbox
	outbox       OutboxWriter
	registry     *Registry
	logger       *slog.Logger
	triggerC     chan struct{}
	offlineCheck func() bool
	history      HistoryWriter
	logWriter    OperationalLogWriter
}

// NewCommandProcessor creates a CommandProcessor.
func NewCommandProcessor(inbox *comms.Inbox, outbox OutboxWriter, registry *Registry, logger *slog.Logger) *CommandProcessor {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &CommandProcessor{
		inbox:    inbox,
		outbox:   outbox,
		registry: registry,
		logger:   logger,
		triggerC: make(chan struct{}, 1),
	}
}

// SetHistoryWriter sets the local history store for persisting command results.
// Must be called before Run.
func (p *CommandProcessor) SetHistoryWriter(h HistoryWriter) {
	p.history = h
}

// SetOfflineCheck sets a function called before each processing cycle.
// When it returns true the cycle is skipped (agent is in offline mode,
// commands from the server inbox are not processed until back online).
// Must be called before Run.
func (p *CommandProcessor) SetOfflineCheck(f func() bool) {
	p.offlineCheck = f
}

// SetLogWriter sets the persistent log writer for recording command execution events.
// Must be called before Run.
func (p *CommandProcessor) SetLogWriter(lw OperationalLogWriter) {
	p.logWriter = lw
}

// Trigger requests an immediate processing cycle. Non-blocking; coalesces
// if a trigger is already pending.
func (p *CommandProcessor) Trigger() {
	select {
	case p.triggerC <- struct{}{}:
	default:
		// Already triggered, coalesce.
	}
}

// Run starts the command processor loop. Blocks until ctx is cancelled.
func (p *CommandProcessor) Run(ctx context.Context) {
	p.logger.InfoContext(ctx, "command processor started", "interval", commandProcessorInterval)

	// Process immediately on start to drain any commands queued while offline.
	p.processOnce(ctx)

	ticker := time.NewTicker(commandProcessorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.InfoContext(ctx, "command processor stopped")
			return
		case <-ticker.C:
			p.processOnce(ctx)
		case <-p.triggerC:
			p.processOnce(ctx)
		}
	}
}

// processOnce fetches pending inbox items and dispatches each one.
func (p *CommandProcessor) processOnce(ctx context.Context) {
	if p.offlineCheck != nil && p.offlineCheck() {
		p.logger.DebugContext(ctx, "command processor: skipping cycle, agent is offline")
		return
	}
	items, err := p.inbox.Pending(ctx, commandProcessorBatchSize)
	if err != nil {
		p.logger.ErrorContext(ctx, "command processor: fetch pending", "error", err)
		return
	}
	if len(items) == 0 {
		return
	}

	p.logger.InfoContext(ctx, "command processor: dispatching commands", "count", len(items))
	for _, item := range items {
		p.dispatch(ctx, item)
	}
}

// dispatch executes a single inbox item and records the result.
func (p *CommandProcessor) dispatch(ctx context.Context, item comms.InboxItem) {
	cmdLogger := p.logger.With("command_id", item.ID, "command_type", item.CommandType)

	// Normalize the command type: inbox stores the protobuf enum string
	// (e.g. "COMMAND_TYPE_INSTALL_PATCH") but registry keys use short names
	// (e.g. "install_patch").
	cmdType := normalizeCommandType(item.CommandType)

	cmd := Command{
		ID:      item.ID,
		Type:    cmdType,
		Payload: item.Payload,
	}

	cmdLogger.InfoContext(ctx, "command processor: dispatching", "normalized_type", cmdType)

	p.writeOperationalLog(ctx, "info", fmt.Sprintf("Executing command %s (id: %s)", cmdType, item.ID), "command_processor")

	result, err := p.registry.HandleCommand(ctx, cmd)
	if err != nil {
		cmdLogger.ErrorContext(ctx, "command processor: dispatch failed", "error", err)
		p.writeOperationalLog(ctx, "error", fmt.Sprintf("Command %s failed: %v", cmdType, err), "command_processor")
		if markErr := p.inbox.MarkFailed(ctx, item.ID, err.Error()); markErr != nil {
			cmdLogger.ErrorContext(ctx, "command processor: mark inbox failed", "error", markErr)
		}
		if outboxErr := p.writeResult(ctx, item.ID, pb.CommandStatus_COMMAND_STATUS_FAILED, nil, err.Error()); outboxErr != nil {
			cmdLogger.ErrorContext(ctx, "command processor: write failure result to outbox", "error", outboxErr)
		}
		return
	}

	// A non-empty ErrorMessage in Result means the command ran but reported failure
	// (e.g. a package install exit code != 0). This is a "soft" failure — the
	// command did execute.
	var status pb.CommandStatus
	if result.ErrorMessage != "" {
		status = pb.CommandStatus_COMMAND_STATUS_FAILED
		cmdLogger.WarnContext(ctx, "command processor: command reported failure", "error_message", result.ErrorMessage)
		p.writeOperationalLog(ctx, "warn", fmt.Sprintf("Command %s reported failure: %s", cmdType, result.ErrorMessage), "command_processor")
	} else {
		status = pb.CommandStatus_COMMAND_STATUS_SUCCEEDED
		cmdLogger.InfoContext(ctx, "command processor: command succeeded")
		p.writeOperationalLog(ctx, "info", fmt.Sprintf("Command %s completed successfully", cmdType), "command_processor")
	}

	resultBytes, _ := proto.Marshal(&pb.CommandResponse{
		CommandId:    item.ID,
		Status:       status,
		Output:       result.Output,
		ErrorMessage: result.ErrorMessage,
		CompletedAt:  timestamppb.Now(),
	})
	if markErr := p.inbox.MarkCompleted(ctx, item.ID, resultBytes); markErr != nil {
		cmdLogger.ErrorContext(ctx, "command processor: mark inbox completed", "error", markErr)
	}
	if outboxErr := p.writeResult(ctx, item.ID, status, result.Output, result.ErrorMessage); outboxErr != nil {
		cmdLogger.ErrorContext(ctx, "command processor: write result to outbox", "error", outboxErr)
	}

	// Persist to local history so the agent UI shows real deployment results.
	if p.history != nil && cmdType == "install_patch" {
		p.recordInstallHistory(ctx, item.ID, cmdType, result, status, cmdLogger)
	}
}

// writeResult serializes a CommandResponse and adds it to the outbox.
func (p *CommandProcessor) writeResult(ctx context.Context, commandID string, status pb.CommandStatus, output []byte, errMsg string) error {
	resp := &pb.CommandResponse{
		CommandId:    commandID,
		Status:       status,
		Output:       output,
		ErrorMessage: errMsg,
		CompletedAt:  timestamppb.Now(),
	}
	payload, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("command processor: marshal command response: %w", err)
	}
	if _, err := p.outbox.Add(ctx, "command_result", payload); err != nil {
		return fmt.Errorf("command processor: add to outbox: %w", err)
	}
	return nil
}

// recordInstallHistory extracts package details from an install_patch result
// and persists a history record locally.
func (p *CommandProcessor) recordInstallHistory(ctx context.Context, cmdID, _ string, result Result, status pb.CommandStatus, logger *slog.Logger) {
	var output pb.InstallPatchOutput
	if err := proto.Unmarshal(result.Output, &output); err != nil {
		logger.DebugContext(ctx, "command processor: could not parse install output for history", "error", err)
		return
	}

	resultStr := "success"
	if status == pb.CommandStatus_COMMAND_STATUS_FAILED {
		resultStr = "failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)

	for _, detail := range output.Results {
		rec := HistoryRecord{
			ID:             fmt.Sprintf("%s-%s", cmdID, detail.PackageName),
			PatchName:      detail.PackageName,
			PatchVersion:   detail.Version,
			Action:         "install",
			Result:         resultStr,
			ErrorMessage:   result.ErrorMessage,
			CompletedAt:    now,
			RebootRequired: detail.RebootRequired,
			ExitCode:       int(detail.ExitCode),
			Stdout:         detail.Stdout,
			Stderr:         detail.Stderr,
		}
		if err := p.history.InsertHistoryRecord(ctx, rec); err != nil {
			logger.WarnContext(ctx, "command processor: persist history record", "package", detail.PackageName, "error", err)
		}
	}
}

// writeOperationalLog persists an operational log entry if a log writer is configured.
func (p *CommandProcessor) writeOperationalLog(ctx context.Context, level, message, source string) {
	if p.logWriter == nil {
		return
	}
	if err := p.logWriter.WriteLog(ctx, level, message, source); err != nil {
		p.logger.WarnContext(ctx, "write operational log", "error", err)
	}
}

// normalizeCommandType converts a protobuf enum string like
// "COMMAND_TYPE_INSTALL_PATCH" to the short form "install_patch" expected by
// module SupportedCommands(). If the string doesn't have the prefix it is
// returned lowercased as-is, so already-normalised values pass through.
func normalizeCommandType(s string) string {
	const prefix = "COMMAND_TYPE_"
	if strings.HasPrefix(s, prefix) {
		return strings.ToLower(s[len(prefix):])
	}
	return strings.ToLower(s)
}
