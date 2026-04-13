package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// GeneralSettingsQuerier is the DB interface required by GeneralSettingsHandler.
type GeneralSettingsQuerier interface {
	GetGeneralSettings(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetGeneralSettingsRow, error)
	UpdateGeneralSettings(ctx context.Context, arg sqlcgen.UpdateGeneralSettingsParams) (sqlcgen.UpdateGeneralSettingsRow, error)
}

// GeneralSettingsHandler handles GET/PUT /api/v1/settings.
type GeneralSettingsHandler struct {
	q        GeneralSettingsQuerier
	eventBus domain.EventBus
}

// NewGeneralSettingsHandler creates a new GeneralSettingsHandler.
func NewGeneralSettingsHandler(q GeneralSettingsQuerier, eventBus domain.EventBus) *GeneralSettingsHandler {
	if q == nil {
		panic("settings_general: NewGeneralSettingsHandler called with nil querier")
	}
	if eventBus == nil {
		panic("settings_general: NewGeneralSettingsHandler called with nil eventBus")
	}
	return &GeneralSettingsHandler{q: q, eventBus: eventBus}
}

// generalSettingsResponse is the API response shape for general settings.
type generalSettingsResponse struct {
	OrgName           string `json:"org_name"`
	Timezone          string `json:"timezone"`
	DateFormat        string `json:"date_format"`
	ScanIntervalHours int32  `json:"scan_interval_hours"`
}

var validDateFormats = map[string]bool{
	"YYYY-MM-DD":  true,
	"MM/DD/YYYY":  true,
	"DD/MM/YYYY":  true,
	"DD MMM YYYY": true,
}

var validScanIntervals = map[int32]bool{
	1: true, 2: true, 4: true, 6: true, 12: true, 24: true,
}

// Get returns the current general settings for the tenant.
// If no row exists, it returns safe defaults.
func (h *GeneralSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())
	slog.InfoContext(r.Context(), "general settings get", "tenant_id", tid)

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "general settings get: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.GetGeneralSettings(r.Context(), pgTID)
	if err != nil {
		if isNotFound(err) {
			WriteJSON(w, http.StatusOK, generalSettingsResponse{
				OrgName:           "",
				Timezone:          "UTC",
				DateFormat:        "YYYY-MM-DD",
				ScanIntervalHours: 24,
			})
			return
		}
		slog.ErrorContext(r.Context(), "general settings get: query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve general settings")
		return
	}

	WriteJSON(w, http.StatusOK, generalSettingsResponse{
		OrgName:           row.OrgName,
		Timezone:          row.Timezone,
		DateFormat:        row.DateFormat,
		ScanIntervalHours: row.ScanIntervalHours,
	})
}

// Update validates and persists general settings for the tenant.
func (h *GeneralSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())

	var req generalSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	// Validate org_name
	req.OrgName = strings.TrimSpace(req.OrgName)
	if req.OrgName == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_ORG_NAME", "org_name must not be empty")
		return
	}
	if len(req.OrgName) > 255 {
		WriteError(w, http.StatusBadRequest, "INVALID_ORG_NAME", "org_name must not exceed 255 characters")
		return
	}

	// Validate timezone
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_TIMEZONE", "timezone must be a valid IANA timezone")
		return
	}

	// Validate date_format
	if !validDateFormats[req.DateFormat] {
		WriteError(w, http.StatusBadRequest, "INVALID_DATE_FORMAT", "date_format must be one of: YYYY-MM-DD, MM/DD/YYYY, DD/MM/YYYY, DD MMM YYYY")
		return
	}

	// Validate scan_interval_hours
	if !validScanIntervals[req.ScanIntervalHours] {
		WriteError(w, http.StatusBadRequest, "INVALID_SCAN_INTERVAL", "scan_interval_hours must be one of: 1, 2, 4, 6, 12, 24")
		return
	}

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "general settings update: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.UpdateGeneralSettings(r.Context(), sqlcgen.UpdateGeneralSettingsParams{
		TenantID:          pgTID,
		OrgName:           req.OrgName,
		Timezone:          req.Timezone,
		DateFormat:        req.DateFormat,
		ScanIntervalHours: req.ScanIntervalHours,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "general settings update: query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update general settings")
		return
	}

	emitEvent(r.Context(), h.eventBus, events.SettingsGeneralUpdated, "settings", tid, tid, row)

	WriteJSON(w, http.StatusOK, generalSettingsResponse{
		OrgName:           row.OrgName,
		Timezone:          row.Timezone,
		DateFormat:        row.DateFormat,
		ScanIntervalHours: row.ScanIntervalHours,
	})
}
