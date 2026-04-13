package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/shared/organization"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// ErrMissingTenantID is returned when the context does not carry a tenant ID.
var ErrMissingTenantID = errors.New("check permission: missing tenant ID in context")

// ErrMissingUserID is returned when the context does not carry a user ID.
var ErrMissingUserID = errors.New("check permission: missing user ID in context")

// PermissionStore loads the effective permissions for a user within a tenant,
// including permissions inherited through role hierarchy.
type PermissionStore interface {
	GetUserPermissions(ctx context.Context, tenantID, userID string) ([]Permission, error)
}

// OrgPermissionStore loads the effective permissions for a user within an
// organization, via org-scoped role assignments (e.g. MSP Admin). This is
// orthogonal to PermissionStore — org-scoped grants cover all tenants in the
// org, while tenant-scoped grants only cover one tenant.
//
// Implementations should tolerate missing platform tenants (empty result) and
// use an RLS-bypassing data path because role_permissions is tenant-scoped
// while the evaluator's active tenant context may not be the platform tenant.
type OrgPermissionStore interface {
	GetUserOrgPermissions(ctx context.Context, orgID, userID string) ([]Permission, error)
}

// Evaluator checks whether a user has a required permission by consulting
// tenant-scoped grants and (if configured) org-scoped grants.
type Evaluator struct {
	store    PermissionStore
	orgStore OrgPermissionStore // optional; if nil, org-scoped grants are ignored
}

// NewEvaluator creates an Evaluator with a tenant-scoped permission store.
// Call WithOrgStore to add org-scoped RBAC support for MSP deployments.
func NewEvaluator(store PermissionStore) *Evaluator {
	if store == nil {
		panic("auth: NewEvaluator called with nil store")
	}
	return &Evaluator{store: store}
}

// WithOrgStore returns the evaluator with org-scoped RBAC enabled. Safe to
// call with nil — a nil org store disables org checks (same as unconfigured).
func (e *Evaluator) WithOrgStore(orgStore OrgPermissionStore) *Evaluator {
	e.orgStore = orgStore
	return e
}

// HasPermission checks if the user (from context) holds the required permission.
// Checks org-scoped grants first (when an org ID is in context and an org store
// is configured); an org-scoped match short-circuits. Falls back to the
// tenant-scoped check on miss.
func (e *Evaluator) HasPermission(ctx context.Context, required Permission) (bool, error) {
	userID, ok := user.UserIDFromContext(ctx)
	if !ok || userID == "" {
		return false, ErrMissingUserID
	}

	// Org-scoped check (optional): a matching org-wide grant satisfies the
	// requirement regardless of which tenant is currently active.
	if e.orgStore != nil {
		if orgID, ok := organization.OrgIDFromContext(ctx); ok && orgID != "" {
			orgPerms, err := e.orgStore.GetUserOrgPermissions(ctx, orgID, userID)
			if err != nil {
				return false, fmt.Errorf("check permission: load org permissions: %w", err)
			}
			for _, p := range orgPerms {
				if p.Covers(required) {
					return true, nil
				}
			}
		}
	}

	// Tenant-scoped check (existing behavior).
	tenantID, ok := tenant.TenantIDFromContext(ctx)
	if !ok || tenantID == "" {
		return false, ErrMissingTenantID
	}

	held, err := e.store.GetUserPermissions(ctx, tenantID, userID)
	if err != nil {
		return false, fmt.Errorf("check permission: load user permissions: %w", err)
	}

	for _, h := range held {
		if h.Covers(required) {
			return true, nil
		}
	}

	return false, nil
}
