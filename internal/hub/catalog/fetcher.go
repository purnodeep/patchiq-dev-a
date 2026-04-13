package catalog

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// BinaryFetcher downloads patch binaries from vendor URLs and uploads them to object storage.
type BinaryFetcher struct {
	store  ObjectStore
	bucket string
	client *http.Client
}

// NewBinaryFetcher creates a fetcher that downloads from vendor URLs and stores in the given object store.
func NewBinaryFetcher(store ObjectStore, bucket string, client *http.Client) *BinaryFetcher {
	if client == nil {
		client = &http.Client{}
	}
	return &BinaryFetcher{
		store:  store,
		bucket: bucket,
		client: client,
	}
}

// FetchAndStore downloads a binary from vendorURL, uploads it to object storage,
// and returns the object key (binary_ref), SHA256 checksum, and file size in bytes.
func (f *BinaryFetcher) FetchAndStore(ctx context.Context, vendorURL, osFamily, osVersion, filename string) (string, string, int64, error) {
	slog.InfoContext(ctx, "fetching binary from vendor",
		"url", vendorURL,
		"os_family", osFamily,
		"os_version", osVersion,
		"filename", filename,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, vendorURL, nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch binary %s: create request: %w", filename, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch binary %s: download: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("fetch binary %s: unexpected status %d", filename, resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "text/html") {
		return "", "", 0, fmt.Errorf("fetch binary %s: got HTML response instead of binary (content-type: %s)", filename, ct)
	}

	size := resp.ContentLength

	ref, checksum, err := uploadBinary(ctx, f.store, f.bucket, osFamily, osVersion, filename, resp.Body, resp.ContentLength)
	if err != nil {
		return "", "", 0, fmt.Errorf("fetch binary %s: upload: %w", filename, err)
	}

	slog.InfoContext(ctx, "binary fetched and stored",
		"binary_ref", ref,
		"checksum_sha256", checksum,
		"size_bytes", size,
	)

	return ref, checksum, size, nil
}
