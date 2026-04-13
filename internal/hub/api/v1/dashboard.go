package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// DashboardQuerier abstracts the sqlcgen queries used by DashboardHandler.
type DashboardQuerier interface {
	GetDashboardStats(ctx context.Context) (sqlcgen.GetDashboardStatsRow, error)
	GetDashboardLicenseBreakdown(ctx context.Context) ([]sqlcgen.GetDashboardLicenseBreakdownRow, error)
	GetDashboardCatalogGrowth(ctx context.Context, days int32) ([]sqlcgen.GetDashboardCatalogGrowthRow, error)
	GetDashboardClientSummary(ctx context.Context) ([]sqlcgen.GetDashboardClientSummaryRow, error)
	ListAuditEventsByTenant(ctx context.Context, arg sqlcgen.ListAuditEventsByTenantParams) ([]sqlcgen.AuditEvent, error)
}

// DashboardHandler serves dashboard endpoints.
type DashboardHandler struct {
	queries DashboardQuerier
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(queries DashboardQuerier) *DashboardHandler {
	return &DashboardHandler{queries: queries}
}

// Stats handles GET /api/v1/dashboard/stats.
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.queries.GetDashboardStats(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "get dashboard stats", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get dashboard stats: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"total_catalog_entries": stats.TotalCatalogEntries,
		"active_feeds":          stats.ActiveFeeds,
		"connected_clients":     stats.ConnectedClients,
		"pending_clients":       stats.PendingClients,
		"active_licenses":       stats.ActiveLicenses,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode dashboard stats response", "error", err)
	}
}

// LicenseBreakdown handles GET /api/v1/dashboard/license-breakdown.
func (h *DashboardHandler) LicenseBreakdown(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.GetDashboardLicenseBreakdown(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "get license breakdown", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get license breakdown: internal error")
		return
	}

	items := make([]map[string]any, len(rows))
	for i, row := range rows {
		items[i] = map[string]any{
			"tier":            row.Tier,
			"status":          row.Status,
			"count":           row.Count,
			"total_endpoints": row.TotalEndpoints,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		slog.ErrorContext(r.Context(), "encode license breakdown response", "error", err)
	}
}

// CatalogGrowth handles GET /api/v1/dashboard/catalog-growth?days=90.
func (h *DashboardHandler) CatalogGrowth(w http.ResponseWriter, r *http.Request) {
	days := int32(queryParamInt(r, "days", 90))
	if days < 1 || days > 365 {
		days = 90
	}

	rows, err := h.queries.GetDashboardCatalogGrowth(r.Context(), days)
	if err != nil {
		slog.ErrorContext(r.Context(), "get catalog growth", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get catalog growth: internal error")
		return
	}

	items := make([]map[string]any, len(rows))
	for i, row := range rows {
		items[i] = map[string]any{
			"day":           row.Day.Time.Format("2006-01-02"),
			"entries_added": row.EntriesAdded,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		slog.ErrorContext(r.Context(), "encode catalog growth response", "error", err)
	}
}

// ClientSummary handles GET /api/v1/dashboard/clients.
func (h *DashboardHandler) ClientSummary(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.GetDashboardClientSummary(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "get client summary", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get client summary: internal error")
		return
	}

	items := make([]map[string]any, len(rows))
	for i, row := range rows {
		item := map[string]any{
			"id":             uuidToString(row.ID),
			"hostname":       row.Hostname,
			"status":         row.Status,
			"endpoint_count": row.EndpointCount,
			"last_sync_at":   nil,
			"version":        nil,
			"os":             nil,
		}
		if row.LastSyncAt.Valid {
			s := row.LastSyncAt.Time.Format("2006-01-02T15:04:05Z07:00")
			item["last_sync_at"] = s
		}
		if row.Version.Valid {
			item["version"] = row.Version.String
		}
		if row.Os.Valid {
			item["os"] = row.Os.String
		}
		items[i] = item
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		slog.ErrorContext(r.Context(), "encode client summary response", "error", err)
	}
}

// Activity handles GET /api/v1/dashboard/activity.
// Returns the 15 most recent audit events for the current tenant.
func (h *DashboardHandler) Activity(w http.ResponseWriter, r *http.Request) {
	tenantID := tenant.MustTenantID(r.Context())
	tenantUUID, err := parseUUID(tenantID)
	if err != nil {
		slog.ErrorContext(r.Context(), "parse tenant ID", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "parse tenant ID: internal error")
		return
	}

	events, err := h.queries.ListAuditEventsByTenant(r.Context(), sqlcgen.ListAuditEventsByTenantParams{
		TenantID: tenantUUID,
		Limit:    15,
		Offset:   0,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list dashboard activity", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list dashboard activity: internal error")
		return
	}

	items := make([]map[string]any, len(events))
	for i, evt := range events {
		payload := json.RawMessage(evt.Payload)
		if len(payload) == 0 {
			payload = json.RawMessage("{}")
		}
		metadata := json.RawMessage(evt.Metadata)
		if len(metadata) == 0 {
			metadata = json.RawMessage("{}")
		}
		item := map[string]any{
			"id":          evt.ID,
			"type":        evt.Type,
			"actor_id":    evt.ActorID,
			"actor_type":  evt.ActorType,
			"resource":    evt.Resource,
			"resource_id": evt.ResourceID,
			"action":      evt.Action,
			"payload":     payload,
			"metadata":    metadata,
		}
		if evt.Timestamp != (pgtype.Timestamptz{}) && evt.Timestamp.Valid {
			item["timestamp"] = evt.Timestamp.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		items[i] = item
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(items); err != nil {
		slog.ErrorContext(r.Context(), "encode dashboard activity response", "error", err)
	}
}
