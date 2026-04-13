package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// HubSyncHandler triggers on-demand catalog sync from Hub into PM.
type HubSyncHandler struct {
	hubURL    string
	hubAPIKey string
	eventBus  domain.EventBus
	store     *store.Store
	client    *http.Client
}

// NewHubSyncHandler creates a handler for Hub catalog sync.
func NewHubSyncHandler(hubURL, hubAPIKey string, eventBus domain.EventBus, st *store.Store) *HubSyncHandler {
	return &HubSyncHandler{
		hubURL:    hubURL,
		hubAPIKey: hubAPIKey,
		eventBus:  eventBus,
		store:     st,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// hubCVELink maps a catalog entry UUID to its associated CVE IDs.
type hubCVELink struct {
	CatalogID string   `json:"catalog_id"`
	CVEIDs    []string `json:"cve_ids"`
}

type hubSyncResponse struct {
	Entries    []json.RawMessage `json:"entries"`
	DeletedIDs []string          `json:"deleted_ids"`
	CVELinks   []hubCVELink      `json:"cve_links"`
	ServerTime string            `json:"server_time"`
}

// catalogEntry represents a single patch catalog entry from the Hub sync response.
type catalogEntry struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Vendor         string `json:"vendor"`
	OsFamily       string `json:"os_family"`
	Version        string `json:"version"`
	Severity       string `json:"severity"`
	Description    string `json:"description"`
	Product        string `json:"product"`
	BinaryRef      string `json:"binary_ref"`
	ChecksumSha256 string `json:"checksum_sha256"`
}

// TriggerSync calls Hub's sync endpoint and returns the sync summary.
// Accepts optional "since" query param (RFC3339); defaults to epoch (full sync).
func (h *HubSyncHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	since := r.URL.Query().Get("since")
	if since == "" {
		since = time.Unix(0, 0).UTC().Format(time.RFC3339)
	} else if _, err := time.Parse(time.RFC3339, since); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid since parameter: must be RFC3339 format")
		return
	}

	syncURL := fmt.Sprintf("%s/api/v1/sync?since=%s", h.hubURL, url.QueryEscape(since))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncURL, nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("build hub request: %v", err))
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.hubAPIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "hub sync: request failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("hub sync request: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
		slog.ErrorContext(ctx, "hub sync: non-200 response", "status", resp.StatusCode, "body", string(body))
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("hub returned status %d", resp.StatusCode))
		return
	}

	var syncResp hubSyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		slog.ErrorContext(ctx, "hub sync: decode response", "error", err)
		writeJSONError(w, http.StatusBadGateway, fmt.Sprintf("hub sync decode: %v", err))
		return
	}

	synced := len(syncResp.Entries)
	deleted := len(syncResp.DeletedIDs)

	// Upsert catalog entries into the PM patches table within a tenant-scoped transaction.
	if h.store != nil && synced > 0 {
		tx, err := h.store.BeginTx(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "hub sync: begin tx for patch upsert", "error", err)
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("hub sync: begin tx: %v", err))
			return
		}
		qtx := sqlcgen.New(tx)

		for i, raw := range syncResp.Entries {
			var entry catalogEntry
			if err := json.Unmarshal(raw, &entry); err != nil {
				slog.WarnContext(ctx, "hub sync: skip malformed catalog entry", "index", i, "error", err)
				continue
			}

			tid, err := scanUUID(tenantID)
			if err != nil {
				slog.ErrorContext(ctx, "hub sync: invalid tenant ID", "tenant_id", tenantID, "error", err)
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					slog.ErrorContext(ctx, "hub sync: rollback after tenant ID error", "error", rbErr)
				}
				writeJSONError(w, http.StatusInternalServerError, "hub sync: invalid tenant ID in context")
				return
			}

			var hubCatalogID pgtype.UUID
			if entry.ID != "" {
				if scanErr := hubCatalogID.Scan(entry.ID); scanErr != nil {
					slog.WarnContext(ctx, "hub sync: invalid hub catalog ID",
						"entry_id", entry.ID, "name", entry.Name, "error", scanErr)
				}
			}

			_, err = qtx.UpsertDiscoveredPatch(ctx, sqlcgen.UpsertDiscoveredPatchParams{
				TenantID:       tid,
				Name:           entry.Name,
				Version:        entry.Version,
				Severity:       entry.Severity,
				OsFamily:       entry.OsFamily,
				Description:    pgtype.Text{String: entry.Description, Valid: entry.Description != ""},
				SourceRepo:     pgtype.Text{String: entry.Vendor, Valid: entry.Vendor != ""},
				PackageUrl:     pgtype.Text{String: entry.BinaryRef, Valid: entry.BinaryRef != ""},
				ChecksumSha256: pgtype.Text{String: entry.ChecksumSha256, Valid: entry.ChecksumSha256 != ""},
				PackageName:    entry.Product,
				HubCatalogID:   hubCatalogID,
			})
			if err != nil {
				slog.ErrorContext(ctx, "hub sync: upsert patch", "name", entry.Name, "version", entry.Version, "error", err)
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					slog.ErrorContext(ctx, "hub sync: rollback after upsert error", "error", rbErr)
				}
				writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("hub sync: upsert patch %q: %v", entry.Name, err))
				return
			}
		}

		if err := tx.Commit(ctx); err != nil {
			slog.ErrorContext(ctx, "hub sync: commit patch upserts", "error", err)
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("hub sync: commit: %v", err))
			return
		}
		slog.InfoContext(ctx, "hub sync: upserted patches", "count", synced, "tenant_id", tenantID)
	}

	// Soft-delete patches that Hub reports as deleted.
	if h.store != nil && deleted > 0 {
		delCount, delErr := h.softDeletePatches(ctx, tenantID, syncResp.DeletedIDs)
		if delErr != nil {
			slog.ErrorContext(ctx, "hub sync: soft-delete patches failed", "error", delErr)
		} else {
			slog.InfoContext(ctx, "hub sync: soft-deleted patches",
				"tenant_id", tenantID, "requested", deleted, "deleted", delCount)
		}
	}

	payload := map[string]any{
		"synced":      synced,
		"deleted":     deleted,
		"server_time": syncResp.ServerTime,
	}
	evt := domain.NewSystemEvent(events.CatalogSynced, tenantID, "patch_catalog", "", "sync", payload)
	if err := h.eventBus.Emit(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "hub sync: emit event", "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.ErrorContext(ctx, "hub sync: encode response", "error", err)
	}
}

// softDeletePatches marks patches as deleted by their Hub catalog IDs within a tenant-scoped transaction.
func (h *HubSyncHandler) softDeletePatches(ctx context.Context, tenantID string, hubIDs []string) (int64, error) {
	tid, err := scanUUID(tenantID)
	if err != nil {
		return 0, fmt.Errorf("soft-delete patches: parse tenant ID: %w", err)
	}

	tx, err := h.store.BeginTx(ctx)
	if err != nil {
		return 0, fmt.Errorf("soft-delete patches: begin tx: %w", err)
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			slog.ErrorContext(ctx, "soft-delete patches: rollback failed", "error", rbErr)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		return 0, fmt.Errorf("soft-delete patches: set tenant context: %w", err)
	}

	uuids := make([]pgtype.UUID, 0, len(hubIDs))
	for _, id := range hubIDs {
		u, parseErr := scanUUID(id)
		if parseErr != nil {
			slog.WarnContext(ctx, "soft-delete patches: skip invalid hub ID", "hub_id", id, "error", parseErr)
			continue
		}
		uuids = append(uuids, u)
	}

	if len(uuids) == 0 {
		return 0, nil
	}

	qtx := sqlcgen.New(tx)
	deleted, err := qtx.SoftDeletePatchesByHubIDs(ctx, sqlcgen.SoftDeletePatchesByHubIDsParams{
		TenantID: tid,
		HubIds:   uuids,
	})
	if err != nil {
		return 0, fmt.Errorf("soft-delete patches: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("soft-delete patches: commit: %w", err)
	}
	return deleted, nil
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		slog.Error("write error response", "error", err)
	}
}
