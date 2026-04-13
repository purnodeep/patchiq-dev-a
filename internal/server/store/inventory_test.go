package store_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

func TestInventoryQueries(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := appPool(t, superPool)
	defer app.Close()

	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(defaultTenant); err != nil {
		t.Fatalf("parse default tenant UUID: %v", err)
	}

	st := store.NewStore(app)
	tenantCtx := tenant.WithTenantID(ctx, defaultTenant)

	beginTx := func(t *testing.T) (pgx.Tx, *sqlcgen.Queries) {
		t.Helper()
		tx, err := st.BeginTx(tenantCtx)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		return tx, sqlcgen.New(tx)
	}

	// Seed an endpoint via superuser for FK constraints.
	var endpointID pgtype.UUID
	if err := superPool.QueryRow(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'inv-host', 'linux', '22.04', 'online') RETURNING id",
		defaultTenant,
	).Scan(&endpointID); err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	var createdInvID pgtype.UUID

	t.Run("CreateEndpointInventory", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		inv, err := q.CreateEndpointInventory(ctx, sqlcgen.CreateEndpointInventoryParams{
			TenantID:     tenantUUID,
			EndpointID:   endpointID,
			ScannedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			PackageCount: 42,
		})
		if err != nil {
			t.Fatalf("CreateEndpointInventory: %v", err)
		}
		if inv.PackageCount != 42 {
			t.Errorf("package_count = %d, want 42", inv.PackageCount)
		}
		if !inv.ID.Valid {
			t.Error("expected valid ID")
		}
		createdInvID = inv.ID

		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
	})

	t.Run("GetEndpointInventoryByID", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		got, err := q.GetEndpointInventoryByID(ctx, sqlcgen.GetEndpointInventoryByIDParams{
			ID:       createdInvID,
			TenantID: tenantUUID,
		})
		if err != nil {
			t.Fatalf("GetEndpointInventoryByID: %v", err)
		}
		if got.ID != createdInvID {
			t.Errorf("ID mismatch")
		}
	})

	t.Run("GetLatestEndpointInventory", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		inv, err := q.GetLatestEndpointInventory(ctx, sqlcgen.GetLatestEndpointInventoryParams{
			EndpointID: endpointID,
			TenantID:   tenantUUID,
		})
		if err != nil {
			t.Fatalf("GetLatestEndpointInventory: %v", err)
		}
		if !inv.ID.Valid {
			t.Error("expected valid ID")
		}
	})

	t.Run("ListEndpointInventories", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		invs, err := q.ListEndpointInventories(ctx, sqlcgen.ListEndpointInventoriesParams{
			EndpointID: endpointID,
			TenantID:   tenantUUID,
		})
		if err != nil {
			t.Fatalf("ListEndpointInventories: %v", err)
		}
		if len(invs) < 1 {
			t.Errorf("expected >= 1 inventory, got %d", len(invs))
		}
	})

	t.Run("CreateEndpointPackage_and_ListByInventory", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		pkg, err := q.CreateEndpointPackage(ctx, sqlcgen.CreateEndpointPackageParams{
			TenantID:    tenantUUID,
			EndpointID:  endpointID,
			InventoryID: createdInvID,
			PackageName: "test-pkg",
			Version:     "2.0.0",
			Arch:        pgtype.Text{String: "amd64", Valid: true},
			Source:      pgtype.Text{String: "apt", Valid: true},
		})
		if err != nil {
			t.Fatalf("CreateEndpointPackage: %v", err)
		}
		if pkg.PackageName != "test-pkg" {
			t.Errorf("package_name = %q, want test-pkg", pkg.PackageName)
		}

		pkgs, err := q.ListEndpointPackages(ctx, sqlcgen.ListEndpointPackagesParams{
			InventoryID: createdInvID,
			TenantID:    tenantUUID,
		})
		if err != nil {
			t.Fatalf("ListEndpointPackages: %v", err)
		}
		if len(pkgs) < 1 {
			t.Errorf("expected >= 1 package, got %d", len(pkgs))
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
	})

	t.Run("ListEndpointPackagesByEndpoint", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		pkgs, err := q.ListEndpointPackagesByEndpoint(ctx, sqlcgen.ListEndpointPackagesByEndpointParams{
			EndpointID: endpointID,
			TenantID:   tenantUUID,
		})
		if err != nil {
			t.Fatalf("ListEndpointPackagesByEndpoint: %v", err)
		}
		if len(pkgs) < 1 {
			t.Errorf("expected >= 1 package, got %d", len(pkgs))
		}
	})
}

func TestBulkInsertEndpointPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}

	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := appPool(t, superPool)
	defer app.Close()

	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(defaultTenant); err != nil {
		t.Fatalf("parse default tenant UUID: %v", err)
	}

	st := store.NewStore(app)
	tenantCtx := tenant.WithTenantID(ctx, defaultTenant)

	// Seed an endpoint via superuser for FK constraints.
	var endpointID pgtype.UUID
	if err := superPool.QueryRow(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'bulk-host', 'linux', '22.04', 'online') RETURNING id",
		defaultTenant,
	).Scan(&endpointID); err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	t.Run("inserts_3_packages_and_verifies", func(t *testing.T) {
		tx, err := st.BeginTx(tenantCtx)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		q := sqlcgen.New(tx)
		inv, err := q.CreateEndpointInventory(ctx, sqlcgen.CreateEndpointInventoryParams{
			TenantID:     tenantUUID,
			EndpointID:   endpointID,
			ScannedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			PackageCount: 3,
		})
		if err != nil {
			t.Fatalf("CreateEndpointInventory: %v", err)
		}

		packages := []*pb.PackageInfo{
			{Name: "curl", Version: "7.88.1", Architecture: "amd64", Source: "apt"},
			{Name: "openssl", Version: "3.0.2", Architecture: "amd64", Source: "apt", Release: "1ubuntu1"},
			{Name: "bash", Version: "5.2.15", Source: "apt"},
		}

		count, err := st.BulkInsertEndpointPackages(ctx, tx, tenantUUID, endpointID, inv.ID, packages)
		if err != nil {
			t.Fatalf("BulkInsertEndpointPackages: %v", err)
		}
		if count != 3 {
			t.Errorf("inserted count = %d, want 3", count)
		}

		// Verify via ListEndpointPackages.
		pkgs, err := q.ListEndpointPackages(ctx, sqlcgen.ListEndpointPackagesParams{
			InventoryID: inv.ID,
			TenantID:    tenantUUID,
		})
		if err != nil {
			t.Fatalf("ListEndpointPackages: %v", err)
		}
		if len(pkgs) != 3 {
			t.Fatalf("ListEndpointPackages returned %d rows, want 3", len(pkgs))
		}

		// Verify packages are sorted by package_name (sqlc query ORDER BY package_name).
		expectedNames := []string{"bash", "curl", "openssl"}
		for i, pkg := range pkgs {
			if pkg.PackageName != expectedNames[i] {
				t.Errorf("pkgs[%d].PackageName = %q, want %q", i, pkg.PackageName, expectedNames[i])
			}
		}

		// Verify nullable fields.
		// openssl should have release set.
		for _, pkg := range pkgs {
			if pkg.PackageName == "openssl" {
				if !pkg.Release.Valid || pkg.Release.String != "1ubuntu1" {
					t.Errorf("openssl release = %v, want '1ubuntu1'", pkg.Release)
				}
			}
			// bash has no architecture set.
			if pkg.PackageName == "bash" {
				if pkg.Arch.Valid {
					t.Errorf("bash arch should be null, got %q", pkg.Arch.String)
				}
			}
		}

		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
	})

	t.Run("empty_slice_returns_zero", func(t *testing.T) {
		tx, err := st.BeginTx(tenantCtx)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		count, err := st.BulkInsertEndpointPackages(ctx, tx, tenantUUID, endpointID, pgtype.UUID{}, nil)
		if err != nil {
			t.Fatalf("BulkInsertEndpointPackages with nil: %v", err)
		}
		if count != 0 {
			t.Errorf("count = %d, want 0 for empty slice", count)
		}
	})
}

func TestBulkInsertEndpointPackages_Scale(t *testing.T) {
	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}

	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := appPool(t, superPool)
	defer app.Close()

	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(defaultTenant); err != nil {
		t.Fatalf("parse default tenant UUID: %v", err)
	}

	st := store.NewStore(app)
	tenantCtx := tenant.WithTenantID(ctx, defaultTenant)

	var endpointID pgtype.UUID
	if err := superPool.QueryRow(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'scale-host', 'linux', '22.04', 'online') RETURNING id",
		defaultTenant,
	).Scan(&endpointID); err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}

	tx, err := st.BeginTx(tenantCtx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	q := sqlcgen.New(tx)
	inv, err := q.CreateEndpointInventory(ctx, sqlcgen.CreateEndpointInventoryParams{
		TenantID:     tenantUUID,
		EndpointID:   endpointID,
		ScannedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
		PackageCount: 500,
	})
	if err != nil {
		t.Fatalf("CreateEndpointInventory: %v", err)
	}

	packages := make([]*pb.PackageInfo, 500)
	for i := range packages {
		packages[i] = &pb.PackageInfo{
			Name:         fmt.Sprintf("pkg-%04d", i),
			Version:      fmt.Sprintf("%d.0.0", i),
			Architecture: "amd64",
			Source:       "apt",
		}
	}

	start := time.Now()
	count, err := st.BulkInsertEndpointPackages(ctx, tx, tenantUUID, endpointID, inv.ID, packages)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("BulkInsertEndpointPackages (500 rows): %v", err)
	}
	if count != 500 {
		t.Errorf("inserted count = %d, want 500", count)
	}
	t.Logf("inserted 500 packages via batched INSERT in %v", elapsed)

	// Verify row count via ListEndpointPackages.
	pkgs, err := q.ListEndpointPackages(ctx, sqlcgen.ListEndpointPackagesParams{
		InventoryID: inv.ID,
		TenantID:    tenantUUID,
	})
	if err != nil {
		t.Fatalf("ListEndpointPackages: %v", err)
	}
	if len(pkgs) != 500 {
		t.Errorf("ListEndpointPackages returned %d rows, want 500", len(pkgs))
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}
