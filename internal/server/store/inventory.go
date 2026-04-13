package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// BulkInsertEndpointPackages inserts packages using a batched multi-row INSERT.
// PostgreSQL COPY protocol is incompatible with RLS, so we use a batch of
// multi-row INSERT statements instead. Each batch statement inserts up to 100
// rows, keeping the parameter count well under PostgreSQL's 65535 limit.
// Must be called within an existing transaction that has tenant context set.
func (s *Store) BulkInsertEndpointPackages(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, endpointID, inventoryID pgtype.UUID,
	packages []*pb.PackageInfo,
) (int64, error) {
	if len(packages) == 0 {
		return 0, nil
	}

	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}

	// Build batched multi-row INSERTs. Each row uses 15 parameters;
	// batch size of 100 rows = 1500 params per statement (well under 65535).
	const batchSize = 100
	var totalInserted int64

	batch := &pgx.Batch{}
	for i := 0; i < len(packages); i += batchSize {
		end := i + batchSize
		if end > len(packages) {
			end = len(packages)
		}
		query, args := buildMultiRowInsert(packages[i:end], tenantID, endpointID, inventoryID, now)
		batch.Queue(query, args...)
	}

	results := tx.SendBatch(ctx, batch)

	for i := range batch.Len() {
		tag, err := results.Exec()
		if err != nil {
			results.Close()
			return totalInserted, fmt.Errorf("bulk insert endpoint packages chunk %d: %w", i, err)
		}
		totalInserted += tag.RowsAffected()
	}

	if err := results.Close(); err != nil {
		return totalInserted, fmt.Errorf("bulk insert endpoint packages close: %w", err)
	}

	return totalInserted, nil
}

// buildMultiRowInsert constructs a single INSERT ... VALUES (...), (...), ...
// statement for the given chunk of packages. Returns the SQL and flattened args.
func buildMultiRowInsert(
	packages []*pb.PackageInfo,
	tenantID, endpointID, inventoryID pgtype.UUID,
	now pgtype.Timestamptz,
) (string, []any) {
	const colsPerRow = 15
	args := make([]any, 0, len(packages)*colsPerRow)

	// Pre-allocate a reasonable buffer for the SQL string.
	// ~180 bytes per row for the VALUES clause.
	query := make([]byte, 0, 300+len(packages)*180)
	query = append(query, "INSERT INTO endpoint_packages (id, tenant_id, endpoint_id, inventory_id, package_name, version, arch, source, release, created_at, kb_article, severity, install_date, category, publisher) VALUES "...)

	for i, pkg := range packages {
		if i > 0 {
			query = append(query, ", "...)
		}
		base := i * colsPerRow
		query = append(query, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base+1, base+2, base+3, base+4, base+5,
			base+6, base+7, base+8, base+9, base+10,
			base+11, base+12, base+13, base+14, base+15,
		)...)

		args = append(args,
			pgtype.UUID{Bytes: uuid.New(), Valid: true},
			tenantID, endpointID, inventoryID,
			pkg.GetName(), pkg.GetVersion(),
			nullableText(pkg.GetArchitecture()),
			nullableText(pkg.GetSource()),
			nullableText(pkg.GetRelease()),
			now,
			nullableText(pkg.GetKbArticle()),
			nullableText(pkg.GetSeverity()),
			nullableText(pkg.GetInstallDate()),
			nullableText(pkg.GetCategory()),
			nullableText(pkg.GetPublisher()),
		)
	}

	// Skip duplicates: multiple collectors (e.g., wua + wua_installed) may
	// report the same package_name+version within one inventory snapshot.
	query = append(query, " ON CONFLICT (tenant_id, inventory_id, package_name, version) DO NOTHING"...)

	return string(query), args
}

// nullableText returns a valid pgtype.Text when s is non-empty, or a null Text otherwise.
func nullableText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}
