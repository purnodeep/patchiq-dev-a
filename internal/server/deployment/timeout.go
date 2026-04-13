package deployment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// TimeoutJobArgs defines the River periodic job for checking timed-out commands.
type TimeoutJobArgs struct{}

// Kind implements river.JobArgs.
func (TimeoutJobArgs) Kind() string { return "deployment_timeout_checker" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TimeoutJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// TimeoutQuerier combines the queries needed by the timeout checker.
type TimeoutQuerier interface {
	ListTimedOutCommands(ctx context.Context) ([]sqlcgen.Command, error)
	UpdateCommandStatus(ctx context.Context, arg sqlcgen.UpdateCommandStatusParams) (sqlcgen.Command, error)
	UpdateDeploymentTargetStatus(ctx context.Context, arg sqlcgen.UpdateDeploymentTargetStatusParams) (sqlcgen.DeploymentTarget, error)
	IncrementDeploymentCounters(ctx context.Context, arg sqlcgen.IncrementDeploymentCountersParams) (sqlcgen.Deployment, error)
	CompleteQuerier
	FailQuerier
}

// TimeoutTxFactory creates a tenant-scoped transaction and returns a querier bound to it,
// along with commit and rollback functions. This ensures all writes happen within
// a transaction with the correct tenant context for RLS enforcement.
type TimeoutTxFactory func(ctx context.Context, tenantID string) (q TimeoutQuerier, commit func() error, rollback func() error, err error)

// TimeoutChecker processes commands that have exceeded their deadline.
type TimeoutChecker struct {
	q         TimeoutQuerier
	txFactory TimeoutTxFactory
	sm        *StateMachine
	eventBus  domain.EventBus
}

// NewTimeoutChecker creates a TimeoutChecker. The StateMachine handles DB transitions
// and returns pending events; the eventBus emits them post-commit.
// Use WithTimeoutTxFactory to enable tenant-scoped transactions for writes (required for RLS).
func NewTimeoutChecker(q TimeoutQuerier, sm *StateMachine, eb domain.EventBus, opts ...func(*TimeoutChecker)) *TimeoutChecker {
	if q == nil {
		panic("deployment: NewTimeoutChecker called with nil querier")
	}
	if sm == nil {
		panic("deployment: NewTimeoutChecker called with nil stateMachine")
	}
	if eb == nil {
		panic("deployment: NewTimeoutChecker called with nil eventBus")
	}
	tc := &TimeoutChecker{q: q, sm: sm, eventBus: eb}
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}

// WithTimeoutTxFactory sets the transaction factory for tenant-scoped writes.
func WithTimeoutTxFactory(f TimeoutTxFactory) func(*TimeoutChecker) {
	return func(tc *TimeoutChecker) {
		tc.txFactory = f
	}
}

// Check finds timed-out commands and transitions them (and their targets/deployments) to failed.
func (tc *TimeoutChecker) Check(ctx context.Context) error {
	commands, err := tc.q.ListTimedOutCommands(ctx)
	if err != nil {
		return fmt.Errorf("timeout checker: list timed-out commands: %w", err)
	}

	var errs []error
	for _, cmd := range commands {
		if err := tc.processTimedOutCommand(ctx, cmd); err != nil {
			slog.ErrorContext(ctx, "timeout checker: process command failed",
				"command_id", cmd.ID,
				"error", err,
			)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("timeout checker: %d of %d commands failed to process: %w", len(errs), len(commands), errors.Join(errs...))
	}

	return nil
}

func (tc *TimeoutChecker) processTimedOutCommand(ctx context.Context, cmd sqlcgen.Command) error {
	tenantIDStr := uuid.UUID(cmd.TenantID.Bytes).String()

	// Get the write querier (tenant-scoped tx if configured, otherwise fallback to pool querier).
	writeQ, commit, rollback, err := tc.beginWriteTx(ctx, tenantIDStr)
	if err != nil {
		return fmt.Errorf("begin tenant tx: %w", err)
	}
	defer func() {
		if rbErr := rollback(); rbErr != nil {
			slog.ErrorContext(ctx, "timeout checker: rollback failed", "error", rbErr)
		}
	}()

	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	// Mark command as failed.
	if _, err := writeQ.UpdateCommandStatus(ctx, sqlcgen.UpdateCommandStatusParams{
		ID:           cmd.ID,
		Status:       string(CommandFailed),
		CompletedAt:  now,
		ErrorMessage: pgtype.Text{String: "command timed out", Valid: true},
		TenantID:     cmd.TenantID,
	}); err != nil {
		return fmt.Errorf("update command status: %w", err)
	}

	// Collect events to emit after successful commit (avoid phantom events on rollback).
	cmdIDStr := uuid.UUID(cmd.ID.Bytes).String()
	var pendingEvents []domain.DomainEvent
	pendingEvents = append(pendingEvents,
		domain.NewSystemEvent(events.CommandTimedOut, tenantIDStr, "command", cmdIDStr, events.CommandTimedOut, nil),
	)

	// Mark linked target as failed if present.
	if cmd.TargetID.Valid {
		if _, err := writeQ.UpdateDeploymentTargetStatus(ctx, sqlcgen.UpdateDeploymentTargetStatusParams{
			ID:           cmd.TargetID,
			Status:       string(TargetFailed),
			CompletedAt:  now,
			ErrorMessage: pgtype.Text{String: "command timed out", Valid: true},
			TenantID:     cmd.TenantID,
		}); err != nil {
			return fmt.Errorf("update target status: %w", err)
		}

		targetIDStr := uuid.UUID(cmd.TargetID.Bytes).String()
		pendingEvents = append(pendingEvents,
			domain.NewSystemEvent(events.DeploymentTargetTimedOut, tenantIDStr, "deployment_target", targetIDStr, events.DeploymentTargetTimedOut, nil),
		)
	}

	// Increment deployment counters if linked to a deployment.
	if cmd.DeploymentID.Valid {
		d, err := writeQ.IncrementDeploymentCounters(ctx, sqlcgen.IncrementDeploymentCountersParams{
			IsSuccess: false,
			ID:        cmd.DeploymentID,
			TenantID:  cmd.TenantID,
		})
		if err != nil {
			return fmt.Errorf("increment deployment counters: %w", err)
		}

		thresholdEvents, err := checkDeploymentThreshold(ctx, tc.sm, writeQ, d, cmd.DeploymentID, cmd.TenantID)
		if err != nil {
			return fmt.Errorf("timeout: %w", err)
		}
		pendingEvents = append(pendingEvents, thresholdEvents...)
	}

	// Commit transaction (no-op if no txFactory configured).
	if err := commit(); err != nil {
		return fmt.Errorf("commit tenant tx: %w", err)
	}

	// Post-commit event emission is best-effort: the DB state is authoritative.
	EmitBestEffort(ctx, tc.eventBus, pendingEvents)

	return nil
}

// beginWriteTx returns a querier for writes. If txFactory is set, it starts a tenant-scoped
// transaction. Otherwise, it returns the default querier with no-op commit/rollback.
func (tc *TimeoutChecker) beginWriteTx(ctx context.Context, tenantID string) (TimeoutQuerier, func() error, func() error, error) {
	if tc.txFactory != nil {
		return tc.txFactory(ctx, tenantID)
	}
	// No txFactory — writes bypass RLS. This is only safe in tests.
	slog.WarnContext(ctx, "TimeoutChecker: no txFactory configured, writes bypass RLS tenant isolation")
	noop := func() error { return nil }
	return tc.q, noop, noop, nil
}

// TimeoutWorker wraps TimeoutChecker as a River worker.
type TimeoutWorker struct {
	river.WorkerDefaults[TimeoutJobArgs]
	checker *TimeoutChecker
}

func NewTimeoutWorker(checker *TimeoutChecker) *TimeoutWorker {
	return &TimeoutWorker{checker: checker}
}

// Work implements river.Worker.
func (w *TimeoutWorker) Work(ctx context.Context, _ *river.Job[TimeoutJobArgs]) error {
	return w.checker.Check(ctx)
}
