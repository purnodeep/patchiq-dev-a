package workers

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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// CatalogSyncJobArgs defines the River job for syncing the patch catalog from Hub.
type CatalogSyncJobArgs struct {
	TenantID string `json:"tenant_id,omitempty"`
}

// Kind implements river.JobArgs.
func (CatalogSyncJobArgs) Kind() string { return "catalog_sync" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (CatalogSyncJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// CatalogSyncStore defines the queries needed by CatalogSyncWorker.
type CatalogSyncStore interface {
	GetHubSyncState(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.HubSyncState, error)
	ListAllHubSyncStates(ctx context.Context) ([]sqlcgen.HubSyncState, error)
	UpdateHubSyncStarted(ctx context.Context, tenantID pgtype.UUID) error
	UpdateHubSyncCompleted(ctx context.Context, arg sqlcgen.UpdateHubSyncCompletedParams) error
	UpdateHubSyncFailed(ctx context.Context, arg sqlcgen.UpdateHubSyncFailedParams) error
	UpdateHubCVESyncCompleted(ctx context.Context, tenantID pgtype.UUID) error
	UpdateHubCVESyncFailed(ctx context.Context, arg sqlcgen.UpdateHubCVESyncFailedParams) error
	GetEndpointOsSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetEndpointOsSummaryRow, error)
	GetEndpointStatusSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetEndpointStatusSummaryRow, error)
	GetFrameworkScoreSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetFrameworkScoreSummaryRow, error)
}

// CVESyncer triggers a CVE sync for a given tenant.
type CVESyncer interface {
	SyncNVD(ctx context.Context, tenantID string) error
}

// catalogEntry represents a single patch catalog entry from the Hub sync response.
type catalogEntry struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Vendor         string     `json:"vendor"`
	OsFamily       string     `json:"os_family"`
	Version        string     `json:"version"`
	Severity       string     `json:"severity"`
	Description    string     `json:"description"`
	BinaryRef      string     `json:"binary_ref"`
	ChecksumSha256 string     `json:"checksum_sha256"`
	SourceUrl      string     `json:"source_url"`
	Product        string     `json:"product"`
	OsPackageName  string     `json:"os_package_name"`
	InstallerType  string     `json:"installer_type"`
	SilentArgs     string     `json:"silent_args"`
	ReleaseDate    *time.Time `json:"release_date"`
}

// CatalogSyncWorker fetches catalog entries from Hub and updates the sync state.
type CatalogSyncWorker struct {
	river.WorkerDefaults[CatalogSyncJobArgs]
	store      CatalogSyncStore
	pool       *pgxpool.Pool
	eventBus   domain.EventBus
	client     *http.Client
	cveSyncSvc CVESyncer // optional; when set, triggered after each successful catalog sync
}

// WithCVESync sets an optional CVESyncer that is called after each successful catalog sync.
func (w *CatalogSyncWorker) WithCVESync(svc CVESyncer) {
	w.cveSyncSvc = svc
}

// NewCatalogSyncWorker creates a new CatalogSyncWorker.
func NewCatalogSyncWorker(store CatalogSyncStore, pool *pgxpool.Pool, eventBus domain.EventBus) *CatalogSyncWorker {
	return &CatalogSyncWorker{
		store:    store,
		pool:     pool,
		eventBus: eventBus,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

// hubCVELink maps a catalog entry UUID to its associated CVE IDs.
type hubCVELink struct {
	CatalogID string   `json:"catalog_id"`
	CVEIDs    []string `json:"cve_ids"`
}

// hubCatalogResponse represents the response from the Hub sync endpoint.
type hubCatalogResponse struct {
	Entries    []json.RawMessage `json:"entries"`
	DeletedIDs []string          `json:"deleted_ids"`
	CVELinks   []hubCVELink      `json:"cve_links"`
	ServerTime string            `json:"server_time"`
}

// Work implements river.Worker. It fetches catalog entries from Hub,
// logs the count received, and updates the sync state.
func (w *CatalogSyncWorker) Work(ctx context.Context, job *river.Job[CatalogSyncJobArgs]) error {
	// If a specific tenant ID is provided in job args, sync only that tenant.
	if job.Args.TenantID != "" {
		var tid pgtype.UUID
		if err := tid.Scan(job.Args.TenantID); err != nil {
			return fmt.Errorf("catalog sync: parse tenant ID %q: %w", job.Args.TenantID, err)
		}
		return w.SyncForTenant(ctx, tid)
	}

	// Otherwise iterate all configured tenants.
	states, err := w.getAllSyncStates(ctx)
	if err != nil {
		return fmt.Errorf("catalog sync: get sync states: %w", err)
	}

	if len(states) == 0 {
		slog.InfoContext(ctx, "catalog sync: no hub sync states configured, skipping")
		return nil
	}

	var lastErr error
	for _, state := range states {
		if syncErr := w.syncTenant(ctx, state); syncErr != nil {
			lastErr = syncErr
			slog.ErrorContext(ctx, "catalog sync: tenant sync failed",
				"tenant_id", uuidToStr(state.TenantID),
				"error", syncErr,
			)
		}
	}
	return lastErr
}

func (w *CatalogSyncWorker) getAllSyncStates(ctx context.Context) ([]sqlcgen.HubSyncState, error) {
	return w.store.ListAllHubSyncStates(ctx)
}

// SyncForTenant performs a catalog sync for a specific tenant. This is called
// by the API handler when a manual sync is triggered.
func (w *CatalogSyncWorker) SyncForTenant(ctx context.Context, tenantID pgtype.UUID) error {
	state, err := w.store.GetHubSyncState(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("catalog sync: get hub sync state: %w", err)
	}
	return w.syncTenant(ctx, state)
}

func (w *CatalogSyncWorker) syncTenant(ctx context.Context, state sqlcgen.HubSyncState) error {
	tenantIDStr := uuidToStr(state.TenantID)

	// Mark sync as started
	if err := w.store.UpdateHubSyncStarted(ctx, state.TenantID); err != nil {
		return fmt.Errorf("catalog sync: mark started: %w", err)
	}
	w.emitEvent(ctx, events.CatalogSyncStarted, tenantIDStr, nil)

	// Build Hub URL with since parameter
	since := time.Unix(0, 0).UTC().Format(time.RFC3339)
	if state.LastSyncAt.Valid {
		since = state.LastSyncAt.Time.UTC().Format(time.RFC3339)
	}

	syncURL := fmt.Sprintf("%s/api/v1/sync?since=%s", state.HubUrl, url.QueryEscape(since))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncURL, nil)
	if err != nil {
		return w.failSync(ctx, state.TenantID, tenantIDStr, fmt.Errorf("catalog sync: build request: %w", err))
	}
	req.Header.Set("Authorization", "Bearer "+state.ApiKey)

	// Best-effort: gather tenant summary data and send as headers.
	w.setSummaryHeaders(ctx, req, state.TenantID, tenantIDStr)

	resp, err := w.client.Do(req)
	if err != nil {
		return w.failSync(ctx, state.TenantID, tenantIDStr, fmt.Errorf("catalog sync: hub request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return w.failSync(ctx, state.TenantID, tenantIDStr,
			fmt.Errorf("catalog sync: hub returned status %d: %s", resp.StatusCode, string(body)))
	}

	var syncResp hubCatalogResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return w.failSync(ctx, state.TenantID, tenantIDStr, fmt.Errorf("catalog sync: decode response: %w", err))
	}

	entryCount := int32(len(syncResp.Entries))
	slog.InfoContext(ctx, "catalog sync: entries received",
		"tenant_id", tenantIDStr,
		"entry_count", entryCount,
		"deleted_count", len(syncResp.DeletedIDs),
		"server_time", syncResp.ServerTime,
	)

	// Persist catalog entries into the PM patches table.
	var catalogToPatch map[string]pgtype.UUID
	if w.pool != nil && entryCount > 0 {
		upserted, ctpMap, upsertErr := w.upsertEntries(ctx, state.TenantID, syncResp.Entries)
		if upsertErr != nil {
			slog.ErrorContext(ctx, "catalog sync: upsert entries failed",
				"tenant_id", tenantIDStr,
				"error", upsertErr,
			)
			return w.failSync(ctx, state.TenantID, tenantIDStr,
				fmt.Errorf("catalog sync: upsert entries: %w", upsertErr))
		}
		catalogToPatch = ctpMap
		slog.InfoContext(ctx, "catalog sync: upserted patches",
			"tenant_id", tenantIDStr,
			"upserted", upserted,
		)
	}

	// Soft-delete patches that Hub reports as deleted.
	if w.pool != nil && len(syncResp.DeletedIDs) > 0 {
		deleted, delErr := w.softDeletePatches(ctx, state.TenantID, syncResp.DeletedIDs)
		if delErr != nil {
			slog.ErrorContext(ctx, "catalog sync: soft-delete patches failed",
				"tenant_id", tenantIDStr, "error", delErr)
		} else {
			slog.InfoContext(ctx, "catalog sync: soft-deleted patches",
				"tenant_id", tenantIDStr, "requested", len(syncResp.DeletedIDs), "deleted", deleted)
		}
	}

	// Trigger CVE sync from Hub so CVEs exist before linking.
	if w.cveSyncSvc != nil {
		if cveErr := w.cveSyncSvc.SyncNVD(ctx, tenantIDStr); cveErr != nil {
			slog.ErrorContext(ctx, "catalog sync: post-sync CVE sync failed",
				"tenant_id", tenantIDStr, "error", cveErr)
			_ = w.store.UpdateHubCVESyncFailed(ctx, sqlcgen.UpdateHubCVESyncFailedParams{
				LastError: pgtype.Text{String: cveErr.Error(), Valid: true},
				TenantID:  state.TenantID,
			})
		} else {
			_ = w.store.UpdateHubCVESyncCompleted(ctx, state.TenantID)
			slog.InfoContext(ctx, "catalog sync: post-sync CVE sync completed", "tenant_id", tenantIDStr)
		}
	}

	// Link CVEs to patches based on Hub-provided CVE linkages.
	if w.pool != nil && len(syncResp.CVELinks) > 0 && len(catalogToPatch) > 0 {
		linked, linkErr := w.linkCVEs(ctx, state.TenantID, syncResp.CVELinks, catalogToPatch)
		if linkErr != nil {
			slog.ErrorContext(ctx, "catalog sync: link CVEs failed",
				"tenant_id", tenantIDStr, "error", linkErr)
		} else {
			slog.InfoContext(ctx, "catalog sync: linked CVEs to patches",
				"tenant_id", tenantIDStr, "linked", linked)
		}
	}

	// Calculate next sync time
	nextSync := time.Now().UTC().Add(time.Duration(state.SyncInterval) * time.Second)

	if err := w.store.UpdateHubSyncCompleted(ctx, sqlcgen.UpdateHubSyncCompletedParams{
		NextSyncAt: pgtype.Timestamptz{Time: nextSync, Valid: true},
		EntryCount: entryCount,
		TenantID:   state.TenantID,
	}); err != nil {
		return fmt.Errorf("catalog sync: update completed: %w", err)
	}

	w.emitEvent(ctx, events.CatalogSynced, tenantIDStr, map[string]any{
		"entries_received": entryCount,
		"deleted_count":    len(syncResp.DeletedIDs),
		"server_time":      syncResp.ServerTime,
	})

	return nil
}

// upsertEntries parses raw catalog entries and upserts them into the patches table
// within a tenant-scoped transaction. Returns upsert count and a map of hub_catalog_id → patch_id.
func (w *CatalogSyncWorker) upsertEntries(ctx context.Context, tenantID pgtype.UUID, entries []json.RawMessage) (int, map[string]pgtype.UUID, error) {
	tenantIDStr := uuidToStr(tenantID)

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err.Error() != "tx is closed" {
			slog.ErrorContext(ctx, "catalog sync: rollback failed", "error", err)
		}
	}()

	// Set RLS tenant context for this transaction.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantIDStr); err != nil {
		return 0, nil, fmt.Errorf("set tenant context: %w", err)
	}

	qtx := sqlcgen.New(tx)
	upserted := 0
	catalogToPatch := make(map[string]pgtype.UUID)

	for i, raw := range entries {
		var entry catalogEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			slog.WarnContext(ctx, "catalog sync: skip malformed entry", "index", i, "error", err)
			continue
		}

		var hubCatalogID pgtype.UUID
		if entry.ID != "" {
			if scanErr := hubCatalogID.Scan(entry.ID); scanErr != nil {
				slog.WarnContext(ctx, "catalog sync: invalid hub catalog ID",
					"entry_id", entry.ID, "name", entry.Name, "error", scanErr)
			}
		}

		result, err := qtx.UpsertDiscoveredPatch(ctx, sqlcgen.UpsertDiscoveredPatchParams{
			TenantID:       tenantID,
			Name:           entry.Name,
			Version:        entry.Version,
			Severity:       entry.Severity,
			OsFamily:       entry.OsFamily,
			Description:    pgtype.Text{String: entry.Description, Valid: entry.Description != ""},
			SourceRepo:     pgtype.Text{String: entry.Vendor, Valid: entry.Vendor != ""},
			PackageUrl:     pgtype.Text{String: entry.BinaryRef, Valid: entry.BinaryRef != ""},
			ChecksumSha256: pgtype.Text{String: entry.ChecksumSha256, Valid: entry.ChecksumSha256 != ""},
			PackageName:    resolvePackageName(entry),
			ReleasedAt:     pgtype.Timestamptz{Time: derefTime(entry.ReleaseDate), Valid: entry.ReleaseDate != nil},
			InstallerType:  entry.InstallerType,
			SilentArgs:     entry.SilentArgs,
			HubCatalogID:   hubCatalogID,
		})
		if err != nil {
			slog.ErrorContext(ctx, "catalog sync: upsert patch failed",
				"name", entry.Name, "version", entry.Version, "error", err)
			continue
		}
		if entry.ID != "" {
			catalogToPatch[entry.ID] = result.ID
		}
		upserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, fmt.Errorf("commit tx: %w", err)
	}
	return upserted, catalogToPatch, nil
}

// setSummaryHeaders queries endpoint/compliance aggregates and sets them as
// HTTP headers on the hub sync request. All queries are best-effort; failures
// are logged and the header is set to an empty JSON object.
func (w *CatalogSyncWorker) setSummaryHeaders(ctx context.Context, req *http.Request, tenantID pgtype.UUID, tenantIDStr string) {
	// OS summary
	osSummary, err := w.store.GetEndpointOsSummary(ctx, tenantID)
	if err != nil {
		slog.WarnContext(ctx, "catalog sync: failed to query OS summary", "tenant_id", tenantIDStr, "error", err)
		req.Header.Set("X-Os-Summary", "{}")
	} else {
		req.Header.Set("X-Os-Summary", mustJSON(osSummary))
	}

	// Endpoint status summary
	statusSummary, err := w.store.GetEndpointStatusSummary(ctx, tenantID)
	if err != nil {
		slog.WarnContext(ctx, "catalog sync: failed to query endpoint status summary", "tenant_id", tenantIDStr, "error", err)
		req.Header.Set("X-Endpoint-Status-Summary", "{}")
	} else {
		req.Header.Set("X-Endpoint-Status-Summary", mustJSON(statusSummary))
	}

	// Compliance framework scores
	complianceSummary, err := w.store.GetFrameworkScoreSummary(ctx, tenantID)
	if err != nil {
		slog.WarnContext(ctx, "catalog sync: failed to query compliance summary", "tenant_id", tenantIDStr, "error", err)
		req.Header.Set("X-Compliance-Summary", "{}")
	} else {
		req.Header.Set("X-Compliance-Summary", mustJSON(complianceSummary))
	}

	// Total endpoint count (derived from status summary to avoid an extra query)
	totalEndpoints := 0
	for _, s := range statusSummary {
		totalEndpoints += int(s.Count)
	}
	req.Header.Set("X-Endpoint-Count", fmt.Sprintf("%d", totalEndpoints))
}

// mustJSON marshals v to a JSON string; returns "{}" on error.
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func (w *CatalogSyncWorker) failSync(ctx context.Context, tenantID pgtype.UUID, tenantIDStr string, syncErr error) error {
	if updateErr := w.store.UpdateHubSyncFailed(ctx, sqlcgen.UpdateHubSyncFailedParams{
		ErrorMessage: pgtype.Text{String: syncErr.Error(), Valid: true},
		TenantID:     tenantID,
	}); updateErr != nil {
		slog.ErrorContext(ctx, "catalog sync: update failed state", "error", updateErr)
	}
	w.emitEvent(ctx, events.CatalogSyncFailed, tenantIDStr, map[string]any{
		"error": syncErr.Error(),
	})
	return syncErr
}

func (w *CatalogSyncWorker) emitEvent(ctx context.Context, eventType, tenantID string, payload any) {
	if w.eventBus == nil {
		return
	}
	evt := domain.NewSystemEvent(eventType, tenantID, "hub_sync_state", "", eventType, payload)
	if err := w.eventBus.Emit(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "catalog sync: emit event failed",
			"event_type", eventType,
			"tenant_id", tenantID,
			"error", err,
		)
	}
}

// linkCVEs creates patch_cves links based on Hub-provided CVE linkage data.
// It looks up CVE DB IDs by cve_id strings and links them to the corresponding patches.
func (w *CatalogSyncWorker) linkCVEs(ctx context.Context, tenantID pgtype.UUID, cveLinks []hubCVELink, catalogToPatch map[string]pgtype.UUID) (int, error) {
	tenantIDStr := uuidToStr(tenantID)

	// Collect all unique CVE IDs across all links.
	cveIDSet := make(map[string]struct{})
	for _, link := range cveLinks {
		for _, cveID := range link.CVEIDs {
			cveIDSet[cveID] = struct{}{}
		}
	}
	if len(cveIDSet) == 0 {
		return 0, nil
	}

	cveIDs := make([]string, 0, len(cveIDSet))
	for id := range cveIDSet {
		cveIDs = append(cveIDs, id)
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("link CVEs: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err.Error() != "tx is closed" {
			slog.ErrorContext(ctx, "link CVEs: rollback failed", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantIDStr); err != nil {
		return 0, fmt.Errorf("link CVEs: set tenant context: %w", err)
	}

	qtx := sqlcgen.New(tx)

	// Look up CVE DB IDs.
	dbCVEs, err := qtx.GetCVEDBIDsByCVEIDs(ctx, sqlcgen.GetCVEDBIDsByCVEIDsParams{
		TenantID: tenantID,
		CveIds:   cveIDs,
	})
	if err != nil {
		return 0, fmt.Errorf("link CVEs: get CVE DB IDs: %w", err)
	}

	cveDBIDMap := make(map[string]pgtype.UUID, len(dbCVEs))
	for _, c := range dbCVEs {
		cveDBIDMap[c.CveID] = c.ID
	}

	linked := 0
	for _, link := range cveLinks {
		patchID, ok := catalogToPatch[link.CatalogID]
		if !ok {
			continue
		}
		for _, cveID := range link.CVEIDs {
			cveDBID, ok := cveDBIDMap[cveID]
			if !ok {
				continue // CVE not yet synced to server — skip
			}
			if err := qtx.LinkPatchCVE(ctx, sqlcgen.LinkPatchCVEParams{
				TenantID: tenantID,
				PatchID:  patchID,
				CveID:    cveDBID,
			}); err != nil {
				slog.WarnContext(ctx, "link CVEs: link failed",
					"patch_id", uuidToStr(patchID), "cve_id", cveID, "error", err)
				continue
			}
			linked++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("link CVEs: commit: %w", err)
	}
	return linked, nil
}

// softDeletePatches marks patches as deleted by their Hub catalog IDs within a tenant-scoped transaction.
func (w *CatalogSyncWorker) softDeletePatches(ctx context.Context, tenantID pgtype.UUID, hubIDs []string) (int64, error) {
	tenantIDStr := uuidToStr(tenantID)

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("soft-delete patches: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err.Error() != "tx is closed" {
			slog.ErrorContext(ctx, "soft-delete patches: rollback failed", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantIDStr); err != nil {
		return 0, fmt.Errorf("soft-delete patches: set tenant context: %w", err)
	}

	uuids := make([]pgtype.UUID, 0, len(hubIDs))
	for _, id := range hubIDs {
		var u pgtype.UUID
		if err := u.Scan(id); err != nil {
			slog.WarnContext(ctx, "soft-delete patches: skip invalid hub ID", "hub_id", id, "error", err)
			continue
		}
		uuids = append(uuids, u)
	}

	if len(uuids) == 0 {
		return 0, nil
	}

	qtx := sqlcgen.New(tx)
	deleted, err := qtx.SoftDeletePatchesByHubIDs(ctx, sqlcgen.SoftDeletePatchesByHubIDsParams{
		TenantID: tenantID,
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

// resolvePackageName returns the os_package_name if set, falling back to product, then name.
func resolvePackageName(entry catalogEntry) string {
	if entry.OsPackageName != "" {
		return entry.OsPackageName
	}
	if entry.Product != "" {
		return entry.Product
	}
	return entry.Name
}

// derefTime dereferences a *time.Time, returning zero value if nil.
func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
