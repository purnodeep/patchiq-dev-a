package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// PatchUpserter abstracts batch patch upsert operations.
type PatchUpserter interface {
	BeginBatch(ctx context.Context, tenantID string) (BatchUpserter, error)
}

// BatchUpserter abstracts a transactional batch of patch upserts.
type BatchUpserter interface {
	UpsertPatch(ctx context.Context, p DiscoveredPatch) (string, bool, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context)
}

// HubAliasReporter reports discovered packages to Hub for alias resolution.
type HubAliasReporter interface {
	ReportAliases(ctx context.Context, packages []DiscoveredAlias) error
}

// DiscoveredAlias represents a package discovered on an endpoint.
type DiscoveredAlias struct {
	OsPackageName  string `json:"os_package_name"`
	OsFamily       string `json:"os_family"`
	OsDistribution string `json:"os_distribution"`
}

// StoreAdapter implements PatchUpserter using pgxpool and sqlcgen.
type StoreAdapter struct {
	pool        *pgxpool.Pool
	hubReporter HubAliasReporter
}

// NewStoreAdapter creates a StoreAdapter backed by the given connection pool.
func NewStoreAdapter(pool *pgxpool.Pool) *StoreAdapter {
	return &StoreAdapter{pool: pool}
}

// WithHubReporter sets an optional HubAliasReporter on the adapter.
func (a *StoreAdapter) WithHubReporter(r HubAliasReporter) {
	a.hubReporter = r
}

// ReportDiscoveredAliases sends discovered packages to Hub for alias resolution.
// This is best-effort — failures are logged but don't block discovery.
func (a *StoreAdapter) ReportDiscoveredAliases(ctx context.Context, packages []DiscoveredAlias) {
	if a.hubReporter == nil || len(packages) == 0 {
		return
	}
	if err := a.hubReporter.ReportAliases(ctx, packages); err != nil {
		slog.ErrorContext(ctx, "discovery: report aliases to hub failed", "count", len(packages), "error", err)
	}
}

// BeginBatch starts a new transaction scoped to the given tenant for batch upserts.
func (a *StoreAdapter) BeginBatch(ctx context.Context, tenantID string) (BatchUpserter, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("begin batch: parse tenant ID: %w", err)
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin batch: begin tx: %w", err)
	}

	// Set tenant context for RLS
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			slog.ErrorContext(ctx, "begin batch: rollback failed", "error", rbErr)
		}
		return nil, fmt.Errorf("begin batch: set tenant context: %w", err)
	}

	return &storeBatchUpserter{tx: tx, tenantUUID: tenantUUID}, nil
}

type storeBatchUpserter struct {
	tx         pgx.Tx
	tenantUUID pgtype.UUID
}

func (b *storeBatchUpserter) UpsertPatch(ctx context.Context, p DiscoveredPatch) (string, bool, error) {
	q := sqlcgen.New(b.tx)
	result, err := q.UpsertDiscoveredPatch(ctx, sqlcgen.UpsertDiscoveredPatchParams{
		TenantID:       b.tenantUUID,
		Name:           p.Name,
		Version:        p.Version,
		Severity:       p.Priority,
		OsFamily:       p.OsFamily,
		OsDistribution: pgtype.Text{String: p.OsDistro, Valid: p.OsDistro != ""},
		PackageUrl:     pgtype.Text{String: p.Filename, Valid: p.Filename != ""},
		ChecksumSha256: pgtype.Text{String: p.Checksum, Valid: p.Checksum != ""},
		SourceRepo:     pgtype.Text{String: p.SourceRepo, Valid: p.SourceRepo != ""},
		Description:    pgtype.Text{String: p.Description, Valid: p.Description != ""},
		PackageName:    p.Name,
	})
	if err != nil {
		return "", false, fmt.Errorf("upsert patch %s: %w", p.Name, err)
	}

	patchID := uuidToString(result.ID)
	isNew := result.CreatedAt.Time.Equal(result.UpdatedAt.Time)
	return patchID, isNew, nil
}

func (b *storeBatchUpserter) Commit(ctx context.Context) error {
	return b.tx.Commit(ctx)
}

func (b *storeBatchUpserter) Rollback(ctx context.Context) {
	if err := b.tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.ErrorContext(ctx, "batch rollback failed", "error", err)
	}
}

func parsePgUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
