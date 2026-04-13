package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// PermissionQuerier is the subset of sqlcgen.Queries needed by SQLPermissionStore.
type PermissionQuerier interface {
	GetUserPermissions(ctx context.Context, arg sqlcgen.GetUserPermissionsParams) ([]sqlcgen.GetUserPermissionsRow, error)
}

// TxBeginner starts a tenant-scoped transaction that sets app.current_tenant_id
// for RLS policies.
type TxBeginner interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

// OrgPermissionLoader retrieves org-scoped permission grants from the store
// layer. It is implemented by *store.Store and called through the bypass pool
// because role_permissions is RLS-protected by tenant_id, and the evaluator's
// active tenant context may not match the platform tenant where org-scoped
// roles live.
type OrgPermissionLoader interface {
	GetUserOrgPermissions(ctx context.Context, orgID, userID string) ([]sqlcgen.GetUserOrgPermissionsRow, error)
}

// SQLPermissionStore implements PermissionStore by querying the database
// via sqlc-generated code. It runs tenant-scoped queries inside a tenant
// transaction (so RLS resolves correctly) and delegates org-scoped lookups
// to an optional OrgPermissionLoader injected via WithOrgLoader.
type SQLPermissionStore struct {
	txb       TxBeginner
	orgLoader OrgPermissionLoader // optional; nil means org-scoped grants are disabled
}

// NewSQLPermissionStore creates a SQLPermissionStore with tenant-scoped
// permission lookup only. Chain .WithOrgLoader(...) to enable org-scoped
// RBAC (MSP model).
func NewSQLPermissionStore(txb TxBeginner) *SQLPermissionStore {
	if txb == nil {
		panic("auth: NewSQLPermissionStore called with nil TxBeginner")
	}
	return &SQLPermissionStore{txb: txb}
}

// WithOrgLoader returns the store with org-scoped permission lookup enabled.
// Pass the same *store.Store that was used for TxBeginner; it satisfies both
// interfaces.
func (s *SQLPermissionStore) WithOrgLoader(loader OrgPermissionLoader) *SQLPermissionStore {
	s.orgLoader = loader
	return s
}

// GetUserOrgPermissions loads org-scoped permissions for the user. Returns
// an empty slice (no error) when no org loader is configured, which makes
// the evaluator fall through to the tenant-scoped check.
func (s *SQLPermissionStore) GetUserOrgPermissions(ctx context.Context, orgID, userID string) ([]Permission, error) {
	if s.orgLoader == nil {
		return nil, nil
	}
	rows, err := s.orgLoader.GetUserOrgPermissions(ctx, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("get user org permissions: %w", err)
	}
	perms := make([]Permission, len(rows))
	for i, r := range rows {
		perms[i] = Permission{Resource: r.Resource, Action: r.Action, Scope: r.Scope}
	}
	return perms, nil
}

// GetUserPermissions loads the effective permissions for a user, including inherited permissions.
// It opens a tenant-scoped transaction to satisfy RLS policies.
func (s *SQLPermissionStore) GetUserPermissions(ctx context.Context, tenantID, userID string) ([]Permission, error) {
	parsed, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("parse tenant ID %q: %w", tenantID, err)
	}

	// Ensure tenant ID is in context for BeginTx.
	ctx = tenant.WithTenantID(ctx, tenantID)

	tx, err := s.txb.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user permissions: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tid := pgtype.UUID{Bytes: parsed, Valid: true}
	q := sqlcgen.New(tx)

	rows, err := q.GetUserPermissions(ctx, sqlcgen.GetUserPermissionsParams{
		UserID:   userID,
		TenantID: tid,
	})
	if err != nil {
		return nil, fmt.Errorf("get user permissions: %w", err)
	}

	perms := make([]Permission, len(rows))
	for i, r := range rows {
		perms[i] = Permission{Resource: r.Resource, Action: r.Action, Scope: r.Scope}
	}
	return perms, nil
}

// GetUserRoles loads the role names assigned to a user within a tenant.
func (s *SQLPermissionStore) GetUserRoles(ctx context.Context, tenantID, userID string) ([]string, error) {
	parsed, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, fmt.Errorf("parse tenant ID %q: %w", tenantID, err)
	}

	ctx = tenant.WithTenantID(ctx, tenantID)

	tx, err := s.txb.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("get user roles: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tid := pgtype.UUID{Bytes: parsed, Valid: true}
	q := sqlcgen.New(tx)

	rows, err := q.ListUserRoles(ctx, sqlcgen.ListUserRolesParams{
		UserID:   userID,
		TenantID: tid,
	})
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = r.Name
	}
	return names, nil
}
