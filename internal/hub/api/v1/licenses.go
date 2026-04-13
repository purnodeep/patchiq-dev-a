package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// maskLicenseKey returns a masked version of the license key.
// The full key is only returned on Create so the user can copy it once.
func maskLicenseKey(key string) string {
	if len(key) <= 8 {
		return "****-****"
	}
	// Extract a short fingerprint from the key for identification.
	// Use first 4 alphanumeric chars + last 4 alphanumeric chars.
	var alphanumeric []byte
	for _, b := range []byte(key) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') {
			alphanumeric = append(alphanumeric, b)
		}
	}
	if len(alphanumeric) < 8 {
		return "****-****"
	}
	return string(alphanumeric[:4]) + "-****-" + string(alphanumeric[len(alphanumeric)-4:])
}

// LicenseQuerier abstracts the sqlcgen queries used by LicenseHandler.
type LicenseQuerier interface {
	CreateLicense(ctx context.Context, arg sqlcgen.CreateLicenseParams) (sqlcgen.License, error)
	GetLicenseByID(ctx context.Context, id pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error)
	ListLicenses(ctx context.Context, arg sqlcgen.ListLicensesParams) ([]sqlcgen.ListLicensesRow, error)
	CountLicenses(ctx context.Context, arg sqlcgen.CountLicensesParams) (int64, error)
	RevokeLicense(ctx context.Context, id pgtype.UUID) (sqlcgen.License, error)
	AssignLicenseToClient(ctx context.Context, arg sqlcgen.AssignLicenseToClientParams) (sqlcgen.License, error)
	RenewLicense(ctx context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error)
	GetLicenseUsageHistory(ctx context.Context, arg sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error)
	ListAuditEventsByResourceID(ctx context.Context, arg sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error)
	CountAuditEventsByResourceID(ctx context.Context, arg sqlcgen.CountAuditEventsByResourceIDParams) (int64, error)
}

// LicenseHandler serves license management endpoints.
type LicenseHandler struct {
	queries  LicenseQuerier
	eventBus domain.EventBus
}

// NewLicenseHandler creates a new LicenseHandler.
func NewLicenseHandler(queries LicenseQuerier, eventBus domain.EventBus) *LicenseHandler {
	return &LicenseHandler{queries: queries, eventBus: eventBus}
}

// validLicenseTiers contains the allowed license tier values.
var validLicenseTiers = map[string]bool{
	"community":    true,
	"professional": true,
	"enterprise":   true,
	"msp":          true,
}

type createLicenseRequest struct {
	CustomerName  string  `json:"customer_name"`
	CustomerEmail string  `json:"customer_email"`
	Tier          string  `json:"tier"`
	MaxEndpoints  int32   `json:"max_endpoints"`
	ExpiresAt     string  `json:"expires_at"`
	ClientID      *string `json:"client_id,omitempty"`
	Notes         string  `json:"notes"`
}

func (cr *createLicenseRequest) validate() error {
	if cr.CustomerName == "" {
		return fmt.Errorf("customer_name is required")
	}
	if cr.Tier == "" {
		return fmt.Errorf("tier is required")
	}
	if !validLicenseTiers[cr.Tier] {
		return fmt.Errorf("tier must be one of: community, professional, enterprise, msp")
	}
	if cr.MaxEndpoints <= 0 {
		return fmt.Errorf("max_endpoints must be greater than 0")
	}
	if cr.ExpiresAt == "" {
		return fmt.Errorf("expires_at is required")
	}
	return nil
}

// Create handles POST /api/v1/licenses.
func (h *LicenseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse expires_at: %s", err))
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant ID", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant ID: internal error")
		return
	}

	now := time.Now().UTC()
	// TODO(NH10): Replace placeholder license key with RSA-signed license using license.Generator.
	licenseKeyData, err := json.Marshal(map[string]any{
		"tier":          req.Tier,
		"max_endpoints": req.MaxEndpoints,
		"issued_at":     now.Format(time.RFC3339),
		"expires_at":    req.ExpiresAt,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "marshal license key", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "generate license key: internal error")
		return
	}

	params := sqlcgen.CreateLicenseParams{
		TenantID:     tenantUUID,
		LicenseKey:   string(licenseKeyData),
		Tier:         req.Tier,
		MaxEndpoints: req.MaxEndpoints,
		IssuedAt:     pgtype.Timestamptz{Time: now, Valid: true},
		ExpiresAt:    pgtype.Timestamptz{Time: expiresAt, Valid: true},
		CustomerName: req.CustomerName,
	}

	if req.CustomerEmail != "" {
		params.CustomerEmail = pgtype.Text{String: req.CustomerEmail, Valid: true}
	}
	if req.ClientID != nil {
		clientUUID, err := parseUUID(*req.ClientID)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client_id: %s", err))
			return
		}
		params.ClientID = clientUUID
	}
	if req.Notes != "" {
		params.Notes = pgtype.Text{String: req.Notes, Valid: true}
	}

	license, err := h.queries.CreateLicense(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "create license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "create license: internal error")
		return
	}

	licenseIDStr := uuidToString(license.ID)
	evt := domain.NewSystemEvent(events.LicenseIssued, tenantID, "license", licenseIDStr, "create", map[string]any{
		"tier":          req.Tier,
		"customer_name": req.CustomerName,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit license.issued event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]any{"license": license}); err != nil {
		slog.ErrorContext(r.Context(), "encode create license response", "error", err)
	}
}

// List handles GET /api/v1/licenses.
func (h *LicenseHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := queryParamInt(r, "limit", 50)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 50
	}
	offset := queryParamInt(r, "offset", 0)

	listParams := sqlcgen.ListLicensesParams{
		QueryLimit:  int32(limit),
		QueryOffset: int32(offset),
	}
	countParams := sqlcgen.CountLicensesParams{}

	if v := r.URL.Query().Get("tier"); v != "" {
		listParams.Tier = pgtype.Text{String: v, Valid: true}
		countParams.Tier = pgtype.Text{String: v, Valid: true}
	}
	if v := r.URL.Query().Get("status"); v != "" {
		listParams.StatusFilter = pgtype.Text{String: v, Valid: true}
		countParams.StatusFilter = pgtype.Text{String: v, Valid: true}
	}

	licenses, err := h.queries.ListLicenses(r.Context(), listParams)
	if err != nil {
		slog.ErrorContext(r.Context(), "list licenses", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list licenses: internal error")
		return
	}

	total, err := h.queries.CountLicenses(r.Context(), countParams)
	if err != nil {
		slog.ErrorContext(r.Context(), "count licenses", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count licenses: internal error")
		return
	}

	if licenses == nil {
		licenses = []sqlcgen.ListLicensesRow{}
	}

	// Mask license keys — only last 4 chars shown; full key returned only on Create.
	for i := range licenses {
		licenses[i].LicenseKey = maskLicenseKey(licenses[i].LicenseKey)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"licenses": licenses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode list licenses response", "error", err)
	}
}

// Get handles GET /api/v1/licenses/{id}.
func (h *LicenseHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	license, err := h.queries.GetLicenseByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found")
			return
		}
		slog.ErrorContext(r.Context(), "get license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get license: internal error")
		return
	}

	// Mask license key — only last 4 chars shown; full key returned only on Create.
	license.LicenseKey = maskLicenseKey(license.LicenseKey)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"license": license}); err != nil {
		slog.ErrorContext(r.Context(), "encode get license response", "error", err)
	}
}

// Revoke handles POST /api/v1/licenses/{id}/revoke.
func (h *LicenseHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	license, err := h.queries.RevokeLicense(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found or already revoked")
			return
		}
		slog.ErrorContext(r.Context(), "revoke license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "revoke license: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	licenseIDStr := uuidToString(license.ID)
	evt := domain.NewSystemEvent(events.LicenseRevoked, tenantID, "license", licenseIDStr, "revoke", map[string]any{
		"tier":          license.Tier,
		"customer_name": license.CustomerName,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit license.revoked event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"status": "revoked"}); err != nil {
		slog.ErrorContext(r.Context(), "encode revoke license response", "error", err)
	}
}

type renewLicenseRequest struct {
	Tier         *string `json:"tier,omitempty"`
	MaxEndpoints *int32  `json:"max_endpoints,omitempty"`
	ExpiresAt    string  `json:"expires_at"`
}

func (rr *renewLicenseRequest) validate() error {
	if rr.ExpiresAt == "" {
		return fmt.Errorf("expires_at is required")
	}
	expiresAt, err := time.Parse(time.RFC3339, rr.ExpiresAt)
	if err != nil {
		return fmt.Errorf("parse expires_at: %w", err)
	}
	if !expiresAt.After(time.Now()) {
		return fmt.Errorf("expires_at must be in the future")
	}
	if rr.Tier != nil && !validLicenseTiers[*rr.Tier] {
		return fmt.Errorf("tier must be one of: community, professional, enterprise, msp")
	}
	if rr.MaxEndpoints != nil && *rr.MaxEndpoints <= 0 {
		return fmt.Errorf("max_endpoints must be greater than 0")
	}
	return nil
}

// Renew handles PUT /api/v1/licenses/{id}/renew.
func (h *LicenseHandler) Renew(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	var req renewLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	expiresAt, _ := time.Parse(time.RFC3339, req.ExpiresAt) // already validated

	params := sqlcgen.RenewLicenseParams{
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		ID:        id,
	}
	if req.Tier != nil {
		params.NewTier = pgtype.Text{String: *req.Tier, Valid: true}
	}
	if req.MaxEndpoints != nil {
		params.NewMaxEndpoints = pgtype.Int4{Int32: *req.MaxEndpoints, Valid: true}
	}

	license, err := h.queries.RenewLicense(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found")
			return
		}
		slog.ErrorContext(r.Context(), "renew license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "renew license: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	licenseIDStr := uuidToString(license.ID)
	evt := domain.NewSystemEvent(events.LicenseRenewed, tenantID, "license", licenseIDStr, "renew", map[string]any{
		"tier":       license.Tier,
		"expires_at": license.ExpiresAt.Time.Format(time.RFC3339),
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit license.renewed event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"license": license}); err != nil {
		slog.ErrorContext(r.Context(), "encode renew license response", "error", err)
	}
}

// UsageHistory handles GET /api/v1/licenses/{id}/usage-history.
func (h *LicenseHandler) UsageHistory(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	// First fetch the license to get max_endpoints.
	license, err := h.queries.GetLicenseByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found")
			return
		}
		slog.ErrorContext(r.Context(), "get license for usage history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get license: internal error")
		return
	}

	days := queryParamInt(r, "days", 90)
	if days < 1 {
		days = 90
	}
	if days > 365 {
		days = 365
	}

	tenantID := tenant.MustTenantID(r.Context())
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant ID", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant ID: internal error")
		return
	}

	points, err := h.queries.GetLicenseUsageHistory(r.Context(), sqlcgen.GetLicenseUsageHistoryParams{
		LicenseID: id,
		TenantID:  tenantUUID,
		Days:      int32(days),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "get license usage history", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get license usage history: internal error")
		return
	}

	if points == nil {
		points = []sqlcgen.GetLicenseUsageHistoryRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"max_endpoints": license.MaxEndpoints,
		"points":        points,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode usage history response", "error", err)
	}
}

// AuditTrail handles GET /api/v1/licenses/{id}/audit-trail.
func (h *LicenseHandler) AuditTrail(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	limit := queryParamInt(r, "limit", 50)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 50
	}
	offset := queryParamInt(r, "offset", 0)

	tenantID := tenant.MustTenantID(r.Context())
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant ID", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant ID: internal error")
		return
	}

	resourceID := uuidToString(id)

	items, err := h.queries.ListAuditEventsByResourceID(r.Context(), sqlcgen.ListAuditEventsByResourceIDParams{
		TenantID:    tenantUUID,
		Resource:    "license",
		ResourceID:  resourceID,
		QueryOffset: int32(offset),
		QueryLimit:  int32(limit),
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list audit events for license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list audit trail: internal error")
		return
	}

	total, err := h.queries.CountAuditEventsByResourceID(r.Context(), sqlcgen.CountAuditEventsByResourceIDParams{
		TenantID:   tenantUUID,
		Resource:   "license",
		ResourceID: resourceID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "count audit events for license", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count audit trail: internal error")
		return
	}

	if items == nil {
		items = []sqlcgen.AuditEvent{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"items": items,
		"total": total,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode audit trail response", "error", err)
	}
}

type assignLicenseRequest struct {
	ClientID string `json:"client_id"`
}

// Assign handles POST /api/v1/licenses/{id}/assign.
func (h *LicenseHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse license id: %s", err))
		return
	}

	var req assignLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if req.ClientID == "" {
		writeJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}

	clientUUID, err := parseUUID(req.ClientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse client_id: %s", err))
		return
	}

	license, err := h.queries.AssignLicenseToClient(r.Context(), sqlcgen.AssignLicenseToClientParams{
		ID:       id,
		ClientID: clientUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "license not found")
			return
		}
		slog.ErrorContext(r.Context(), "assign license to client", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "assign license to client: internal error")
		return
	}

	licenseIDStr := uuidToString(license.ID)
	evt := domain.NewSystemEvent(events.LicenseAssigned, uuidToString(license.TenantID), "license", licenseIDStr, "assign", map[string]any{
		"client_id": req.ClientID,
	})
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit license.assigned event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"license": license}); err != nil {
		slog.ErrorContext(r.Context(), "encode assign license response", "error", err)
	}
}
