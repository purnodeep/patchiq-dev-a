package store

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// RunRetention periodically deletes old records from logs, outbox, and history
// tables based on the retention period returned by retentionDaysFunc. It runs
// once per hour and stops when ctx is cancelled.
func RunRetention(ctx context.Context, db *sql.DB, retentionDaysFunc func() int, logger *slog.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run once immediately on startup.
	runRetentionOnce(ctx, db, retentionDaysFunc(), logger)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runRetentionOnce(ctx, db, retentionDaysFunc(), logger)
		}
	}
}

func runRetentionOnce(ctx context.Context, db *sql.DB, days int, logger *slog.Logger) {
	if days <= 0 {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -days).Format(time.RFC3339)

	logsDeleted, err := execDelete(ctx, db, `DELETE FROM agent_logs WHERE timestamp < ?`, cutoff)
	if err != nil {
		logger.WarnContext(ctx, "retention: delete agent_logs failed", "error", err)
	}

	outboxDeleted, err := execDelete(ctx, db, `DELETE FROM outbox WHERE created_at < ? AND status = 'sent'`, cutoff)
	if err != nil {
		logger.WarnContext(ctx, "retention: delete outbox failed", "error", err)
	}

	historyDeleted, err := execDelete(ctx, db, `DELETE FROM patch_history WHERE completed_at < ?`, cutoff)
	if err != nil {
		logger.WarnContext(ctx, "retention: delete patch_history failed", "error", err)
	}

	total := logsDeleted + outboxDeleted + historyDeleted
	if total > 0 {
		logger.InfoContext(ctx, "retention: cleaned old records",
			"retention_days", days,
			"logs_deleted", logsDeleted,
			"outbox_deleted", outboxDeleted,
			"history_deleted", historyDeleted,
		)
	}
}

func execDelete(ctx context.Context, db *sql.DB, query string, args ...any) (int64, error) {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return n, nil
}
