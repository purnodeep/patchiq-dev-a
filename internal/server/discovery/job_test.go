package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/shared/config"
)

func TestDiscoveryJobArgs_Kind(t *testing.T) {
	args := DiscoveryJobArgs{TenantID: "tenant-1"}
	if got := args.Kind(); got != "patch_discovery" {
		t.Errorf("Kind() = %q, want %q", got, "patch_discovery")
	}
}

// Compile-time check that DiscoveryJobArgs implements river.JobArgs.
var _ river.JobArgs = DiscoveryJobArgs{}

func TestDiscoveryWorker_DisabledRepoSkipped(t *testing.T) {
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	svc := NewService(upserter, emitter, nil)

	cfg := config.DiscoveryConfig{
		Repositories: []config.RepositoryConfig{
			{Name: "disabled-repo", Type: "apt", Enabled: false},
		},
	}
	w := NewDiscoveryWorker(svc, cfg)

	job := &river.Job[DiscoveryJobArgs]{
		Args: DiscoveryJobArgs{TenantID: "tenant-1"},
	}

	err := w.Work(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No batches should have been created since the only repo is disabled
	if len(upserter.batches) != 0 {
		t.Errorf("expected 0 batches, got %d", len(upserter.batches))
	}
}

func TestDiscoveryWorker_RepoNameFilter(t *testing.T) {
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	fetcher := NewFetcher(5*time.Second, 1)
	svc := NewService(upserter, emitter, fetcher)

	cfg := config.DiscoveryConfig{
		Repositories: []config.RepositoryConfig{
			{Name: "repo-a", Type: "apt", Enabled: true, URL: "http://192.0.2.1:1/invalid"},
			{Name: "repo-b", Type: "apt", Enabled: true, URL: "http://192.0.2.1:1/invalid"},
		},
	}
	w := NewDiscoveryWorker(svc, cfg)

	job := &river.Job[DiscoveryJobArgs]{
		Args: DiscoveryJobArgs{TenantID: "tenant-1", RepoName: "repo-a"},
	}

	// repo-b is filtered out by RepoName. Only repo-a is attempted (and fails).
	// All matched+enabled repos fail = error returned.
	err := w.Work(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when repo fetch fails")
	}
}

func TestDiscoveryWorker_PartialFailureReturnsNil(t *testing.T) {
	// One repo succeeds (via mock discoverFromReader), one fails (invalid URL).
	// Since not all enabled repos fail, Work() should return nil.
	raw := "Package: curl\nVersion: 7.81.0\nArchitecture: amd64\nSHA256: abc123\n"
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	fetcher := NewFetcher(2*time.Second, 1)
	svc := NewService(upserter, emitter, fetcher)

	// We can't use discoverFromReader here since Work() calls DiscoverRepo which uses Fetch.
	// Instead, set up one repo with a working httptest server and one with an invalid URL.
	srv := newTestGzipServer(t, raw)
	defer srv.Close()

	cfg := config.DiscoveryConfig{
		Repositories: []config.RepositoryConfig{
			{Name: "good-repo", Type: "apt", Enabled: true, URL: srv.URL, OsFamily: "debian", OsDistro: "ubuntu-22.04"},
			{Name: "bad-repo", Type: "apt", Enabled: true, URL: "http://192.0.2.1:1/invalid", OsFamily: "debian", OsDistro: "ubuntu-22.04"},
		},
	}
	w := NewDiscoveryWorker(svc, cfg)

	job := &river.Job[DiscoveryJobArgs]{
		Args: DiscoveryJobArgs{TenantID: "tenant-1"},
	}

	err := w.Work(context.Background(), job)
	if err != nil {
		t.Fatalf("expected nil error for partial failure, got: %v", err)
	}
	// At least one batch should have been created from the good repo.
	if len(upserter.batches) < 1 {
		t.Errorf("expected >= 1 batch, got %d", len(upserter.batches))
	}
}

func newTestGzipServer(t *testing.T, data string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(gzipBytes(t, data)) //nolint:errcheck
	}))
}

type failingUpserter struct{}

func (f *failingUpserter) BeginBatch(_ context.Context, _ string) (BatchUpserter, error) {
	return &mockBatchUpserter{}, nil
}

func TestDiscoveryWorker_AllReposFailReturnsError(t *testing.T) {
	upserter := &failingUpserter{}
	emitter := &mockEventEmitter{}
	fetcher := NewFetcher(5*time.Second, 1)
	svc := NewService(upserter, emitter, fetcher)

	cfg := config.DiscoveryConfig{
		Repositories: []config.RepositoryConfig{
			{Name: "repo-a", Type: "apt", Enabled: true, URL: "http://192.0.2.1:1/invalid"},
			{Name: "repo-b", Type: "apt", Enabled: true, URL: "http://192.0.2.1:1/invalid"},
		},
	}
	w := NewDiscoveryWorker(svc, cfg)

	job := &river.Job[DiscoveryJobArgs]{
		Args: DiscoveryJobArgs{TenantID: "tenant-1"},
	}

	err := w.Work(context.Background(), job)
	if err == nil {
		t.Fatal("expected error when all repos fail")
	}
}
