package v1

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// TxBeginner starts a new database transaction. Used by handlers that need
// to wrap multi-statement workflows (idempotent upserts, cross-table
// updates) in a single tx. Was previously declared alongside the groups
// handler; groups.go was removed in Phase 2 so the definition moved here.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
