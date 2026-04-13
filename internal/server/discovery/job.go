package discovery

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/shared/config"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// DiscoveryJobArgs is the payload for a patch discovery River job.
type DiscoveryJobArgs struct {
	TenantID string `json:"tenant_id"`
	RepoName string `json:"repo_name,omitempty"`
}

// Kind returns the unique job type identifier for River.
func (DiscoveryJobArgs) Kind() string { return "patch_discovery" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (DiscoveryJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "background"}
}

// DiscoveryWorker processes patch discovery jobs via River.
type DiscoveryWorker struct {
	river.WorkerDefaults[DiscoveryJobArgs]
	svc *Service
	cfg config.DiscoveryConfig
}

// NewDiscoveryWorker creates a DiscoveryWorker with the given service and config.
func NewDiscoveryWorker(svc *Service, cfg config.DiscoveryConfig) *DiscoveryWorker {
	return &DiscoveryWorker{svc: svc, cfg: cfg}
}

// Work processes a single discovery job: iterates configured repositories,
// optionally filtering by RepoName, skipping disabled repos.
func (w *DiscoveryWorker) Work(ctx context.Context, job *river.Job[DiscoveryJobArgs]) error {
	ctx = tenant.WithTenantID(ctx, job.Args.TenantID)

	var failCount, enabledCount int
	for _, repo := range w.cfg.Repositories {
		if !repo.Enabled {
			slog.InfoContext(ctx, "discovery worker: skipping disabled repo", "repo", repo.Name)
			continue
		}
		if job.Args.RepoName != "" && repo.Name != job.Args.RepoName {
			continue
		}
		enabledCount++

		if _, err := w.svc.DiscoverRepo(ctx, job.Args.TenantID, repo); err != nil {
			slog.ErrorContext(ctx, "discovery worker: repo sync failed",
				"repo", repo.Name,
				"error", err,
			)
			failCount++
			continue
		}
	}

	if failCount > 0 && failCount == enabledCount {
		return fmt.Errorf("discovery worker: all %d repos failed", failCount)
	}
	return nil
}
