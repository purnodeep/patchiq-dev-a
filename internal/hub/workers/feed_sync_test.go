package workers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/hub/feeds"
)

// stubFeed implements feeds.Feed for testing.
type stubFeed struct {
	name string
}

func (f *stubFeed) Name() string { return f.name }

func (f *stubFeed) Fetch(_ context.Context, _ string) ([]feeds.RawEntry, string, error) {
	return nil, "", nil
}

// spySyncer records calls to Sync.
type spySyncer struct {
	called   bool
	feedName string
	err      error
}

func (s *spySyncer) Sync(_ context.Context, feed feeds.Feed) error {
	s.called = true
	s.feedName = feed.Name()
	return s.err
}

var _ river.JobArgs = FeedSyncJobArgs{}

func TestFeedSyncJobArgs_Kind(t *testing.T) {
	args := FeedSyncJobArgs{FeedName: "nvd"}
	if got := args.Kind(); got != "hub_feed_sync" {
		t.Errorf("Kind() = %q, want hub_feed_sync", got)
	}
}

func TestFeedSyncWorker_Work(t *testing.T) {
	t.Run("calls syncer with correct feed", func(t *testing.T) {
		feed := &stubFeed{name: "nvd"}
		syncer := &spySyncer{}
		worker := NewFeedSyncWorker(map[string]feeds.Feed{"nvd": feed}, syncer)

		err := worker.Work(context.Background(), &river.Job[FeedSyncJobArgs]{
			Args: FeedSyncJobArgs{FeedName: "nvd"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !syncer.called {
			t.Fatal("expected syncer.Sync to be called")
		}
		if syncer.feedName != "nvd" {
			t.Errorf("syncer called with feed %q, want nvd", syncer.feedName)
		}
	})

	t.Run("returns error for unknown feed", func(t *testing.T) {
		worker := NewFeedSyncWorker(map[string]feeds.Feed{}, nil)

		err := worker.Work(context.Background(), &river.Job[FeedSyncJobArgs]{
			Args: FeedSyncJobArgs{FeedName: "unknown"},
		})
		if err == nil {
			t.Fatal("expected error for unknown feed")
		}
		if !strings.Contains(err.Error(), "unknown feed") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("propagates syncer error", func(t *testing.T) {
		feed := &stubFeed{name: "cisa_kev"}
		syncErr := errors.New("connection refused")
		syncer := &spySyncer{err: syncErr}
		worker := NewFeedSyncWorker(map[string]feeds.Feed{"cisa_kev": feed}, syncer)

		err := worker.Work(context.Background(), &river.Job[FeedSyncJobArgs]{
			Args: FeedSyncJobArgs{FeedName: "cisa_kev"},
		})
		if err == nil {
			t.Fatal("expected error from syncer")
		}
		if !errors.Is(err, syncErr) {
			t.Errorf("expected wrapped syncErr, got: %v", err)
		}
	})
}

func TestRegisterWorkers(t *testing.T) {
	feed := &stubFeed{name: "test"}
	syncer := &spySyncer{}
	feedWorker := NewFeedSyncWorker(map[string]feeds.Feed{"test": feed}, syncer)

	workers := RegisterWorkers(feedWorker, nil)
	if workers == nil {
		t.Fatal("expected non-nil workers bundle")
	}
}
