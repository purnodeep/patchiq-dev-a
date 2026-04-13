package deployment_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

type fakeScanQuerier struct {
	endpoints []sqlcgen.Endpoint
	endpErr   error
	createCmd sqlcgen.Command
	createErr error
}

func (f *fakeScanQuerier) ListActiveEndpointsByTenant(_ context.Context, _ pgtype.UUID) ([]sqlcgen.Endpoint, error) {
	return f.endpoints, f.endpErr
}

func (f *fakeScanQuerier) CreateCommand(_ context.Context, _ sqlcgen.CreateCommandParams) (sqlcgen.Command, error) {
	return f.createCmd, f.createErr
}

func TestScanJobArgs_Kind(t *testing.T) {
	t.Parallel()
	args := deployment.ScanJobArgs{TenantID: "abc"}
	if got := args.Kind(); got != "scan_scheduler" {
		t.Fatalf("Kind() = %q, want %q", got, "scan_scheduler")
	}
}

func TestScanScheduler_CreatesCommands(t *testing.T) {
	t.Parallel()
	epID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeScanQuerier{
		endpoints: []sqlcgen.Endpoint{{ID: epID, TenantID: tenantID}},
		createCmd: sqlcgen.Command{ID: cmdID, TenantID: tenantID},
	}
	bus := &fakeEventBus{}
	sched := deployment.NewScanScheduler(q, bus)

	err := sched.ScanAll(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ScanAll() error = %v", err)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.events))
	}
}

func TestScanScheduler_NoEndpoints(t *testing.T) {
	t.Parallel()
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")
	q := &fakeScanQuerier{}
	bus := &fakeEventBus{}
	sched := deployment.NewScanScheduler(q, bus)

	err := sched.ScanAll(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ScanAll() error = %v", err)
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(bus.events))
	}
}

type fakeTenantLister struct {
	tenants []sqlcgen.Tenant
	err     error
}

func (f *fakeTenantLister) ListTenants(_ context.Context) ([]sqlcgen.Tenant, error) {
	return f.tenants, f.err
}

func TestScanWorker_EmptyTenantID_ScansAllTenants(t *testing.T) {
	t.Parallel()
	tenant1 := validUUID("00000000-0000-0000-0000-000000000001")
	tenant2 := validUUID("00000000-0000-0000-0000-000000000002")
	epID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")

	q := &fakeScanQuerier{
		endpoints: []sqlcgen.Endpoint{{ID: epID, TenantID: tenant1}},
		createCmd: sqlcgen.Command{ID: cmdID, TenantID: tenant1},
	}
	bus := &fakeEventBus{}
	sched := deployment.NewScanScheduler(q, bus)

	lister := &fakeTenantLister{
		tenants: []sqlcgen.Tenant{
			{ID: tenant1, Name: "t1"},
			{ID: tenant2, Name: "t2"},
		},
	}
	worker := deployment.NewScanWorker(sched, lister)

	job := &river.Job[deployment.ScanJobArgs]{
		Args: deployment.ScanJobArgs{TenantID: ""},
	}
	err := worker.Work(context.Background(), job)
	if err != nil {
		t.Fatalf("Work() error = %v", err)
	}
	// Should have emitted events for both tenants (1 endpoint each call)
	if len(bus.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(bus.events))
	}
}

func TestScanSingle_Success(t *testing.T) {
	t.Parallel()
	epID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeScanQuerier{
		createCmd: sqlcgen.Command{ID: cmdID, TenantID: tenantID},
	}
	bus := &fakeEventBus{}
	sched := deployment.NewScanScheduler(q, bus)

	gotID, err := sched.ScanSingle(context.Background(), epID, tenantID, "", "")
	if err != nil {
		t.Fatalf("ScanSingle() error = %v", err)
	}
	if gotID != cmdID {
		t.Fatalf("ScanSingle() returned command ID %v, want %v", gotID, cmdID)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.events))
	}
	// Empty actorID/actorType must fall through to the system actor so the
	// audit row never has an empty attribution.
	evt := bus.events[0]
	if evt.ActorType != domain.ActorSystem || evt.ActorID != domain.ActorSystem {
		t.Errorf("system fallback: ActorID=%q ActorType=%q, want both %q", evt.ActorID, evt.ActorType, domain.ActorSystem)
	}
	if evt.Type != events.ScanTriggered {
		t.Errorf("Type = %q, want %q", evt.Type, events.ScanTriggered)
	}
}

func TestScanSingle_UserActorPropagatesToEvent(t *testing.T) {
	t.Parallel()
	epID := validUUID("00000000-0000-0000-0000-000000000010")
	cmdID := validUUID("00000000-0000-0000-0000-000000000020")
	tenantID := validUUID("00000000-0000-0000-0000-000000000002")

	q := &fakeScanQuerier{
		createCmd: sqlcgen.Command{ID: cmdID, TenantID: tenantID},
	}
	bus := &fakeEventBus{}
	sched := deployment.NewScanScheduler(q, bus)

	const userID = "user-42"
	if _, err := sched.ScanSingle(context.Background(), epID, tenantID, userID, domain.ActorUser); err != nil {
		t.Fatalf("ScanSingle() error = %v", err)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.events))
	}
	evt := bus.events[0]
	if evt.ActorID != userID {
		t.Errorf("ActorID = %q, want %q", evt.ActorID, userID)
	}
	if evt.ActorType != domain.ActorUser {
		t.Errorf("ActorType = %q, want %q", evt.ActorType, domain.ActorUser)
	}
}
