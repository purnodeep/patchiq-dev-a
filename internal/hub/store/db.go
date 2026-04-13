package store

// IMPORTANT: This file mirrors internal/server/store/db.go.
// Both files must be kept in sync until extracted to internal/shared/store/.
// NOTE(PIQ-75): server/store/db.go no longer has Pool(); hub retains it until
// hub router is refactored to accept *Store (same fix as PIQ-75 for hub).
// TODO(PIQ-14): Extract to shared/store when a third caller appears (CLAUDE.md rule 5).

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// NewPool creates a pgx connection pool from the given database URL.
// maxConns and minConns configure the pool size; pass 0 to use pgx defaults.
// The caller is responsible for calling pool.Close() on shutdown.
func NewPool(ctx context.Context, databaseURL string, maxConns, minConns int32) (*pgxpool.Pool, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("create database pool: empty database URL")
	}

	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	if maxConns > 0 {
		poolCfg.MaxConns = maxConns
	}
	if minConns > 0 {
		poolCfg.MinConns = minConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

// Store wraps a pgx connection pool and provides tenant-aware transactions.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store from an existing connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	if pool == nil {
		panic("store: NewStore called with nil pool")
	}
	return &Store{pool: pool}
}

// BeginTx starts a transaction and sets the transaction-local PostgreSQL
// parameter app.current_tenant_id from the tenant ID in ctx. This enables
// RLS policies to enforce tenant isolation for the duration of this transaction.
//
// The caller must commit or rollback the returned transaction.
func (s *Store) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tenantID, ok := tenant.TenantIDFromContext(ctx)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("begin tx: missing tenant ID in context")
	}

	if _, err := uuid.Parse(tenantID); err != nil {
		return nil, fmt.Errorf("begin tx: invalid tenant ID %q: %w", tenantID, err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	// Use LOCAL scope (true) so the variable resets when the transaction ends,
	// preventing tenant context from leaking into other requests on the same connection.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.ErrorContext(ctx, "rollback failed after set_config error",
				"rollback_error", rbErr,
				"original_error", err,
				"tenant_id", tenantID,
			)
		}
		return nil, fmt.Errorf("begin tx: set tenant context: %w", err)
	}

	return tx, nil
}

// Ping checks database connectivity (for health checks).
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Stat returns pool statistics (for observability).
func (s *Store) Stat() *pgxpool.Stat {
	return s.pool.Stat()
}

// Close closes the underlying connection pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Pool returns the underlying connection pool.
// Deprecated(PIQ-14): Prefer Ping() for health checks and Stat() for metrics.
// Direct pool access bypasses tenant-scoped BeginTx. Use only during M0
// bootstrapping; this method will be removed when Store is fully wired.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}
