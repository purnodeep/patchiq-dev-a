package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// CatalogQuerier abstracts the sqlcgen queries used by CatalogHandler.
type CatalogQuerier interface {
	CreateCatalogEntry(ctx context.Context, arg sqlcgen.CreateCatalogEntryParams) (sqlcgen.PatchCatalog, error)
	GetCatalogEntryByID(ctx context.Context, id pgtype.UUID) (sqlcgen.PatchCatalog, error)
	ListCatalogEntries(ctx context.Context, arg sqlcgen.ListCatalogEntriesParams) ([]sqlcgen.PatchCatalog, error)
	CountCatalogEntries(ctx context.Context, arg sqlcgen.CountCatalogEntriesParams) (int64, error)
	UpdateCatalogEntry(ctx context.Context, arg sqlcgen.UpdateCatalogEntryParams) (sqlcgen.PatchCatalog, error)
	SoftDeleteCatalogEntry(ctx context.Context, id pgtype.UUID) error
	LinkCatalogCVE(ctx context.Context, arg sqlcgen.LinkCatalogCVEParams) error
	UnlinkAllCatalogCVEs(ctx context.Context, catalogID pgtype.UUID) error
	ListCVEsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) ([]sqlcgen.ListCVEsForCatalogEntryRow, error)
	CountCVEsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) (int64, error)
	GetCatalogStats(ctx context.Context) (sqlcgen.GetCatalogStatsRow, error)
	CountApprovedClients(ctx context.Context) (int64, error)
	ListCatalogEntriesEnriched(ctx context.Context, arg sqlcgen.ListCatalogEntriesEnrichedParams) ([]sqlcgen.ListCatalogEntriesEnrichedRow, error)
	CountCatalogEntriesEnriched(ctx context.Context, arg sqlcgen.CountCatalogEntriesEnrichedParams) (int64, error)
	CountSyncedClientsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) (int64, error)
	ListSyncsForCatalogEntry(ctx context.Context, catalogID pgtype.UUID) ([]sqlcgen.ListSyncsForCatalogEntryRow, error)
	ListApprovedClientsBasic(ctx context.Context) ([]sqlcgen.ListApprovedClientsBasicRow, error)
	GetFeedSourceByID(ctx context.Context, id pgtype.UUID) (sqlcgen.FeedSource, error)
}

// CatalogHandler serves catalog CRUD endpoints.
type CatalogHandler struct {
	queries  CatalogQuerier
	eventBus domain.EventBus
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(queries CatalogQuerier, eventBus domain.EventBus) *CatalogHandler {
	return &CatalogHandler{queries: queries, eventBus: eventBus}
}

type catalogRequest struct {
	Name        string   `json:"name"`
	Vendor      string   `json:"vendor"`
	OsFamily    string   `json:"os_family"`
	Version     string   `json:"version"`
	Severity    string   `json:"severity"`
	ReleaseDate *string  `json:"release_date,omitempty"`
	Description *string  `json:"description,omitempty"`
	CVEIDs      []string `json:"cve_ids,omitempty"`
}

func (cr *catalogRequest) validate() error {
	if cr.Name == "" {
		return fmt.Errorf("name is required")
	}
	if cr.Vendor == "" {
		return fmt.Errorf("vendor is required")
	}
	if cr.OsFamily == "" {
		return fmt.Errorf("os_family is required")
	}
	if cr.Version == "" {
		return fmt.Errorf("version is required")
	}
	if cr.Severity == "" {
		return fmt.Errorf("severity is required")
	}
	return nil
}

// Stats handles GET /api/v1/catalog/stats.
func (h *CatalogHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.queries.GetCatalogStats(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "get catalog stats", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get catalog stats: internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"total_entries":      stats.TotalEntries,
		"new_this_week":      stats.NewThisWeek,
		"cves_tracked":       stats.CvesTracked,
		"synced_entries":     stats.SyncedEntries,
		"total_for_sync_pct": stats.TotalForSyncPct,
		"by_severity": map[string]any{
			"critical": stats.CriticalCount,
			"high":     stats.HighCount,
			"medium":   stats.MediumCount,
			"low":      stats.LowCount,
		},
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode catalog stats response", "error", err)
	}
}

// Create handles POST /api/v1/catalog.
func (h *CatalogHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req catalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := sqlcgen.CreateCatalogEntryParams{
		Name:     req.Name,
		Vendor:   req.Vendor,
		OsFamily: req.OsFamily,
		Version:  req.Version,
		Severity: req.Severity,
	}

	if req.ReleaseDate != nil {
		t, err := time.Parse(time.RFC3339, *req.ReleaseDate)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse release_date: %s", err))
			return
		}
		params.ReleaseDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	entry, err := h.queries.CreateCatalogEntry(r.Context(), params)
	if err != nil {
		slog.ErrorContext(r.Context(), "create catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "create catalog entry: internal error")
		return
	}

	var cveWarnings []string
	for _, cveIDStr := range req.CVEIDs {
		cveID, err := parseUUID(cveIDStr)
		if err != nil {
			slog.ErrorContext(r.Context(), "parse cve_id for linking", "cve_id", cveIDStr, "error", err)
			cveWarnings = append(cveWarnings, fmt.Sprintf("invalid CVE ID %q", cveIDStr))
			continue
		}
		if err := h.queries.LinkCatalogCVE(r.Context(), sqlcgen.LinkCatalogCVEParams{
			CatalogID: entry.ID,
			CveID:     cveID,
		}); err != nil {
			slog.ErrorContext(r.Context(), "link catalog CVE", "catalog_id", entry.ID, "cve_id", cveIDStr, "error", err)
			cveWarnings = append(cveWarnings, fmt.Sprintf("failed to link CVE %q", cveIDStr))
		}
	}

	tenantID := tenant.MustTenantID(r.Context())
	entryIDStr := uuidToString(entry.ID)
	evt := domain.NewSystemEvent(events.CatalogCreated, tenantID, "patch_catalog", entryIDStr, "create", entry)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit catalog.created event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	resp := map[string]any{"entry": entry}
	if len(cveWarnings) > 0 {
		resp["warnings"] = cveWarnings
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "encode create catalog response", "error", err)
	}
}

// List handles GET /api/v1/catalog.
func (h *CatalogHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := queryParamInt(r, "limit", 50)
	limit = min(limit, 100)
	if limit < 1 {
		limit = 50
	}
	offset := queryParamInt(r, "offset", 0)

	enrichedParams := sqlcgen.ListCatalogEntriesEnrichedParams{
		QueryLimit:  int32(limit),
		QueryOffset: int32(offset),
	}
	countParams := sqlcgen.CountCatalogEntriesEnrichedParams{}

	if v := r.URL.Query().Get("os_family"); v != "" {
		enrichedParams.OsFamily = pgtype.Text{String: v, Valid: true}
		countParams.OsFamily = pgtype.Text{String: v, Valid: true}
	}
	if v := r.URL.Query().Get("severity"); v != "" {
		enrichedParams.Severity = pgtype.Text{String: v, Valid: true}
		countParams.Severity = pgtype.Text{String: v, Valid: true}
	}
	if v := r.URL.Query().Get("search"); v != "" {
		enrichedParams.Search = pgtype.Text{String: v, Valid: true}
		countParams.Search = pgtype.Text{String: v, Valid: true}
	}
	if v := r.URL.Query().Get("feed_source_id"); v != "" {
		id, err := parseUUID(v)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse feed_source_id: %s", err))
			return
		}
		enrichedParams.FeedSourceID = id
		countParams.FeedSourceID = id
	}
	if v := r.URL.Query().Get("date_range"); v != "" {
		switch v {
		case "7d", "30d", "90d":
			enrichedParams.DateRange = pgtype.Text{String: v, Valid: true}
			countParams.DateRange = pgtype.Text{String: v, Valid: true}
		default:
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid date_range %q: must be 7d, 30d, or 90d", v))
			return
		}
	}
	if v := r.URL.Query().Get("entry_type"); v != "" {
		switch v {
		case "cve", "patch":
			enrichedParams.EntryType = pgtype.Text{String: v, Valid: true}
			countParams.EntryType = pgtype.Text{String: v, Valid: true}
		default:
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid entry_type %q: must be cve or patch", v))
			return
		}
	}

	entries, err := h.queries.ListCatalogEntriesEnriched(r.Context(), enrichedParams)
	if err != nil {
		slog.ErrorContext(r.Context(), "list catalog entries enriched", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list catalog entries: internal error")
		return
	}

	total, err := h.queries.CountCatalogEntriesEnriched(r.Context(), countParams)
	if err != nil {
		slog.ErrorContext(r.Context(), "count catalog entries enriched", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count catalog entries: internal error")
		return
	}

	totalClients, err := h.queries.CountApprovedClients(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "count approved clients", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count approved clients: internal error")
		return
	}

	type entryResponse struct {
		ID             pgtype.UUID        `json:"id"`
		Name           string             `json:"name"`
		Vendor         string             `json:"vendor"`
		OsFamily       string             `json:"os_family"`
		Version        string             `json:"version"`
		Severity       string             `json:"severity"`
		ReleaseDate    pgtype.Timestamptz `json:"release_date"`
		Description    pgtype.Text        `json:"description"`
		CreatedAt      pgtype.Timestamptz `json:"created_at"`
		UpdatedAt      pgtype.Timestamptz `json:"updated_at"`
		FeedSourceID   pgtype.UUID        `json:"feed_source_id"`
		SourceUrl      string             `json:"source_url"`
		InstallerType  string             `json:"installer_type"`
		BinaryRef      string             `json:"binary_ref"`
		ChecksumSha256 string             `json:"checksum_sha256"`
		FeedSourceName pgtype.Text        `json:"feed_source_name"`
		CveCount       int64              `json:"cve_count"`
		SyncedCount    int64              `json:"synced_count"`
		TotalClients   int64              `json:"total_clients"`
	}

	result := make([]entryResponse, len(entries))
	for i, e := range entries {
		result[i] = entryResponse{
			ID:             e.ID,
			Name:           e.Name,
			Vendor:         e.Vendor,
			OsFamily:       e.OsFamily,
			Version:        e.Version,
			Severity:       e.Severity,
			ReleaseDate:    e.ReleaseDate,
			Description:    e.Description,
			CreatedAt:      e.CreatedAt,
			UpdatedAt:      e.UpdatedAt,
			FeedSourceID:   e.FeedSourceID,
			SourceUrl:      e.SourceUrl,
			InstallerType:  e.InstallerType,
			BinaryRef:      e.BinaryRef,
			ChecksumSha256: e.ChecksumSha256,
			FeedSourceName: e.FeedSourceName,
			CveCount:       e.CveCount,
			SyncedCount:    e.SyncedCount,
			TotalClients:   totalClients,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"entries": result,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode list catalog response", "error", err)
	}
}

// Get handles GET /api/v1/catalog/{id}.
func (h *CatalogHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse catalog id: %s", err))
		return
	}

	entry, err := h.queries.GetCatalogEntryByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "catalog entry not found")
			return
		}
		slog.ErrorContext(r.Context(), "get catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get catalog entry: internal error")
		return
	}

	cves, err := h.queries.ListCVEsForCatalogEntry(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "list CVEs for catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list CVEs for catalog entry: internal error")
		return
	}
	if cves == nil {
		cves = []sqlcgen.ListCVEsForCatalogEntryRow{}
	}

	syncedCount, err := h.queries.CountSyncedClientsForCatalogEntry(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "count synced clients for catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count synced clients: internal error")
		return
	}

	totalClients, err := h.queries.CountApprovedClients(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "count approved clients", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "count approved clients: internal error")
		return
	}

	rawSyncs, err := h.queries.ListSyncsForCatalogEntry(r.Context(), id)
	if err != nil {
		slog.ErrorContext(r.Context(), "list syncs for catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list syncs for catalog entry: internal error")
		return
	}

	// Get client info separately (clients table has RLS).
	clients, err := h.queries.ListApprovedClientsBasic(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "list approved clients basic", "error", err)
		// Non-fatal: proceed with empty client names.
		clients = nil
	}

	// Build client lookup map.
	clientMap := make(map[string]sqlcgen.ListApprovedClientsBasicRow)
	for _, c := range clients {
		clientMap[uuidToString(c.ID)] = c
	}

	// Merge sync records with client info.
	type syncResponse struct {
		ID            string     `json:"id"`
		CatalogID     string     `json:"catalog_id"`
		ClientID      string     `json:"client_id"`
		ClientName    string     `json:"client_name"`
		EndpointCount int32      `json:"endpoint_count"`
		Status        string     `json:"status"`
		SyncedAt      *time.Time `json:"synced_at"`
		CreatedAt     time.Time  `json:"created_at"`
	}

	syncs := make([]syncResponse, 0, len(rawSyncs)+len(clients))
	syncedClientIDs := make(map[string]bool)
	for _, s := range rawSyncs {
		sr := syncResponse{
			ID:        uuidToString(s.ID),
			CatalogID: uuidToString(s.CatalogID),
			ClientID:  uuidToString(s.ClientID),
			Status:    s.Status,
			CreatedAt: s.CreatedAt.Time,
		}
		if s.SyncedAt.Valid {
			t := s.SyncedAt.Time.UTC()
			sr.SyncedAt = &t
		}
		if c, ok := clientMap[sr.ClientID]; ok {
			sr.ClientName = c.Hostname
			sr.EndpointCount = c.EndpointCount
		}
		syncs = append(syncs, sr)
		syncedClientIDs[sr.ClientID] = true
	}
	// Add "not_pushed" entries for approved clients without a sync record.
	for _, c := range clients {
		cid := uuidToString(c.ID)
		if !syncedClientIDs[cid] {
			syncs = append(syncs, syncResponse{
				ID:            cid,
				CatalogID:     uuidToString(id),
				ClientID:      cid,
				ClientName:    c.Hostname,
				EndpointCount: c.EndpointCount,
				Status:        "not_pushed",
				CreatedAt:     time.Now(),
			})
		}
	}

	var feedSourceID *string
	var feedSourceName, feedSourceDisplayName *string
	if entry.FeedSourceID.Valid {
		s := uuidToString(entry.FeedSourceID)
		feedSourceID = &s
		fs, err := h.queries.GetFeedSourceByID(r.Context(), entry.FeedSourceID)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				slog.ErrorContext(r.Context(), "get feed source by id", "feed_source_id", s, "error", err)
			}
		} else {
			feedSourceName = &fs.Name
			feedSourceDisplayName = &fs.DisplayName
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":                       entry.ID,
		"name":                     entry.Name,
		"vendor":                   entry.Vendor,
		"os_family":                entry.OsFamily,
		"version":                  entry.Version,
		"severity":                 entry.Severity,
		"release_date":             entry.ReleaseDate,
		"description":              entry.Description,
		"created_at":               entry.CreatedAt,
		"updated_at":               entry.UpdatedAt,
		"feed_source_id":           feedSourceID,
		"feed_source_name":         feedSourceName,
		"feed_source_display_name": feedSourceDisplayName,
		"source_url":               entry.SourceUrl,
		"installer_type":           entry.InstallerType,
		"binary_ref":               entry.BinaryRef,
		"checksum_sha256":          entry.ChecksumSha256,
		"cves":                     cves,
		"synced_count":             syncedCount,
		"total_clients":            totalClients,
		"syncs":                    syncs,
	}); err != nil {
		slog.ErrorContext(r.Context(), "encode get catalog response", "error", err)
	}
}

// Update handles PUT /api/v1/catalog/{id}.
func (h *CatalogHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse catalog id: %s", err))
		return
	}

	var req catalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("decode request body: %s", err))
		return
	}

	if err := req.validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	params := sqlcgen.UpdateCatalogEntryParams{
		ID:       id,
		Name:     req.Name,
		Vendor:   req.Vendor,
		OsFamily: req.OsFamily,
		Version:  req.Version,
		Severity: req.Severity,
	}

	if req.ReleaseDate != nil {
		t, err := time.Parse(time.RFC3339, *req.ReleaseDate)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse release_date: %s", err))
			return
		}
		params.ReleaseDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	if req.Description != nil {
		params.Description = pgtype.Text{String: *req.Description, Valid: true}
	}

	entry, err := h.queries.UpdateCatalogEntry(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "catalog entry not found")
			return
		}
		slog.ErrorContext(r.Context(), "update catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "update catalog entry: internal error")
		return
	}

	// Re-link CVEs
	if err := h.queries.UnlinkAllCatalogCVEs(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "unlink catalog CVEs", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "update catalog CVEs: internal error")
		return
	}
	var cveWarnings []string
	for _, cveIDStr := range req.CVEIDs {
		cveID, err := parseUUID(cveIDStr)
		if err != nil {
			slog.ErrorContext(r.Context(), "parse cve_id for linking", "cve_id", cveIDStr, "error", err)
			cveWarnings = append(cveWarnings, fmt.Sprintf("invalid CVE ID %q", cveIDStr))
			continue
		}
		if err := h.queries.LinkCatalogCVE(r.Context(), sqlcgen.LinkCatalogCVEParams{
			CatalogID: id,
			CveID:     cveID,
		}); err != nil {
			slog.ErrorContext(r.Context(), "link catalog CVE", "catalog_id", id, "cve_id", cveIDStr, "error", err)
			cveWarnings = append(cveWarnings, fmt.Sprintf("failed to link CVE %q", cveIDStr))
		}
	}

	tenantID := tenant.MustTenantID(r.Context())
	entryIDStr := uuidToString(entry.ID)
	evt := domain.NewSystemEvent(events.CatalogUpdated, tenantID, "patch_catalog", entryIDStr, "update", entry)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit catalog.updated event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{"entry": entry}
	if len(cveWarnings) > 0 {
		resp["warnings"] = cveWarnings
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "encode update catalog response", "error", err)
	}
}

// Delete handles DELETE /api/v1/catalog/{id}.
func (h *CatalogHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("parse catalog id: %s", err))
		return
	}

	if err := h.queries.SoftDeleteCatalogEntry(r.Context(), id); err != nil {
		slog.ErrorContext(r.Context(), "soft delete catalog entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "delete catalog entry: internal error")
		return
	}

	tenantID := tenant.MustTenantID(r.Context())
	entryIDStr := uuidToString(id)
	evt := domain.NewSystemEvent(events.CatalogDeleted, tenantID, "patch_catalog", entryIDStr, "delete", nil)
	if err := h.eventBus.Emit(r.Context(), evt); err != nil {
		slog.ErrorContext(r.Context(), "emit catalog.deleted event", "error", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseUUID parses a UUID string into a pgtype.UUID.
func parseUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return u, nil
}

// uuidToString converts a pgtype.UUID to its string representation.
func uuidToString(u pgtype.UUID) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// queryParamInt reads an integer query parameter with a default value.
func queryParamInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// writeJSONError writes a JSON error response.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		slog.Error("write error response", "error", err)
	}
}
