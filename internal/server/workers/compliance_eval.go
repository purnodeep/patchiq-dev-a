package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/compliance"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// ComplianceEvalJobArgs defines the River periodic job for compliance evaluation.
type ComplianceEvalJobArgs struct{}

// Kind implements river.JobArgs.
func (ComplianceEvalJobArgs) Kind() string { return "compliance_evaluation" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ComplianceEvalJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// TenantLister queries tenant IDs for the compliance evaluation cycle.
type TenantLister interface {
	ListAllTenantIDs(ctx context.Context) ([]pgtype.UUID, error)
}

// ComplianceEvaluator runs the full compliance evaluation cycle across all tenants.
// It delegates per-tenant evaluation to compliance.Service.RunEvaluation, which
// handles built-in frameworks, custom frameworks, control results, and SLA overrides.
type ComplianceEvaluator struct {
	lister TenantLister
	svc    *compliance.Service
	st     *store.Store
	bus    domain.EventBus
}

// NewComplianceEvaluator creates a new ComplianceEvaluator that delegates to the compliance Service.
func NewComplianceEvaluator(lister TenantLister, svc *compliance.Service, st *store.Store, bus domain.EventBus) *ComplianceEvaluator {
	if lister == nil {
		panic("compliance: NewComplianceEvaluator called with nil lister")
	}
	if svc == nil {
		panic("compliance: NewComplianceEvaluator called with nil service")
	}
	if st == nil {
		panic("compliance: NewComplianceEvaluator called with nil store")
	}
	if bus == nil {
		panic("compliance: NewComplianceEvaluator called with nil eventBus")
	}
	return &ComplianceEvaluator{lister: lister, svc: svc, st: st, bus: bus}
}

// retentionDays is how long evaluation and score data is kept.
const retentionDays = 30

// Evaluate runs the full compliance evaluation cycle for all tenants.
func (e *ComplianceEvaluator) Evaluate(ctx context.Context) error {
	tenantIDs, err := e.lister.ListAllTenantIDs(ctx)
	if err != nil {
		return fmt.Errorf("compliance evaluation: list tenant IDs: %w", err)
	}

	now := time.Now().UTC()

	var failures int
	for _, tenantID := range tenantIDs {
		if err := e.evaluateTenant(ctx, tenantID, now); err != nil {
			failures++
			tenantStr := uuid.UUID(tenantID.Bytes).String()
			slog.ErrorContext(ctx, "compliance evaluation: tenant evaluation failed",
				"tenant_id", tenantStr,
				"error", err,
			)
			continue
		}
	}

	slog.InfoContext(ctx, "compliance evaluation complete",
		"tenants_evaluated", len(tenantIDs),
		"tenants_failed", failures,
	)

	if failures > 0 && failures == len(tenantIDs) {
		return fmt.Errorf("compliance evaluation: all %d tenants failed", failures)
	}

	return nil
}

func (e *ComplianceEvaluator) evaluateTenant(ctx context.Context, tenantID pgtype.UUID, now time.Time) error {
	tenantStr := uuid.UUID(tenantID.Bytes).String()

	// Inject tenant ID into context so store.BeginTx can set RLS.
	tenantCtx := tenant.WithTenantID(ctx, tenantStr)

	// Begin a transaction with RLS set for this tenant.
	tx, err := e.st.BeginTx(tenantCtx)
	if err != nil {
		return fmt.Errorf("begin tx for tenant %s: %w", tenantStr, err)
	}
	defer func() {
		if err := tx.Rollback(tenantCtx); err != nil {
			// Rollback after commit is a no-op; only log real errors.
			slog.DebugContext(ctx, "compliance eval: rollback after eval", "error", err)
		}
	}()

	txQ := sqlcgen.New(tx)

	result, err := e.svc.RunEvaluation(tenantCtx, tenantID, txQ)
	if err != nil {
		return fmt.Errorf("evaluate tenant %s: %w", tenantStr, err)
	}

	if err := tx.Commit(tenantCtx); err != nil {
		return fmt.Errorf("commit evaluation for tenant %s: %w", tenantStr, err)
	}

	// Cleanup old data (outside the main tx, best-effort).
	cutoff := pgtype.Timestamptz{Time: now.AddDate(0, 0, -retentionDays), Valid: true}
	cleanupCtx := tenant.WithTenantID(ctx, tenantStr)
	cleanupTx, err := e.st.BeginTx(cleanupCtx)
	if err != nil {
		slog.WarnContext(ctx, "compliance eval: failed to begin cleanup tx", "tenant_id", tenantStr, "error", err)
	} else {
		cleanupQ := sqlcgen.New(cleanupTx)
		if err := cleanupQ.DeleteOldEvaluations(cleanupCtx, sqlcgen.DeleteOldEvaluationsParams{
			TenantID: tenantID,
			Before:   cutoff,
		}); err != nil {
			slog.WarnContext(ctx, "compliance eval: failed to cleanup old evaluations", "tenant_id", tenantStr, "error", err)
		}
		if err := cleanupQ.DeleteOldScores(cleanupCtx, sqlcgen.DeleteOldScoresParams{
			TenantID: tenantID,
			Before:   cutoff,
		}); err != nil {
			slog.WarnContext(ctx, "compliance eval: failed to cleanup old scores", "tenant_id", tenantStr, "error", err)
		}
		if err := cleanupTx.Commit(cleanupCtx); err != nil {
			slog.WarnContext(ctx, "compliance eval: cleanup commit failed", "tenant_id", tenantStr, "error", err)
		}
	}

	// Emit domain event.
	if err := e.bus.Emit(ctx, domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       "compliance.evaluation_completed",
		TenantID:   tenantStr,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "compliance_evaluation",
		ResourceID: result.RunID,
		Action:     "completed",
		Payload: map[string]any{
			"frameworks_evaluated": result.FrameworksEvaluated,
		},
		Timestamp: now,
	}); err != nil {
		slog.ErrorContext(ctx, "compliance eval: failed to emit event", "tenant_id", tenantStr, "error", err)
	}

	return nil
}

// ComplianceEvalWorker wraps ComplianceEvaluator as a River worker.
type ComplianceEvalWorker struct {
	river.WorkerDefaults[ComplianceEvalJobArgs]
	evaluator *ComplianceEvaluator
}

// NewComplianceEvalWorker creates a new ComplianceEvalWorker.
func NewComplianceEvalWorker(evaluator *ComplianceEvaluator) *ComplianceEvalWorker {
	return &ComplianceEvalWorker{evaluator: evaluator}
}

// Work implements river.Worker.
func (w *ComplianceEvalWorker) Work(ctx context.Context, _ *river.Job[ComplianceEvalJobArgs]) error {
	return w.evaluator.Evaluate(ctx)
}
