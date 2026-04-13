// TODO(#329): Wire MSUFetcher into feed registry when Windows Update binary fetching is supported.
package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"time"
)

// MSUFetcher downloads Windows MSU/CAB update packages from Microsoft servers
// and uploads them to object storage.
type MSUFetcher struct {
	client *http.Client
	store  ObjectStore
	bucket string
}

// NewMSUFetcher creates an MSUFetcher that stores binaries in the given object store.
func NewMSUFetcher(store ObjectStore, bucket string) *MSUFetcher {
	return &MSUFetcher{
		client: &http.Client{Timeout: 10 * time.Minute},
		store:  store,
		bucket: bucket,
	}
}

// FetchBinary downloads a Windows MSU/CAB package from fetchURL, computes its SHA256
// checksum during upload, and stores it under key msu/{filename}.
// Returns (binary_ref, checksum_sha256, file_size, error).
func (f *MSUFetcher) FetchBinary(ctx context.Context, fetchURL, osFamily, osVersion, filename string) (string, string, int64, error) {
	slog.InfoContext(ctx, "msu: fetching windows update package",
		"url", fetchURL,
		"os_family", osFamily,
		"os_version", osVersion,
		"filename", filename,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("msu: fetch %s: create request: %w", filename, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("msu: fetch %s: download: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("msu: fetch %s: unexpected status %d", filename, resp.StatusCode)
	}

	key := path.Join("msu", filename)
	counted := &countingReader{r: resp.Body}
	ref, checksum, err := uploadBinaryWithKey(ctx, f.store, f.bucket, key, counted, resp.ContentLength)
	if err != nil {
		return "", "", 0, fmt.Errorf("msu: fetch %s: upload: %w", filename, err)
	}

	slog.InfoContext(ctx, "msu: binary fetched and stored",
		"binary_ref", ref,
		"checksum_sha256", checksum,
		"size", counted.n,
	)

	return ref, checksum, counted.n, nil
}
