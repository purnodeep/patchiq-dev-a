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

// --- Fake TimeoutQuerier ---

type fakeTimeoutQuerier struct {
	commands          []sqlcgen.Command
	listErr           error
	updateCmdCalls    int
	updateTargetCalls int
	incrementCalls    int
	failDeployCalls   int
	incrementResult   sqlcgen.Deployment
	incrementErr      error
	failDeployResult  sqlcgen.Deployment
	failDeployErr     error
	updateCmdErr      error
	updateTargetErr   error
}

func (f *fakeTimeoutQuerier) ListTimedOutCommands(_ context.Context) ([]sqlcgen.Command, error) {
	return f.commands, f.listErr
}

func (f *fakeTimeoutQuerier) UpdateCommandStatus(_ context.Context, _ sqlcgen.UpdateCommandStatusParams) (sqlcgen.Command, error) {
	f.updateCmdCalls++
	return sqlcgen.Command{}, f.updateCmdErr
}

func (f *fakeTimeoutQuerier) UpdateDeploymentTargetStatus(_ context.Context, _ sqlcgen.UpdateDeploymentTargetStatusParams) (sqlcgen.DeploymentTarget, error) {
	f.updateTargetCalls++
	return sqlcgen.DeploymentTarget{}, f.updateTargetErr
}

func (f *fakeTimeoutQuerier) IncrementDeploymentCounters(_ context.Context, _ sqlcgen.IncrementDeploymentCountersParams) (sqlcgen.Deployment, error) {
	f.incrementCalls++
	return f.incrementResult, f.incrementErr
}

func (f *fakeTimeoutQuerier) SetDeploymentCompleted(_ context.Context, _ sqlcgen.SetDeploymentCompletedParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, nil
}

func (f *fakeTimeoutQuerier) SetDeploymentFailed(_ context.Context, _ sqlcgen.SetDeploymentFailedParams) (sqlcgen.Deployment, error) {
	f.failDeployCalls++
	return f.failDeployResult, f.failDeployErr
}

// --- Fake TxFactory for timeout tests ---

func fakeTimeoutTxFactory(writeQ deployment.TimeoutQuerier, committed *int) deployment.TimeoutTxFactory {
	return func(_ context.Context, _ string) (deployment.TimeoutQuerier, func() error, func() error, error) {
		noop := func() error { return nil }
		commit := func() error {
			*committed++
			return nil
		}
		return writeQ, commit, noop, nil
	}
}

func fakeTimeoutTxFactoryWithCommitErr(writeQ deployment.TimeoutQuerier, commitErr error) deployment.TimeoutTxFactory {
	return func(_ context.Context, _ string) (deployment.TimeoutQuerier, func() error, func() error, error) {
		noop := func() error { return nil }
		commit := func() error { return commitErr }
		return writeQ, commit, noop, nil
	}
}

// --- Tests ---

func TestTimeoutJobArgs_Kind(t *testing.T) {
	t.Parallel()
	args := deployment.TimeoutJobArgs{}
	if got := args.Kind(); got != "deployment_timeout_checker" {
		t.Fatalf("expected kind %q, got %q", "deployment_timeout_checker", got)
	}
}

func TestNewTimeoutChecker_NilEventBus_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil eventBus, got none")
		}
	}()
	q := &fakeTimeoutQuerier{}
	sm := deployment.NewStateMachine()
	deployment.NewTimeoutChecker(q, sm, nil)
}

func TestTimeoutChecker_NoTimedOut(t *testing.T) {
	t.Parallel()
	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()
	q := &fakeTimeoutQuerier{commands: []sqlcgen.Command{}}
	tc := deployment.NewTimeoutChecker(q, sm, bus)

	err := tc.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected no events, got %d", len(bus.events))
	}
}

func TestTimeoutChecker_TimesOutCommand(t *testing.T) {
	t.Parallel()
	deployID := validUUID("00000000-0000-0000-0000-000000000010")
	targetID := validUUID("00000000-0000-0000-0000-000000000011")
	tenantID := validUUID("00000000-0000-0000-0000-000000000012")
	cmdID := validUUID("00000000-0000-0000-0000-000000000013")

	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	// Set up threshold as pgtype.Numeric (0.5 = 50%)
	var threshold pgtype.Numeric
	_ = threshold.Scan("0.5")

	q := &fakeTimeoutQuerier{
		commands: []sqlcgen.Command{
			{
				ID:           cmdID,
				TenantID:     tenantID,
				DeploymentID: deployID,
				TargetID:     targetID,
			},
		},
		incrementResult: sqlcgen.Deployment{
			Status:           "running",
			TotalTargets:     2,
			CompletedCount:   1,
			FailedCount:      1,
			FailureThreshold: threshold,
		},
	}

	tc := deployment.NewTimeoutChecker(q, sm, bus)
	err := tc.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.updateCmdCalls != 1 {
		t.Fatalf("expected 1 UpdateCommandStatus call, got %d", q.updateCmdCalls)
	}
	if q.updateTargetCalls != 1 {
		t.Fatalf("expected 1 UpdateDeploymentTargetStatus call, got %d", q.updateTargetCalls)
	}
	if q.incrementCalls != 1 {
		t.Fatalf("expected 1 IncrementDeploymentCounters call, got %d", q.incrementCalls)
	}
	// failureRate = 1/2 = 0.5, threshold = 0.5 → 0.5 > 0.5 is false, so no fail
	if q.failDeployCalls != 0 {
		t.Fatalf("expected 0 SetDeploymentFailed calls (rate == threshold), got %d", q.failDeployCalls)
	}
}

func TestTimeoutChecker_TriggersDeploymentFailed(t *testing.T) {
	t.Parallel()
	deployID := validUUID("00000000-0000-0000-0000-000000000010")
	targetID := validUUID("00000000-0000-0000-0000-000000000011")
	tenantID := validUUID("00000000-0000-0000-0000-000000000012")
	cmdID := validUUID("00000000-0000-0000-0000-000000000013")

	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	var threshold pgtype.Numeric
	_ = threshold.Scan("0.3")

	q := &fakeTimeoutQuerier{
		commands: []sqlcgen.Command{
			{
				ID:           cmdID,
				TenantID:     tenantID,
				DeploymentID: deployID,
				TargetID:     targetID,
			},
		},
		incrementResult: sqlcgen.Deployment{
			Status:           "running",
			TotalTargets:     2,
			CompletedCount:   1,
			FailedCount:      1,
			FailureThreshold: threshold,
		},
	}

	tc := deployment.NewTimeoutChecker(q, sm, bus)
	err := tc.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// failureRate = 1/2 = 0.5 > 0.3 threshold → should fail deployment
	if q.failDeployCalls != 1 {
		t.Fatalf("expected 1 SetDeploymentFailed call, got %d", q.failDeployCalls)
	}
	// Should have deployment.failed event from FailDeployment
	found := false
	for _, e := range bus.events {
		if e.Type == events.DeploymentFailed {
			found = true
		}
	}
	if !found {
		t.Fatal("expected deployment.failed event")
	}
}

func TestTimeoutChecker_WithTxFactory(t *testing.T) {
	t.Parallel()
	tenantID := validUUID("00000000-0000-0000-0000-000000000012")
	cmdID := validUUID("00000000-0000-0000-0000-000000000013")

	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	// readQ only handles ListTimedOutCommands
	readQ := &fakeTimeoutQuerier{
		commands: []sqlcgen.Command{
			{
				ID:       cmdID,
				TenantID: tenantID,
				// No deployment or target links — simplest case
			},
		},
	}

	// writeQ handles writes via the tx factory
	writeQ := &fakeTimeoutQuerier{}

	var commitCount int
	tc := deployment.NewTimeoutChecker(readQ, sm, bus,
		deployment.WithTimeoutTxFactory(fakeTimeoutTxFactory(writeQ, &commitCount)),
	)
	err := tc.Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Writes should go through writeQ, not readQ
	if readQ.updateCmdCalls != 0 {
		t.Fatal("expected read querier NOT to receive writes")
	}
	if writeQ.updateCmdCalls != 1 {
		t.Fatalf("expected 1 UpdateCommandStatus on write querier, got %d", writeQ.updateCmdCalls)
	}
	if commitCount != 1 {
		t.Fatalf("expected 1 commit, got %d", commitCount)
	}
}

func TestProcessTimedOutCommand_EventsEmittedAfterCommit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		hasTarget    bool
		commitErr    error
		wantEvents   int
		wantCheckErr bool
	}{
		{
			name:         "commit succeeds emits events",
			commitErr:    nil,
			wantEvents:   1, // CommandTimedOut event
			wantCheckErr: false,
		},
		{
			name:         "commit fails emits no events",
			commitErr:    errors.New("commit failed: serialization error"),
			wantEvents:   0,
			wantCheckErr: true,
		},
		{
			name:         "target-linked command emits both events",
			hasTarget:    true,
			commitErr:    nil,
			wantEvents:   2, // CommandTimedOut + DeploymentTargetTimedOut
			wantCheckErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenantID := validUUID("00000000-0000-0000-0000-000000000012")
			cmdID := validUUID("00000000-0000-0000-0000-000000000013")

			cmd := sqlcgen.Command{ID: cmdID, TenantID: tenantID}
			if tt.hasTarget {
				cmd.TargetID = validUUID("00000000-0000-0000-0000-000000000014")
			}

			bus := &fakeEventBus{}
			sm := deployment.NewStateMachine()

			readQ := &fakeTimeoutQuerier{
				commands: []sqlcgen.Command{cmd},
			}

			writeQ := &fakeTimeoutQuerier{}

			var txFactory deployment.TimeoutTxFactory
			if tt.commitErr != nil {
				txFactory = fakeTimeoutTxFactoryWithCommitErr(writeQ, tt.commitErr)
			} else {
				var committed int
				txFactory = fakeTimeoutTxFactory(writeQ, &committed)
			}

			tc := deployment.NewTimeoutChecker(readQ, sm, bus,
				deployment.WithTimeoutTxFactory(txFactory),
			)
			err := tc.Check(context.Background())

			if tt.wantCheckErr && err == nil {
				t.Fatal("expected Check to return an error")
			}
			if !tt.wantCheckErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := len(bus.events); got != tt.wantEvents {
				t.Fatalf("expected %d events on bus, got %d", tt.wantEvents, got)
			}

			// When events are emitted, verify the correct event types.
			if tt.wantEvents > 0 {
				foundCmd := false
				foundTarget := false
				for _, evt := range bus.events {
					switch evt.Type {
					case events.CommandTimedOut:
						foundCmd = true
					case events.DeploymentTargetTimedOut:
						foundTarget = true
					}
				}
				if !foundCmd {
					t.Fatal("expected command.timed_out event")
				}
				if tt.hasTarget && !foundTarget {
					t.Fatal("expected deployment_target.timed_out event")
				}
			}
		})
	}
}

func TestTimeoutChecker_TxFactoryError(t *testing.T) {
	t.Parallel()
	tenantID := validUUID("00000000-0000-0000-0000-000000000012")
	cmdID := validUUID("00000000-0000-0000-0000-000000000013")

	bus := &fakeEventBus{}
	sm := deployment.NewStateMachine()

	readQ := &fakeTimeoutQuerier{
		commands: []sqlcgen.Command{
			{ID: cmdID, TenantID: tenantID},
		},
	}

	txFactory := func(_ context.Context, _ string) (deployment.TimeoutQuerier, func() error, func() error, error) {
		return nil, nil, nil, errors.New("pool exhausted")
	}

	tc := deployment.NewTimeoutChecker(readQ, sm, bus, deployment.WithTimeoutTxFactory(txFactory))
	// Single command fails → returns error
	err := tc.Check(context.Background())
	if err == nil {
		t.Fatal("expected error when tx factory fails for all commands")
	}
}
