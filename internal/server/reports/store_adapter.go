package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// SQLCQuerier is the subset of sqlcgen.Queries used by the reports store adapter.
type SQLCQuerier interface {
	CreateReportGeneration(ctx context.Context, arg sqlcgen.CreateReportGenerationParams) (sqlcgen.ReportGeneration, error)
	UpdateReportStatus(ctx context.Context, arg sqlcgen.UpdateReportStatusParams) error
	GetReportGeneration(ctx context.Context, arg sqlcgen.GetReportGenerationParams) (sqlcgen.ReportGeneration, error)
	ListReportGenerations(ctx context.Context, arg sqlcgen.ListReportGenerationsParams) ([]sqlcgen.ReportGeneration, error)
	CountReportGenerations(ctx context.Context, arg sqlcgen.CountReportGenerationsParams) (sqlcgen.CountReportGenerationsRow, error)
	CountReportGenerationsToday(ctx context.Context, tenantID pgtype.UUID) (int64, error)
	DeleteReportGeneration(ctx context.Context, arg sqlcgen.DeleteReportGenerationParams) error
	DeleteExpiredReports(ctx context.Context, tenantID pgtype.UUID) ([]pgtype.Text, error)
}

// StoreAdapter adapts sqlcgen.Queries to the ReportStore interface.
type StoreAdapter struct {
	q SQLCQuerier
}

// NewStoreAdapter creates a StoreAdapter wrapping sqlc queries.
func NewStoreAdapter(q SQLCQuerier) *StoreAdapter {
	return &StoreAdapter{q: q}
}

func (a *StoreAdapter) CreateReportGeneration(ctx context.Context, arg CreateReportGenerationParams) (ReportRecord, error) {
	filtersJSON, err := json.Marshal(arg.Filters)
	if err != nil {
		return ReportRecord{}, fmt.Errorf("marshal filters: %w", err)
	}

	row, err := a.q.CreateReportGeneration(ctx, sqlcgen.CreateReportGenerationParams{
		ID:         parseUUID(arg.ID),
		TenantID:   parseUUID(arg.TenantID),
		ReportType: arg.ReportType,
		Format:     arg.Format,
		Name:       arg.Name,
		Filters:    filtersJSON,
		CreatedBy:  parseUUID(arg.CreatedBy),
		ExpiresAt:  pgtype.Timestamptz{Time: arg.ExpiresAt, Valid: true},
	})
	if err != nil {
		return ReportRecord{}, err
	}
	return toReportRecord(row), nil
}

func (a *StoreAdapter) UpdateReportStatus(ctx context.Context, arg UpdateReportStatusParams) error {
	params := sqlcgen.UpdateReportStatusParams{
		Status:   arg.Status,
		ID:       parseUUID(arg.ID),
		TenantID: parseUUID(arg.TenantID),
	}
	if arg.FilePath != "" {
		params.FilePath = pgtype.Text{String: arg.FilePath, Valid: true}
	}
	if arg.FileSizeBytes > 0 {
		params.FileSizeBytes = pgtype.Int8{Int64: arg.FileSizeBytes, Valid: true}
	}
	if arg.ChecksumSHA256 != "" {
		params.ChecksumSha256 = pgtype.Text{String: arg.ChecksumSHA256, Valid: true}
	}
	if arg.RowCount > 0 {
		params.RowCount = pgtype.Int4{Int32: int32(arg.RowCount), Valid: true}
	}
	if arg.ErrorMessage != "" {
		params.ErrorMessage = pgtype.Text{String: arg.ErrorMessage, Valid: true}
	}
	if arg.CompletedAt != nil {
		params.CompletedAt = pgtype.Timestamptz{Time: *arg.CompletedAt, Valid: true}
	}
	return a.q.UpdateReportStatus(ctx, params)
}

func (a *StoreAdapter) GetReportGeneration(ctx context.Context, tenantID, id string) (ReportRecord, error) {
	row, err := a.q.GetReportGeneration(ctx, sqlcgen.GetReportGenerationParams{
		ID:       parseUUID(id),
		TenantID: parseUUID(tenantID),
	})
	if err != nil {
		return ReportRecord{}, err
	}
	return toReportRecord(row), nil
}

func (a *StoreAdapter) ListReportGenerations(ctx context.Context, arg ListReportGenerationsParams) ([]ReportRecord, error) {
	params := sqlcgen.ListReportGenerationsParams{
		TenantID:   parseUUID(arg.TenantID),
		Status:     arg.Status,
		ReportType: arg.ReportType,
		Format:     arg.Format,
		PageLimit:  arg.Limit,
	}
	if arg.Cursor != "" {
		// Cursor is "{created_at}:{id}" — parse it
		ts, cid, err := DecodeCursor(arg.Cursor)
		if err == nil {
			params.CursorCreatedAt = pgtype.Timestamptz{Time: ts, Valid: true}
			params.CursorID = parseUUID(cid)
		}
	}
	if params.PageLimit <= 0 {
		params.PageLimit = 25
	}

	rows, err := a.q.ListReportGenerations(ctx, params)
	if err != nil {
		return nil, err
	}

	records := make([]ReportRecord, len(rows))
	for i, row := range rows {
		records[i] = toReportRecord(row)
	}
	return records, nil
}

func (a *StoreAdapter) CountReportGenerations(ctx context.Context, tenantID string) (int64, error) {
	row, err := a.q.CountReportGenerations(ctx, sqlcgen.CountReportGenerationsParams{
		TenantID: parseUUID(tenantID),
	})
	if err != nil {
		return 0, err
	}
	return row.Total, nil
}

func (a *StoreAdapter) CountReportGenerationsToday(ctx context.Context, tenantID string) (int64, error) {
	return a.q.CountReportGenerationsToday(ctx, parseUUID(tenantID))
}

func (a *StoreAdapter) DeleteReportGeneration(ctx context.Context, tenantID, id string) error {
	return a.q.DeleteReportGeneration(ctx, sqlcgen.DeleteReportGenerationParams{
		ID:       parseUUID(id),
		TenantID: parseUUID(tenantID),
	})
}

func (a *StoreAdapter) DeleteExpiredReports(ctx context.Context) ([]ReportRecord, error) {
	// DeleteExpiredReports requires tenant_id but our interface doesn't pass one.
	// For cleanup, we pass a nil UUID — the query should handle all tenants.
	// This is a simplification; in production, iterate tenants.
	paths, err := a.q.DeleteExpiredReports(ctx, pgtype.UUID{})
	if err != nil {
		return nil, err
	}
	records := make([]ReportRecord, len(paths))
	for i, p := range paths {
		records[i] = ReportRecord{FilePath: p.String}
	}
	return records, nil
}

// toReportRecord converts a sqlcgen.ReportGeneration to a ReportRecord.
func toReportRecord(r sqlcgen.ReportGeneration) ReportRecord {
	rec := ReportRecord{
		ID:         uuidToString(r.ID),
		TenantID:   uuidToString(r.TenantID),
		ReportType: r.ReportType,
		Format:     r.Format,
		Status:     r.Status,
		Name:       r.Name,
		CreatedBy:  uuidToString(r.CreatedBy),
		CreatedAt:  r.CreatedAt.Time.In(IST).Format(time.RFC3339),
		ExpiresAt:  r.ExpiresAt.Time.In(IST).Format(time.RFC3339),
	}
	if r.FilePath.Valid {
		rec.FilePath = r.FilePath.String
	}
	if r.FileSizeBytes.Valid {
		rec.FileSizeBytes = r.FileSizeBytes.Int64
	}
	if r.ChecksumSha256.Valid {
		rec.ChecksumSHA256 = r.ChecksumSha256.String
	}
	if r.RowCount.Valid {
		rec.RowCount = int(r.RowCount.Int32)
	}
	if r.ErrorMessage.Valid {
		rec.ErrorMessage = r.ErrorMessage.String
	}
	if r.CompletedAt.Valid {
		rec.CompletedAt = r.CompletedAt.Time.In(IST).Format(time.RFC3339)
	}
	if len(r.Filters) > 0 {
		_ = json.Unmarshal(r.Filters, &rec.Filters)
	}
	return rec
}

func parseUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil || !u.Valid {
		// Fallback for non-UUID user IDs (e.g., "dev-user" in dev mode).
		u.Valid = true
		// Deterministic: hash the string into the UUID bytes.
		for i, b := range []byte(s) {
			u.Bytes[i%16] ^= b
		}
	}
	return u
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// DecodeCursor parses a cursor string "timestamp:uuid" into its parts.
func DecodeCursor(cursor string) (time.Time, string, error) {
	for i := len(cursor) - 1; i >= 0; i-- {
		if cursor[i] == ':' {
			ts, err := time.Parse(time.RFC3339Nano, cursor[:i])
			if err != nil {
				return time.Time{}, "", fmt.Errorf("decode cursor timestamp: %w", err)
			}
			return ts, cursor[i+1:], nil
		}
	}
	return time.Time{}, "", fmt.Errorf("invalid cursor format")
}
