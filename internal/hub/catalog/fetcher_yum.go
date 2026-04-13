// TODO(#329): Wire YUMFetcher into feed registry when RPM binary fetching is supported.
package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"time"
)

// YUMFetcher downloads .rpm packages from RHEL/CentOS/Fedora YUM/DNF repositories
// and uploads them to object storage.
type YUMFetcher struct {
	client *http.Client
	store  ObjectStore
	bucket string
}

// NewYUMFetcher creates a YUMFetcher that stores binaries in the given object store.
func NewYUMFetcher(store ObjectStore, bucket string) *YUMFetcher {
	return &YUMFetcher{
		client: &http.Client{Timeout: 10 * time.Minute},
		store:  store,
		bucket: bucket,
	}
}

// FetchBinary downloads an .rpm package from fetchURL, computes its SHA256 checksum
// during upload, and stores it under key yum/{osVersion}/{filename}.
// Returns (binary_ref, checksum_sha256, file_size, error).
func (f *YUMFetcher) FetchBinary(ctx context.Context, fetchURL, osFamily, osVersion, filename string) (string, string, int64, error) {
	slog.InfoContext(ctx, "yum: fetching rpm package",
		"url", fetchURL,
		"os_family", osFamily,
		"os_version", osVersion,
		"filename", filename,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("yum: fetch %s: create request: %w", filename, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("yum: fetch %s: download: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("yum: fetch %s: unexpected status %d", filename, resp.StatusCode)
	}

	key := path.Join("yum", osVersion, filename)
	counted := &countingReader{r: resp.Body}
	ref, checksum, err := uploadBinaryWithKey(ctx, f.store, f.bucket, key, counted, resp.ContentLength)
	if err != nil {
		return "", "", 0, fmt.Errorf("yum: fetch %s: upload: %w", filename, err)
	}

	slog.InfoContext(ctx, "yum: binary fetched and stored",
		"binary_ref", ref,
		"checksum_sha256", checksum,
		"size", counted.n,
	)

	return ref, checksum, counted.n, nil
}
