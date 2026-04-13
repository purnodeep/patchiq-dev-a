package deployment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// ScanJobArgs are the arguments for a River scan scheduler job.
type ScanJobArgs struct {
	TenantID string `json:"tenant_id"`
}

// Kind implements river.JobArgs.
func (ScanJobArgs) Kind() string { return "scan_scheduler" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ScanJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// ScanQuerier defines the store methods needed by ScanScheduler.
type ScanQuerier interface {
	ListActiveEndpointsByTenant(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.Endpoint, error)
	CreateCommand(ctx context.Context, arg sqlcgen.CreateCommandParams) (sqlcgen.Command, error)
}

// TenantLister lists all tenants for system-wide scans.
type TenantLister interface {
	ListTenants(ctx context.Context) ([]sqlcgen.Tenant, error)
}

// ScanScheduler creates run_scan commands for active endpoints.
type ScanScheduler struct {
	q        ScanQuerier
	eventBus domain.EventBus
}

func NewScanScheduler(q ScanQuerier, eventBus domain.EventBus) *ScanScheduler {
	if q == nil {
		panic("deployment: NewScanScheduler called with nil querier")
	}
	if eventBus == nil {
		panic("deployment: NewScanScheduler called with nil eventBus")
	}
	return &ScanScheduler{q: q, eventBus: eventBus}
}

// ScanAll creates a run_scan command for every active endpoint in the tenant.
// This path is only invoked by the periodic scheduler worker, so the event is
// always attributed to the system actor.
func (s *ScanScheduler) ScanAll(ctx context.Context, tenantID pgtype.UUID) error {
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()

	endpoints, err := s.q.ListActiveEndpointsByTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("scan scheduler: list endpoints: %w", err)
	}

	var errs []error
	for _, ep := range endpoints {
		cmd, err := s.q.CreateCommand(ctx, sqlcgen.CreateCommandParams{
			TenantID: tenantID,
			AgentID:  ep.ID,
			Type:     string(CommandTypeRunScan),
			Status:   string(CommandPending),
		})
		if err != nil {
			slog.ErrorContext(ctx, "scan scheduler: create scan command",
				"endpoint_id", uuid.UUID(ep.ID.Bytes).String(), "error", err)
			errs = append(errs, err)
			continue
		}

		endpointIDStr := uuid.UUID(ep.ID.Bytes).String()
		// Post-write event emission is best-effort: the command was already created.
		evt := domain.NewSystemEvent(events.ScanTriggered, tenantIDStr, "endpoint",
			endpointIDStr, events.ScanTriggered, events.ScanTriggeredPayload{
				CommandID:  uuid.UUID(cmd.ID.Bytes).String(),
				EndpointID: endpointIDStr,
			})
		EmitBestEffort(ctx, s.eventBus, []domain.DomainEvent{evt})
	}
	if len(errs) > 0 {
		return fmt.Errorf("scan scheduler: %d of %d scan commands failed: %w", len(errs), len(endpoints), errors.Join(errs...))
	}
	return nil
}

// ScanSingle creates a run_scan command for a single endpoint. actorID/actorType
// are attributed on the emitted scan.triggered event; an empty actorID falls back
// to the system actor (used by internal callers without request context).
// Returns the ID of the created command so callers can track its status.
func (s *ScanScheduler) ScanSingle(ctx context.Context, endpointID, tenantID pgtype.UUID, actorID, actorType string) (pgtype.UUID, error) {
	tenantIDStr := uuid.UUID(tenantID.Bytes).String()

	cmd, err := s.q.CreateCommand(ctx, sqlcgen.CreateCommandParams{
		TenantID: tenantID,
		AgentID:  endpointID,
		Type:     string(CommandTypeRunScan),
		Status:   string(CommandPending),
	})
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("scan single: create command: %w", err)
	}

	if actorID == "" {
		actorID = domain.ActorSystem
		actorType = domain.ActorSystem
	}

	endpointIDStr := uuid.UUID(endpointID.Bytes).String()
	// Post-write event emission is best-effort: the command was already created.
	evt := domain.NewAuditEvent(
		events.ScanTriggered,
		tenantIDStr,
		actorID,
		actorType,
		"endpoint",
		endpointIDStr,
		events.ScanTriggered,
		events.ScanTriggeredPayload{
			CommandID:  uuid.UUID(cmd.ID.Bytes).String(),
			EndpointID: endpointIDStr,
		},
		domain.EventMeta{},
	)
	EmitBestEffort(ctx, s.eventBus, []domain.DomainEvent{evt})
	return cmd.ID, nil
}

// ScanWorker wraps ScanScheduler as a River worker.
type ScanWorker struct {
	river.WorkerDefaults[ScanJobArgs]
	scheduler    *ScanScheduler
	tenantLister TenantLister
}

func NewScanWorker(scheduler *ScanScheduler, tenantLister TenantLister) *ScanWorker {
	if scheduler == nil {
		panic("deployment: NewScanWorker called with nil scheduler")
	}
	if tenantLister == nil {
		panic("deployment: NewScanWorker called with nil tenantLister")
	}
	return &ScanWorker{scheduler: scheduler, tenantLister: tenantLister}
}

// Work implements river.Worker.
func (w *ScanWorker) Work(ctx context.Context, job *river.Job[ScanJobArgs]) error {
	// Periodic system-wide scan: TenantID is empty, scan all tenants.
	if job.Args.TenantID == "" {
		tenants, err := w.tenantLister.ListTenants(ctx)
		if err != nil {
			return fmt.Errorf("scan worker: list tenants: %w", err)
		}
		var errs []error
		for _, t := range tenants {
			if err := w.scheduler.ScanAll(ctx, t.ID); err != nil {
				slog.ErrorContext(ctx, "scan worker: scan tenant failed",
					"tenant_id", uuid.UUID(t.ID.Bytes).String(), "error", err)
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	tenantUUID, err := uuid.Parse(job.Args.TenantID)
	if err != nil {
		return fmt.Errorf("scan worker: invalid tenant_id: %w", err)
	}
	return w.scheduler.ScanAll(ctx, pgtype.UUID{Bytes: tenantUUID, Valid: true})
}
