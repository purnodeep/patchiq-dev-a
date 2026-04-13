package catalog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"time"
)

// APTFetcher downloads .deb packages from Ubuntu/Debian APT repositories
// and uploads them to object storage.
type APTFetcher struct {
	client *http.Client
	store  ObjectStore
	bucket string
}

// NewAPTFetcher creates an APTFetcher that stores binaries in the given object store.
func NewAPTFetcher(store ObjectStore, bucket string) *APTFetcher {
	return &APTFetcher{
		client: &http.Client{Timeout: 10 * time.Minute},
		store:  store,
		bucket: bucket,
	}
}

// FetchBinary downloads a .deb package from fetchURL, computes its SHA256 checksum
// during upload, and stores it under key apt/{osVersion}/{filename}.
// Returns (binary_ref, checksum_sha256, file_size, error).
func (f *APTFetcher) FetchBinary(ctx context.Context, fetchURL, osFamily, osVersion, filename string) (string, string, int64, error) {
	slog.InfoContext(ctx, "apt: fetching deb package",
		"url", fetchURL,
		"os_family", osFamily,
		"os_version", osVersion,
		"filename", filename,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("apt: fetch %s: create request: %w", filename, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("apt: fetch %s: download: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("apt: fetch %s: unexpected status %d", filename, resp.StatusCode)
	}

	key := path.Join("apt", osVersion, filename)
	counted := &countingReader{r: resp.Body}
	ref, checksum, err := uploadBinaryWithKey(ctx, f.store, f.bucket, key, counted, resp.ContentLength)
	if err != nil {
		return "", "", 0, fmt.Errorf("apt: fetch %s: upload: %w", filename, err)
	}

	slog.InfoContext(ctx, "apt: binary fetched and stored",
		"binary_ref", ref,
		"checksum_sha256", checksum,
		"size", counted.n,
	)

	return ref, checksum, counted.n, nil
}

// countingReader wraps an io.Reader and counts total bytes read.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
