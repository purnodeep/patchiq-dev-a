// TODO(#329): Wire AppleFetcher into feed registry when macOS binary fetching is supported.
package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"time"
)

// AppleFetcher downloads macOS PKG/DMG update packages from Apple servers
// and uploads them to object storage.
type AppleFetcher struct {
	client *http.Client
	store  ObjectStore
	bucket string
}

// NewAppleFetcher creates an AppleFetcher that stores binaries in the given object store.
func NewAppleFetcher(store ObjectStore, bucket string) *AppleFetcher {
	return &AppleFetcher{
		client: &http.Client{Timeout: 10 * time.Minute},
		store:  store,
		bucket: bucket,
	}
}

// FetchBinary downloads a macOS PKG/DMG package from fetchURL, computes its SHA256
// checksum during upload, and stores it under key apple/{filename}.
// Returns (binary_ref, checksum_sha256, file_size, error).
func (f *AppleFetcher) FetchBinary(ctx context.Context, fetchURL, osFamily, osVersion, filename string) (string, string, int64, error) {
	slog.InfoContext(ctx, "apple: fetching macos update package",
		"url", fetchURL,
		"os_family", osFamily,
		"os_version", osVersion,
		"filename", filename,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("apple: fetch %s: create request: %w", filename, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("apple: fetch %s: download: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("apple: fetch %s: unexpected status %d", filename, resp.StatusCode)
	}

	key := path.Join("apple", filename)
	counted := &countingReader{r: resp.Body}
	ref, checksum, err := uploadBinaryWithKey(ctx, f.store, f.bucket, key, counted, resp.ContentLength)
	if err != nil {
		return "", "", 0, fmt.Errorf("apple: fetch %s: upload: %w", filename, err)
	}

	slog.InfoContext(ctx, "apple: binary fetched and stored",
		"binary_ref", ref,
		"checksum_sha256", checksum,
		"size", counted.n,
	)

	return ref, checksum, counted.n, nil
}
