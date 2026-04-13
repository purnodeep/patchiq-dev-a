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

// --- Fake result querier ---

type fakeResultQuerier struct {
	command         sqlcgen.Command
	getCommandErr   error
	updateCmdErr    error
	updateTargetErr error
	incrementResult sqlcgen.Deployment
	incrementErr    error
	completeErr     error
	failErr         error

	updatedCmdStatus    string
	updatedTargetStatus string
	incrementCalled     bool
	completeCalled      bool
	failCalled          bool
}

func (f *fakeResultQuerier) GetCommandByID(_ context.Context, _ sqlcgen.GetCommandByIDParams) (sqlcgen.Command, error) {
	return f.command, f.getCommandErr
}

func (f *fakeResultQuerier) UpdateCommandStatus(_ context.Context, arg sqlcgen.UpdateCommandStatusParams) (sqlcgen.Command, error) {
	f.updatedCmdStatus = arg.Status
	return sqlcgen.Command{}, f.updateCmdErr
}

func (f *fakeResultQuerier) UpdateDeploymentTargetStatus(_ context.Context, arg sqlcgen.UpdateDeploymentTargetStatusParams) (sqlcgen.DeploymentTarget, error) {
	f.updatedTargetStatus = arg.Status
	return sqlcgen.DeploymentTarget{}, f.updateTargetErr
}

func (f *fakeResultQuerier) IncrementDeploymentCounters(_ context.Context, _ sqlcgen.IncrementDeploymentCountersParams) (sqlcgen.Deployment, error) {
	f.incrementCalled = true
	return f.incrementResult, f.incrementErr
}

func (f *fakeResultQuerier) SetDeploymentCompleted(_ context.Context, _ sqlcgen.SetDeploymentCompletedParams) (sqlcgen.Deployment, error) {
	f.completeCalled = true
	return sqlcgen.Deployment{}, f.completeErr
}

func (f *fakeResultQuerier) SetDeploymentFailed(_ context.Context, _ sqlcgen.SetDeploymentFailedParams) (sqlcgen.Deployment, error) {
	f.failCalled = true
	return sqlcgen.Deployment{}, f.failErr
}

func (f *fakeResultQuerier) GetDeploymentTargetWaveID(_ context.Context, _ sqlcgen.GetDeploymentTargetWaveIDParams) (pgtype.UUID, error) {
	return pgtype.UUID{}, nil
}

func (f *fakeResultQuerier) IncrementWaveCounters(_ context.Context, _ sqlcgen.IncrementWaveCountersParams) (sqlcgen.DeploymentWave, error) {
	return sqlcgen.DeploymentWave{}, nil
}

// --- Fake TxFactory for testing ---

func fakeResultTxFactoryWithCommitTracker(writeQ deployment.ResultQuerier, committed *bool) deployment.ResultTxFactory {
	return func(_ context.Context, _ string) (deployment.ResultQuerier, func() error, func() error, error) {
		noop := func() error { return nil }
		commit := func() error {
			*committed = true
			return nil
		}
		return writeQ, commit, noop, nil
	}
}

func fakeResultTxFactoryError(err error) deployment.ResultTxFactory {
	return func(_ context.Context, _ string) (deployment.ResultQuerier, func() error, func() error, error) {
		return nil, nil, nil, err
	}
}

// --- Tests ---

func TestHandleResult_Success(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	deployID := validUUID("00000000-0000-0000-0000-000000000001")
	targetID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeResultQuerier{
		command: sqlcgen.Command{
			DeploymentID: deployID,
			TargetID:     targetID,
		},
		incrementResult: sqlcgen.Deployment{
			Status:         "running",
			TotalTargets:   5,
			CompletedCount: 2,
			SuccessCount:   2,
			FailedCount:    0,
		},
	}

	rh := deployment.NewResultHandler(q, sm, bus)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "ok", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.updatedCmdStatus != "succeeded" {
		t.Fatalf("expected command status %q, got %q", "succeeded", q.updatedCmdStatus)
	}
	if q.updatedTargetStatus != "succeeded" {
		t.Fatalf("expected target status %q, got %q", "succeeded", q.updatedTargetStatus)
	}
	if !q.incrementCalled {
		t.Fatal("expected IncrementDeploymentCounters to be called")
	}
	if q.completeCalled {
		t.Fatal("expected deployment NOT to be completed (only 2/5)")
	}
}

func TestHandleResult_Failure(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	deployID := validUUID("00000000-0000-0000-0000-000000000001")
	targetID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeResultQuerier{
		command: sqlcgen.Command{
			DeploymentID: deployID,
			TargetID:     targetID,
		},
		incrementResult: sqlcgen.Deployment{
			Status:         "running",
			TotalTargets:   10,
			CompletedCount: 1,
			FailedCount:    1,
		},
	}

	rh := deployment.NewResultHandler(q, sm, bus)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, false, "", "", "install failed", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.updatedCmdStatus != "failed" {
		t.Fatalf("expected command status %q, got %q", "failed", q.updatedCmdStatus)
	}
	if q.updatedTargetStatus != "failed" {
		t.Fatalf("expected target status %q, got %q", "failed", q.updatedTargetStatus)
	}
}

func TestHandleResult_CompletesDeployment(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	deployID := validUUID("00000000-0000-0000-0000-000000000001")
	targetID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeResultQuerier{
		command: sqlcgen.Command{
			DeploymentID: deployID,
			TargetID:     targetID,
		},
		incrementResult: sqlcgen.Deployment{
			Status:         "running",
			TotalTargets:   3,
			CompletedCount: 3,
			SuccessCount:   3,
			FailedCount:    0,
		},
	}

	rh := deployment.NewResultHandler(q, sm, bus)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "done", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !q.completeCalled {
		t.Fatal("expected deployment to be completed")
	}
}

func TestHandleResult_FailsDeployment(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	deployID := validUUID("00000000-0000-0000-0000-000000000001")
	targetID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeResultQuerier{
		command: sqlcgen.Command{
			DeploymentID: deployID,
			TargetID:     targetID,
		},
		incrementResult: sqlcgen.Deployment{
			Status:         "running",
			TotalTargets:   10,
			CompletedCount: 5,
			SuccessCount:   2,
			FailedCount:    3, // 3/10 = 0.3 > default threshold 0.2
		},
	}

	rh := deployment.NewResultHandler(q, sm, bus)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, false, "", "", "timeout", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !q.failCalled {
		t.Fatal("expected deployment to be failed (threshold exceeded)")
	}
	if q.completeCalled {
		t.Fatal("expected deployment NOT to be completed")
	}
}

func TestHandleResult_NoDeployment(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeResultQuerier{
		command: sqlcgen.Command{
			// DeploymentID and TargetID are zero-value (Valid=false)
		},
	}

	rh := deployment.NewResultHandler(q, sm, bus)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "scan output", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.updatedCmdStatus != "succeeded" {
		t.Fatalf("expected command status %q, got %q", "succeeded", q.updatedCmdStatus)
	}
	if q.updatedTargetStatus != "" {
		t.Fatal("expected no target update for scan command")
	}
	if q.incrementCalled {
		t.Fatal("expected no counter increment for scan command")
	}
}

func TestHandleResult_GetCommandError(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	q := &fakeResultQuerier{
		getCommandErr: errors.New("not found"),
	}

	rh := deployment.NewResultHandler(q, sm, bus)
	err := rh.HandleResult(context.Background(), testDeployID, testTenantID, true, "", "", "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHandleResult_WithTxFactory(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	deployID := validUUID("00000000-0000-0000-0000-000000000001")
	targetID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	// readQ only handles GetCommandByID
	readQ := &fakeResultQuerier{
		command: sqlcgen.Command{
			DeploymentID: deployID,
			TargetID:     targetID,
		},
	}

	// writeQ handles all write operations via the tx factory
	writeQ := &fakeResultQuerier{
		incrementResult: sqlcgen.Deployment{
			Status:         "running",
			TotalTargets:   5,
			CompletedCount: 2,
			SuccessCount:   2,
			FailedCount:    0,
		},
	}

	var committed bool
	rh := deployment.NewResultHandler(readQ, sm, bus,
		deployment.WithResultTxFactory(fakeResultTxFactoryWithCommitTracker(writeQ, &committed)),
	)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "ok", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Writes should go through writeQ, not readQ
	if readQ.updatedCmdStatus != "" {
		t.Fatal("expected reads querier NOT to receive writes")
	}
	if writeQ.updatedCmdStatus != "succeeded" {
		t.Fatalf("expected write querier command status %q, got %q", "succeeded", writeQ.updatedCmdStatus)
	}
	if writeQ.updatedTargetStatus != "succeeded" {
		t.Fatalf("expected write querier target status %q, got %q", "succeeded", writeQ.updatedTargetStatus)
	}
	if !writeQ.incrementCalled {
		t.Fatal("expected IncrementDeploymentCounters on write querier")
	}
	if !committed {
		t.Fatal("expected transaction to be committed")
	}
}

func TestHandleResult_TxFactoryError(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	readQ := &fakeResultQuerier{
		command: sqlcgen.Command{},
	}

	rh := deployment.NewResultHandler(readQ, sm, bus,
		deployment.WithResultTxFactory(fakeResultTxFactoryError(errors.New("pool exhausted"))),
	)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "", "", "", nil)
	if err == nil {
		t.Fatal("expected error when tx factory fails")
	}
}

// --- Event-after-commit tests ---

// fakeResultTxFactoryWithCommitErr returns a tx factory where commit returns the given error.
func fakeResultTxFactoryWithCommitErr(writeQ deployment.ResultQuerier, commitErr error) deployment.ResultTxFactory {
	return func(_ context.Context, _ string) (deployment.ResultQuerier, func() error, func() error, error) {
		noop := func() error { return nil }
		commit := func() error { return commitErr }
		return writeQ, commit, noop, nil
	}
}

func TestHandleResult_EventsEmittedAfterCommit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		commitErr     error
		wantEvents    int
		wantHandleErr bool
	}{
		{
			name:          "commit succeeds emits events",
			commitErr:     nil,
			wantEvents:    1, // DeploymentEndpointCompleted for the target
			wantHandleErr: false,
		},
		{
			name:          "commit fails emits no events",
			commitErr:     errors.New("commit failed: serialization error"),
			wantEvents:    0,
			wantHandleErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bus := &fakeEventBus{}
			sm := deployment.NewStateMachine()

			deployID := validUUID("00000000-0000-0000-0000-000000000001")
			targetID := validUUID("00000000-0000-0000-0000-000000000010")
			cmdID := validUUID("00000000-0000-0000-0000-000000000020")
			tenantID := validUUID("00000000-0000-0000-0000-000000000002")

			// readQ handles GetCommandByID — command has a target so an event is expected.
			readQ := &fakeResultQuerier{
				command: sqlcgen.Command{
					DeploymentID: deployID,
					TargetID:     targetID,
				},
			}

			// writeQ handles writes; deployment is mid-flight so no completion events.
			writeQ := &fakeResultQuerier{
				incrementResult: sqlcgen.Deployment{
					Status:         "running",
					TotalTargets:   5,
					CompletedCount: 2,
					SuccessCount:   2,
					FailedCount:    0,
				},
			}

			var txFactory deployment.ResultTxFactory
			if tt.commitErr != nil {
				txFactory = fakeResultTxFactoryWithCommitErr(writeQ, tt.commitErr)
			} else {
				var committed bool
				txFactory = fakeResultTxFactoryWithCommitTracker(writeQ, &committed)
			}

			rh := deployment.NewResultHandler(readQ, sm, bus,
				deployment.WithResultTxFactory(txFactory),
			)
			err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "ok", "", "", nil)

			if tt.wantHandleErr && err == nil {
				t.Fatal("expected HandleResult to return an error")
			}
			if !tt.wantHandleErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := len(bus.events); got != tt.wantEvents {
				t.Fatalf("expected %d events on bus, got %d", tt.wantEvents, got)
			}

			// When events are emitted, verify the correct event type.
			if tt.wantEvents > 0 {
				found := false
				for _, evt := range bus.events {
					if evt.Type == events.DeploymentEndpointCompleted {
						found = true
					}
				}
				if !found {
					t.Fatal("expected deployment.endpoint_completed event")
				}
			}
		})
	}
}

func TestHandleResult_NoTargetNoEvents(t *testing.T) {
	t.Parallel()

	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	// Command with no target and no deployment — no pending events expected.
	readQ := &fakeResultQuerier{
		command: sqlcgen.Command{},
	}

	rh := deployment.NewResultHandler(readQ, sm, bus)
	err := rh.HandleResult(context.Background(), cmdID, tenantID, true, "", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bus.events) != 0 {
		t.Fatalf("expected 0 events for command without target, got %d", len(bus.events))
	}
}
