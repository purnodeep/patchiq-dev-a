package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
)

// BinaryStore abstracts downloading a binary from object storage.
// It complements ObjectStore (write-side) with a read path.
type BinaryStore interface {
	// GetObject returns a ReadCloser for the object at bucket/key.
	// It also returns the object size (-1 if unknown).
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error)
}

// CatalogBinaryQuerier abstracts catalog queries needed for binary serving.
type CatalogBinaryQuerier interface {
	GetCatalogEntryByID(ctx context.Context, id pgtype.UUID) (sqlcgen.PatchCatalog, error)
}

// CatalogBinaryHandler serves binary download endpoints for catalog entries.
type CatalogBinaryHandler struct {
	queries     CatalogBinaryQuerier
	binaryStore BinaryStore
	bucket      string
}

// NewCatalogBinaryHandler creates a handler that proxies MinIO objects to callers.
func NewCatalogBinaryHandler(queries CatalogBinaryQuerier, binaryStore BinaryStore, bucket string) *CatalogBinaryHandler {
	return &CatalogBinaryHandler{queries: queries, binaryStore: binaryStore, bucket: bucket}
}

// GetBinary handles GET /catalog/{id}/binary.
// It looks up the catalog entry, fetches the binary from MinIO, and streams it to the client.
// Authentication is expected to be handled by upstream middleware (same sync API key pattern).
func (h *CatalogBinaryHandler) GetBinary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid catalog id: %s", err))
		return
	}

	entry, err := h.queries.GetCatalogEntryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "catalog entry not found")
			return
		}
		slog.ErrorContext(ctx, "catalog binary: get entry", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "get catalog entry: internal error")
		return
	}

	if entry.BinaryRef == "" {
		writeJSONError(w, http.StatusNotFound, "no binary available for this catalog entry")
		return
	}

	rc, size, err := h.binaryStore.GetObject(ctx, h.bucket, entry.BinaryRef)
	if err != nil {
		slog.ErrorContext(ctx, "catalog binary: fetch from object store",
			"binary_ref", entry.BinaryRef,
			"error", err,
		)
		writeJSONError(w, http.StatusInternalServerError, "fetch binary: internal error")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	if size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	}

	if _, err := io.Copy(w, rc); err != nil {
		slog.ErrorContext(ctx, "catalog binary: stream to client",
			"binary_ref", entry.BinaryRef,
			"error", err,
		)
	}
}
