package deployment_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// --- Fake executor querier ---

type fakeExecutorQuerier struct {
	startErr       error
	failErr        error
	getWaveResult  sqlcgen.DeploymentWave
	getWaveErr     error
	setEligibleErr error

	setEligibleCalled bool
	failCalled        bool
}

func (f *fakeExecutorQuerier) SetDeploymentStarted(_ context.Context, _ sqlcgen.SetDeploymentStartedParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, f.startErr
}

func (f *fakeExecutorQuerier) SetDeploymentFailed(_ context.Context, _ sqlcgen.SetDeploymentFailedParams) (sqlcgen.Deployment, error) {
	f.failCalled = true
	return sqlcgen.Deployment{}, f.failErr
}

func (f *fakeExecutorQuerier) GetCurrentWave(_ context.Context, _ sqlcgen.GetCurrentWaveParams) (sqlcgen.DeploymentWave, error) {
	return f.getWaveResult, f.getWaveErr
}

func (f *fakeExecutorQuerier) SetWaveEligibleAt(_ context.Context, _ sqlcgen.SetWaveEligibleAtParams) error {
	f.setEligibleCalled = true
	return f.setEligibleErr
}

// --- Tests ---

func TestExecutorJobArgs_Kind(t *testing.T) {
	t.Parallel()
	args := deployment.ExecutorJobArgs{}
	if got := args.Kind(); got != "deployment_executor" {
		t.Fatalf("expected Kind() = %q, got %q", "deployment_executor", got)
	}
}

func TestExecutor_Success(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")

	q := &fakeExecutorQuerier{
		getWaveResult: sqlcgen.DeploymentWave{
			ID:     waveID,
			Status: "pending",
		},
	}

	executor := deployment.NewExecutor(q, sm, bus)
	err := executor.Execute(context.Background(), testDeployID, testTenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify wave eligible_at was set.
	if !q.setEligibleCalled {
		t.Fatal("expected SetWaveEligibleAt to be called")
	}

	// Verify deployment.started event emitted.
	if len(bus.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.events))
	}
	if bus.events[0].Type != events.DeploymentStarted {
		t.Fatalf("expected event %s, got %s", events.DeploymentStarted, bus.events[0].Type)
	}
}

func TestExecutor_StartFailure(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	q := &fakeExecutorQuerier{
		startErr: errors.New("optimistic lock failed"),
	}

	executor := deployment.NewExecutor(q, sm, bus)
	err := executor.Execute(context.Background(), testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when start deployment fails")
	}

	// No wave operations should happen.
	if q.setEligibleCalled {
		t.Fatal("expected SetWaveEligibleAt not to be called")
	}
}

func TestExecutor_GetWaveFailure_FailsDeployment(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	q := &fakeExecutorQuerier{
		getWaveErr: errors.New("no waves found"),
	}

	executor := deployment.NewExecutor(q, sm, bus)
	err := executor.Execute(context.Background(), testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when get wave fails")
	}

	// Deployment should be marked as failed.
	if !q.failCalled {
		t.Fatal("expected deployment to be marked as failed")
	}

	// Events: deployment.started + deployment.failed
	if len(bus.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(bus.events))
	}
	if bus.events[0].Type != events.DeploymentStarted {
		t.Fatalf("expected first event %s, got %s", events.DeploymentStarted, bus.events[0].Type)
	}
	if bus.events[1].Type != events.DeploymentFailed {
		t.Fatalf("expected second event %s, got %s", events.DeploymentFailed, bus.events[1].Type)
	}
}

func TestExecutor_SetEligibleAtFailure(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}

	q := &fakeExecutorQuerier{
		getWaveResult:  sqlcgen.DeploymentWave{ID: waveID, Status: "pending"},
		setEligibleErr: errors.New("db error"),
	}

	executor := deployment.NewExecutor(q, sm, bus)
	err := executor.Execute(context.Background(), testDeployID, testTenantID)
	if err == nil {
		t.Fatal("expected error when set eligible_at fails")
	}
}
