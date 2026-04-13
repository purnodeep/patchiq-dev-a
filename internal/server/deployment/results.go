package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// ResultQuerier defines queries needed by the result handler.
type ResultQuerier interface {
	GetCommandByID(ctx context.Context, arg sqlcgen.GetCommandByIDParams) (sqlcgen.Command, error)
	UpdateCommandStatus(ctx context.Context, arg sqlcgen.UpdateCommandStatusParams) (sqlcgen.Command, error)
	UpdateDeploymentTargetStatus(ctx context.Context, arg sqlcgen.UpdateDeploymentTargetStatusParams) (sqlcgen.DeploymentTarget, error)
	IncrementDeploymentCounters(ctx context.Context, arg sqlcgen.IncrementDeploymentCountersParams) (sqlcgen.Deployment, error)
	GetDeploymentTargetWaveID(ctx context.Context, arg sqlcgen.GetDeploymentTargetWaveIDParams) (pgtype.UUID, error)
	IncrementWaveCounters(ctx context.Context, arg sqlcgen.IncrementWaveCountersParams) (sqlcgen.DeploymentWave, error)
	CompleteQuerier
	FailQuerier
}

// ResultTxFactory creates a tenant-scoped transaction and returns a querier bound to it,
// along with commit and rollback functions. This ensures all writes happen within
// a transaction with the correct tenant context for RLS enforcement.
type ResultTxFactory func(ctx context.Context, tenantID string) (q ResultQuerier, commit func() error, rollback func() error, err error)

// ResultHandler processes command results and updates deployment state.
type ResultHandler struct {
	q         ResultQuerier
	txFactory ResultTxFactory
	sm        *StateMachine
	eventBus  domain.EventBus
}

// NewResultHandler creates a ResultHandler. The querier q is used for reads.
// Use WithResultTxFactory to enable tenant-scoped transactions for writes (required for RLS).
// Without a TxFactory, writes use the querier directly — only safe in tests without RLS.
func NewResultHandler(q ResultQuerier, sm *StateMachine, eventBus domain.EventBus, opts ...func(*ResultHandler)) *ResultHandler {
	if q == nil {
		panic("deployment: NewResultHandler called with nil querier")
	}
	if sm == nil {
		panic("deployment: NewResultHandler called with nil stateMachine")
	}
	if eventBus == nil {
		panic("deployment: NewResultHandler called with nil eventBus")
	}
	rh := &ResultHandler{q: q, sm: sm, eventBus: eventBus}
	for _, opt := range opts {
		opt(rh)
	}
	return rh
}

// WithResultTxFactory sets the transaction factory for tenant-scoped writes.
func WithResultTxFactory(f ResultTxFactory) func(*ResultHandler) {
	return func(rh *ResultHandler) {
		rh.txFactory = f
	}
}

// HandleResult processes a command result from an agent.
func (rh *ResultHandler) HandleResult(ctx context.Context, commandID, tenantID pgtype.UUID, succeeded bool, stdout, stderr, errMsg string, exitCode *int32) error {
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()

	// 1. Look up command.
	cmd, err := rh.q.GetCommandByID(ctx, sqlcgen.GetCommandByIDParams{ID: commandID, TenantID: tenantID})
	if err != nil {
		return fmt.Errorf("handle result: get command: %w", err)
	}

	// 2. Determine which querier to use for writes.
	// If a TxFactory is configured, all writes go through a tenant-scoped transaction.
	writeQ, commit, rollback, err := rh.beginWriteTx(ctx, tenantIDStr)
	if err != nil {
		return fmt.Errorf("handle result: begin tenant tx: %w", err)
	}
	defer func() {
		if rbErr := rollback(); rbErr != nil {
			slog.ErrorContext(ctx, "handle result: rollback failed", "error", rbErr)
		}
	}()

	// 3. Update command status.
	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	cmdStatus := string(CommandSucceeded)
	if !succeeded {
		cmdStatus = string(CommandFailed)
	}
	var errorMessage pgtype.Text
	if errMsg != "" {
		errorMessage = pgtype.Text{String: errMsg, Valid: true}
	}

	if _, err := writeQ.UpdateCommandStatus(ctx, sqlcgen.UpdateCommandStatusParams{
		ID:           commandID,
		Status:       cmdStatus,
		CompletedAt:  now,
		ErrorMessage: errorMessage,
		TenantID:     tenantID,
	}); err != nil {
		return fmt.Errorf("handle result: update command: %w", err)
	}

	// Collect events to emit after successful commit (avoid phantom events on rollback).
	var pendingEvents []domain.DomainEvent

	// 4. Update deployment target if linked.
	if cmd.TargetID.Valid {
		targetStatus := string(TargetSucceeded)
		if !succeeded {
			targetStatus = string(TargetFailed)
		}
		// Use the exit code passed from the event payload (extracted at the gRPC layer
		// before the protobuf bytes go through JSON serialization in the event bus).
		var ec pgtype.Int4
		if exitCode != nil {
			ec = pgtype.Int4{Int32: *exitCode, Valid: true}
		}

		// Use the command's created_at as a proxy for started_at.
		startedAt := pgtype.Timestamptz{Time: cmd.CreatedAt.Time, Valid: cmd.CreatedAt.Valid}

		if _, err := writeQ.UpdateDeploymentTargetStatus(ctx, sqlcgen.UpdateDeploymentTargetStatusParams{
			ID:           cmd.TargetID,
			Status:       targetStatus,
			StartedAt:    startedAt,
			CompletedAt:  now,
			ErrorMessage: errorMessage,
			Stdout:       pgtype.Text{String: stdout, Valid: stdout != ""},
			Stderr:       pgtype.Text{String: stderr, Valid: stderr != ""},
			ExitCode:     ec,
			TenantID:     tenantID,
		}); err != nil {
			return fmt.Errorf("handle result: update target: %w", err)
		}

		targetIDStr := uuid.UUID(cmd.TargetID.Bytes).String()
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(events.DeploymentEndpointCompleted, tenantIDStr, "deployment_target", targetIDStr, events.DeploymentEndpointCompleted, nil),
		)

		// 4b. Increment wave counters if target belongs to a wave.
		waveID, waveErr := writeQ.GetDeploymentTargetWaveID(ctx, sqlcgen.GetDeploymentTargetWaveIDParams{
			ID:       cmd.TargetID,
			TenantID: tenantID,
		})
		if waveErr != nil {
			slog.WarnContext(ctx, "handle result: get target wave_id", "target_id", targetIDStr, "error", waveErr)
		} else if waveID.Valid {
			if _, waveErr = writeQ.IncrementWaveCounters(ctx, sqlcgen.IncrementWaveCountersParams{
				IsSuccess: succeeded,
				WaveID:    waveID,
				TenantID:  tenantID,
			}); waveErr != nil {
				return fmt.Errorf("handle result: increment wave counters: %w", waveErr)
			}
		}
	}

	// scan.completed is emitted in addition to CommandResultReceived so the
	// endpoint audit timeline has a terminal event for ad-hoc scans (which are
	// not tied to a deployment and therefore skip the deployment events below).
	if cmd.Type == string(CommandTypeRunScan) {
		endpointIDStr := uuid.UUID(cmd.AgentID.Bytes).String()
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(
				events.ScanCompleted,
				tenantIDStr,
				"endpoint",
				endpointIDStr,
				events.ScanCompleted,
				events.ScanCompletedPayload{
					CommandID:    uuid.UUID(commandID.Bytes).String(),
					EndpointID:   endpointIDStr,
					Succeeded:    succeeded,
					ErrorMessage: errMsg,
				},
			),
		)
	}

	// 5. Update deployment counters and check completion/failure.
	if cmd.DeploymentID.Valid {
		d, err := writeQ.IncrementDeploymentCounters(ctx, sqlcgen.IncrementDeploymentCountersParams{
			IsSuccess: succeeded,
			ID:        cmd.DeploymentID,
			TenantID:  tenantID,
		})
		if err != nil {
			return fmt.Errorf("handle result: increment counters: %w", err)
		}

		thresholdEvents, err := checkDeploymentThreshold(ctx, rh.sm, writeQ, d, cmd.DeploymentID, tenantID)
		if err != nil {
			return fmt.Errorf("handle result: check threshold: %w", err)
		}
		pendingEvents = append(pendingEvents, thresholdEvents...)
	}

	// 6. Commit transaction (no-op if no txFactory configured).
	if err := commit(); err != nil {
		return fmt.Errorf("handle result: commit tx: %w", err)
	}

	// Post-commit event emission is best-effort: the DB state is authoritative.
	EmitBestEffort(ctx, rh.eventBus, pendingEvents)

	return nil
}

// beginWriteTx returns a querier for writes. If txFactory is set, it starts a tenant-scoped
// transaction. Otherwise, it returns the default querier with no-op commit/rollback.
func (rh *ResultHandler) beginWriteTx(ctx context.Context, tenantID string) (ResultQuerier, func() error, func() error, error) {
	if rh.txFactory != nil {
		return rh.txFactory(ctx, tenantID)
	}
	// No txFactory — writes bypass RLS. This is only safe in tests.
	slog.WarnContext(ctx, "ResultHandler: no txFactory configured, writes bypass RLS tenant isolation")
	noop := func() error { return nil }
	return rh.q, noop, noop, nil
}
