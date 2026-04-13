package deployment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// ExecutorJobArgs is the payload for a deployment executor River job.
type ExecutorJobArgs struct {
	DeploymentID string `json:"deployment_id"`
	TenantID     string `json:"tenant_id"`
}

func (ExecutorJobArgs) Kind() string { return "deployment_executor" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ExecutorJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// ExecutorQuerier defines the queries needed by the executor.
type ExecutorQuerier interface {
	StartQuerier
	FailQuerier
	GetCurrentWave(ctx context.Context, arg sqlcgen.GetCurrentWaveParams) (sqlcgen.DeploymentWave, error)
	SetWaveEligibleAt(ctx context.Context, arg sqlcgen.SetWaveEligibleAtParams) error
}

// Executor transitions a deployment from CREATED to RUNNING and activates wave 1.
// The WaveDispatcher periodic job handles actual target dispatch.
type Executor struct {
	q        ExecutorQuerier
	sm       *StateMachine
	eventBus domain.EventBus
}

func NewExecutor(q ExecutorQuerier, sm *StateMachine, eventBus domain.EventBus) *Executor {
	if q == nil {
		panic("deployment: NewExecutor called with nil querier")
	}
	if sm == nil {
		panic("deployment: NewExecutor called with nil stateMachine")
	}
	if eventBus == nil {
		panic("deployment: NewExecutor called with nil eventBus")
	}
	return &Executor{q: q, sm: sm, eventBus: eventBus}
}

// Execute transitions the deployment from CREATED to RUNNING and activates wave 1.
// The WaveDispatcher periodic job handles actual target dispatch.
func (e *Executor) Execute(ctx context.Context, deployID, tenantID pgtype.UUID) error {
	// Transition CREATED -> RUNNING.
	_, startEvents, startErr := e.sm.StartDeployment(ctx, e.q, deployID, tenantID)
	if startErr != nil {
		return fmt.Errorf("executor: start deployment: %w", startErr)
	}
	EmitBestEffort(ctx, e.eventBus, startEvents)

	// Activate wave 1 by setting eligible_at to now.
	wave, err := e.q.GetCurrentWave(ctx, sqlcgen.GetCurrentWaveParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "executor: get current wave",
			"deployment_id", uuid.UUID(deployID.Bytes).String(), "error", err)
		if _, failEvts, failErr := e.sm.FailDeployment(ctx, e.q, deployID, tenantID); failErr != nil {
			slog.ErrorContext(ctx, "executor: failed to mark deployment as failed",
				"deployment_id", uuid.UUID(deployID.Bytes).String(), "error", failErr)
		} else {
			EmitBestEffort(ctx, e.eventBus, failEvts)
		}
		return fmt.Errorf("executor: get current wave: %w", err)
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	if err := e.q.SetWaveEligibleAt(ctx, sqlcgen.SetWaveEligibleAtParams{
		ID:         wave.ID,
		EligibleAt: now,
		TenantID:   tenantID,
	}); err != nil {
		return fmt.Errorf("executor: set wave eligible_at: %w", err)
	}

	return nil
}

// ExecutorWorker wraps Executor as a River worker.
type ExecutorWorker struct {
	river.WorkerDefaults[ExecutorJobArgs]
	executor *Executor
}

func NewExecutorWorker(executor *Executor) *ExecutorWorker {
	return &ExecutorWorker{executor: executor}
}

// Work implements river.Worker.
func (w *ExecutorWorker) Work(ctx context.Context, job *river.Job[ExecutorJobArgs]) error {
	deployUUID, err := uuid.Parse(job.Args.DeploymentID)
	if err != nil {
		return fmt.Errorf("executor worker: invalid deployment_id: %w", err)
	}
	tenantUUID, err := uuid.Parse(job.Args.TenantID)
	if err != nil {
		return fmt.Errorf("executor worker: invalid tenant_id: %w", err)
	}
	return w.executor.Execute(ctx,
		pgtype.UUID{Bytes: deployUUID, Valid: true},
		pgtype.UUID{Bytes: tenantUUID, Valid: true},
	)
}
