package v1

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// SyncQuerier abstracts the sqlcgen queries used by SyncHandler.
type SyncQuerier interface {
	ListCatalogEntriesUpdatedSince(ctx context.Context, updatedAt pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error)
	ListCatalogEntriesDeletedSince(ctx context.Context, deletedAt pgtype.Timestamptz) ([]pgtype.UUID, error)
	ListCVEFeedsUpdatedSince(ctx context.Context, arg sqlcgen.ListCVEFeedsUpdatedSinceParams) ([]sqlcgen.CVEFeed, error)
	ListCatalogCVELinks(ctx context.Context, catalogIds []pgtype.UUID) ([]sqlcgen.ListCatalogCVELinksRow, error)
	GetClientByAPIKeyHash(ctx context.Context, apiKeyHash pgtype.Text) (sqlcgen.Client, error)
	UpdateClientSummaries(ctx context.Context, arg sqlcgen.UpdateClientSummariesParams) (sqlcgen.Client, error)
	InsertClientSyncHistory(ctx context.Context, arg sqlcgen.InsertClientSyncHistoryParams) (sqlcgen.ClientSyncHistory, error)
}

// hashAPIKey returns a deterministic SHA-256 hex digest of the plaintext API key.
// This is used to look up the calling client by their API key without iterating
// all clients (bcrypt hashes are non-deterministic and cannot be used for lookups).
func hashAPIKey(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

// SyncHandler serves the catalog delta sync endpoint for Patch Manager polling.
type SyncHandler struct {
	queries  SyncQuerier
	apiKey   string
	eventBus domain.EventBus
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(queries SyncQuerier, apiKey string, eventBus domain.EventBus) *SyncHandler {
	return &SyncHandler{queries: queries, apiKey: apiKey, eventBus: eventBus}
}

// catalogCVELink maps a catalog entry UUID to its associated CVE IDs.
type catalogCVELink struct {
	CatalogID string   `json:"catalog_id"`
	CVEIDs    []string `json:"cve_ids"`
}

// syncResponse is the JSON response for GET /api/v1/sync.
type syncResponse struct {
	Entries    []sqlcgen.PatchCatalog `json:"entries"`
	DeletedIDs []string               `json:"deleted_ids"`
	CVELinks   []catalogCVELink       `json:"cve_links,omitempty"`
	ServerTime string                 `json:"server_time"`
}

// SyncAuthMiddleware returns a middleware that validates the Bearer token against the sync API key.
// It mirrors the auth check inside SyncHandler.Sync so that other endpoints can reuse it.
func SyncAuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				writeJSONError(w, http.StatusServiceUnavailable, "endpoint not configured: API key missing")
				return
			}
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSONError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if subtle.ConstantTimeCompare([]byte(token), []byte(apiKey)) != 1 {
				writeJSONError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Sync handles GET /api/v1/sync?since=<RFC3339>.
func (h *SyncHandler) Sync(w http.ResponseWriter, r *http.Request) {
	syncStartedAt := time.Now()

	// Reject requests when API key is not configured.
	if h.apiKey == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "sync endpoint not configured: API key missing")
		return
	}

	// Auth: check Bearer token.
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if subtle.ConstantTimeCompare([]byte(token), []byte(h.apiKey)) != 1 {
		writeJSONError(w, http.StatusUnauthorized, "invalid API key")
		return
	}

	// Identify the calling client by API key hash (best-effort).
	var clientID pgtype.UUID
	var tenantID pgtype.UUID
	apiKeyHashStr := hashAPIKey(token)
	if client, err := h.queries.GetClientByAPIKeyHash(r.Context(), pgtype.Text{String: apiKeyHashStr, Valid: true}); err == nil {
		clientID = client.ID
		tenantID = client.TenantID
	} else {
		slog.DebugContext(r.Context(), "client lookup by API key hash failed (non-fatal)", "error", err)
	}

	// Parse since query param.
	sinceStr := r.URL.Query().Get("since")
	if sinceStr == "" {
		writeJSONError(w, http.StatusBadRequest, "since query parameter is required")
		return
	}
	sinceTime, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid since parameter: must be RFC3339 format")
		return
	}

	// Parse summary headers from Patch Manager.
	endpointCount := int32(0)
	if v := r.Header.Get("X-Endpoint-Count"); v != "" {
		slog.InfoContext(r.Context(), "sync request received", "endpoint_count", v)
		if n, parseErr := strconv.Atoi(v); parseErr == nil {
			endpointCount = int32(n)
		}
	}

	osSummary := []byte("{}")
	if v := r.Header.Get("X-Os-Summary"); v != "" {
		if json.Valid([]byte(v)) {
			osSummary = []byte(v)
		}
	}

	statusSummary := []byte("{}")
	if v := r.Header.Get("X-Endpoint-Status-Summary"); v != "" {
		if json.Valid([]byte(v)) {
			statusSummary = []byte(v)
		}
	}

	complianceSummary := []byte("{}")
	if v := r.Header.Get("X-Compliance-Summary"); v != "" {
		if json.Valid([]byte(v)) {
			complianceSummary = []byte(v)
		}
	}

	since := pgtype.Timestamptz{Time: sinceTime, Valid: true}

	entries, err := h.queries.ListCatalogEntriesUpdatedSince(r.Context(), since)
	if err != nil {
		slog.ErrorContext(r.Context(), "list catalog entries updated since", "since", sinceStr, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list updated catalog entries: internal error")
		return
	}

	deletedUUIDs, err := h.queries.ListCatalogEntriesDeletedSince(r.Context(), since)
	if err != nil {
		slog.ErrorContext(r.Context(), "list catalog entries deleted since", "since", sinceStr, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "list deleted catalog entries: internal error")
		return
	}

	if entries == nil {
		entries = []sqlcgen.PatchCatalog{}
	}

	deletedIDs := make([]string, 0, len(deletedUUIDs))
	for _, u := range deletedUUIDs {
		deletedIDs = append(deletedIDs, uuidToString(u))
	}

	// Update client summaries if we identified the client.
	if clientID.Valid {
		if _, updateErr := h.queries.UpdateClientSummaries(r.Context(), sqlcgen.UpdateClientSummariesParams{
			ID:                    clientID,
			EndpointCount:         endpointCount,
			OsSummary:             osSummary,
			EndpointStatusSummary: statusSummary,
			ComplianceSummary:     complianceSummary,
		}); updateErr != nil {
			slog.ErrorContext(r.Context(), "update client summaries", "error", updateErr)
		}
	}

	// Fetch CVE linkages for the returned catalog entries.
	var cveLinks []catalogCVELink
	if len(entries) > 0 {
		catalogIDs := make([]pgtype.UUID, len(entries))
		for i, e := range entries {
			catalogIDs[i] = e.ID
		}
		links, linkErr := h.queries.ListCatalogCVELinks(r.Context(), catalogIDs)
		if linkErr != nil {
			slog.ErrorContext(r.Context(), "list catalog CVE links", "error", linkErr)
		} else if len(links) > 0 {
			linkMap := make(map[string][]string)
			for _, l := range links {
				catID := uuidToString(l.CatalogID)
				linkMap[catID] = append(linkMap[catID], l.CveID)
			}
			cveLinks = make([]catalogCVELink, 0, len(linkMap))
			for catID, cveIDs := range linkMap {
				cveLinks = append(cveLinks, catalogCVELink{CatalogID: catID, CVEIDs: cveIDs})
			}
		}
	}

	resp := syncResponse{
		Entries:    entries,
		DeletedIDs: deletedIDs,
		CVELinks:   cveLinks,
		ServerTime: time.Now().UTC().Format(time.RFC3339Nano),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "encode sync response", "error", err)
	}

	// Insert sync history record.
	syncFinishedAt := time.Now()
	if clientID.Valid {
		if _, histErr := h.queries.InsertClientSyncHistory(r.Context(), sqlcgen.InsertClientSyncHistoryParams{
			TenantID:         tenantID,
			ClientID:         clientID,
			StartedAt:        pgtype.Timestamptz{Time: syncStartedAt, Valid: true},
			FinishedAt:       pgtype.Timestamptz{Time: syncFinishedAt, Valid: true},
			DurationMs:       pgtype.Int4{Int32: int32(syncFinishedAt.Sub(syncStartedAt).Milliseconds()), Valid: true},
			EntriesDelivered: int32(len(entries)),
			DeletesDelivered: int32(len(deletedIDs)),
			EndpointCount:    endpointCount,
			Status:           "success",
		}); histErr != nil {
			slog.ErrorContext(r.Context(), "insert client sync history", "error", histErr)
		}
	}

	// Emit sync.completed event after successful response.
	if h.eventBus != nil {
		tenantIDStr := uuidToString(tenantID)
		evt := domain.NewSystemEvent(events.SyncCompleted, tenantIDStr, "sync", uuidToString(clientID), "completed", map[string]any{
			"entries_count": len(entries),
			"deleted_count": len(deletedIDs),
			"since":         sinceStr,
		})
		if err := h.eventBus.Emit(r.Context(), evt); err != nil {
			slog.ErrorContext(r.Context(), "emit sync.completed event", "error", err)
		}
	}
}

// cveSyncResponse is the JSON response for GET /api/v1/sync/cves.
type cveSyncResponse struct {
	CVEs       []sqlcgen.CVEFeed `json:"cves"`
	ServerTime string            `json:"server_time"`
}

// SyncCVEs handles GET /api/v1/sync/cves?since=<RFC3339>.
// Returns CVE records updated since the given timestamp.
// Protected by the same Bearer token auth as the catalog sync endpoint.
func (h *SyncHandler) SyncCVEs(w http.ResponseWriter, r *http.Request) {
	// Auth: reuse same Bearer token check.
	if h.apiKey == "" {
		writeJSONError(w, http.StatusServiceUnavailable, "sync endpoint not configured: API key missing")
		return
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
		return
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if subtle.ConstantTimeCompare([]byte(token), []byte(h.apiKey)) != 1 {
		writeJSONError(w, http.StatusUnauthorized, "invalid API key")
		return
	}

	// Identify calling client for event context (best-effort).
	var cveSyncTenantID string
	apiKeyHashStr := hashAPIKey(token)
	if client, err := h.queries.GetClientByAPIKeyHash(r.Context(), pgtype.Text{String: apiKeyHashStr, Valid: true}); err == nil {
		cveSyncTenantID = uuidToString(client.TenantID)
	}

	sinceStr := r.URL.Query().Get("since")
	if sinceStr == "" {
		writeJSONError(w, http.StatusBadRequest, "since query parameter is required")
		return
	}
	sinceTime, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid since parameter: must be RFC3339 format")
		return
	}

	since := pgtype.Timestamptz{Time: sinceTime, Valid: true}
	cves, err := h.queries.ListCVEFeedsUpdatedSince(r.Context(), sqlcgen.ListCVEFeedsUpdatedSinceParams{
		UpdatedAt: since,
		Limit:     10000,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "list CVE feeds updated since", "since", sinceStr, "error", err)
		if h.eventBus != nil {
			evt := domain.NewSystemEvent(events.CVESyncFailed, cveSyncTenantID, "cve_sync", "", "failed", map[string]any{
				"error": err.Error(),
				"since": sinceStr,
			})
			_ = h.eventBus.Emit(r.Context(), evt)
		}
		writeJSONError(w, http.StatusInternalServerError, "list CVE feeds: internal error")
		return
	}

	if cves == nil {
		cves = []sqlcgen.CVEFeed{}
	}

	resp := cveSyncResponse{
		CVEs:       cves,
		ServerTime: time.Now().UTC().Format(time.RFC3339Nano),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.ErrorContext(r.Context(), "encode CVE sync response", "error", err)
	}

	if h.eventBus != nil {
		evt := domain.NewSystemEvent(events.CVESyncCompleted, cveSyncTenantID, "cve_sync", "", "completed", map[string]any{
			"cve_count": len(cves),
			"since":     sinceStr,
		})
		if err := h.eventBus.Emit(r.Context(), evt); err != nil {
			slog.ErrorContext(r.Context(), "emit cve_sync.completed event", "error", err)
		}
	}
}
