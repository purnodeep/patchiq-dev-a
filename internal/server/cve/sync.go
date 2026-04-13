package cve

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	syncSourceNVD = "nvd"
	// defaultSyncLookback is how far back to look on the first sync when no cursor exists.
	defaultSyncLookback = 120 * 24 * time.Hour // 120 days
)

// CVEUpserter persists CVE records and manages sync cursors.
type CVEUpserter interface {
	UpsertCVE(ctx context.Context, tenantID string, rec CVERecord) (string, bool, error)
	UpsertCVEWithKEV(ctx context.Context, tenantID string, rec CVERecord, kevDueDate string, exploitAvailable bool) (string, bool, error)
	GetSyncCursor(ctx context.Context, tenantID, source string) (time.Time, error)
	UpdateSyncCursor(ctx context.Context, tenantID, source string, lastSynced time.Time) error
}

// CVEEventEmitter emits domain events for CVE operations.
type CVEEventEmitter interface {
	EmitCVEDiscovered(ctx context.Context, tenantID, cveDBID, cveID, severity string, cvss float64) error
	EmitCVERemediationAvailable(ctx context.Context, tenantID, cveID, patchID, packageName string) error
}

// CVEFetcher retrieves CVE data from external sources.
type CVEFetcher interface {
	FetchCVEs(ctx context.Context, since time.Time) ([]CVERecord, error)
	FetchKEV(ctx context.Context) (map[string]KEVVulnerability, error)
}

// NVDSyncService orchestrates NVD CVE feed ingestion: fetch, upsert, correlate, emit events.
type NVDSyncService struct {
	fetcher    CVEFetcher
	store      CVEUpserter
	events     CVEEventEmitter
	correlator *Correlator
}

// NewNVDSyncService creates an NVDSyncService with the required dependencies.
func NewNVDSyncService(fetcher CVEFetcher, store CVEUpserter, events CVEEventEmitter, correlator *Correlator) *NVDSyncService {
	return &NVDSyncService{
		fetcher:    fetcher,
		store:      store,
		events:     events,
		correlator: correlator,
	}
}

// SyncNVD fetches CVEs from NVD since the last sync cursor, upserts them,
// correlates with patches, and emits domain events.
func (s *NVDSyncService) SyncNVD(ctx context.Context, tenantID string) error {
	since, err := s.store.GetSyncCursor(ctx, tenantID, syncSourceNVD)
	if err != nil {
		return fmt.Errorf("sync NVD: get cursor: %w", err)
	}
	if since.IsZero() {
		since = time.Now().UTC().Add(-defaultSyncLookback)
		slog.InfoContext(ctx, "cve sync: no cursor found, using lookback default",
			"tenant_id", tenantID,
			"since", since,
		)
	}

	slog.InfoContext(ctx, "cve sync: fetching CVEs from NVD",
		"tenant_id", tenantID,
		"since", since,
	)

	records, err := s.fetcher.FetchCVEs(ctx, since)
	if err != nil {
		return fmt.Errorf("sync NVD: fetch CVEs: %w", err)
	}

	if len(records) == 0 {
		slog.InfoContext(ctx, "cve sync: no new CVEs", "tenant_id", tenantID)
		return s.store.UpdateSyncCursor(ctx, tenantID, syncSourceNVD, time.Now().UTC())
	}

	// Fetch KEV catalog to enrich CVEs with exploit/CISA data.
	kevMap, err := s.fetcher.FetchKEV(ctx)
	if err != nil {
		slog.WarnContext(ctx, "cve sync: KEV fetch failed, proceeding without KEV enrichment",
			"tenant_id", tenantID,
			"error", err,
		)
		kevMap = nil
	}

	cveDBIDs := make(map[string]string, len(records))
	var newCount, updatedCount int

	for _, rec := range records {
		var dbID string
		var isNew bool

		if kev, ok := kevMap[rec.CVEID]; ok {
			dbID, isNew, err = s.store.UpsertCVEWithKEV(ctx, tenantID, rec, kev.DueDate, true)
		} else {
			dbID, isNew, err = s.store.UpsertCVE(ctx, tenantID, rec)
		}
		if err != nil {
			return fmt.Errorf("sync NVD: upsert CVE %s: %w", rec.CVEID, err)
		}

		cveDBIDs[rec.CVEID] = dbID

		if isNew {
			newCount++
			if emitErr := s.events.EmitCVEDiscovered(ctx, tenantID, dbID, rec.CVEID, rec.Severity, rec.CVSSv3Score); emitErr != nil {
				slog.ErrorContext(ctx, "cve sync: emit cve.discovered failed",
					"cve_id", rec.CVEID,
					"error", emitErr,
				)
			}
		} else {
			updatedCount++
		}
	}

	// Correlate CVEs with patches.
	if s.correlator != nil {
		linked, correlateErr := s.correlator.Correlate(ctx, tenantID, records, cveDBIDs)
		if correlateErr != nil {
			slog.ErrorContext(ctx, "cve sync: correlation failed",
				"tenant_id", tenantID,
				"error", correlateErr,
			)
		} else if linked > 0 {
			slog.InfoContext(ctx, "cve sync: correlated CVEs with patches",
				"tenant_id", tenantID,
				"links_created", linked,
			)
		}
	}

	if err := s.store.UpdateSyncCursor(ctx, tenantID, syncSourceNVD, time.Now().UTC()); err != nil {
		return fmt.Errorf("sync NVD: update cursor: %w", err)
	}

	slog.InfoContext(ctx, "cve sync: complete",
		"tenant_id", tenantID,
		"total", len(records),
		"new", newCount,
		"updated", updatedCount,
	)

	return nil
}
