package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/hub/feeds"
)

// Syncer is the interface for feed sync operations.
type Syncer interface {
	Sync(ctx context.Context, feed feeds.Feed) error
}

// FeedSyncJobArgs identifies which feed to sync.
type FeedSyncJobArgs struct {
	FeedName string `json:"feed_name"`
}

// Kind returns the unique job kind identifier.
func (FeedSyncJobArgs) Kind() string { return "hub_feed_sync" }

// FeedSyncWorker runs a single feed sync via the pipeline.
type FeedSyncWorker struct {
	river.WorkerDefaults[FeedSyncJobArgs]
	feeds  map[string]feeds.Feed
	syncer Syncer
}

// NewFeedSyncWorker creates a FeedSyncWorker with the given feed registry and syncer.
func NewFeedSyncWorker(feedRegistry map[string]feeds.Feed, syncer Syncer) *FeedSyncWorker {
	return &FeedSyncWorker{
		feeds:  feedRegistry,
		syncer: syncer,
	}
}

// Timeout overrides River's default 1-minute job timeout. NVD full scans
// (no API key, 6s delay per page, ~135 pages) take ~14 minutes; 30 minutes
// provides safe margin for all feed types.
func (w *FeedSyncWorker) Timeout(*river.Job[FeedSyncJobArgs]) time.Duration {
	return 30 * time.Minute
}

// Work executes the feed sync for the feed specified in the job args.
func (w *FeedSyncWorker) Work(ctx context.Context, job *river.Job[FeedSyncJobArgs]) error {
	feed, ok := w.feeds[job.Args.FeedName]
	if !ok {
		return fmt.Errorf("feed sync: unknown feed %q", job.Args.FeedName)
	}
	return w.syncer.Sync(ctx, feed)
}
