package deployment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// --- Fake wave dispatcher querier ---

type fakeWaveDispatcherQuerier struct {
	// Return values
	runningDeployments     []sqlcgen.Deployment
	runningDeploymentsErr  error
	currentWave            sqlcgen.DeploymentWave
	currentWaveErr         error
	pendingTargets         []sqlcgen.DeploymentTarget
	pendingTargetsErr      error
	activeCount            int64
	activeCountErr         error
	maintenanceWindow      []byte
	maintenanceWindowErr   error
	setWaveRunningResult   sqlcgen.DeploymentWave
	setWaveRunningErr      error
	setWaveCompletedResult sqlcgen.DeploymentWave
	setWaveCompletedErr    error
	setWaveFailedResult    sqlcgen.DeploymentWave
	setWaveFailedErr       error
	setWaveEligibleAtErr   error
	deploymentWaves        []sqlcgen.DeploymentWave
	deploymentWavesErr     error

	// Capture calls
	createdCommands          []sqlcgen.CreateCommandParams
	updatedTargets           []sqlcgen.UpdateDeploymentTargetStatusParams
	setWaveRunningCalls      []sqlcgen.SetWaveRunningParams
	setWaveCompletedCalls    []sqlcgen.SetWaveCompletedParams
	setWaveFailedCalls       []sqlcgen.SetWaveFailedParams
	setWaveEligibleAtCalls   []sqlcgen.SetWaveEligibleAtParams
	listDeploymentWavesCalls []sqlcgen.ListDeploymentWavesParams

	// Rollback querier fields
	rollingBackErr       error
	rolledBackErr        error
	rollbackFailedErr    error
	cancelWavesErr       error
	cancelWaveTargetsErr error
	cancelCommandsErr    error
	cancelWavesCalled    bool
	cancelTargetsCalled  bool
	cancelCommandsCalled bool

	// Complete/fail deployment
	completeDeploymentErr error
	failDeploymentErr     error
}

func (f *fakeWaveDispatcherQuerier) ListTenantIDsWithRunningDeployments(_ context.Context) ([]pgtype.UUID, error) {
	seen := make(map[pgtype.UUID]bool)
	var ids []pgtype.UUID
	for _, d := range f.runningDeployments {
		if !seen[d.TenantID] {
			seen[d.TenantID] = true
			ids = append(ids, d.TenantID)
		}
	}
	return ids, f.runningDeploymentsErr
}

func (f *fakeWaveDispatcherQuerier) ListRunningDeployments(_ context.Context, tenantID pgtype.UUID) ([]sqlcgen.Deployment, error) {
	var result []sqlcgen.Deployment
	for _, d := range f.runningDeployments {
		if d.TenantID == tenantID {
			result = append(result, d)
		}
	}
	return result, f.runningDeploymentsErr
}

func (f *fakeWaveDispatcherQuerier) GetCurrentWave(_ context.Context, arg sqlcgen.GetCurrentWaveParams) (sqlcgen.DeploymentWave, error) {
	return f.currentWave, f.currentWaveErr
}

func (f *fakeWaveDispatcherQuerier) ListPendingWaveTargets(_ context.Context, arg sqlcgen.ListPendingWaveTargetsParams) ([]sqlcgen.DeploymentTarget, error) {
	return f.pendingTargets, f.pendingTargetsErr
}

func (f *fakeWaveDispatcherQuerier) CountActiveTargets(_ context.Context, arg sqlcgen.CountActiveTargetsParams) (int64, error) {
	return f.activeCount, f.activeCountErr
}

func (f *fakeWaveDispatcherQuerier) GetEndpointMaintenanceWindow(_ context.Context, arg sqlcgen.GetEndpointMaintenanceWindowParams) ([]byte, error) {
	return f.maintenanceWindow, f.maintenanceWindowErr
}

func (f *fakeWaveDispatcherQuerier) SetWaveRunning(_ context.Context, arg sqlcgen.SetWaveRunningParams) (sqlcgen.DeploymentWave, error) {
	f.setWaveRunningCalls = append(f.setWaveRunningCalls, arg)
	return f.setWaveRunningResult, f.setWaveRunningErr
}

func (f *fakeWaveDispatcherQuerier) SetWaveCompleted(_ context.Context, arg sqlcgen.SetWaveCompletedParams) (sqlcgen.DeploymentWave, error) {
	f.setWaveCompletedCalls = append(f.setWaveCompletedCalls, arg)
	return f.setWaveCompletedResult, f.setWaveCompletedErr
}

func (f *fakeWaveDispatcherQuerier) SetWaveFailed(_ context.Context, arg sqlcgen.SetWaveFailedParams) (sqlcgen.DeploymentWave, error) {
	f.setWaveFailedCalls = append(f.setWaveFailedCalls, arg)
	return f.setWaveFailedResult, f.setWaveFailedErr
}

func (f *fakeWaveDispatcherQuerier) SetWaveEligibleAt(_ context.Context, arg sqlcgen.SetWaveEligibleAtParams) error {
	f.setWaveEligibleAtCalls = append(f.setWaveEligibleAtCalls, arg)
	return f.setWaveEligibleAtErr
}

func (f *fakeWaveDispatcherQuerier) CreateCommand(_ context.Context, arg sqlcgen.CreateCommandParams) (sqlcgen.Command, error) {
	f.createdCommands = append(f.createdCommands, arg)
	return sqlcgen.Command{
		ID: validUUID("00000000-0000-0000-0000-000000000099"),
	}, nil
}

func (f *fakeWaveDispatcherQuerier) GetPatchByID(_ context.Context, arg sqlcgen.GetPatchByIDParams) (sqlcgen.Patch, error) {
	return sqlcgen.Patch{
		ID:       arg.ID,
		TenantID: arg.TenantID,
		Name:     "test-patch",
		Version:  "1.0.0",
		Severity: "high",
		OsFamily: "linux",
		Status:   "available",
	}, nil
}

func (f *fakeWaveDispatcherQuerier) UpdateDeploymentTargetStatus(_ context.Context, arg sqlcgen.UpdateDeploymentTargetStatusParams) (sqlcgen.DeploymentTarget, error) {
	f.updatedTargets = append(f.updatedTargets, arg)
	return sqlcgen.DeploymentTarget{}, nil
}

func (f *fakeWaveDispatcherQuerier) ListDeploymentWaves(_ context.Context, arg sqlcgen.ListDeploymentWavesParams) ([]sqlcgen.DeploymentWave, error) {
	f.listDeploymentWavesCalls = append(f.listDeploymentWavesCalls, arg)
	return f.deploymentWaves, f.deploymentWavesErr
}

// RollbackQuerier methods
func (f *fakeWaveDispatcherQuerier) SetDeploymentRollingBack(_ context.Context, _ sqlcgen.SetDeploymentRollingBackParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, f.rollingBackErr
}

func (f *fakeWaveDispatcherQuerier) SetDeploymentRolledBack(_ context.Context, _ sqlcgen.SetDeploymentRolledBackParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, f.rolledBackErr
}

func (f *fakeWaveDispatcherQuerier) SetDeploymentRollbackFailed(_ context.Context, _ sqlcgen.SetDeploymentRollbackFailedParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, f.rollbackFailedErr
}

func (f *fakeWaveDispatcherQuerier) CancelRemainingWaves(_ context.Context, _ sqlcgen.CancelRemainingWavesParams) error {
	f.cancelWavesCalled = true
	return f.cancelWavesErr
}

func (f *fakeWaveDispatcherQuerier) CancelWaveTargets(_ context.Context, _ sqlcgen.CancelWaveTargetsParams) error {
	f.cancelTargetsCalled = true
	return f.cancelWaveTargetsErr
}

func (f *fakeWaveDispatcherQuerier) CancelCommandsByDeployment(_ context.Context, _ sqlcgen.CancelCommandsByDeploymentParams) error {
	f.cancelCommandsCalled = true
	return f.cancelCommandsErr
}

// CompleteQuerier
func (f *fakeWaveDispatcherQuerier) SetDeploymentCompleted(_ context.Context, _ sqlcgen.SetDeploymentCompletedParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, f.completeDeploymentErr
}

// FailQuerier
func (f *fakeWaveDispatcherQuerier) SetDeploymentFailed(_ context.Context, _ sqlcgen.SetDeploymentFailedParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, f.failDeploymentErr
}

// --- Helper to build pgtype.Numeric from float ---

func numericFromFloat(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(f)
	return n
}

// --- Tests ---

func TestWaveDispatcherJobArgs_Kind(t *testing.T) {
	t.Parallel()
	args := deployment.WaveDispatcherJobArgs{}
	if got := args.Kind(); got != "wave_dispatcher" {
		t.Fatalf("expected Kind() = %q, got %q", "wave_dispatcher", got)
	}
}

func TestWaveDispatcher_NoRunningDeployments(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()
	q := &fakeWaveDispatcherQuerier{
		runningDeployments: nil,
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(bus.events))
	}
}

func TestWaveDispatcher_DispatchPendingTargets(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")
	target1ID := validUUID("00000000-0000-0000-0000-000000000010")
	target2ID := validUUID("00000000-0000-0000-0000-000000000011")
	endpoint1ID := validUUID("00000000-0000-0000-0000-000000000020")
	endpoint2ID := validUUID("00000000-0000-0000-0000-000000000021")

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:       testDeployID,
				TenantID: testTenantID,
				Status:   "running",
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      2,
			SuccessCount:     0,
			FailedCount:      0,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets: []sqlcgen.DeploymentTarget{
			{
				ID:           target1ID,
				TenantID:     testTenantID,
				DeploymentID: testDeployID,
				EndpointID:   endpoint1ID,
				WaveID:       waveID,
				Status:       "pending",
			},
			{
				ID:           target2ID,
				TenantID:     testTenantID,
				DeploymentID: testDeployID,
				EndpointID:   endpoint2ID,
				WaveID:       waveID,
				Status:       "pending",
			},
		},
		// No maintenance window restriction, no throttle
		activeCount:       0,
		maintenanceWindow: nil,
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both targets should be dispatched
	if len(q.createdCommands) != 2 {
		t.Fatalf("expected 2 commands created, got %d", len(q.createdCommands))
	}
	if len(q.updatedTargets) != 2 {
		t.Fatalf("expected 2 target updates, got %d", len(q.updatedTargets))
	}
	for _, ut := range q.updatedTargets {
		if ut.Status != "sent" {
			t.Fatalf("expected target status 'sent', got %q", ut.Status)
		}
	}

	// Verify commands have correct type
	for _, cmd := range q.createdCommands {
		if cmd.Type != "install_patch" {
			t.Fatalf("expected command type 'install_patch', got %q", cmd.Type)
		}
		if cmd.Status != "pending" {
			t.Fatalf("expected command status 'pending', got %q", cmd.Status)
		}
	}

	// Verify events: command.dispatched + deployment_target.sent for each target
	if len(bus.events) != 4 {
		t.Fatalf("expected 4 events (2 dispatched + 2 sent), got %d", len(bus.events))
	}
}

func TestWaveDispatcher_RespectsMaxConcurrent(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")
	target1ID := validUUID("00000000-0000-0000-0000-000000000010")
	endpoint1ID := validUUID("00000000-0000-0000-0000-000000000020")

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:            testDeployID,
				TenantID:      testTenantID,
				Status:        "running",
				MaxConcurrent: pgtype.Int4{Int32: 1, Valid: true},
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      2,
			SuccessCount:     0,
			FailedCount:      0,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets: []sqlcgen.DeploymentTarget{
			{
				ID:           target1ID,
				TenantID:     testTenantID,
				DeploymentID: testDeployID,
				EndpointID:   endpoint1ID,
				WaveID:       waveID,
				Status:       "pending",
			},
		},
		activeCount:       1, // Already 1 active, max_concurrent is 1
		maintenanceWindow: nil,
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No targets should be dispatched due to throttle
	if len(q.createdCommands) != 0 {
		t.Fatalf("expected 0 commands (throttled), got %d", len(q.createdCommands))
	}
	if len(q.updatedTargets) != 0 {
		t.Fatalf("expected 0 target updates (throttled), got %d", len(q.updatedTargets))
	}
}

func TestWaveDispatcher_SkipsMaintenanceWindow(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")
	target1ID := validUUID("00000000-0000-0000-0000-000000000010")
	endpoint1ID := validUUID("00000000-0000-0000-0000-000000000020")

	// Maintenance window that does not include current time
	// Use a window far in the past day-of-week to ensure mismatch
	mwData := []byte(`{"days":[6],"start":"03:00","end":"04:00","tz":"UTC"}`) // Saturday 3-4 AM UTC only

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:       testDeployID,
				TenantID: testTenantID,
				Status:   "running",
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      1,
			SuccessCount:     0,
			FailedCount:      0,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets: []sqlcgen.DeploymentTarget{
			{
				ID:           target1ID,
				TenantID:     testTenantID,
				DeploymentID: testDeployID,
				EndpointID:   endpoint1ID,
				WaveID:       waveID,
				Status:       "pending",
			},
		},
		activeCount:       0,
		maintenanceWindow: mwData,
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Target should not be dispatched because endpoint is outside maintenance window
	if len(q.createdCommands) != 0 {
		t.Fatalf("expected 0 commands (outside maintenance window), got %d", len(q.createdCommands))
	}
}

func TestWaveDispatcher_AdvancesWaveOnCompletion(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")
	nextWaveID := validUUID("00000000-0000-0000-0000-000000000031")

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:       testDeployID,
				TenantID: testTenantID,
				Status:   "running",
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      2,
			SuccessCount:     2, // All succeeded
			FailedCount:      0,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets: nil, // No pending targets — all completed
		activeCount:    0,
		// Return next wave when listing all waves
		deploymentWaves: []sqlcgen.DeploymentWave{
			{
				ID:               waveID,
				TenantID:         testTenantID,
				DeploymentID:     testDeployID,
				WaveNumber:       1,
				Status:           "running", // Will be completed
				TargetCount:      2,
				SuccessCount:     2,
				FailedCount:      0,
				SuccessThreshold: numericFromFloat(0.8),
				ErrorRateMax:     numericFromFloat(0.2),
			},
			{
				ID:                nextWaveID,
				TenantID:          testTenantID,
				DeploymentID:      testDeployID,
				WaveNumber:        2,
				Status:            "pending",
				TargetCount:       3,
				SuccessThreshold:  numericFromFloat(0.8),
				ErrorRateMax:      numericFromFloat(0.2),
				DelayAfterMinutes: 5,
			},
		},
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wave should be completed
	if len(q.setWaveCompletedCalls) != 1 {
		t.Fatalf("expected 1 SetWaveCompleted call, got %d", len(q.setWaveCompletedCalls))
	}
	if q.setWaveCompletedCalls[0].ID != waveID {
		t.Fatalf("expected wave %v to be completed, got %v", waveID, q.setWaveCompletedCalls[0].ID)
	}

	// Next wave should get eligible_at set
	if len(q.setWaveEligibleAtCalls) != 1 {
		t.Fatalf("expected 1 SetWaveEligibleAt call, got %d", len(q.setWaveEligibleAtCalls))
	}
	if q.setWaveEligibleAtCalls[0].ID != nextWaveID {
		t.Fatalf("expected next wave %v to get eligible_at, got %v", nextWaveID, q.setWaveEligibleAtCalls[0].ID)
	}

	// Verify wave completed event
	hasWaveCompleted := false
	for _, evt := range bus.events {
		if evt.Type == events.DeploymentWaveCompleted {
			hasWaveCompleted = true
		}
	}
	if !hasWaveCompleted {
		t.Fatal("expected DeploymentWaveCompleted event")
	}
}

func TestWaveDispatcher_TriggersRollbackOnFailure(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:       testDeployID,
				TenantID: testTenantID,
				Status:   "running",
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      10,
			SuccessCount:     5,
			FailedCount:      5, // 50% failure rate > 20% threshold
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets: nil, // All targets completed
		activeCount:    0,
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wave should be marked failed
	if len(q.setWaveFailedCalls) != 1 {
		t.Fatalf("expected 1 SetWaveFailed call, got %d", len(q.setWaveFailedCalls))
	}

	// Rollback should be triggered
	hasRollbackTriggered := false
	hasRolledBack := false
	hasWaveFailed := false
	for _, evt := range bus.events {
		if evt.Type == events.DeploymentRollbackTriggered {
			hasRollbackTriggered = true
		}
		if evt.Type == events.DeploymentRolledBack {
			hasRolledBack = true
		}
		if evt.Type == events.DeploymentWaveFailed {
			hasWaveFailed = true
		}
	}
	if !hasWaveFailed {
		t.Fatal("expected DeploymentWaveFailed event")
	}
	if !hasRollbackTriggered {
		t.Fatal("expected DeploymentRollbackTriggered event")
	}
	if !hasRolledBack {
		t.Fatal("expected DeploymentRolledBack event")
	}
}

func TestWaveDispatcher_ListRunningDeploymentsError(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	q := &fakeWaveDispatcherQuerier{
		runningDeploymentsErr: errors.New("db connection lost"),
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err == nil {
		t.Fatal("expected error when listing running deployments fails")
	}
}

func TestWaveDispatcher_PendingWaveBecomesRunning(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:       testDeployID,
				TenantID: testTenantID,
				Status:   "running",
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "pending",
			EligibleAt:       pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Minute), Valid: true}, // Past
			TargetCount:      1,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		setWaveRunningResult: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      1,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets:    nil,
		activeCount:       0,
		maintenanceWindow: nil,
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wave should have been transitioned to running
	if len(q.setWaveRunningCalls) != 1 {
		t.Fatalf("expected 1 SetWaveRunning call, got %d", len(q.setWaveRunningCalls))
	}

	// Verify wave started event
	hasWaveStarted := false
	for _, evt := range bus.events {
		if evt.Type == events.DeploymentWaveStarted {
			hasWaveStarted = true
		}
	}
	if !hasWaveStarted {
		t.Fatal("expected DeploymentWaveStarted event")
	}
}

func TestWaveDispatcher_CompletesDeploymentWhenLastWaveDone(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	waveID := validUUID("00000000-0000-0000-0000-000000000030")

	q := &fakeWaveDispatcherQuerier{
		runningDeployments: []sqlcgen.Deployment{
			{
				ID:       testDeployID,
				TenantID: testTenantID,
				Status:   "running",
			},
		},
		currentWave: sqlcgen.DeploymentWave{
			ID:               waveID,
			TenantID:         testTenantID,
			DeploymentID:     testDeployID,
			WaveNumber:       1,
			Status:           "running",
			TargetCount:      2,
			SuccessCount:     2,
			FailedCount:      0,
			SuccessThreshold: numericFromFloat(0.8),
			ErrorRateMax:     numericFromFloat(0.2),
		},
		pendingTargets: nil,
		activeCount:    0,
		// Only one wave, already completed — no next wave
		deploymentWaves: []sqlcgen.DeploymentWave{
			{
				ID:               waveID,
				TenantID:         testTenantID,
				DeploymentID:     testDeployID,
				WaveNumber:       1,
				Status:           "running",
				TargetCount:      2,
				SuccessCount:     2,
				SuccessThreshold: numericFromFloat(0.8),
				ErrorRateMax:     numericFromFloat(0.2),
			},
		},
	}

	wd := deployment.NewWaveDispatcher(q, sm, bus, 30*time.Minute)
	err := wd.Dispatch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wave should be completed
	if len(q.setWaveCompletedCalls) != 1 {
		t.Fatalf("expected 1 SetWaveCompleted call, got %d", len(q.setWaveCompletedCalls))
	}

	// Deployment should be completed
	hasDeploymentCompleted := false
	for _, evt := range bus.events {
		if evt.Type == events.DeploymentCompleted {
			hasDeploymentCompleted = true
		}
	}
	if !hasDeploymentCompleted {
		t.Fatal("expected DeploymentCompleted event when last wave finishes")
	}
}
