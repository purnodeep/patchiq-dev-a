package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/feeds"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// defaultTenantID is the well-known tenant for global/system operations.
const defaultTenantID = "00000000-0000-0000-0000-000000000001"

// PipelineStore abstracts the sqlc queries required by the normalization pipeline.
type PipelineStore interface {
	GetFeedSourceByName(ctx context.Context, name string) (sqlcgen.FeedSource, error)
	GetFeedSyncState(ctx context.Context, feedSourceID pgtype.UUID) (sqlcgen.FeedSyncState, error)
	UpdateFeedSyncStateStart(ctx context.Context, feedSourceID pgtype.UUID) error
	UpdateFeedSyncStateSuccess(ctx context.Context, arg sqlcgen.UpdateFeedSyncStateSuccessParams) error
	UpdateFeedSyncStateError(ctx context.Context, arg sqlcgen.UpdateFeedSyncStateErrorParams) error
	UpsertCatalogEntryFromFeed(ctx context.Context, arg sqlcgen.UpsertCatalogEntryFromFeedParams) (sqlcgen.PatchCatalog, error)
	GetCVEFeedByCVEID(ctx context.Context, cveID string) (sqlcgen.CVEFeed, error)
	CreateCVEFeed(ctx context.Context, arg sqlcgen.CreateCVEFeedParams) (sqlcgen.CVEFeed, error)
	UpsertCVEFeed(ctx context.Context, arg sqlcgen.UpsertCVEFeedParams) (sqlcgen.CVEFeed, error)
	LinkCatalogCVE(ctx context.Context, arg sqlcgen.LinkCatalogCVEParams) error
	CreateBinaryFetchState(ctx context.Context, arg sqlcgen.CreateBinaryFetchStateParams) (sqlcgen.BinaryFetchState, error)
}

// Pipeline normalizes raw feed entries into the patch catalog.
type Pipeline struct {
	store       PipelineStore
	bus         domain.EventBus
	aptResolver *APTPackageResolver
}

// NewPipeline creates a normalization pipeline with the given store, event bus, and APT resolver.
// aptResolver may be nil, in which case deb/apt entries will not have binary fetch states created.
func NewPipeline(store PipelineStore, bus domain.EventBus, aptResolver *APTPackageResolver) *Pipeline {
	return &Pipeline{store: store, bus: bus, aptResolver: aptResolver}
}

// Sync fetches entries from the given feed and normalizes them into the catalog.
func (p *Pipeline) Sync(ctx context.Context, feed feeds.Feed) error {
	source, err := p.store.GetFeedSourceByName(ctx, feed.Name())
	if err != nil {
		return fmt.Errorf("pipeline sync %s: get feed source: %w", feed.Name(), err)
	}

	state, err := p.store.GetFeedSyncState(ctx, source.ID)
	if err != nil {
		return fmt.Errorf("pipeline sync %s: get sync state: %w", feed.Name(), err)
	}

	if err := p.store.UpdateFeedSyncStateStart(ctx, source.ID); err != nil {
		return fmt.Errorf("pipeline sync %s: mark sync start: %w", feed.Name(), err)
	}

	rawEntries, nextCursor, err := feed.Fetch(ctx, state.Cursor)
	if err != nil {
		p.recordSyncError(ctx, source.ID, feed.Name(), err)
		return fmt.Errorf("pipeline sync %s: fetch: %w", feed.Name(), err)
	}

	var ingested int64
	for i := range rawEntries {
		entry := &rawEntries[i]
		if vErr := entry.Validate(); vErr != nil {
			slog.WarnContext(ctx, "pipeline sync: skipping invalid entry",
				"feed", feed.Name(),
				"error", vErr.Error(),
			)
			continue
		}

		// CVE-only entries (NVD, CISA KEV) contribute vulnerability data but
		// should NOT create patch catalog entries. They only upsert CVE feed records.
		if entry.CVEOnly {
			for _, cveID := range entry.CVEs {
				if _, cErr := p.ensureCVEFeed(ctx, cveID, feed.Name(), entry); cErr != nil {
					slog.ErrorContext(ctx, "pipeline sync: ensure CVE feed failed",
						"feed", feed.Name(),
						"cve", cveID,
						"error", cErr.Error(),
					)
				}
			}
			ingested++
			continue
		}

		catalogEntry, uErr := p.store.UpsertCatalogEntryFromFeed(ctx, sqlcgen.UpsertCatalogEntryFromFeedParams{
			Name:          entry.Name,
			Vendor:        entry.Vendor,
			OsFamily:      entry.OSFamily,
			Version:       entry.Version,
			Severity:      entry.Severity,
			ReleaseDate:   pgtype.Timestamptz{Time: entry.ReleaseDate, Valid: !entry.ReleaseDate.IsZero()},
			Description:   pgtype.Text{String: entry.Summary, Valid: entry.Summary != ""},
			FeedSourceID:  source.ID,
			SourceUrl:     entry.SourceURL,
			InstallerType: entry.InstallerType,
			Product:       entry.Product,
			OsPackageName: resolveOsPackageName(entry),
			SilentArgs:    entry.SilentArgs,
		})
		if uErr != nil {
			slog.ErrorContext(ctx, "pipeline sync: upsert catalog entry failed",
				"feed", feed.Name(),
				"entry", entry.Name,
				"error", uErr.Error(),
			)
			continue
		}

		// Only create binary fetch state for entries with resolvable download URLs.
		// rpm/yum (Red Hat) require subscriptions, windows_update requires browser scraping,
		// pkg (Apple) CDN URLs are not constructable — skip all of these.
		if entry.InstallerType == "deb" || entry.InstallerType == "apt" {
			if p.aptResolver != nil {
				if fetchURL := p.aptResolver.Resolve(ctx, entry.Product, entry.Version); fetchURL != "" {
					// Map vendor to distribution name for binary storage path.
					dist := entry.OSFamily
					switch entry.Vendor {
					case "canonical":
						dist = "ubuntu"
					}
					osVer := ""
					if len(entry.OSVersions) > 0 {
						osVer = entry.OSVersions[0]
					}
					if _, bfsErr := p.store.CreateBinaryFetchState(ctx, sqlcgen.CreateBinaryFetchStateParams{
						CatalogID:      catalogEntry.ID,
						OsDistribution: dist,
						OsVersion:      osVer,
						FetchUrl:       pgtype.Text{String: fetchURL, Valid: true},
					}); bfsErr != nil {
						// ON CONFLICT DO NOTHING — duplicates are expected and safe.
						slog.DebugContext(ctx, "pipeline sync: create binary fetch state (may already exist)",
							"entry", entry.Name, "error", bfsErr)
					}
				}
			}
		}

		for _, cveID := range entry.CVEs {
			cveFeed, cErr := p.ensureCVEFeed(ctx, cveID, feed.Name(), entry)
			if cErr != nil {
				slog.ErrorContext(ctx, "pipeline sync: ensure CVE feed failed",
					"feed", feed.Name(),
					"cve", cveID,
					"error", cErr.Error(),
				)
				continue
			}
			if lErr := p.store.LinkCatalogCVE(ctx, sqlcgen.LinkCatalogCVEParams{
				CatalogID: catalogEntry.ID,
				CveID:     cveFeed.ID,
			}); lErr != nil {
				slog.ErrorContext(ctx, "pipeline sync: link catalog CVE failed",
					"feed", feed.Name(),
					"cve", cveID,
					"error", lErr.Error(),
				)
			}
		}

		p.emitEvent(ctx, events.CatalogCreated, "patch_catalog", uuidToString(catalogEntry.ID), "created", nil)
		ingested++
	}

	nextSyncAt := time.Now().UTC().Add(time.Duration(source.SyncIntervalSeconds) * time.Second)
	if err := p.store.UpdateFeedSyncStateSuccess(ctx, sqlcgen.UpdateFeedSyncStateSuccessParams{
		FeedSourceID:    source.ID,
		NextSyncAt:      pgtype.Timestamptz{Time: nextSyncAt, Valid: true},
		Cursor:          nextCursor,
		EntriesIngested: ingested,
	}); err != nil {
		return fmt.Errorf("pipeline sync %s: update sync state success: %w", feed.Name(), err)
	}

	p.emitEvent(ctx, events.FeedSyncCompleted, "feed_source", uuidToString(source.ID), "sync_completed", map[string]any{
		"feed":     feed.Name(),
		"ingested": ingested,
		"cursor":   nextCursor,
	})

	return nil
}

// ensureCVEFeed upserts a CVE record with all available enrichment data from the entry.
// On conflict, non-empty fields from the new entry merge with existing data.
func (p *Pipeline) ensureCVEFeed(ctx context.Context, cveID string, feedName string, entry *feeds.RawEntry) (sqlcgen.CVEFeed, error) {
	refs, marshalErr := json.Marshal(entry.References)
	if marshalErr != nil {
		slog.ErrorContext(ctx, "pipeline: marshal CVE references",
			"cve_id", cveID, "error", marshalErr)
		refs = []byte("[]")
	}

	params := sqlcgen.UpsertCVEFeedParams{
		CveID:              cveID,
		Severity:           entry.Severity,
		Description:        pgtype.Text{String: entry.Summary, Valid: entry.Summary != ""},
		Source:             feedName,
		CvssV3Vector:       pgtype.Text{String: entry.CVSSv3Vector, Valid: entry.CVSSv3Vector != ""},
		AttackVector:       pgtype.Text{String: entry.AttackVector, Valid: entry.AttackVector != ""},
		CweID:              pgtype.Text{String: entry.CweID, Valid: entry.CweID != ""},
		ExternalReferences: refs,
		InKev:              feedName == "cisa_kev",
	}

	if entry.CVSSScore > 0 {
		params.CvssV3Score = pgtype.Numeric{
			Int:   big.NewInt(int64(entry.CVSSScore * 10)),
			Exp:   -1,
			Valid: true,
		}
	}

	if !entry.ReleaseDate.IsZero() {
		params.PublishedAt = pgtype.Timestamptz{Time: entry.ReleaseDate, Valid: true}
	}

	if entry.CISAKEVDueDate != nil {
		params.CisaKevDueDate = pgtype.Date{Time: *entry.CISAKEVDueDate, Valid: true}
	}

	if entry.NVDLastModified != nil {
		params.NvdLastModified = pgtype.Timestamptz{Time: *entry.NVDLastModified, Valid: true}
	}

	result, err := p.store.UpsertCVEFeed(ctx, params)
	if err != nil {
		return sqlcgen.CVEFeed{}, fmt.Errorf("pipeline sync: upsert CVE feed %s: %w", cveID, err)
	}

	p.emitEvent(ctx, events.CVEFeedEnriched, "cve_feed", cveID, "enriched", map[string]any{
		"feed":   feedName,
		"cve_id": cveID,
	})

	return result, nil
}

// recordSyncError updates sync state to error and emits a feed.sync_failed event.
func (p *Pipeline) recordSyncError(ctx context.Context, feedSourceID pgtype.UUID, feedName string, syncErr error) {
	if err := p.store.UpdateFeedSyncStateError(ctx, sqlcgen.UpdateFeedSyncStateErrorParams{
		FeedSourceID: feedSourceID,
		LastError:    pgtype.Text{String: syncErr.Error(), Valid: true},
	}); err != nil {
		slog.ErrorContext(ctx, "pipeline sync: failed to record sync error",
			"feed", feedName,
			"error", err.Error(),
		)
	}

	p.emitEvent(ctx, events.FeedSyncFailed, "feed_source", uuidToString(feedSourceID), "sync_failed", map[string]any{
		"feed":  feedName,
		"error": syncErr.Error(),
	})
}

// emitEvent publishes a domain event, logging any emission errors without failing the sync.
func (p *Pipeline) emitEvent(ctx context.Context, eventType, resource, resourceID, action string, payload any) {
	evt := domain.NewSystemEvent(eventType, defaultTenantID, resource, resourceID, action, payload)
	if err := p.bus.Emit(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "pipeline sync: failed to emit event",
			"event_type", eventType,
			"error", err.Error(),
		)
	}
}

// resolveOsPackageName returns the best package name for catalog matching.
// For KB-prefixed entries (MSRC/Windows), use the KB name since that's what
// agents report in endpoint_packages.package_name.
func resolveOsPackageName(entry *feeds.RawEntry) string {
	if strings.HasPrefix(entry.Name, "KB") {
		return entry.Name
	}
	if entry.Product != "" {
		return entry.Product
	}
	return entry.Name
}

// uuidToString converts a pgtype.UUID to its string representation.
func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	b := id.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
