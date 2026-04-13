package discovery

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// Enqueuer implements the api/v1.JobEnqueuer interface using a River client.
type Enqueuer struct {
	client *river.Client[pgx.Tx]
}

// NewEnqueuer creates an Enqueuer backed by the given River client.
func NewEnqueuer(client *river.Client[pgx.Tx]) *Enqueuer {
	return &Enqueuer{client: client}
}

// EnqueueDiscovery inserts a patch discovery job into the River queue.
// Duplicate jobs with the same tenant/repo args are deduplicated via River's unique job support.
// Returns the job ID as a string.
func (e *Enqueuer) EnqueueDiscovery(ctx context.Context, tenantID, repoName string) (string, error) {
	result, err := e.client.Insert(ctx, DiscoveryJobArgs{TenantID: tenantID, RepoName: repoName}, &river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
		},
	})
	if err != nil {
		return "", fmt.Errorf("enqueue discovery job: %w", err)
	}
	return fmt.Sprintf("%d", result.Job.ID), nil
}
