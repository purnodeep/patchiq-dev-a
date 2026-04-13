package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/organization"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// OrganizationHandler serves the /api/v1/organizations endpoints.
//
// The handler takes a direct *store.Store dependency because org-level
// operations (cross-tenant aggregation, user-accessible tenant resolution)
// are defined on the store rather than on sqlcgen.Queries. Coverage lives
// in integration tests that run against a real database.
type OrganizationHandler struct {
	s        *store.Store
	eventBus domain.EventBus
}

// NewOrganizationHandler creates an OrganizationHandler. Panics on nil args.
func NewOrganizationHandler(s *store.Store, eventBus domain.EventBus) *OrganizationHandler {
	if s == nil {
		panic("organizations: NewOrganizationHandler called with nil store")
	}
	if eventBus == nil {
		panic("organizations: NewOrganizationHandler called with nil eventBus")
	}
	return &OrganizationHandler{s: s, eventBus: eventBus}
}

// OrganizationDTO is the JSON response shape for an organization.
type OrganizationDTO struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Slug             string  `json:"slug"`
	Type             string  `json:"type"`
	ParentOrgID      *string `json:"parent_org_id"`
	ZitadelOrgID     *string `json:"zitadel_org_id"`
	LicenseID        *string `json:"license_id"`
	PlatformTenantID *string `json:"platform_tenant_id"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func orgToDTO(o sqlcgen.Organization) OrganizationDTO {
	var parentOrg, licenseID, platformTenantID *string
	if o.ParentOrgID.Valid {
		s := uuid.UUID(o.ParentOrgID.Bytes).String()
		parentOrg = &s
	}
	if o.LicenseID.Valid {
		s := uuid.UUID(o.LicenseID.Bytes).String()
		licenseID = &s
	}
	if o.PlatformTenantID.Valid {
		s := uuid.UUID(o.PlatformTenantID.Bytes).String()
		platformTenantID = &s
	}
	return OrganizationDTO{
		ID:               uuid.UUID(o.ID.Bytes).String(),
		Name:             o.Name,
		Slug:             o.Slug,
		Type:             o.Type,
		ParentOrgID:      parentOrg,
		ZitadelOrgID:     nullableText(o.ZitadelOrgID),
		LicenseID:        licenseID,
		PlatformTenantID: platformTenantID,
		CreatedAt:        o.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        o.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// List handles GET /api/v1/organizations.
func (h *OrganizationHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit := int32(ParseLimit(r.URL.Query().Get("limit")))
	if limit <= 0 {
		limit = 50
	}

	q := sqlcgen.New(h.s.Pool())
	orgs, err := q.ListOrganizations(ctx, sqlcgen.ListOrganizationsParams{Limit: limit, Offset: 0})
	if err != nil {
		slog.ErrorContext(ctx, "list organizations", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list organizations")
		return
	}
	total, err := q.CountOrganizations(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "count organizations", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count organizations")
		return
	}

	data := make([]OrganizationDTO, len(orgs))
	for i, o := range orgs {
		data[i] = orgToDTO(o)
	}
	WriteList(w, data, "", total)
}

// Get handles GET /api/v1/organizations/{id}.
func (h *OrganizationHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}
	q := sqlcgen.New(h.s.Pool())
	org, err := q.GetOrganizationByID(ctx, id)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		slog.ErrorContext(ctx, "get organization", "id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get organization")
		return
	}
	WriteJSON(w, http.StatusOK, orgToDTO(org))
}

// CreateOrganizationRequest is the POST body for organization creation.
type CreateOrganizationRequest struct {
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Type         string  `json:"type"`
	ParentOrgID  *string `json:"parent_org_id,omitempty"`
	ZitadelOrgID *string `json:"zitadel_org_id,omitempty"`
}

// Create handles POST /api/v1/organizations.
func (h *OrganizationHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateOrganizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Slug == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_FIELDS", "name and slug are required")
		return
	}
	if req.Type == "" {
		req.Type = "direct"
	}
	if req.Type != "direct" && req.Type != "msp" && req.Type != "reseller" {
		WriteError(w, http.StatusBadRequest, "INVALID_TYPE", "type must be one of: direct, msp, reseller")
		return
	}

	params := sqlcgen.CreateOrganizationParams{
		Name: req.Name,
		Slug: req.Slug,
		Type: req.Type,
	}
	if req.ParentOrgID != nil && *req.ParentOrgID != "" {
		parent, err := scanUUID(*req.ParentOrgID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PARENT", "invalid parent_org_id")
			return
		}
		params.ParentOrgID = parent
	}
	if req.ZitadelOrgID != nil && *req.ZitadelOrgID != "" {
		params.ZitadelOrgID = pgtype.Text{String: *req.ZitadelOrgID, Valid: true}
	}

	q := sqlcgen.New(h.s.Pool())
	org, err := q.CreateOrganization(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "SLUG_TAKEN", "slug or zitadel_org_id already in use")
			return
		}
		slog.ErrorContext(ctx, "create organization", "error", err, "slug", req.Slug)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create organization")
		return
	}

	// MSP and reseller orgs need a hidden "platform tenant" to host their
	// org-scoped role definitions (MSP Admin, MSP Technician, MSP Auditor).
	// Direct-type orgs do not — their single working tenant is enough.
	// If platform tenant bootstrap fails we roll back the org create so the
	// caller doesn't see a half-provisioned org.
	if req.Type == "msp" || req.Type == "reseller" {
		orgIDStr := uuid.UUID(org.ID.Bytes).String()
		if _, err := h.s.BootstrapPlatformTenant(ctx, orgIDStr, org.Name); err != nil {
			slog.ErrorContext(ctx, "bootstrap platform tenant", "org_id", orgIDStr, "error", err)
			// Compensating delete must survive client cancellation, otherwise a
			// disconnect during bootstrap leaves the org row orphaned with no
			// platform_tenant_id. WithoutCancel preserves trace context but
			// detaches the deadline; bound it explicitly so a hung pool can't
			// stall the response.
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			rollbackFailed := false
			if _, delErr := q.DeleteOrganization(cleanupCtx, org.ID); delErr != nil {
				rollbackFailed = true
				slog.ErrorContext(ctx, "rollback org after platform tenant failure",
					"org_id", orgIDStr, "delete_error", delErr, "original_error", err)
			}
			cancel()
			if isUniqueViolation(err) {
				WriteError(w, http.StatusConflict, "PLATFORM_SLUG_TAKEN",
					"derived platform tenant slug already in use; pick a different organization slug")
				return
			}
			if rollbackFailed {
				WriteError(w, http.StatusInternalServerError, "BOOTSTRAP_ORPHANED",
					"platform tenant bootstrap failed and the org row could not be cleaned up; contact support")
				return
			}
			WriteError(w, http.StatusInternalServerError, "BOOTSTRAP_FAILED",
				"failed to provision platform tenant for organization")
			return
		}
		// Reload so the response carries the freshly-linked platform_tenant_id.
		// A reload failure here is unexpected (the row must exist — we just
		// committed it inside the bypass-pool transaction), so surface it
		// instead of silently returning the pre-bootstrap snapshot.
		reloaded, reloadErr := q.GetOrganizationByID(ctx, org.ID)
		if reloadErr != nil {
			slog.ErrorContext(ctx, "reload org after bootstrap", "org_id", orgIDStr, "error", reloadErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"failed to reload organization after platform tenant bootstrap")
			return
		}
		org = reloaded
	}

	emitEvent(ctx, h.eventBus, "organization.created", "organization",
		uuid.UUID(org.ID.Bytes).String(), "", orgToDTO(org))
	WriteJSON(w, http.StatusCreated, orgToDTO(org))
}

// Delete handles DELETE /api/v1/organizations/{id}.
func (h *OrganizationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}
	q := sqlcgen.New(h.s.Pool())
	rows, err := q.DeleteOrganization(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "delete organization", "id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete organization")
		return
	}
	if rows == 0 {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
		return
	}
	emitEvent(ctx, h.eventBus, "organization.deleted", "organization", chi.URLParam(r, "id"), "", nil)
	w.WriteHeader(http.StatusNoContent)
}

// TenantSummaryDTO is the response shape for tenants listed under an organization.
type TenantSummaryDTO struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
}

func tenantToSummary(t sqlcgen.Tenant) TenantSummaryDTO {
	return TenantSummaryDTO{
		ID:        uuid.UUID(t.ID.Bytes).String(),
		Name:      t.Name,
		Slug:      t.Slug,
		CreatedAt: t.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// ListTenants handles GET /api/v1/organizations/{id}/tenants.
func (h *OrganizationHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}
	// Use the client-tenant query so we hide the org's platform tenant from
	// MSP operators (it exists only to host org-scoped role definitions).
	q := sqlcgen.New(h.s.Pool())
	tenants, err := q.ListClientTenantsByOrganization(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "list client tenants by org", "id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list tenants")
		return
	}
	out := make([]TenantSummaryDTO, len(tenants))
	for i, t := range tenants {
		out[i] = tenantToSummary(t)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": out})
}

// ProvisionTenantRequest is the POST body for tenant provisioning.
type ProvisionTenantRequest struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	LicenseID string `json:"license_id,omitempty"`
}

// ProvisionTenant handles POST /api/v1/organizations/{id}/tenants.
// Creates a new child tenant belonging to the organization.
func (h *OrganizationHandler) ProvisionTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}

	var req ProvisionTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	if req.Name == "" || req.Slug == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_FIELDS", "name and slug are required")
		return
	}

	q := sqlcgen.New(h.s.Pool())
	created, err := q.CreateTenantInOrganization(ctx, sqlcgen.CreateTenantInOrganizationParams{
		Name:           req.Name,
		Slug:           req.Slug,
		LicenseID:      textFromString(req.LicenseID),
		OrganizationID: orgID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "SLUG_TAKEN", "tenant slug already in use")
			return
		}
		slog.ErrorContext(ctx, "provision tenant", "org_id", chi.URLParam(r, "id"), "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to provision tenant")
		return
	}

	emitEvent(ctx, h.eventBus, "tenant.provisioned_under_org", "tenant",
		uuid.UUID(created.ID.Bytes).String(), uuid.UUID(created.ID.Bytes).String(),
		map[string]string{"organization_id": chi.URLParam(r, "id")})
	WriteJSON(w, http.StatusCreated, tenantToSummary(created))
}

// AssignOrgRoleRequest is the POST body for assigning an org-scoped role.
type AssignOrgRoleRequest struct {
	RoleID string `json:"role_id"`
}

// AssignUserRole handles POST /api/v1/organizations/{id}/users/{user_id}/roles.
func (h *OrganizationHandler) AssignUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_USER", "user_id is required")
		return
	}

	var req AssignOrgRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}
	roleID, err := scanUUID(req.RoleID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "invalid role_id")
		return
	}

	q := sqlcgen.New(h.s.Pool())
	if err := q.AssignOrgUserRole(ctx, sqlcgen.AssignOrgUserRoleParams{
		OrganizationID: orgID,
		UserID:         userID,
		RoleID:         roleID,
	}); err != nil {
		slog.ErrorContext(ctx, "assign org user role",
			"org_id", chi.URLParam(r, "id"), "user_id", userID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign role")
		return
	}

	emitEvent(ctx, h.eventBus, "org_user_role.assigned", "org_user_role",
		userID, "",
		map[string]string{"organization_id": chi.URLParam(r, "id"), "user_id": userID, "role_id": req.RoleID})
	w.WriteHeader(http.StatusNoContent)
}

// RevokeUserRole handles DELETE /api/v1/organizations/{id}/users/{user_id}/roles/{role_id}.
func (h *OrganizationHandler) RevokeUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_USER", "user_id is required")
		return
	}
	roleID, err := scanUUID(chi.URLParam(r, "role_id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "invalid role_id")
		return
	}

	q := sqlcgen.New(h.s.Pool())
	rows, err := q.RevokeOrgUserRole(ctx, sqlcgen.RevokeOrgUserRoleParams{
		OrganizationID: orgID,
		UserID:         userID,
		RoleID:         roleID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "revoke org user role", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to revoke role")
		return
	}
	if rows == 0 {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "grant not found")
		return
	}
	emitEvent(ctx, h.eventBus, "org_user_role.revoked", "org_user_role",
		userID, "",
		map[string]string{
			"organization_id": chi.URLParam(r, "id"),
			"user_id":         userID,
			"role_id":         chi.URLParam(r, "role_id"),
		})
	w.WriteHeader(http.StatusNoContent)
}

// OrgDashboardTenantRow is a per-tenant summary row in the org dashboard.
type OrgDashboardTenantRow struct {
	TenantID      string `json:"tenant_id"`
	TenantName    string `json:"tenant_name"`
	EndpointCount int64  `json:"endpoint_count"`
}

// OrgDashboardDTO is the response shape for GET /api/v1/organizations/{id}/dashboard.
type OrgDashboardDTO struct {
	OrganizationID string                  `json:"organization_id"`
	TotalTenants   int                     `json:"total_tenants"`
	TotalEndpoints int64                   `json:"total_endpoints"`
	Tenants        []OrgDashboardTenantRow `json:"tenants"`
}

// Dashboard handles GET /api/v1/organizations/{id}/dashboard.
// Aggregates endpoint counts across every tenant the caller can access in
// the given organization. Authorization is enforced by UserAccessibleTenants
// (which honors org-scoped and tenant-scoped grants), so rows returned here
// are guaranteed to be visible to the caller.
func (h *OrganizationHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgIDStr := chi.URLParam(r, "id")
	if _, err := uuid.Parse(orgIDStr); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid organization ID")
		return
	}

	// Authorization: the session must belong to this organization (if any
	// org is set in context at all). Users without an org context (legacy
	// single-tenant deployments) are allowed through — the tenant-level
	// RBAC check gate is still applied by RequirePermission middleware.
	if sessionOrgID, ok := organization.OrgIDFromContext(ctx); ok && sessionOrgID != orgIDStr {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "organization scope mismatch")
		return
	}

	userID, ok := user.UserIDFromContext(ctx)
	if !ok || userID == "" {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}

	tenants, err := h.s.UserAccessibleTenants(ctx, orgIDStr, userID)
	if err != nil {
		slog.ErrorContext(ctx, "dashboard: list accessible tenants", "org_id", orgIDStr, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load tenants")
		return
	}

	out := OrgDashboardDTO{
		OrganizationID: orgIDStr,
		Tenants:        make([]OrgDashboardTenantRow, 0, len(tenants)),
	}

	// Count endpoints per tenant using the bypass pool so we don't need to
	// set app.current_tenant_id for each call. This is safe because
	// UserAccessibleTenants has already authorized the caller's view of
	// every tenant in the result set.
	for _, t := range tenants {
		var count int64
		row := h.s.BypassPool().QueryRow(ctx,
			"SELECT count(*) FROM endpoints WHERE tenant_id = $1",
			t.ID,
		)
		if err := row.Scan(&count); err != nil {
			slog.ErrorContext(ctx, "dashboard: count endpoints",
				"tenant_id", uuid.UUID(t.ID.Bytes).String(), "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to aggregate dashboard")
			return
		}
		out.Tenants = append(out.Tenants, OrgDashboardTenantRow{
			TenantID:      uuid.UUID(t.ID.Bytes).String(),
			TenantName:    t.Name,
			EndpointCount: count,
		})
		out.TotalEndpoints += count
	}
	out.TotalTenants = len(out.Tenants)

	WriteJSON(w, http.StatusOK, out)
}
