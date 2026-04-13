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

// --- Fake ScheduleCheckerQuerier ---

type fakeScheduleCheckerQuerier struct {
	schedules        []sqlcgen.DeploymentSchedule
	listErr          error
	hasActive        bool
	hasActiveErr     error
	updateCalls      int
	updateErr        error
	createCalls      int
	createResult     sqlcgen.Deployment
	createErr        error
	lastCreateParams sqlcgen.CreateDeploymentWithWaveConfigParams
	lastUpdateParams sqlcgen.UpdateScheduleAfterRunParams
	hasActiveCalls   int
	hasActiveResults map[string]bool // keyed by policy UUID string for per-schedule control
}

func (f *fakeScheduleCheckerQuerier) ListDueSchedules(_ context.Context) ([]sqlcgen.DeploymentSchedule, error) {
	return f.schedules, f.listErr
}

func (f *fakeScheduleCheckerQuerier) HasActiveDeploymentForSchedule(_ context.Context, arg sqlcgen.HasActiveDeploymentForScheduleParams) (bool, error) {
	f.hasActiveCalls++
	if f.hasActiveResults != nil {
		key := pgUUIDToString(arg.PolicyID)
		if v, ok := f.hasActiveResults[key]; ok {
			return v, f.hasActiveErr
		}
	}
	return f.hasActive, f.hasActiveErr
}

func (f *fakeScheduleCheckerQuerier) UpdateScheduleAfterRun(_ context.Context, arg sqlcgen.UpdateScheduleAfterRunParams) error {
	f.updateCalls++
	f.lastUpdateParams = arg
	return f.updateErr
}

func (f *fakeScheduleCheckerQuerier) CreateDeploymentWithWaveConfig(_ context.Context, arg sqlcgen.CreateDeploymentWithWaveConfigParams) (sqlcgen.Deployment, error) {
	f.createCalls++
	f.lastCreateParams = arg
	return f.createResult, f.createErr
}

func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return string([]byte{
		hexChar(b[0] >> 4), hexChar(b[0] & 0xf),
		hexChar(b[1] >> 4), hexChar(b[1] & 0xf),
		hexChar(b[2] >> 4), hexChar(b[2] & 0xf),
		hexChar(b[3] >> 4), hexChar(b[3] & 0xf),
		'-',
		hexChar(b[4] >> 4), hexChar(b[4] & 0xf),
		hexChar(b[5] >> 4), hexChar(b[5] & 0xf),
		'-',
		hexChar(b[6] >> 4), hexChar(b[6] & 0xf),
		hexChar(b[7] >> 4), hexChar(b[7] & 0xf),
		'-',
		hexChar(b[8] >> 4), hexChar(b[8] & 0xf),
		hexChar(b[9] >> 4), hexChar(b[9] & 0xf),
		'-',
		hexChar(b[10] >> 4), hexChar(b[10] & 0xf),
		hexChar(b[11] >> 4), hexChar(b[11] & 0xf),
		hexChar(b[12] >> 4), hexChar(b[12] & 0xf),
		hexChar(b[13] >> 4), hexChar(b[13] & 0xf),
		hexChar(b[14] >> 4), hexChar(b[14] & 0xf),
		hexChar(b[15] >> 4), hexChar(b[15] & 0xf),
	})
}

func hexChar(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + b - 10
}

// --- Tests ---

func TestScheduleCheckerJobArgs_Kind(t *testing.T) {
	t.Parallel()
	args := deployment.ScheduleCheckerJobArgs{}
	if got := args.Kind(); got != "schedule_checker" {
		t.Fatalf("expected kind %q, got %q", "schedule_checker", got)
	}
}

func TestNewScheduleChecker_NilQuerier_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil querier, got none")
		}
	}()
	bus := &fakeEventBus{}
	deployment.NewScheduleChecker(nil, bus)
}

func TestNewScheduleChecker_NilEventBus_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil eventBus, got none")
		}
	}()
	q := &fakeScheduleCheckerQuerier{}
	deployment.NewScheduleChecker(q, nil)
}

func TestScheduleChecker_Check(t *testing.T) {
	t.Parallel()

	tenantID := validUUID("00000000-0000-0000-0000-000000000001")
	policyID := validUUID("00000000-0000-0000-0000-000000000002")
	createdBy := validUUID("00000000-0000-0000-0000-000000000003")
	scheduleID := validUUID("00000000-0000-0000-0000-000000000004")
	deployID := validUUID("00000000-0000-0000-0000-000000000010")

	policyID2 := validUUID("00000000-0000-0000-0000-000000000005")
	scheduleID2 := validUUID("00000000-0000-0000-0000-000000000006")

	baseSchedule := sqlcgen.DeploymentSchedule{
		ID:             scheduleID,
		TenantID:       tenantID,
		PolicyID:       policyID,
		CronExpression: "0 2 * * *",
		WaveConfig:     []byte(`[{"percentage":100}]`),
		MaxConcurrent:  pgtype.Int4{Int32: 10, Valid: true},
		Enabled:        true,
		NextRunAt:      pgtype.Timestamptz{Time: time.Now().Add(-1 * time.Hour), Valid: true},
		CreatedBy:      createdBy,
	}

	tests := []struct {
		name             string
		schedules        []sqlcgen.DeploymentSchedule
		listErr          error
		hasActive        bool
		hasActiveResults map[string]bool
		hasActiveErr     error
		createResult     sqlcgen.Deployment
		createErr        error
		updateErr        error
		wantCreateCalls  int
		wantUpdateCalls  int
		wantEvents       int
		wantErr          bool
	}{
		{
			name:            "no due schedules is a no-op",
			schedules:       []sqlcgen.DeploymentSchedule{},
			wantCreateCalls: 0,
			wantUpdateCalls: 0,
			wantEvents:      0,
		},
		{
			name:       "list error returns error",
			listErr:    errors.New("db connection lost"),
			wantErr:    true,
			wantEvents: 0,
		},
		{
			name:      "due schedule with no active deployment creates deployment",
			schedules: []sqlcgen.DeploymentSchedule{baseSchedule},
			hasActive: false,
			createResult: sqlcgen.Deployment{
				ID:       deployID,
				TenantID: tenantID,
				PolicyID: policyID,
				Status:   "created",
			},
			wantCreateCalls: 1,
			wantUpdateCalls: 1,
			wantEvents:      1,
		},
		{
			name:            "due schedule with active deployment skips creation",
			schedules:       []sqlcgen.DeploymentSchedule{baseSchedule},
			hasActive:       true,
			wantCreateCalls: 0,
			wantUpdateCalls: 0,
			wantEvents:      0,
		},
		{
			name: "multiple due schedules processed independently",
			schedules: []sqlcgen.DeploymentSchedule{
				baseSchedule,
				{
					ID:             scheduleID2,
					TenantID:       tenantID,
					PolicyID:       policyID2,
					CronExpression: "30 3 * * *",
					WaveConfig:     []byte(`[{"percentage":50}]`),
					MaxConcurrent:  pgtype.Int4{Int32: 5, Valid: true},
					Enabled:        true,
					NextRunAt:      pgtype.Timestamptz{Time: time.Now().Add(-30 * time.Minute), Valid: true},
					CreatedBy:      createdBy,
				},
			},
			hasActive: false,
			createResult: sqlcgen.Deployment{
				ID:       deployID,
				TenantID: tenantID,
				Status:   "created",
			},
			wantCreateCalls: 2,
			wantUpdateCalls: 2,
			wantEvents:      2,
		},
		{
			name:      "create deployment error logs and continues",
			schedules: []sqlcgen.DeploymentSchedule{baseSchedule},
			hasActive: false,
			createErr: errors.New("unique constraint violation"),
			// Should NOT return error — individual failures are logged
			wantCreateCalls: 1,
			wantUpdateCalls: 0,
			wantEvents:      0,
		},
		{
			name:            "has active check error logs and continues",
			schedules:       []sqlcgen.DeploymentSchedule{baseSchedule},
			hasActiveErr:    errors.New("db timeout"),
			wantCreateCalls: 0,
			wantUpdateCalls: 0,
			wantEvents:      0,
		},
		{
			name:      "update schedule error logs and continues (deployment already created)",
			schedules: []sqlcgen.DeploymentSchedule{baseSchedule},
			hasActive: false,
			createResult: sqlcgen.Deployment{
				ID:       deployID,
				TenantID: tenantID,
				PolicyID: policyID,
				Status:   "created",
			},
			updateErr:       errors.New("update failed"),
			wantCreateCalls: 1,
			wantUpdateCalls: 1,
			// Event still emitted since deployment was created
			wantEvents: 1,
		},
		{
			name: "mixed: one active one not — only creates for non-active",
			schedules: []sqlcgen.DeploymentSchedule{
				baseSchedule,
				{
					ID:             scheduleID2,
					TenantID:       tenantID,
					PolicyID:       policyID2,
					CronExpression: "30 3 * * *",
					WaveConfig:     []byte(`[{"percentage":50}]`),
					MaxConcurrent:  pgtype.Int4{Int32: 5, Valid: true},
					Enabled:        true,
					NextRunAt:      pgtype.Timestamptz{Time: time.Now().Add(-30 * time.Minute), Valid: true},
					CreatedBy:      createdBy,
				},
			},
			hasActiveResults: map[string]bool{
				pgUUIDToString(policyID):  true,
				pgUUIDToString(policyID2): false,
			},
			createResult: sqlcgen.Deployment{
				ID:       deployID,
				TenantID: tenantID,
				Status:   "created",
			},
			wantCreateCalls: 1,
			wantUpdateCalls: 1,
			wantEvents:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bus := &fakeEventBus{}
			q := &fakeScheduleCheckerQuerier{
				schedules:        tt.schedules,
				listErr:          tt.listErr,
				hasActive:        tt.hasActive,
				hasActiveResults: tt.hasActiveResults,
				hasActiveErr:     tt.hasActiveErr,
				createResult:     tt.createResult,
				createErr:        tt.createErr,
				updateErr:        tt.updateErr,
			}

			sc := deployment.NewScheduleChecker(q, bus)
			err := sc.Check(context.Background())

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if q.createCalls != tt.wantCreateCalls {
				t.Fatalf("expected %d CreateDeploymentWithWaveConfig calls, got %d", tt.wantCreateCalls, q.createCalls)
			}
			if q.updateCalls != tt.wantUpdateCalls {
				t.Fatalf("expected %d UpdateScheduleAfterRun calls, got %d", tt.wantUpdateCalls, q.updateCalls)
			}
			if got := len(bus.events); got != tt.wantEvents {
				t.Fatalf("expected %d events, got %d", tt.wantEvents, got)
			}

			// Verify event types when events are expected.
			for _, evt := range bus.events {
				if evt.Type != events.DeploymentCreated {
					t.Fatalf("expected event type %q, got %q", events.DeploymentCreated, evt.Type)
				}
			}
		})
	}
}
