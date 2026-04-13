package deployment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// --- Test helpers ---

type fakeEventBus struct {
	events  []domain.DomainEvent
	emitErr error
}

func (f *fakeEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	f.events = append(f.events, event)
	return f.emitErr
}

func (f *fakeEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (f *fakeEventBus) Close() error                                    { return nil }

func validUUID(hex string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(hex)
	return u
}

// --- Fake queriers ---

type fakeStartQuerier struct {
	result sqlcgen.Deployment
	err    error
}

func (f *fakeStartQuerier) SetDeploymentStarted(_ context.Context, _ sqlcgen.SetDeploymentStartedParams) (sqlcgen.Deployment, error) {
	return f.result, f.err
}

type fakeCompleteQuerier struct {
	result sqlcgen.Deployment
	err    error
}

func (f *fakeCompleteQuerier) SetDeploymentCompleted(_ context.Context, _ sqlcgen.SetDeploymentCompletedParams) (sqlcgen.Deployment, error) {
	return f.result, f.err
}

type fakeFailQuerier struct {
	result sqlcgen.Deployment
	err    error
}

func (f *fakeFailQuerier) SetDeploymentFailed(_ context.Context, _ sqlcgen.SetDeploymentFailedParams) (sqlcgen.Deployment, error) {
	return f.result, f.err
}

type fakeCancelQuerier struct {
	result         sqlcgen.Deployment
	err            error
	targetsCalled  bool
	commandsCalled bool
	targetsErr     error
	commandsErr    error
}

func (f *fakeCancelQuerier) SetDeploymentCancelled(_ context.Context, _ sqlcgen.SetDeploymentCancelledParams) (sqlcgen.Deployment, error) {
	return f.result, f.err
}

func (f *fakeCancelQuerier) CancelDeploymentTargets(_ context.Context, _ sqlcgen.CancelDeploymentTargetsParams) error {
	f.targetsCalled = true
	return f.targetsErr
}

func (f *fakeCancelQuerier) CancelCommandsByDeployment(_ context.Context, _ sqlcgen.CancelCommandsByDeploymentParams) error {
	f.commandsCalled = true
	return f.commandsErr
}

// --- Tests ---

var (
	testDeployID = validUUID("00000000-0000-0000-0000-000000000001")
	testTenantID = validUUID("00000000-0000-0000-0000-000000000002")
)

func TestStartDeployment(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		queryErr  error
		wantErr   bool
		wantEvent string
	}{
		{
			name:      "success returns event",
			wantEvent: events.DeploymentStarted,
		},
		{
			name:     "optimistic lock failure returns error",
			queryErr: errors.New("no rows"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := deployment.NewStateMachine()
			q := &fakeStartQuerier{err: tt.queryErr}

			_, evts, err := sm.StartDeployment(context.Background(), q, testDeployID, testTenantID)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(evts) != 1 || evts[0].Type != tt.wantEvent {
				t.Fatalf("expected event %s, got %v", tt.wantEvent, evts)
			}
		})
	}
}

func TestCompleteDeployment(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeCompleteQuerier{}

	_, evts, err := sm.CompleteDeployment(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 1 || evts[0].Type != events.DeploymentCompleted {
		t.Fatalf("expected event %s, got %v", events.DeploymentCompleted, evts)
	}
}

func TestFailDeployment(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeFailQuerier{}

	_, evts, err := sm.FailDeployment(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 1 || evts[0].Type != events.DeploymentFailed {
		t.Fatalf("expected event %s, got %v", events.DeploymentFailed, evts)
	}
}

func TestCancelDeployment(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeCancelQuerier{}

	_, evts, err := sm.CancelDeployment(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !q.targetsCalled {
		t.Fatal("expected CancelDeploymentTargets to be called")
	}
	if !q.commandsCalled {
		t.Fatal("expected CancelCommandsByDeployment to be called")
	}
	if len(evts) != 1 || evts[0].Type != events.DeploymentCancelled {
		t.Fatalf("expected event %s, got %v", events.DeploymentCancelled, evts)
	}
}

func TestCompleteDeployment_QueryError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeCompleteQuerier{err: errors.New("no rows: deployment not in running state")}

	_, evts, err := sm.CompleteDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(evts) != 0 {
		t.Fatal("expected no events when query fails")
	}
}

func TestFailDeployment_QueryError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeFailQuerier{err: errors.New("no rows: deployment not in running state")}

	_, evts, err := sm.FailDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(evts) != 0 {
		t.Fatal("expected no events when query fails")
	}
}

func TestCancelDeployment_SetCancelledError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeCancelQuerier{err: errors.New("no rows")}

	_, _, err := sm.CancelDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCancelDeployment_CancelTargetsError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeCancelQuerier{targetsErr: errors.New("targets update failed")}

	_, _, err := sm.CancelDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when cancelling targets fails")
	}
}

func TestCancelDeployment_CancelCommandsError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeCancelQuerier{commandsErr: errors.New("commands update failed")}

	_, _, err := sm.CancelDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when cancelling commands fails")
	}
}

// --- Rollback fake querier ---

type fakeRollbackQuerier struct {
	rollingBackResult    sqlcgen.Deployment
	rollingBackErr       error
	rolledBackResult     sqlcgen.Deployment
	rolledBackErr        error
	rollbackFailedResult sqlcgen.Deployment
	rollbackFailedErr    error
	cancelWavesErr       error
	cancelTargetsErr     error
	cancelCommandsErr    error

	cancelWavesCalled    bool
	cancelTargetsCalled  bool
	cancelCommandsCalled bool
}

func (f *fakeRollbackQuerier) SetDeploymentRollingBack(_ context.Context, _ sqlcgen.SetDeploymentRollingBackParams) (sqlcgen.Deployment, error) {
	return f.rollingBackResult, f.rollingBackErr
}

func (f *fakeRollbackQuerier) SetDeploymentRolledBack(_ context.Context, _ sqlcgen.SetDeploymentRolledBackParams) (sqlcgen.Deployment, error) {
	return f.rolledBackResult, f.rolledBackErr
}

func (f *fakeRollbackQuerier) SetDeploymentRollbackFailed(_ context.Context, _ sqlcgen.SetDeploymentRollbackFailedParams) (sqlcgen.Deployment, error) {
	return f.rollbackFailedResult, f.rollbackFailedErr
}

func (f *fakeRollbackQuerier) CancelRemainingWaves(_ context.Context, _ sqlcgen.CancelRemainingWavesParams) error {
	f.cancelWavesCalled = true
	return f.cancelWavesErr
}

func (f *fakeRollbackQuerier) CancelWaveTargets(_ context.Context, _ sqlcgen.CancelWaveTargetsParams) error {
	f.cancelTargetsCalled = true
	return f.cancelTargetsErr
}

func (f *fakeRollbackQuerier) CancelCommandsByDeployment(_ context.Context, _ sqlcgen.CancelCommandsByDeploymentParams) error {
	f.cancelCommandsCalled = true
	return f.cancelCommandsErr
}

// --- Schedule fake querier ---

type fakeScheduleQuerier struct {
	result sqlcgen.Deployment
	err    error
}

func (f *fakeScheduleQuerier) SetDeploymentScheduledToCreated(_ context.Context, _ sqlcgen.SetDeploymentScheduledToCreatedParams) (sqlcgen.Deployment, error) {
	return f.result, f.err
}

// --- Rollback tests ---

func TestStateMachine_RollbackDeployment_Success(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeRollbackQuerier{}

	_, evts, err := sm.RollbackDeployment(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !q.cancelWavesCalled {
		t.Fatal("expected CancelRemainingWaves to be called")
	}
	if !q.cancelTargetsCalled {
		t.Fatal("expected CancelWaveTargets to be called")
	}
	if !q.cancelCommandsCalled {
		t.Fatal("expected CancelCommandsByDeployment to be called")
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	if evts[0].Type != events.DeploymentRollbackTriggered {
		t.Fatalf("expected first event %s, got %s", events.DeploymentRollbackTriggered, evts[0].Type)
	}
	if evts[1].Type != events.DeploymentRolledBack {
		t.Fatalf("expected second event %s, got %s", events.DeploymentRolledBack, evts[1].Type)
	}
}

func TestStateMachine_RollbackDeployment_CommandCancelFails(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeRollbackQuerier{cancelCommandsErr: errors.New("cancel commands failed")}

	_, evts, err := sm.RollbackDeployment(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: rollback should not return error on command cancel failure, got %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	if evts[0].Type != events.DeploymentRollbackTriggered {
		t.Fatalf("expected first event %s, got %s", events.DeploymentRollbackTriggered, evts[0].Type)
	}
	if evts[1].Type != events.DeploymentRollbackFailed {
		t.Fatalf("expected second event %s, got %s", events.DeploymentRollbackFailed, evts[1].Type)
	}
}

func TestStateMachine_RollbackDeployment_RollingBackError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeRollbackQuerier{rollingBackErr: errors.New("no rows")}

	_, _, err := sm.RollbackDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when setting rolling_back fails")
	}
}

// --- Schedule tests ---

func TestStateMachine_ActivateScheduled(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeScheduleQuerier{}

	_, evts, err := sm.ActivateScheduled(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 event, got %d", len(evts))
	}
	if evts[0].Type != events.DeploymentCreated {
		t.Fatalf("expected event %s, got %s", events.DeploymentCreated, evts[0].Type)
	}
}

// --- Retry fake querier ---

type fakeRetryQuerier struct {
	retryingResult sqlcgen.Deployment
	retryingErr    error
	targetsResult  int64
	targetsErr     error
	targetsCalled  bool
}

func (f *fakeRetryQuerier) SetDeploymentRetrying(_ context.Context, _ sqlcgen.SetDeploymentRetryingParams) (sqlcgen.Deployment, error) {
	return f.retryingResult, f.retryingErr
}

func (f *fakeRetryQuerier) RetryFailedTargets(_ context.Context, _ sqlcgen.RetryFailedTargetsParams) (int64, error) {
	f.targetsCalled = true
	return f.targetsResult, f.targetsErr
}

// --- Retry tests ---

func TestRetryDeployment(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeRetryQuerier{
		retryingResult: sqlcgen.Deployment{Status: "running"},
		targetsResult:  3,
	}

	dep, evts, err := sm.RetryDeployment(context.Background(), q, testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dep.Status != "running" {
		t.Fatalf("expected status running, got %s", dep.Status)
	}
	if !q.targetsCalled {
		t.Fatal("expected RetryFailedTargets to be called")
	}
	if len(evts) != 1 || evts[0].Type != events.DeploymentRetryTriggered {
		t.Fatalf("expected event %s, got %v", events.DeploymentRetryTriggered, evts)
	}
}

func TestRetryDeployment_SetRetryingError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeRetryQuerier{retryingErr: errors.New("not in failed state")}

	_, _, err := sm.RetryDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRetryDeployment_RetryTargetsError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeRetryQuerier{targetsErr: errors.New("targets update failed")}

	_, _, err := sm.RetryDeployment(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when retrying targets fails")
	}
}

func TestStateMachine_ActivateScheduled_QueryError(t *testing.T) {
	t.Parallel()
	sm := deployment.NewStateMachine()
	q := &fakeScheduleQuerier{err: errors.New("no rows")}

	_, evts, err := sm.ActivateScheduled(context.Background(), q, testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(evts) != 0 {
		t.Fatal("expected no events when query fails")
	}
}
