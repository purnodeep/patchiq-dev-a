package v1

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/skenzeriq/patchiq/internal/server/reports"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// ReportHandler serves report REST API endpoints.
type ReportHandler struct {
	svc *reports.Service
}

// NewReportHandler creates a ReportHandler.
func NewReportHandler(svc *reports.Service) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// Generate handles POST /api/v1/reports/generate.
func (h *ReportHandler) Generate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	userID, _ := user.UserIDFromContext(ctx)

	var req reports.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid request body")
		return
	}

	if req.ReportType == "" {
		WriteFieldError(w, http.StatusBadRequest, "MISSING_FIELD", "report_type is required", "report_type")
		return
	}
	if req.Format == "" {
		WriteFieldError(w, http.StatusBadRequest, "MISSING_FIELD", "format is required", "format")
		return
	}

	if h.svc == nil {
		WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "report service is not configured")
		return
	}

	resp, err := h.svc.Generate(ctx, tenantID, userID, req)
	if err != nil {
		slog.ErrorContext(ctx, "generate report", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "GENERATION_FAILED", fmt.Sprintf("report generation failed: %v", err))
		return
	}

	WriteJSON(w, http.StatusCreated, resp)
}

// Counts handles GET /api/v1/reports/counts.
func (h *ReportHandler) Counts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	if h.svc == nil {
		WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "report service is not configured")
		return
	}

	counts, err := h.svc.GetCounts(ctx, tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "get report counts", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get report counts")
		return
	}

	WriteJSON(w, http.StatusOK, counts)
}

// List handles GET /api/v1/reports.
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	if h.svc == nil {
		WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "report service is not configured")
		return
	}

	q := r.URL.Query()
	limit := ParseLimit(q.Get("limit"))

	params := reports.ListReportGenerationsParams{
		TenantID:   tenantID,
		Status:     q.Get("status"),
		ReportType: q.Get("report_type"),
		Format:     q.Get("format"),
		Cursor:     q.Get("cursor"),
		Limit:      limit,
	}

	records, total, err := h.svc.List(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list reports", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list reports")
		return
	}

	var nextCursor string
	if len(records) == int(limit) && len(records) > 0 {
		last := records[len(records)-1]
		nextCursor = last.ID
	}

	WriteList(w, records, nextCursor, total)
}

// Get handles GET /api/v1/reports/{id}.
func (h *ReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	if h.svc == nil {
		WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "report service is not configured")
		return
	}

	record, err := h.svc.Get(ctx, tenantID, id)
	if err != nil {
		slog.ErrorContext(ctx, "get report", "tenant_id", tenantID, "id", id, "error", err)
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "report not found")
		return
	}

	WriteJSON(w, http.StatusOK, record)
}

// Download handles GET /api/v1/reports/{id}/download.
func (h *ReportHandler) Download(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	if h.svc == nil {
		WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "report service is not configured")
		return
	}

	data, contentType, filename, err := h.svc.Download(ctx, tenantID, id)
	if err != nil {
		slog.ErrorContext(ctx, "download report", "tenant_id", tenantID, "id", id, "error", err)
		WriteError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("report download failed: %v", err))
		return
	}

	// Get record for checksum header.
	record, err := h.svc.Get(ctx, tenantID, id)
	if err == nil && record.ChecksumSHA256 != "" {
		w.Header().Set("X-Report-Checksum", record.ChecksumSHA256)
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(data); err != nil {
		slog.ErrorContext(ctx, "download report: write response", "id", id, "error", err)
	}
}

// Delete handles DELETE /api/v1/reports/{id}.
func (h *ReportHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	if h.svc == nil {
		WriteError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "report service is not configured")
		return
	}

	if err := h.svc.Delete(ctx, tenantID, id); err != nil {
		slog.ErrorContext(ctx, "delete report", "tenant_id", tenantID, "id", id, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete report")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
