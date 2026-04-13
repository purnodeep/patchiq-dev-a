package workers

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// AuditRetentionJobArgs defines the River periodic job for purging expired audit partitions.
type AuditRetentionJobArgs struct{}

// Kind implements river.JobArgs.
func (AuditRetentionJobArgs) Kind() string { return "audit_retention_purge" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (AuditRetentionJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "background"}
}

// RetentionQuerier queries tenant retention policies.
type RetentionQuerier interface {
	ListTenantRetentionPolicies(ctx context.Context) ([]sqlcgen.ListTenantRetentionPoliciesRow, error)
}

// PartitionDropper lists and drops audit partitions.
type PartitionDropper interface {
	ListAuditPartitions(ctx context.Context) ([]string, error)
	DropPartition(ctx context.Context, name string) error
}

// AuditRetentionPurger drops expired monthly partitions from the audit_events table
// based on tenant retention policies.
type AuditRetentionPurger struct {
	q       RetentionQuerier
	dropper PartitionDropper
}

// NewAuditRetentionPurger creates a new AuditRetentionPurger.
func NewAuditRetentionPurger(q RetentionQuerier, dropper PartitionDropper) *AuditRetentionPurger {
	return &AuditRetentionPurger{q: q, dropper: dropper}
}

// defaultRetentionDays is used when no tenant retention policies exist.
const defaultRetentionDays = 365

// Purge queries all tenant retention policies, determines the longest retention period,
// and drops any monthly audit partitions that are fully expired.
func (p *AuditRetentionPurger) Purge(ctx context.Context) error {
	policies, err := p.q.ListTenantRetentionPolicies(ctx)
	if err != nil {
		return fmt.Errorf("audit retention purge: list tenant retention policies: %w", err)
	}

	maxRetention := defaultRetentionDays
	for _, pol := range policies {
		if int(pol.AuditRetentionDays) > maxRetention {
			maxRetention = int(pol.AuditRetentionDays)
		}
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -maxRetention)

	partitions, err := p.dropper.ListAuditPartitions(ctx)
	if err != nil {
		return fmt.Errorf("audit retention purge: list audit partitions: %w", err)
	}

	var dropped []string
	for _, name := range partitions {
		if name == "audit_events_default" {
			continue
		}

		partDate, err := parsePartitionDate(name)
		if err != nil {
			slog.WarnContext(ctx, "audit retention purge: skipping unrecognized partition",
				"partition", name,
				"error", err,
			)
			continue
		}

		// End of the partition month = first day of next month.
		endOfMonth := partDate.AddDate(0, 1, 0)
		if endOfMonth.Before(cutoff) {
			if err := p.dropper.DropPartition(ctx, name); err != nil {
				return fmt.Errorf("audit retention purge: drop partition %s: %w", name, err)
			}
			dropped = append(dropped, name)
		}
	}

	slog.InfoContext(ctx, "audit retention purge complete",
		"partitions_dropped", len(dropped),
		"dropped", dropped,
	)

	return nil
}

// partitionNameRe validates the format "audit_events_YYYY_MM".
var partitionNameRe = regexp.MustCompile(`^audit_events_(\d{4})_(\d{2})$`)

// parsePartitionDate parses a partition name like "audit_events_2024_01" into a time.Time
// representing the first day of that month in UTC.
func parsePartitionDate(name string) (time.Time, error) {
	matches := partitionNameRe.FindStringSubmatch(name)
	if matches == nil {
		return time.Time{}, fmt.Errorf("partition name %q does not match audit_events_YYYY_MM format", name)
	}

	t, err := time.Parse("2006_01", matches[1]+"_"+matches[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse partition date %q: %w", name, err)
	}
	return t.UTC(), nil
}

// PgPartitionDropper is the production implementation of PartitionDropper using PostgreSQL.
type PgPartitionDropper struct {
	pool *pgxpool.Pool
}

// NewPgPartitionDropper creates a PgPartitionDropper.
func NewPgPartitionDropper(pool *pgxpool.Pool) *PgPartitionDropper {
	return &PgPartitionDropper{pool: pool}
}

// ListAuditPartitions queries pg_inherits to list child tables of audit_events.
func (d *PgPartitionDropper) ListAuditPartitions(ctx context.Context) ([]string, error) {
	const query = `
		SELECT c.relname
		FROM pg_inherits i
		JOIN pg_class c ON c.oid = i.inhrelid
		JOIN pg_class p ON p.oid = i.inhparent
		WHERE p.relname = 'audit_events'
		ORDER BY c.relname`

	rows, err := d.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list audit partitions: %w", err)
	}
	defer rows.Close()

	var partitions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan audit partition name: %w", err)
		}
		partitions = append(partitions, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit partitions: %w", err)
	}

	return partitions, nil
}

// DropPartition drops an audit partition table. It validates the name format
// before executing to prevent SQL injection.
func (d *PgPartitionDropper) DropPartition(ctx context.Context, name string) error {
	if _, err := parsePartitionDate(name); err != nil {
		return fmt.Errorf("drop partition: invalid partition name %q: %w", name, err)
	}

	stmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", name)
	if _, err := d.pool.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("drop partition %s: %w", name, err)
	}

	slog.InfoContext(ctx, "dropped audit partition", "partition", name)
	return nil
}

// AuditRetentionWorker wraps AuditRetentionPurger as a River worker.
type AuditRetentionWorker struct {
	river.WorkerDefaults[AuditRetentionJobArgs]
	purger *AuditRetentionPurger
}

// NewAuditRetentionWorker creates a new AuditRetentionWorker.
func NewAuditRetentionWorker(purger *AuditRetentionPurger) *AuditRetentionWorker {
	return &AuditRetentionWorker{purger: purger}
}

// Work implements river.Worker.
func (w *AuditRetentionWorker) Work(ctx context.Context, _ *river.Job[AuditRetentionJobArgs]) error {
	return w.purger.Purge(ctx)
}
