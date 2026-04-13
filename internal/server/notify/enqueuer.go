package notify

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// RiverEnqueuer enqueues notification send jobs using a River client.
type RiverEnqueuer struct {
	client *river.Client[pgx.Tx]
}

// NewRiverEnqueuer creates an enqueuer backed by the given River client.
func NewRiverEnqueuer(client *river.Client[pgx.Tx]) *RiverEnqueuer {
	return &RiverEnqueuer{client: client}
}

// EnqueueNotification inserts a notification send job into the River queue.
func (e *RiverEnqueuer) EnqueueNotification(ctx context.Context, args SendJobArgs) error {
	_, err := e.client.Insert(ctx, args, &river.InsertOpts{
		MaxAttempts: 3,
	})
	if err != nil {
		return fmt.Errorf("enqueue notification job: %w", err)
	}
	return nil
}
