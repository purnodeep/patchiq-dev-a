package workers

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// BinaryFetchJobArgs defines the River job for fetching patch binaries.
type BinaryFetchJobArgs struct{}

// Kind returns the unique job kind identifier.
func (BinaryFetchJobArgs) Kind() string { return "hub_binary_fetch" }

// BinaryFetchStore defines the queries needed by BinaryFetchWorker.
type BinaryFetchStore interface {
	ListPendingBinaryFetches(ctx context.Context, limit int32) ([]sqlcgen.BinaryFetchState, error)
	UpdateBinaryFetchSuccess(ctx context.Context, arg sqlcgen.UpdateBinaryFetchSuccessParams) error
	UpdateBinaryFetchFailed(ctx context.Context, arg sqlcgen.UpdateBinaryFetchFailedParams) error
	UpdateCatalogEntryBinaryRef(ctx context.Context, arg sqlcgen.UpdateCatalogEntryBinaryRefParams) error
}

// BinaryDownloader downloads a binary from a URL and stores it in object storage.
type BinaryDownloader interface {
	FetchAndStore(ctx context.Context, vendorURL, osFamily, osVersion, filename string) (string, string, int64, error)
}

// BinaryFetchWorker processes pending binary fetch requests.
type BinaryFetchWorker struct {
	river.WorkerDefaults[BinaryFetchJobArgs]
	store      BinaryFetchStore
	downloader BinaryDownloader
	eventBus   domain.EventBus
}

// NewBinaryFetchWorker creates a new BinaryFetchWorker.
func NewBinaryFetchWorker(store BinaryFetchStore, downloader BinaryDownloader, eventBus domain.EventBus) *BinaryFetchWorker {
	return &BinaryFetchWorker{
		store:      store,
		downloader: downloader,
		eventBus:   eventBus,
	}
}

// SetDownloader configures the binary downloader after construction.
// This is needed because MinIO is initialized after River workers are registered.
func (w *BinaryFetchWorker) SetDownloader(d BinaryDownloader) {
	w.downloader = d
}

// Timeout allows up to 30 minutes for binary downloads.
func (w *BinaryFetchWorker) Timeout(*river.Job[BinaryFetchJobArgs]) time.Duration {
	return 30 * time.Minute
}

// Work fetches pending binary downloads and processes them.
func (w *BinaryFetchWorker) Work(ctx context.Context, _ *river.Job[BinaryFetchJobArgs]) error {
	pending, err := w.store.ListPendingBinaryFetches(ctx, 10)
	if err != nil {
		return fmt.Errorf("binary fetch: list pending: %w", err)
	}

	if len(pending) == 0 {
		slog.InfoContext(ctx, "binary fetch: no pending fetches")
		return nil
	}

	slog.InfoContext(ctx, "binary fetch: processing pending", "count", len(pending))

	var failed int
	for _, fetch := range pending {
		if err := w.processFetch(ctx, fetch); err != nil {
			slog.ErrorContext(ctx, "binary fetch: process failed",
				"catalog_id", uuidToStr(fetch.CatalogID),
				"error", err,
			)
			failed++
		}
	}
	if failed > 0 && failed == len(pending) {
		return fmt.Errorf("binary fetch: all %d fetches failed", failed)
	}
	return nil
}

func (w *BinaryFetchWorker) processFetch(ctx context.Context, fetch sqlcgen.BinaryFetchState) error {
	if !fetch.FetchUrl.Valid || fetch.FetchUrl.String == "" {
		return w.failFetch(ctx, fetch.ID, "no fetch URL configured")
	}

	if w.downloader == nil {
		return w.failFetch(ctx, fetch.ID, "binary downloader not configured (MinIO not enabled)")
	}

	fetchURL := fetch.FetchUrl.String
	filename := filepath.Base(fetchURL)
	if filename == "." || filename == "/" {
		filename = fmt.Sprintf("patch-%s", uuidToStr(fetch.CatalogID))
	}
	ref, checksum, size, err := w.downloader.FetchAndStore(ctx, fetchURL, fetch.OsDistribution, fetch.OsVersion, filename)
	if err != nil {
		return w.failFetch(ctx, fetch.ID, err.Error())
	}

	if err := w.store.UpdateBinaryFetchSuccess(ctx, sqlcgen.UpdateBinaryFetchSuccessParams{
		ID:             fetch.ID,
		BinaryRef:      pgtype.Text{String: ref, Valid: true},
		ChecksumSha256: pgtype.Text{String: checksum, Valid: true},
		FileSizeBytes:  pgtype.Int8{Int64: size, Valid: size > 0},
	}); err != nil {
		return fmt.Errorf("binary fetch: update success: %w", err)
	}

	if err := w.store.UpdateCatalogEntryBinaryRef(ctx, sqlcgen.UpdateCatalogEntryBinaryRefParams{
		ID:             fetch.CatalogID,
		BinaryRef:      ref,
		ChecksumSha256: checksum,
	}); err != nil {
		slog.ErrorContext(ctx, "binary fetch: update catalog binary ref", "error", err)
	}

	w.emitEvent(ctx, events.BinaryFetched, uuidToStr(fetch.CatalogID), map[string]any{
		"binary_ref": ref,
		"checksum":   checksum,
	})

	return nil
}

func (w *BinaryFetchWorker) failFetch(ctx context.Context, id pgtype.UUID, errMsg string) error {
	if err := w.store.UpdateBinaryFetchFailed(ctx, sqlcgen.UpdateBinaryFetchFailedParams{
		ID:           id,
		ErrorMessage: pgtype.Text{String: errMsg, Valid: true},
	}); err != nil {
		slog.ErrorContext(ctx, "binary fetch: update failed state", "error", err)
	}

	w.emitEvent(ctx, events.BinaryFetchFailed, uuidToStr(id), map[string]any{
		"error": errMsg,
	})

	return fmt.Errorf("binary fetch failed: %s", errMsg)
}

func (w *BinaryFetchWorker) emitEvent(ctx context.Context, eventType, resourceID string, payload any) {
	if w.eventBus == nil {
		return
	}
	evt := domain.NewSystemEvent(eventType, "", "binary_fetch", resourceID, eventType, payload)
	if err := w.eventBus.Emit(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "binary fetch: emit event failed",
			"event_type", eventType,
			"error", err,
		)
	}
}

func uuidToStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}
