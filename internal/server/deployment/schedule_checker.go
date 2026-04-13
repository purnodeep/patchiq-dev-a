package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/robfig/cron/v3"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// ScheduleCheckerJobArgs are the arguments for a River schedule checker periodic job.
type ScheduleCheckerJobArgs struct{}

// Kind implements river.JobArgs.
func (ScheduleCheckerJobArgs) Kind() string { return "schedule_checker" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ScheduleCheckerJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// ScheduleCheckerQuerier defines the store methods needed by ScheduleChecker.
type ScheduleCheckerQuerier interface {
	ListDueSchedules(ctx context.Context) ([]sqlcgen.DeploymentSchedule, error)
	HasActiveDeploymentForSchedule(ctx context.Context, arg sqlcgen.HasActiveDeploymentForScheduleParams) (bool, error)
	UpdateScheduleAfterRun(ctx context.Context, arg sqlcgen.UpdateScheduleAfterRunParams) error
	CreateDeploymentWithWaveConfig(ctx context.Context, arg sqlcgen.CreateDeploymentWithWaveConfigParams) (sqlcgen.Deployment, error)
}

// ScheduleChecker finds due deployment schedules and creates deployments for them.
type ScheduleChecker struct {
	q        ScheduleCheckerQuerier
	eventBus domain.EventBus
}

// NewScheduleChecker creates a ScheduleChecker.
func NewScheduleChecker(q ScheduleCheckerQuerier, eventBus domain.EventBus) *ScheduleChecker {
	if q == nil {
		panic("deployment: NewScheduleChecker called with nil querier")
	}
	if eventBus == nil {
		panic("deployment: NewScheduleChecker called with nil eventBus")
	}
	return &ScheduleChecker{q: q, eventBus: eventBus}
}

// Check finds due schedules and creates deployments for each one that does not
// already have an active deployment. Individual schedule failures are logged but
// do not stop processing of remaining schedules.
func (sc *ScheduleChecker) Check(ctx context.Context) error {
	schedules, err := sc.q.ListDueSchedules(ctx)
	if err != nil {
		return fmt.Errorf("schedule checker: list due schedules: %w", err)
	}

	if len(schedules) == 0 {
		slog.InfoContext(ctx, "schedule checker: no due schedules")
		return nil
	}

	slog.InfoContext(ctx, "schedule checker: processing due schedules", "count", len(schedules))

	for _, s := range schedules {
		sc.processSchedule(ctx, s)
	}

	return nil
}

func (sc *ScheduleChecker) processSchedule(ctx context.Context, schedule sqlcgen.DeploymentSchedule) {
	scheduleIDStr := uuid.UUID(schedule.ID.Bytes).String()
	tenantIDStr := uuid.UUID(schedule.TenantID.Bytes).String()
	policyIDStr := uuid.UUID(schedule.PolicyID.Bytes).String()

	// Check if there is already an active deployment for this schedule's policy.
	hasActive, err := sc.q.HasActiveDeploymentForSchedule(ctx, sqlcgen.HasActiveDeploymentForScheduleParams{
		PolicyID: schedule.PolicyID,
		TenantID: schedule.TenantID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "schedule checker: check active deployment",
			"schedule_id", scheduleIDStr, "policy_id", policyIDStr, "error", err)
		return
	}

	if hasActive {
		slog.WarnContext(ctx, "schedule checker: skipping schedule with active deployment",
			"schedule_id", scheduleIDStr, "policy_id", policyIDStr)
		return
	}

	// Create a new deployment.
	now := time.Now().UTC()
	dep, err := sc.q.CreateDeploymentWithWaveConfig(ctx, sqlcgen.CreateDeploymentWithWaveConfigParams{
		TenantID:      schedule.TenantID,
		PolicyID:      schedule.PolicyID,
		Status:        string(StatusCreated),
		CreatedBy:     schedule.CreatedBy,
		WaveConfig:    schedule.WaveConfig,
		MaxConcurrent: schedule.MaxConcurrent,
		ScheduledAt:   pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		slog.ErrorContext(ctx, "schedule checker: create deployment",
			"schedule_id", scheduleIDStr, "policy_id", policyIDStr, "error", err)
		return
	}

	deployIDStr := uuid.UUID(dep.ID.Bytes).String()
	slog.InfoContext(ctx, "schedule checker: created deployment",
		"deployment_id", deployIDStr, "schedule_id", scheduleIDStr, "policy_id", policyIDStr)

	// Emit deployment.created event (best-effort).
	evt := domain.NewSystemEvent(events.DeploymentCreated, tenantIDStr, "deployment",
		deployIDStr, events.DeploymentCreated, dep)
	EmitBestEffort(ctx, sc.eventBus, []domain.DomainEvent{evt})

	// Compute next run time from cron expression.
	nextRun, err := computeNextRun(schedule.CronExpression, now)
	if err != nil {
		slog.ErrorContext(ctx, "schedule checker: parse cron expression",
			"schedule_id", scheduleIDStr, "cron", schedule.CronExpression, "error", err)
		return
	}

	// Update schedule with last_run_at and new next_run_at.
	if err := sc.q.UpdateScheduleAfterRun(ctx, sqlcgen.UpdateScheduleAfterRunParams{
		ID:        schedule.ID,
		NextRunAt: pgtype.Timestamptz{Time: nextRun, Valid: true},
		TenantID:  schedule.TenantID,
	}); err != nil {
		slog.ErrorContext(ctx, "schedule checker: update schedule after run",
			"schedule_id", scheduleIDStr, "error", err)
		return
	}
}

// computeNextRun parses a 5-field cron expression and returns the next run time
// after the given reference time.
func computeNextRun(cronExpr string, after time.Time) (time.Time, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cron expression %q: %w", cronExpr, err)
	}
	return sched.Next(after), nil
}

// ScheduleCheckerWorker wraps ScheduleChecker as a River worker.
type ScheduleCheckerWorker struct {
	river.WorkerDefaults[ScheduleCheckerJobArgs]
	checker *ScheduleChecker
}

// NewScheduleCheckerWorker creates a ScheduleCheckerWorker.
func NewScheduleCheckerWorker(checker *ScheduleChecker) *ScheduleCheckerWorker {
	if checker == nil {
		panic("deployment: NewScheduleCheckerWorker called with nil checker")
	}
	return &ScheduleCheckerWorker{checker: checker}
}

// Work implements river.Worker.
func (w *ScheduleCheckerWorker) Work(ctx context.Context, _ *river.Job[ScheduleCheckerJobArgs]) error {
	return w.checker.Check(ctx)
}
