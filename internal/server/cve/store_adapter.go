package cve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// StoreAdapter bridges the CVE package interfaces to sqlcgen queries.
// All write operations use tenant-scoped transactions with SET LOCAL app.current_tenant_id.
type StoreAdapter struct {
	pool *pgxpool.Pool
}

// beginTenantTx starts a transaction and sets the tenant context for RLS.
func (a *StoreAdapter) beginTenantTx(ctx context.Context, tenantID string) (pgx.Tx, *sqlcgen.Queries, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tenant tx: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			slog.WarnContext(ctx, "store adapter: rollback after set_config failure", "error", rbErr)
		}
		return nil, nil, fmt.Errorf("begin tenant tx: set tenant context: %w", err)
	}
	return tx, sqlcgen.New(tx), nil
}

// NewStoreAdapter creates a StoreAdapter backed by the given connection pool.
func NewStoreAdapter(pool *pgxpool.Pool) *StoreAdapter {
	return &StoreAdapter{pool: pool}
}

// UpsertCVE inserts or updates a CVE record for the given tenant.
// Returns the database ID, whether the record was newly created, and any error.
func (a *StoreAdapter) UpsertCVE(ctx context.Context, tenantID string, rec CVERecord) (string, bool, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return "", false, fmt.Errorf("upsert CVE: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return "", false, fmt.Errorf("upsert CVE %s: %w", rec.CVEID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	refsJSON, marshalErr := marshalReferences(rec.References)
	if marshalErr != nil {
		return "", false, fmt.Errorf("upsert CVE %s: marshal references: %w", rec.CVEID, marshalErr)
	}

	var kevDueDate pgtype.Date
	if rec.CisaKEVDueDate != "" {
		t, parseErr := time.Parse("2006-01-02", rec.CisaKEVDueDate)
		if parseErr != nil {
			slog.WarnContext(ctx, "upsert CVE: parse KEV due date",
				"cve_id", rec.CVEID, "value", rec.CisaKEVDueDate, "error", parseErr)
		} else {
			kevDueDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	result, err := q.UpsertCVE(ctx, sqlcgen.UpsertCVEParams{
		TenantID:           tenantUUID,
		CveID:              rec.CVEID,
		Severity:           rec.Severity,
		Description:        pgtype.Text{String: rec.Description, Valid: rec.Description != ""},
		PublishedAt:        pgtype.Timestamptz{Time: rec.PublishedAt, Valid: !rec.PublishedAt.IsZero()},
		CvssV3Score:        float64ToNumeric(rec.CVSSv3Score),
		CvssV3Vector:       pgtype.Text{String: rec.CVSSv3Vector, Valid: rec.CVSSv3Vector != ""},
		CisaKevDueDate:     kevDueDate,
		NvdLastModified:    pgtype.Timestamptz{Time: rec.LastModified, Valid: !rec.LastModified.IsZero()},
		ExploitAvailable:   rec.ExploitAvailable,
		AttackVector:       pgtype.Text{String: rec.AttackVector, Valid: rec.AttackVector != ""},
		ExternalReferences: refsJSON,
		CweID:              pgtype.Text{String: rec.CweID, Valid: rec.CweID != ""},
		Source:             rec.Source,
	})
	if err != nil {
		return "", false, fmt.Errorf("upsert CVE %s: %w", rec.CVEID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("upsert CVE %s: commit: %w", rec.CVEID, err)
	}

	dbID := uuidToString(result.ID)
	isNew := result.CreatedAt.Time.Equal(result.UpdatedAt.Time)
	return dbID, isNew, nil
}

// UpsertCVEWithKEV inserts or updates a CVE record enriched with KEV data.
func (a *StoreAdapter) UpsertCVEWithKEV(ctx context.Context, tenantID string, rec CVERecord, kevDueDate string, exploitAvailable bool) (string, bool, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return "", false, fmt.Errorf("upsert CVE with KEV: parse tenant ID: %w", err)
	}

	var dueDate pgtype.Date
	if kevDueDate != "" {
		t, parseErr := time.Parse("2006-01-02", kevDueDate)
		if parseErr != nil {
			return "", false, fmt.Errorf("upsert CVE with KEV %s: parse due date %q: %w", rec.CVEID, kevDueDate, parseErr)
		}
		dueDate = pgtype.Date{Time: t, Valid: true}
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return "", false, fmt.Errorf("upsert CVE with KEV %s: %w", rec.CVEID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	refsJSON, marshalErr := marshalReferences(rec.References)
	if marshalErr != nil {
		return "", false, fmt.Errorf("upsert CVE with KEV %s: marshal references: %w", rec.CVEID, marshalErr)
	}

	result, err := q.UpsertCVE(ctx, sqlcgen.UpsertCVEParams{
		TenantID:           tenantUUID,
		CveID:              rec.CVEID,
		Severity:           rec.Severity,
		Description:        pgtype.Text{String: rec.Description, Valid: rec.Description != ""},
		PublishedAt:        pgtype.Timestamptz{Time: rec.PublishedAt, Valid: !rec.PublishedAt.IsZero()},
		CvssV3Score:        float64ToNumeric(rec.CVSSv3Score),
		CvssV3Vector:       pgtype.Text{String: rec.CVSSv3Vector, Valid: rec.CVSSv3Vector != ""},
		CisaKevDueDate:     dueDate,
		ExploitAvailable:   exploitAvailable,
		NvdLastModified:    pgtype.Timestamptz{Time: rec.LastModified, Valid: !rec.LastModified.IsZero()},
		AttackVector:       pgtype.Text{String: rec.AttackVector, Valid: rec.AttackVector != ""},
		ExternalReferences: refsJSON,
		CweID:              pgtype.Text{String: rec.CweID, Valid: rec.CweID != ""},
		Source:             rec.Source,
	})
	if err != nil {
		return "", false, fmt.Errorf("upsert CVE with KEV %s: %w", rec.CVEID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("upsert CVE with KEV %s: commit: %w", rec.CVEID, err)
	}

	dbID := uuidToString(result.ID)
	isNew := result.CreatedAt.Time.Equal(result.UpdatedAt.Time)
	return dbID, isNew, nil
}

// GetSyncCursor retrieves the last sync time for the given source and tenant.
func (a *StoreAdapter) GetSyncCursor(ctx context.Context, tenantID, source string) (time.Time, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return time.Time{}, fmt.Errorf("get sync cursor: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return time.Time{}, fmt.Errorf("get sync cursor %s: %w", source, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	cursor, err := q.GetCVESyncCursor(ctx, sqlcgen.GetCVESyncCursorParams{
		TenantID: tenantUUID,
		Source:   source,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("get sync cursor %s: %w", source, err)
	}
	return cursor.LastSynced.Time, nil
}

// UpdateSyncCursor upserts the sync cursor for the given source and tenant.
func (a *StoreAdapter) UpdateSyncCursor(ctx context.Context, tenantID, source string, lastSynced time.Time) error {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return fmt.Errorf("update sync cursor: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("update sync cursor %s: %w", source, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = q.UpsertCVESyncCursor(ctx, sqlcgen.UpsertCVESyncCursorParams{
		TenantID:   tenantUUID,
		Source:     source,
		LastSynced: pgtype.Timestamptz{Time: lastSynced, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update sync cursor %s: %w", source, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("update sync cursor %s: commit: %w", source, err)
	}
	return nil
}

// ListPatchesByName implements PatchLister by querying patches with the given package name.
func (a *StoreAdapter) ListPatchesByName(ctx context.Context, tenantID, packageName string) ([]PatchInfo, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list patches by name: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list patches by name %s: %w", packageName, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	patches, err := q.ListPatchesByName(ctx, sqlcgen.ListPatchesByNameParams{
		PackageName: packageName,
		TenantID:    tenantUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("list patches by name %s: %w", packageName, err)
	}

	result := make([]PatchInfo, len(patches))
	for i, p := range patches {
		result[i] = PatchInfo{
			ID:      uuidToString(p.ID),
			Name:    p.Name,
			Version: p.Version,
		}
	}
	return result, nil
}

// LinkPatchCVE implements CVELinker by inserting a patch-CVE link with version range data.
func (a *StoreAdapter) LinkPatchCVE(ctx context.Context, tenantID, patchID, cveDBID, versionEndExcluding, versionEndIncluding string) error {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return fmt.Errorf("link patch CVE: parse tenant ID: %w", err)
	}
	patchUUID, err := parsePgUUID(patchID)
	if err != nil {
		return fmt.Errorf("link patch CVE: parse patch ID: %w", err)
	}
	cveUUID, err := parsePgUUID(cveDBID)
	if err != nil {
		return fmt.Errorf("link patch CVE: parse CVE DB ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("link patch CVE: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := q.LinkPatchCVE(ctx, sqlcgen.LinkPatchCVEParams{
		TenantID:            tenantUUID,
		PatchID:             patchUUID,
		CveID:               cveUUID,
		VersionEndExcluding: versionEndExcluding,
		VersionEndIncluding: versionEndIncluding,
	}); err != nil {
		return fmt.Errorf("link patch CVE: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("link patch CVE: commit: %w", err)
	}
	return nil
}

// UpsertEndpointCVE implements EndpointCVEUpserter by upserting an endpoint-CVE association.
func (a *StoreAdapter) UpsertEndpointCVE(ctx context.Context, tenantID string, rec EndpointCVERecord) error {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return fmt.Errorf("upsert endpoint CVE: parse tenant ID: %w", err)
	}
	endpointUUID, err := parsePgUUID(rec.EndpointID)
	if err != nil {
		return fmt.Errorf("upsert endpoint CVE: parse endpoint ID: %w", err)
	}
	cveUUID, err := parsePgUUID(rec.CVEDBID)
	if err != nil {
		return fmt.Errorf("upsert endpoint CVE: parse CVE DB ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("upsert endpoint CVE %s: %w", rec.CVEDBID, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = q.UpsertEndpointCVE(ctx, sqlcgen.UpsertEndpointCVEParams{
		TenantID:   tenantUUID,
		EndpointID: endpointUUID,
		CveID:      cveUUID,
		Status:     rec.Status,
		DetectedAt: pgtype.Timestamptz{Time: rec.DetectedAt, Valid: true},
		RiskScore:  float64ToNumeric(rec.RiskScore),
	})
	if err != nil {
		return fmt.Errorf("upsert endpoint CVE %s: %w", rec.CVEDBID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("upsert endpoint CVE %s: commit: %w", rec.CVEDBID, err)
	}
	return nil
}

// ListEndpointPackages implements EndpointPackageLister by querying packages for an endpoint.
func (a *StoreAdapter) ListEndpointPackages(ctx context.Context, tenantID, endpointID string) ([]EndpointPackage, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint packages: parse tenant ID: %w", err)
	}
	endpointUUID, err := parsePgUUID(endpointID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint packages: parse endpoint ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint packages: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dbPkgs, err := q.ListEndpointPackagesByEndpoint(ctx, sqlcgen.ListEndpointPackagesByEndpointParams{
		EndpointID: endpointUUID,
		TenantID:   tenantUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("list endpoint packages for %s: %w", endpointID, err)
	}

	result := make([]EndpointPackage, len(dbPkgs))
	for i, p := range dbPkgs {
		result[i] = EndpointPackage{
			Name:    p.PackageName,
			Version: p.Version,
		}
	}
	return result, nil
}

// ListCVEsForPackage implements CVELookup by querying CVEs linked to a package name.
func (a *StoreAdapter) ListCVEsForPackage(ctx context.Context, tenantID, packageName string) ([]MatchableCVE, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list CVEs for package: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list CVEs for package %s: %w", packageName, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dbCVEs, err := q.ListCVEsByPackageName(ctx, sqlcgen.ListCVEsByPackageNameParams{
		PackageName: packageName,
		TenantID:    tenantUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("list CVEs for package %s: %w", packageName, err)
	}

	result := make([]MatchableCVE, len(dbCVEs))
	for i, c := range dbCVEs {
		var cvss float64
		if c.CvssV3Score.Valid {
			if f, scanErr := numericToFloat64(c.CvssV3Score); scanErr == nil {
				cvss = f
			}
		}
		result[i] = MatchableCVE{
			CVEDBID:             uuidToString(c.ID),
			CVEID:               c.CveID,
			CVSSv3Score:         cvss,
			CISAKev:             c.CisaKevDueDate.Valid,
			ExploitAvailable:    c.ExploitAvailable,
			PublishedAt:         c.PublishedAt.Time,
			VersionEndExcluding: c.VersionEndExcluding,
			VersionEndIncluding: c.VersionEndIncluding,
		}
	}
	return result, nil
}

// ListCVEsByOsFamily implements OsFamilyCVELookup by searching CVE descriptions for the OS family keyword.
// Only "windows" is supported; all other os_family values return an empty slice.
func (a *StoreAdapter) ListCVEsByOsFamily(ctx context.Context, tenantID, osFamily string) ([]MatchableCVE, error) {
	keyword := osFamilyToKeyword(osFamily)
	if keyword == "" {
		return nil, nil
	}

	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list CVEs by os family: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list CVEs by os family %s: %w", osFamily, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dbCVEs, err := q.ListCVEsByOsFamily(ctx, sqlcgen.ListCVEsByOsFamilyParams{
		TenantID: tenantUUID,
		Column2:  pgtype.Text{String: keyword, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("list CVEs by os family %s: %w", osFamily, err)
	}

	result := make([]MatchableCVE, len(dbCVEs))
	for i, c := range dbCVEs {
		var cvss float64
		if c.CvssV3Score.Valid {
			if f, scanErr := numericToFloat64(c.CvssV3Score); scanErr == nil {
				cvss = f
			}
		}
		result[i] = MatchableCVE{
			CVEDBID:             uuidToString(c.ID),
			CVEID:               c.CveID,
			CVSSv3Score:         cvss,
			CISAKev:             c.CisaKevDueDate.Valid,
			ExploitAvailable:    c.ExploitAvailable,
			PublishedAt:         c.PublishedAt.Time,
			VersionEndExcluding: c.VersionEndExcluding,
			VersionEndIncluding: c.VersionEndIncluding,
		}
	}
	return result, nil
}

// osFamilyToKeyword maps an os_family value to a CVE description search keyword.
// Returns empty string for unsupported os_family values.
func osFamilyToKeyword(osFamily string) string {
	switch osFamily {
	case "windows":
		return "microsoft windows"
	default:
		return ""
	}
}

// GetEndpointOsFamily returns the os_family for the given endpoint ID.
func (a *StoreAdapter) GetEndpointOsFamily(ctx context.Context, endpointID string) (string, error) {
	endpointUUID, err := parsePgUUID(endpointID)
	if err != nil {
		return "", fmt.Errorf("get endpoint os family: parse endpoint ID: %w", err)
	}

	q := sqlcgen.New(a.pool)
	ep, err := q.LookupEndpointByID(ctx, endpointUUID)
	if err != nil {
		return "", fmt.Errorf("get endpoint os family for %s: %w", endpointID, err)
	}
	return ep.OsFamily, nil
}

// ListEndpointIDs returns all non-decommissioned endpoint IDs for a tenant.
func (a *StoreAdapter) ListEndpointIDs(ctx context.Context, tenantID string) ([]string, error) {
	tenantUUID, err := parsePgUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint IDs: parse tenant ID: %w", err)
	}

	tx, q, err := a.beginTenantTx(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint IDs: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	endpoints, err := q.ListEndpointsByTenant(ctx, tenantUUID)
	if err != nil {
		return nil, fmt.Errorf("list endpoint IDs: %w", err)
	}

	ids := make([]string, 0, len(endpoints))
	for _, ep := range endpoints {
		if ep.Status == "decommissioned" {
			continue
		}
		ids = append(ids, uuidToString(ep.ID))
	}
	return ids, nil
}

func numericToFloat64(n pgtype.Numeric) (float64, error) {
	f8, err := n.Float64Value()
	if err != nil {
		return 0, err
	}
	if !f8.Valid {
		return 0, nil
	}
	return f8.Float64, nil
}

func parsePgUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	if err := u.Scan(s); err != nil {
		return u, fmt.Errorf("parse UUID %q: %w", s, err)
	}
	return u, nil
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func marshalReferences(refs []CVEReference) ([]byte, error) {
	if len(refs) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(refs)
}

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(fmt.Sprintf("%.2f", f)); err != nil {
		slog.Warn("float64ToNumeric: failed to convert", "value", f, "error", err)
		return pgtype.Numeric{Valid: false}
	}
	return n
}
