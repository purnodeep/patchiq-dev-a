package targeting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPolicyNotFound is returned by ResolveForPolicy when the policy row
// does not exist (or is invisible to the caller under RLS). This lets
// engines distinguish "no targeting row → match all endpoints" from
// "typo/stale id → do not dispatch anywhere", which matters because the
// match-all fallback for a missing policy would otherwise dispatch a
// patch wave to every endpoint in the tenant.
var ErrPolicyNotFound = errors.New("targeting: policy not found")

// ErrInvalidTenantID is returned when the caller passes a tenantID that
// does not parse as a UUID. Failing fast here avoids cryptic RLS errors
// when Postgres tries to cast the setting to ::uuid.
var ErrInvalidTenantID = errors.New("targeting: invalid tenant id")

// ErrInvalidPolicyID is returned when the caller passes a non-UUID policyID.
var ErrInvalidPolicyID = errors.New("targeting: invalid policy id")

// Resolver executes compiled selectors against PostgreSQL. Engines consume
// Resolver — never the unexported compile — so the SQL assembly boundary
// stays internal to this package.
type Resolver struct {
	pool *pgxpool.Pool
}

// NewResolver constructs a Resolver.
func NewResolver(pool *pgxpool.Pool) *Resolver {
	if pool == nil {
		panic("targeting: NewResolver called with nil pool")
	}
	return &Resolver{pool: pool}
}

// Resolve returns endpoint IDs matching the selector within the tenant.
// A nil selector is treated as "match every non-decommissioned endpoint
// in the tenant".
func (r *Resolver) Resolve(ctx context.Context, tenantID string, sel *Selector) ([]uuid.UUID, error) {
	query, args, err := buildQuery(sel, "SELECT "+endpointAlias+".id")
	if err != nil {
		return nil, err
	}
	var ids []uuid.UUID
	err = r.withTenantTx(ctx, tenantID, "Resolve", func(ctx context.Context, tx pgx.Tx) error {
		rows, qerr := tx.Query(ctx, query, args...)
		if qerr != nil {
			return fmt.Errorf("targeting: query: %w", qerr)
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			if serr := rows.Scan(&id); serr != nil {
				return fmt.Errorf("targeting: scan endpoint id: %w", serr)
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		// Discard any partial results so callers cannot accidentally use a
		// half-built slice when the query errored mid-iteration.
		return nil, err
	}
	return ids, nil
}

// Count returns the number of endpoints matching the selector within the
// tenant. Cheaper than Resolve when only the match count is needed (e.g.
// for live-preview in the selector builder UI).
func (r *Resolver) Count(ctx context.Context, tenantID string, sel *Selector) (int, error) {
	query, args, err := buildQuery(sel, "SELECT count(*)")
	if err != nil {
		return 0, err
	}
	var n int
	err = r.withTenantTx(ctx, tenantID, "Count", func(ctx context.Context, tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(&n)
	})
	if err != nil {
		return 0, err
	}
	return n, nil
}

// ResolveForPolicy loads the stored selector for the given policy and
// resolves it. Returns ErrPolicyNotFound if the policy row does not exist
// (or is invisible under RLS). A policy that exists but has no selector
// row targets every non-decommissioned endpoint in the tenant.
func (r *Resolver) ResolveForPolicy(ctx context.Context, tenantID, policyID string) ([]uuid.UUID, error) {
	if _, err := uuid.Parse(policyID); err != nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidPolicyID, policyID)
	}

	var ids []uuid.UUID
	err := r.withTenantTx(ctx, tenantID, "ResolveForPolicy", func(ctx context.Context, tx pgx.Tx) error {
		exists, eerr := policyExistsTx(ctx, tx, tenantID, policyID)
		if eerr != nil {
			return eerr
		}
		if !exists {
			return ErrPolicyNotFound
		}

		sel, lerr := loadPolicySelectorTx(ctx, tx, tenantID, policyID)
		if lerr != nil {
			return lerr
		}

		query, args, berr := buildQuery(sel, "SELECT "+endpointAlias+".id")
		if berr != nil {
			return berr
		}
		rows, qerr := tx.Query(ctx, query, args...)
		if qerr != nil {
			return fmt.Errorf("targeting: query: %w", qerr)
		}
		defer rows.Close()
		for rows.Next() {
			var id uuid.UUID
			if serr := rows.Scan(&id); serr != nil {
				return fmt.Errorf("targeting: scan endpoint id: %w", serr)
			}
			ids = append(ids, id)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// withTenantTx opens a read-only transaction, sets the tenant-local
// parameter so RLS filters correctly, runs fn, and rolls back. Read-only
// rollback is equivalent to commit for query semantics but avoids leaving
// session state on the pooled connection.
//
// tenantID is parsed as a UUID before any SQL runs so that a malformed
// caller input fails fast with a clear error instead of surfacing as a
// cryptic Postgres cast failure inside the RLS USING clause.
//
// op is a short operation label ("Resolve", "Count", "ResolveForPolicy")
// included in teardown log context so an operator debugging a dropped
// connection can tell which caller triggered it.
//
// NOTE on `set_config(...,true)` vs `SET LOCAL`: the CLAUDE.md DB
// convention prescribes `SET LOCAL app.current_tenant_id = $tenant_id`,
// but SET LOCAL does not accept positional parameters — the tenant UUID
// would have to be interpolated into the statement string, defeating
// prepared-statement safety. `set_config('name', $1, true)` is the
// documented parameterised equivalent and is what `store.BeginTx` uses
// elsewhere in the codebase. Both set a transaction-local GUC; the tx
// is already open with ReadOnly access mode above.
func (r *Resolver) withTenantTx(ctx context.Context, tenantID, op string, fn func(context.Context, pgx.Tx) error) error {
	if _, err := uuid.Parse(tenantID); err != nil {
		return fmt.Errorf("%w: %q", ErrInvalidTenantID, tenantID)
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return fmt.Errorf("targeting: begin tx (op=%s): %w", op, err)
	}
	defer func() {
		// Rollback after a successful read-only tx is a no-op in effect but
		// pgx returns ErrTxClosed when the tx has already been finalised.
		// Any other rollback error is worth logging because it signals a
		// dropped connection or cancelled context during teardown — include
		// op and tenant_id so the log line is self-contained for oncall.
		if rerr := tx.Rollback(ctx); rerr != nil && !errors.Is(rerr, pgx.ErrTxClosed) {
			slog.WarnContext(ctx, "targeting: rollback failed",
				"error", rerr,
				"op", op,
				"tenant_id", tenantID,
			)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return fmt.Errorf("targeting: set tenant context (op=%s): %w", op, err)
	}
	return fn(ctx, tx)
}

func buildQuery(sel *Selector, projection string) (string, []any, error) {
	base := projection + " FROM endpoints " + endpointAlias + " WHERE " + endpointAlias + ".status != 'decommissioned'"
	if sel == nil {
		return base, nil, nil
	}
	if err := Validate(*sel); err != nil {
		return "", nil, err
	}
	opt := Optimize(*sel)
	frag, args, err := compile(opt)
	if err != nil {
		return "", nil, err
	}
	return base + " AND " + frag, args, nil
}

// policyExistsTx returns whether a policy row is visible to the current
// tenant context. RLS on `policies` means rows from other tenants are
// invisible, so a cross-tenant probe correctly returns false.
func policyExistsTx(ctx context.Context, tx pgx.Tx, tenantID, policyID string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM policies WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL)",
		policyID, tenantID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("targeting: check policy exists: %w", err)
	}
	return exists, nil
}

// loadPolicySelectorTx fetches the JSONB selector for a policy inside an
// already-open tenant-scoped transaction. Returns (nil, nil) when the
// policy exists but has no selector row — meaning "match all".
func loadPolicySelectorTx(ctx context.Context, tx pgx.Tx, tenantID, policyID string) (*Selector, error) {
	var raw []byte
	err := tx.QueryRow(ctx,
		"SELECT expression FROM policy_tag_selectors WHERE policy_id = $1 AND tenant_id = $2",
		policyID, tenantID,
	).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("targeting: load policy selector: %w", err)
	}
	var sel Selector
	if err := json.Unmarshal(raw, &sel); err != nil {
		return nil, fmt.Errorf("targeting: decode policy selector: %w", err)
	}
	return &sel, nil
}
