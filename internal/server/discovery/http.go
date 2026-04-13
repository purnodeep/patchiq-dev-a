package discovery

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Fetcher downloads repository metadata with timeout and retry.
type Fetcher struct {
	client     *http.Client
	maxRetries int
}

// NewFetcher creates a Fetcher with the given timeout and max retries.
func NewFetcher(timeout time.Duration, maxRetries int) *Fetcher {
	return &Fetcher{
		client:     &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
	}
}

// Fetch downloads the given URL, retrying on 5xx errors with exponential backoff.
// The caller must close the returned ReadCloser.
func (f *Fetcher) Fetch(ctx context.Context, url string) (io.ReadCloser, error) {
	var lastErr error
	for attempt := range f.maxRetries {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("fetch %s: %w", url, ctx.Err())
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: create request: %w", url, err)
		}

		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("fetch %s attempt %d: %w", url, attempt+1, err)
			slog.WarnContext(ctx, "fetch: attempt failed", "url", url, "attempt", attempt+1, "error", err)
			continue
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetch %s attempt %d: HTTP %d", url, attempt+1, resp.StatusCode)
			slog.WarnContext(ctx, "fetch: attempt failed", "url", url, "attempt", attempt+1, "status", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
		}

		return resp.Body, nil
	}
	return nil, fmt.Errorf("fetch %s: all %d retries exhausted: %w", url, f.maxRetries, lastErr)
}
