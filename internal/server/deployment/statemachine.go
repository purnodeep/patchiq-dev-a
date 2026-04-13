package deployment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// StartQuerier transitions a deployment from created to running.
type StartQuerier interface {
	SetDeploymentStarted(ctx context.Context, arg sqlcgen.SetDeploymentStartedParams) (sqlcgen.Deployment, error)
}

// CompleteQuerier transitions a deployment from running to completed.
type CompleteQuerier interface {
	SetDeploymentCompleted(ctx context.Context, arg sqlcgen.SetDeploymentCompletedParams) (sqlcgen.Deployment, error)
}

// FailQuerier transitions a deployment from running to failed.
type FailQuerier interface {
	SetDeploymentFailed(ctx context.Context, arg sqlcgen.SetDeploymentFailedParams) (sqlcgen.Deployment, error)
}

// CancelQuerier transitions a deployment from created or running to cancelled and cleans up targets/commands.
type CancelQuerier interface {
	SetDeploymentCancelled(ctx context.Context, arg sqlcgen.SetDeploymentCancelledParams) (sqlcgen.Deployment, error)
	CancelDeploymentTargets(ctx context.Context, arg sqlcgen.CancelDeploymentTargetsParams) error
	CancelCommandsByDeployment(ctx context.Context, arg sqlcgen.CancelCommandsByDeploymentParams) error
}

// RollbackQuerier defines DB methods needed for deployment rollback.
type RollbackQuerier interface {
	SetDeploymentRollingBack(ctx context.Context, arg sqlcgen.SetDeploymentRollingBackParams) (sqlcgen.Deployment, error)
	SetDeploymentRolledBack(ctx context.Context, arg sqlcgen.SetDeploymentRolledBackParams) (sqlcgen.Deployment, error)
	SetDeploymentRollbackFailed(ctx context.Context, arg sqlcgen.SetDeploymentRollbackFailedParams) (sqlcgen.Deployment, error)
	CancelRemainingWaves(ctx context.Context, arg sqlcgen.CancelRemainingWavesParams) error
	CancelWaveTargets(ctx context.Context, arg sqlcgen.CancelWaveTargetsParams) error
	CancelCommandsByDeployment(ctx context.Context, arg sqlcgen.CancelCommandsByDeploymentParams) error
}

// RetryQuerier resets a failed deployment to running and resets failed targets.
type RetryQuerier interface {
	SetDeploymentRetrying(ctx context.Context, arg sqlcgen.SetDeploymentRetryingParams) (sqlcgen.Deployment, error)
	RetryFailedTargets(ctx context.Context, arg sqlcgen.RetryFailedTargetsParams) (int64, error)
}

// ScheduleQuerier transitions a deployment from scheduled to created.
type ScheduleQuerier interface {
	SetDeploymentScheduledToCreated(ctx context.Context, arg sqlcgen.SetDeploymentScheduledToCreatedParams) (sqlcgen.Deployment, error)
}

// Valid state transitions (enforced by database CHECK constraints):
//   created   → running, cancelled
//   running   → completed, failed, cancelled, rolling_back
//   rolling_back → rolled_back, rollback_failed
//   scheduled → created

// StateMachine manages deployment state transitions and returns domain events
// for callers to emit after a successful commit (avoiding phantom events on rollback).
type StateMachine struct{}

func NewStateMachine() *StateMachine {
	return &StateMachine{}
}

// StartDeployment transitions CREATED → RUNNING.
func (sm *StateMachine) StartDeployment(ctx context.Context, q StartQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	d, err := q.SetDeploymentStarted(ctx, sqlcgen.SetDeploymentStartedParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("start deployment: %w", err)
	}
	return d, []domain.DomainEvent{sm.newEvent(events.DeploymentStarted, deployID, tenantID)}, nil
}

// CompleteDeployment transitions RUNNING → COMPLETED.
func (sm *StateMachine) CompleteDeployment(ctx context.Context, q CompleteQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	d, err := q.SetDeploymentCompleted(ctx, sqlcgen.SetDeploymentCompletedParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("complete deployment: %w", err)
	}
	return d, []domain.DomainEvent{sm.newEvent(events.DeploymentCompleted, deployID, tenantID)}, nil
}

// FailDeployment transitions RUNNING → FAILED.
func (sm *StateMachine) FailDeployment(ctx context.Context, q FailQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	d, err := q.SetDeploymentFailed(ctx, sqlcgen.SetDeploymentFailedParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("fail deployment: %w", err)
	}
	return d, []domain.DomainEvent{sm.newEvent(events.DeploymentFailed, deployID, tenantID)}, nil
}

// CancelDeployment transitions CREATED|RUNNING → CANCELLED and cancels pending targets/commands.
func (sm *StateMachine) CancelDeployment(ctx context.Context, q CancelQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	d, err := q.SetDeploymentCancelled(ctx, sqlcgen.SetDeploymentCancelledParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("cancel deployment: %w", err)
	}

	if err := q.CancelDeploymentTargets(ctx, sqlcgen.CancelDeploymentTargetsParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	}); err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("cancel deployment: cancel targets: %w", err)
	}

	if err := q.CancelCommandsByDeployment(ctx, sqlcgen.CancelCommandsByDeploymentParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	}); err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("cancel deployment: cancel commands: %w", err)
	}

	return d, []domain.DomainEvent{sm.newEvent(events.DeploymentCancelled, deployID, tenantID)}, nil
}

// RollbackDeployment transitions RUNNING → ROLLING_BACK → ROLLED_BACK (or ROLLBACK_FAILED).
// It cancels remaining waves, wave targets, and commands. If cancelling commands fails,
// the deployment is marked as rollback_failed instead of rolled_back.
func (sm *StateMachine) RollbackDeployment(ctx context.Context, q RollbackQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	_, err := q.SetDeploymentRollingBack(ctx, sqlcgen.SetDeploymentRollingBackParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("rollback deployment: set rolling back: %w", err)
	}

	evts := []domain.DomainEvent{sm.newEvent(events.DeploymentRollbackTriggered, deployID, tenantID)}

	if err := q.CancelRemainingWaves(ctx, sqlcgen.CancelRemainingWavesParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	}); err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("rollback deployment: cancel remaining waves: %w", err)
	}

	if err := q.CancelWaveTargets(ctx, sqlcgen.CancelWaveTargetsParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	}); err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("rollback deployment: cancel wave targets: %w", err)
	}

	if err := q.CancelCommandsByDeployment(ctx, sqlcgen.CancelCommandsByDeploymentParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	}); err != nil {
		d, rbErr := q.SetDeploymentRollbackFailed(ctx, sqlcgen.SetDeploymentRollbackFailedParams{
			ID:       deployID,
			TenantID: tenantID,
		})
		if rbErr != nil {
			return sqlcgen.Deployment{}, nil, fmt.Errorf("rollback deployment: set rollback failed: %w", rbErr)
		}
		evts = append(evts, sm.newEvent(events.DeploymentRollbackFailed, deployID, tenantID))
		return d, evts, nil
	}

	d, err := q.SetDeploymentRolledBack(ctx, sqlcgen.SetDeploymentRolledBackParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("rollback deployment: set rolled back: %w", err)
	}

	evts = append(evts, sm.newEvent(events.DeploymentRolledBack, deployID, tenantID))
	return d, evts, nil
}

// ActivateScheduled transitions SCHEDULED → CREATED.
func (sm *StateMachine) ActivateScheduled(ctx context.Context, q ScheduleQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	d, err := q.SetDeploymentScheduledToCreated(ctx, sqlcgen.SetDeploymentScheduledToCreatedParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("activate scheduled deployment: %w", err)
	}
	return d, []domain.DomainEvent{sm.newEvent(events.DeploymentCreated, deployID, tenantID)}, nil
}

// RetryDeployment transitions FAILED → RUNNING and resets failed targets to pending.
func (sm *StateMachine) RetryDeployment(ctx context.Context, q RetryQuerier, deployID, tenantID pgtype.UUID) (sqlcgen.Deployment, []domain.DomainEvent, error) {
	dep, err := q.SetDeploymentRetrying(ctx, sqlcgen.SetDeploymentRetryingParams{
		ID:       deployID,
		TenantID: tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("set deployment retrying: %w", err)
	}

	affected, err := q.RetryFailedTargets(ctx, sqlcgen.RetryFailedTargetsParams{
		DeploymentID: deployID,
		TenantID:     tenantID,
	})
	if err != nil {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("retry failed targets: %w", err)
	}
	if affected == 0 {
		return sqlcgen.Deployment{}, nil, fmt.Errorf("no failed targets found to retry")
	}

	return dep, []domain.DomainEvent{sm.newEvent(events.DeploymentRetryTriggered, deployID, tenantID)}, nil
}

func (sm *StateMachine) newEvent(eventType string, deployID, tenantID pgtype.UUID) domain.DomainEvent {
	deployIDStr := uuid.UUID(deployID.Bytes).String()
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()
	return domain.NewSystemEvent(eventType, tenantIDStr, "deployment", deployIDStr, eventType, nil)
}
