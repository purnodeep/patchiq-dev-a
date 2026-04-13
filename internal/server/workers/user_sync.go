package workers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/riverqueue/river"
)

// UserSyncJobArgs defines the River periodic job for Zitadel user sync.
type UserSyncJobArgs struct{}

// Kind implements river.JobArgs.
func (UserSyncJobArgs) Kind() string { return "user_sync" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (UserSyncJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// ZitadelUser represents a user from the Zitadel API.
type ZitadelUser struct {
	ID          string
	Email       string
	DisplayName string
	Roles       []string
}

// ZitadelUserLister lists users from the Zitadel Management API.
type ZitadelUserLister interface {
	ListUsers(ctx context.Context) ([]ZitadelUser, error)
}

// UserEnsurer provisions or updates a user in PatchIQ.
type UserEnsurer interface {
	EnsureUser(ctx context.Context, u ZitadelUser) error
}

// UserSyncer syncs Zitadel users to PatchIQ.
type UserSyncer struct {
	client      ZitadelUserLister
	provisioner UserEnsurer
}

// NewUserSyncer creates a new UserSyncer.
func NewUserSyncer(client ZitadelUserLister, provisioner UserEnsurer) *UserSyncer {
	if client == nil {
		panic("workers: NewUserSyncer called with nil ZitadelUserLister")
	}
	if provisioner == nil {
		panic("workers: NewUserSyncer called with nil UserEnsurer")
	}
	return &UserSyncer{client: client, provisioner: provisioner}
}

// Sync fetches users from Zitadel and ensures they exist in PatchIQ.
func (s *UserSyncer) Sync(ctx context.Context) error {
	users, err := s.client.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("user sync: list zitadel users: %w", err)
	}

	var synced, failed int
	for _, u := range users {
		if err := s.provisioner.EnsureUser(ctx, u); err != nil {
			slog.ErrorContext(ctx, "user sync: ensure user failed",
				"user_id", u.ID, "email", u.Email, "error", err)
			failed++
			continue
		}
		synced++
	}

	slog.InfoContext(ctx, "user sync complete",
		"total", len(users), "synced", synced, "failed", failed)

	if failed > 0 && synced == 0 {
		return fmt.Errorf("user sync: all %d users failed to sync", failed)
	}
	return nil
}

// UserSyncWorker wraps UserSyncer as a River worker.
type UserSyncWorker struct {
	river.WorkerDefaults[UserSyncJobArgs]
	syncer *UserSyncer
}

// NewUserSyncWorker creates a new UserSyncWorker.
func NewUserSyncWorker(syncer *UserSyncer) *UserSyncWorker {
	if syncer == nil {
		panic("workers: NewUserSyncWorker called with nil UserSyncer")
	}
	return &UserSyncWorker{syncer: syncer}
}

// Work implements river.Worker.
func (w *UserSyncWorker) Work(ctx context.Context, _ *river.Job[UserSyncJobArgs]) error {
	return w.syncer.Sync(ctx)
}
