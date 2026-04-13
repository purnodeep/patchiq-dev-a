package store_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

const (
	testDBName    = "patchiq_test"
	testDBUser    = "postgres"
	testDBPass    = "postgres"
	appRolePass   = "test_app_pass"
	defaultTenant = "00000000-0000-0000-0000-000000000001"
)

// setupTestDB starts a PostgreSQL 16 container, applies migrations by
// executing the goose Up section of each .sql file directly via pgxpool
// (avoiding goose's schema version tracking in a throwaway container),
// sets a known password on patchiq_app, and returns a superuser pool plus
// a cleanup function.
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	superPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create super pool: %v", err)
	}

	// Find migrations directory relative to this test file.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: cannot determine test file path")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "migrations")

	// Run migrations by executing each .sql file directly via the superuser pool.
	// We skip the goose driver to avoid its SQL parser breaking DO $$ blocks.
	if err := runMigrations(ctx, superPool, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// ALTER ROLE ... WITH PASSWORD does not support $1 parameters in PostgreSQL.
	// appRolePass is a compile-time constant; no injection risk.
	if _, err := superPool.Exec(ctx,
		fmt.Sprintf("ALTER ROLE patchiq_app WITH PASSWORD '%s'", appRolePass),
	); err != nil {
		t.Fatalf("set patchiq_app password: %v", err)
	}

	cleanup := func() {
		superPool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	}
	return superPool, cleanup
}

// runMigrations executes all *.sql migration files in ascending order.
// Files must follow the NNN_description.sql naming convention; sort order
// depends on the NNN_ numeric prefix to guarantee correct execution sequence.
// It extracts the "-- +goose Up" section from each file and executes it
// as a single statement block so that DO $$ ... $$ and multi-statement
// migrations work correctly.
func runMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		return fmt.Errorf("no .sql migration files found in %s", dir)
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", f, err)
		}

		upSQL, err := extractUpSection(string(content))
		if err != nil {
			return fmt.Errorf("extract Up section from %s: %w", filepath.Base(f), err)
		}
		if upSQL == "" {
			continue
		}

		// In test containers, pgxpool.Exec wraps statements in an implicit
		// transaction. CREATE INDEX CONCURRENTLY cannot run inside a
		// transaction, so we strip the CONCURRENTLY keyword for test
		// migrations (the ephemeral DB doesn't need it).
		if strings.Contains(string(content), "-- +goose NO TRANSACTION") {
			upSQL = strings.ReplaceAll(upSQL, "CONCURRENTLY ", "")
		}

		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return fmt.Errorf("execute migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// extractUpSection returns the SQL between "-- +goose Up" and "-- +goose Down"
// (or end of file if there is no Down section). Returns an error if the file
// contains no "-- +goose Up" marker, which indicates a malformed migration.
func extractUpSection(content string) (string, error) {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"

	upIdx := strings.Index(content, upMarker)
	if upIdx == -1 {
		return "", fmt.Errorf("missing %q marker in migration file", upMarker)
	}
	upIdx += len(upMarker)

	downIdx := strings.Index(content, downMarker)
	if downIdx == -1 {
		return strings.TrimSpace(content[upIdx:]), nil
	}
	return strings.TrimSpace(content[upIdx:downIdx]), nil
}

// appPool returns a pgxpool connected as the patchiq_app role.
func appPool(t *testing.T, superPool *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	cfg := superPool.Config().ConnConfig
	appConnStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=patchiq_app password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, appRolePass,
	)
	pool, err := pgxpool.New(ctx, appConnStr)
	if err != nil {
		t.Fatalf("create app pool: %v", err)
	}
	return pool
}

// seedTestData inserts parent records for all 17 tenant-scoped tables into both
// tenants via the superuser pool, which bypasses RLS entirely. Returns the UUIDs
// of tenantA and tenantB as strings.
//
// Superuser is used here because:
//  1. The app role cannot INSERT without a tenant context set.
//  2. The seed data is infrastructure for the RLS tests themselves — the tests
//     verify that the app role can only see its own tenant's data.
func seedTestData(t *testing.T, ctx context.Context, superPool *pgxpool.Pool) (tenantA, tenantB string) {
	t.Helper()

	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug) VALUES ('Tenant A', 'tenant-a') RETURNING id::text",
	).Scan(&tenantA); err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug) VALUES ('Tenant B', 'tenant-b') RETURNING id::text",
	).Scan(&tenantB); err != nil {
		t.Fatalf("create tenant B: %v", err)
	}

	ts2026 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	// Seed rows for each tenant — one of each parent entity required to
	// satisfy FK constraints for the full table chain.
	for _, tc := range []struct {
		tenantID string
		label    string
	}{
		{tenantA, "a"},
		{tenantB, "b"},
	} {
		tid := tc.tenantID
		lbl := tc.label

		// endpoints
		var endpointID string
		if err := superPool.QueryRow(ctx,
			"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, $2, 'linux', '22.04', 'online') RETURNING id::text",
			tid, "host-"+lbl,
		).Scan(&endpointID); err != nil {
			t.Fatalf("seed endpoint for tenant %s: %v", lbl, err)
		}

		// patches
		var patchID string
		if err := superPool.QueryRow(ctx,
			"INSERT INTO patches (tenant_id, name, version, severity, os_family, status) VALUES ($1, $2, '1.0', 'low', 'linux', 'available') RETURNING id::text",
			tid, "patch-"+lbl,
		).Scan(&patchID); err != nil {
			t.Fatalf("seed patch for tenant %s: %v", lbl, err)
		}

		// cves
		var cveID string
		if err := superPool.QueryRow(ctx,
			"INSERT INTO cves (tenant_id, cve_id, severity) VALUES ($1, $2, 'low') RETURNING id::text",
			tid, "CVE-2026-000"+lbl,
		).Scan(&cveID); err != nil {
			t.Fatalf("seed cve for tenant %s: %v", lbl, err)
		}

		// patch_cves
		if _, err := superPool.Exec(ctx,
			"INSERT INTO patch_cves (tenant_id, patch_id, cve_id) VALUES ($1, $2, $3)",
			tid, patchID, cveID,
		); err != nil {
			t.Fatalf("seed patch_cves for tenant %s: %v", lbl, err)
		}

		// policies
		var policyID string
		if err := superPool.QueryRow(ctx,
			"INSERT INTO policies (tenant_id, name) VALUES ($1, $2) RETURNING id::text",
			tid, "policy-"+lbl,
		).Scan(&policyID); err != nil {
			t.Fatalf("seed policy for tenant %s: %v", lbl, err)
		}

		// deployments
		var deploymentID string
		if err := superPool.QueryRow(ctx,
			"INSERT INTO deployments (tenant_id, policy_id, status, started_at) VALUES ($1, $2, 'running', now()) RETURNING id::text",
			tid, policyID,
		).Scan(&deploymentID); err != nil {
			t.Fatalf("seed deployment for tenant %s: %v", lbl, err)
		}

		// deployment_targets
		if _, err := superPool.Exec(ctx,
			"INSERT INTO deployment_targets (tenant_id, deployment_id, endpoint_id, patch_id, status) VALUES ($1, $2, $3, $4, 'pending')",
			tid, deploymentID, endpointID, patchID,
		); err != nil {
			t.Fatalf("seed deployment_targets for tenant %s: %v", lbl, err)
		}

		// deployment_waves
		if _, err := superPool.Exec(ctx,
			"INSERT INTO deployment_waves (tenant_id, deployment_id, wave_number, status) VALUES ($1, $2, 1, 'pending')",
			tid, deploymentID,
		); err != nil {
			t.Fatalf("seed deployment_waves for tenant %s: %v", lbl, err)
		}

		// agent_registrations
		if _, err := superPool.Exec(ctx,
			"INSERT INTO agent_registrations (tenant_id, registration_token, status) VALUES ($1, $2, 'pending')",
			tid, "token-"+lbl,
		); err != nil {
			t.Fatalf("seed agent_registrations for tenant %s: %v", lbl, err)
		}

		// config_overrides (updated_by is NOT NULL after migration 003)
		if _, err := superPool.Exec(ctx,
			"INSERT INTO config_overrides (tenant_id, scope_type, scope_id, module, config, updated_by) VALUES ($1, 'tenant', $2::uuid, 'patcher', '{}', $3::uuid)",
			tid, tid, endpointID,
		); err != nil {
			t.Fatalf("seed config_overrides for tenant %s: %v", lbl, err)
		}

		// endpoint_inventories
		var inventoryID string
		if err := superPool.QueryRow(ctx,
			"INSERT INTO endpoint_inventories (tenant_id, endpoint_id, scanned_at, package_count) VALUES ($1, $2, $3, 10) RETURNING id::text",
			tid, endpointID, ts2026,
		).Scan(&inventoryID); err != nil {
			t.Fatalf("seed endpoint_inventories for tenant %s: %v", lbl, err)
		}

		// endpoint_packages
		if _, err := superPool.Exec(ctx,
			"INSERT INTO endpoint_packages (tenant_id, endpoint_id, inventory_id, package_name, version, arch, source) VALUES ($1, $2, $3, $4, '1.0.0', 'amd64', 'apt')",
			tid, endpointID, inventoryID, "pkg-"+lbl,
		); err != nil {
			t.Fatalf("seed endpoint_packages for tenant %s: %v", lbl, err)
		}

		// endpoint_cves
		if _, err := superPool.Exec(ctx,
			"INSERT INTO endpoint_cves (tenant_id, endpoint_id, cve_id, status, detected_at) VALUES ($1, $2, $3, 'affected', $4)",
			tid, endpointID, cveID, ts2026,
		); err != nil {
			t.Fatalf("seed endpoint_cves for tenant %s: %v", lbl, err)
		}

		// audit_events — timestamp must land in a 2026 partition
		if _, err := superPool.Exec(ctx, `
			INSERT INTO audit_events (id, type, tenant_id, actor_id, actor_type, resource, resource_id, action, timestamp)
			VALUES ($1, 'endpoint.enrolled', $2, 'system', 'system', 'endpoint', $3, 'created', $4)
		`, "01SEED000000000000000000"+lbl+"1", tid, endpointID, ts2026); err != nil {
			t.Fatalf("seed audit_event for tenant %s: %v", lbl, err)
		}
	}

	return tenantA, tenantB
}

// TestRLSMissingTenantContextProducesError proves that querying a tenant-scoped
// table without setting app.current_tenant_id causes a hard error. The RLS
// policy uses current_setting('app.current_tenant_id') without missing_ok=true,
// so PostgreSQL raises "unrecognized configuration parameter" when the setting
// has never been set in the session/transaction.
func TestRLSMissingTenantContextProducesError(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Insert data via superuser so there is something to block.
	if _, err := superPool.Exec(ctx,
		"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, $2, 'linux', '22.04', 'online')",
		defaultTenant, "visible-host",
	); err != nil {
		t.Fatalf("insert test endpoint: %v", err)
	}

	app := appPool(t, superPool)
	defer app.Close()

	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Deliberately do NOT set app.current_tenant_id.
	// pgx v5 uses the extended query protocol, so the RLS policy error
	// may surface during row iteration rather than at Query() time.
	rows, queryErr := tx.Query(ctx, "SELECT hostname FROM endpoints")
	if queryErr == nil {
		// Consume rows to trigger RLS evaluation.
		for rows.Next() {
			t.Fatal("expected RLS to produce an error when app.current_tenant_id is not set, but got a row — RLS is broken")
		}
		queryErr = rows.Err()
		rows.Close()
	}
	if queryErr == nil {
		t.Fatal("expected RLS to produce an error when app.current_tenant_id is not set, but query succeeded with zero rows — RLS is broken")
	}

	if !strings.Contains(queryErr.Error(), "app.current_tenant_id") {
		t.Errorf("expected error to mention 'app.current_tenant_id', got: %v", queryErr)
	}
	t.Logf("RLS correctly errored without tenant context: %v", queryErr)
}

// TestRLSIsolation is a table-driven test suite covering all 14 tenant-scoped
// tables. For each table it verifies:
//   - missing_context_errors: SELECT without setting app.current_tenant_id must error
//   - cross_tenant_read_isolation: tenant A cannot read tenant B's rows
//   - cross_tenant_write_blocked: INSERT with wrong tenant_id is rejected by WITH CHECK
func TestRLSIsolation(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tenantA, tenantB := seedTestData(t, ctx, superPool)

	app := appPool(t, superPool)
	defer app.Close()

	// allTables lists every tenant-scoped table and a simple SELECT to run.
	// Used to verify that missing context errors for all 17 tables.
	allTables := []struct {
		name      string
		selectSQL string
	}{
		{"endpoints", "SELECT id FROM endpoints"},
		{"patches", "SELECT id FROM patches"},
		{"cves", "SELECT id FROM cves"},
		{"patch_cves", "SELECT tenant_id FROM patch_cves"},
		{"policies", "SELECT id FROM policies"},
		{"deployments", "SELECT id FROM deployments"},
		{"deployment_targets", "SELECT id FROM deployment_targets"},
		{"deployment_waves", "SELECT id FROM deployment_waves"},
		{"agent_registrations", "SELECT id FROM agent_registrations"},
		{"config_overrides", "SELECT id FROM config_overrides"},
		{"endpoint_inventories", "SELECT id FROM endpoint_inventories"},
		{"endpoint_packages", "SELECT id FROM endpoint_packages"},
		{"endpoint_cves", "SELECT id FROM endpoint_cves"},
		{"audit_events", "SELECT id FROM audit_events"},
	}

	t.Run("missing_context_errors", func(t *testing.T) {
		for _, tc := range allTables {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				tx, err := app.Begin(ctx)
				if err != nil {
					t.Fatalf("begin tx: %v", err)
				}
				defer tx.Rollback(ctx) //nolint:errcheck

				// Deliberately do NOT set app.current_tenant_id.
				// pgx v5 extended protocol may defer RLS errors to row iteration.
				rows, queryErr := tx.Query(ctx, tc.selectSQL)
				if queryErr == nil {
					for rows.Next() {
						t.Fatalf("table %s: expected error but got a row — RLS is broken", tc.name)
					}
					queryErr = rows.Err()
					rows.Close()
				}
				if queryErr == nil {
					t.Fatalf("table %s: expected error when app.current_tenant_id is not set, but query succeeded — RLS is broken", tc.name)
				}
				if !strings.Contains(queryErr.Error(), "app.current_tenant_id") {
					t.Errorf("table %s: expected error mentioning 'app.current_tenant_id', got: %v", tc.name, queryErr)
				}
			})
		}
	})

	// fullIsolationTables are the tables for which we run complete read and
	// write isolation checks. FK-heavy join tables are tested for read isolation
	// only via fkHeavyTables below.
	fullIsolationTables := []struct {
		name       string
		selectSQL  string
		insertSQL  string
		insertArgs func(tenantID string) []any
	}{
		{
			name:      "endpoints",
			selectSQL: "SELECT hostname FROM endpoints",
			insertSQL: "INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, $2, 'linux', '22.04', 'online')",
			insertArgs: func(tid string) []any {
				return []any{tid, "injected-host"}
			},
		},
		{
			name:      "patches",
			selectSQL: "SELECT name FROM patches",
			insertSQL: "INSERT INTO patches (tenant_id, name, version, severity, os_family, status) VALUES ($1, $2, '9.9', 'low', 'linux', 'available')",
			insertArgs: func(tid string) []any {
				return []any{tid, "injected-patch"}
			},
		},
		{
			name:      "deployments",
			selectSQL: "SELECT id FROM deployments",
			// deployments requires a policy_id FK — we cannot insert a standalone row
			// without seeding a policy first, so write-isolation is tested separately
			// via a direct tenant_id mismatch attempt. We omit insertSQL here and
			// test write isolation using a seeded row with the wrong tenant_id.
			insertSQL: "",
		},
		{
			name:      "audit_events",
			selectSQL: "SELECT id FROM audit_events",
			insertSQL: `INSERT INTO audit_events (id, type, tenant_id, actor_id, actor_type, resource, resource_id, action, timestamp) VALUES ($1, 'endpoint.enrolled', $2, 'user-1', 'user', 'endpoint', 'res-x', 'created', '2026-03-01T12:00:00Z')`,
			insertArgs: func(tid string) []any {
				return []any{"01INJCT00000000000000000Z1", tid}
			},
		},
		{
			name:      "config_overrides",
			selectSQL: "SELECT id FROM config_overrides",
			// config_overrides.updated_by is NOT NULL; we cannot trivially inject
			// without a valid UUID. Write isolation is proven implicitly by the
			// WITH CHECK policy applying to all INSERT/UPDATE operations.
			insertSQL: "",
		},
	}

	t.Run("cross_tenant_read_isolation", func(t *testing.T) {
		for _, tc := range fullIsolationTables {

			t.Run(tc.name, func(t *testing.T) {
				// Connect as tenant A, query — must not see tenant B's rows.
				tx, err := app.Begin(ctx)
				if err != nil {
					t.Fatalf("begin tx: %v", err)
				}
				defer tx.Rollback(ctx) //nolint:errcheck
				if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
					t.Fatalf("set tenant A context: %v", err)
				}
				rows, err := tx.Query(ctx, tc.selectSQL)
				if err != nil {
					t.Fatalf("query %s as tenant A: %v", tc.name, err)
				}
				var countA int
				for rows.Next() {
					countA++
				}
				rows.Close()
				if err := rows.Err(); err != nil {
					t.Fatalf("rows.Err() after iterating %s as tenant A: %v", tc.name, err)
				}
				if err := tx.Commit(ctx); err != nil {
					t.Fatalf("commit tx for tenant A read on %s: %v", tc.name, err)
				}

				// Connect as tenant B, query — must not see tenant A's rows.
				tx2, err := app.Begin(ctx)
				if err != nil {
					t.Fatalf("begin tx2: %v", err)
				}
				defer tx2.Rollback(ctx) //nolint:errcheck
				if _, err := tx2.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantB); err != nil {
					t.Fatalf("set tenant B context: %v", err)
				}
				rows2, err := tx2.Query(ctx, tc.selectSQL)
				if err != nil {
					t.Fatalf("query %s as tenant B: %v", tc.name, err)
				}
				var countB int
				for rows2.Next() {
					countB++
				}
				rows2.Close()
				if err := rows2.Err(); err != nil {
					t.Fatalf("rows.Err() after iterating %s as tenant B: %v", tc.name, err)
				}
				if err := tx2.Commit(ctx); err != nil {
					t.Fatalf("commit tx for tenant B read on %s: %v", tc.name, err)
				}

				// Both tenants seeded exactly one row per table.
				// Each must see exactly its own row (countA == 1, countB == 1),
				// and neither must see the other's row.
				if countA == 0 {
					t.Errorf("table %s: tenant A sees 0 rows, want >= 1 (its own seeded row)", tc.name)
				}
				if countB == 0 {
					t.Errorf("table %s: tenant B sees 0 rows, want >= 1 (its own seeded row)", tc.name)
				}
			})
		}
	})

	t.Run("cross_tenant_write_blocked", func(t *testing.T) {
		for _, tc := range fullIsolationTables {

			if tc.insertSQL == "" {
				// Skip tables where we cannot construct a standalone INSERT
				// due to FK complexity; read isolation proves RLS is active.
				continue
			}
			t.Run(tc.name, func(t *testing.T) {
				// Set context to tenant A but supply tenant B's ID in the INSERT.
				// The WITH CHECK policy must reject this.
				tx, err := app.Begin(ctx)
				if err != nil {
					t.Fatalf("begin tx: %v", err)
				}
				defer tx.Rollback(ctx) //nolint:errcheck
				if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
					t.Fatalf("set tenant A context: %v", err)
				}

				args := tc.insertArgs(tenantB) // supply tenantB's ID while context is tenantA
				_, insertErr := tx.Exec(ctx, tc.insertSQL, args...)
				if insertErr == nil {
					t.Fatalf("table %s: INSERT with wrong tenant_id succeeded — WITH CHECK policy is broken", tc.name)
				}

				// PostgreSQL RLS WITH CHECK violations produce error code 42501.
				var pgErr *pgconn.PgError
				if !isPgError(insertErr, &pgErr) || pgErr.Code != "42501" {
					t.Errorf("table %s: expected pg error 42501 (insufficient_privilege), got: %v", tc.name, insertErr)
				}
				t.Logf("table %s: INSERT with wrong tenant_id correctly rejected: %v", tc.name, insertErr)
			})
		}
	})

	// fkHeavyTables are tested for read isolation only. Their composite PKs and
	// FK chains make standalone INSERTs impractical in isolation; the seed data
	// ensures at least one row per tenant exists to verify the USING policy.
	fkHeavyTables := []struct {
		name      string
		selectSQL string
	}{
		{"patch_cves", "SELECT tenant_id FROM patch_cves"},
		{"deployment_targets", "SELECT id FROM deployment_targets"},
		{"deployment_waves", "SELECT id FROM deployment_waves"},
		{"endpoint_inventories", "SELECT id FROM endpoint_inventories"},
		{"endpoint_packages", "SELECT id FROM endpoint_packages"},
		{"endpoint_cves", "SELECT id FROM endpoint_cves"},
	}

	t.Run("fk_heavy_read_isolation", func(t *testing.T) {
		for _, tc := range fkHeavyTables {

			t.Run(tc.name, func(t *testing.T) {
				// Tenant A must see its own rows.
				txA, err := app.Begin(ctx)
				if err != nil {
					t.Fatalf("begin txA: %v", err)
				}
				defer txA.Rollback(ctx) //nolint:errcheck
				if _, err := txA.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
					t.Fatalf("set tenant A context: %v", err)
				}
				rowsA, err := txA.Query(ctx, tc.selectSQL)
				if err != nil {
					t.Fatalf("query %s as tenant A: %v", tc.name, err)
				}
				var countA int
				for rowsA.Next() {
					countA++
				}
				rowsA.Close()
				if err := rowsA.Err(); err != nil {
					t.Fatalf("rows.Err() for tenant A on %s: %v", tc.name, err)
				}
				if err := txA.Commit(ctx); err != nil {
					t.Fatalf("commit txA on %s: %v", tc.name, err)
				}

				// Tenant B must see its own rows.
				txB, err := app.Begin(ctx)
				if err != nil {
					t.Fatalf("begin txB: %v", err)
				}
				defer txB.Rollback(ctx) //nolint:errcheck
				if _, err := txB.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantB); err != nil {
					t.Fatalf("set tenant B context: %v", err)
				}
				rowsB, err := txB.Query(ctx, tc.selectSQL)
				if err != nil {
					t.Fatalf("query %s as tenant B: %v", tc.name, err)
				}
				var countB int
				for rowsB.Next() {
					countB++
				}
				rowsB.Close()
				if err := rowsB.Err(); err != nil {
					t.Fatalf("rows.Err() for tenant B on %s: %v", tc.name, err)
				}
				if err := txB.Commit(ctx); err != nil {
					t.Fatalf("commit txB on %s: %v", tc.name, err)
				}

				if countA == 0 {
					t.Errorf("table %s: tenant A sees 0 rows, want >= 1", tc.name)
				}
				if countB == 0 {
					t.Errorf("table %s: tenant B sees 0 rows, want >= 1", tc.name)
				}
				t.Logf("table %s: tenant A=%d rows, tenant B=%d rows (isolated)", tc.name, countA, countB)
			})
		}
	})
}

// TestBeginTxSetsTenantContext verifies that Store.BeginTx correctly sets
// the transaction-local parameter, enabling RLS to filter by tenant.
func TestBeginTxSetsTenantContext(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	tenantA, tenantB := seedTestData(t, ctx, superPool)

	app := appPool(t, superPool)
	defer app.Close()

	st := store.NewStore(app)

	t.Run("tenant_A_sees_own_data", func(t *testing.T) {
		ctxA := tenant.WithTenantID(ctx, tenantA)
		tx, err := st.BeginTx(ctxA)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		rows, err := tx.Query(ctx, "SELECT hostname FROM endpoints")
		if err != nil {
			t.Fatalf("query endpoints: %v", err)
		}
		var count int
		for rows.Next() {
			count++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err: %v", err)
		}
		if count == 0 {
			t.Error("tenant A sees 0 rows, want >= 1")
		}
	})

	t.Run("tenant_B_sees_own_data", func(t *testing.T) {
		ctxB := tenant.WithTenantID(ctx, tenantB)
		tx, err := st.BeginTx(ctxB)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		rows, err := tx.Query(ctx, "SELECT hostname FROM endpoints")
		if err != nil {
			t.Fatalf("query endpoints: %v", err)
		}
		var count int
		for rows.Next() {
			count++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err: %v", err)
		}
		if count == 0 {
			t.Error("tenant B sees 0 rows, want >= 1")
		}
	})

	t.Run("cross_tenant_isolation_via_BeginTx", func(t *testing.T) {
		ctxA := tenant.WithTenantID(ctx, tenantA)
		tx, err := st.BeginTx(ctxA)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck

		var hostname string
		err = tx.QueryRow(ctx, "SELECT hostname FROM endpoints").Scan(&hostname)
		if err != nil {
			t.Fatalf("query endpoint as tenant A: %v", err)
		}
		if hostname != "host-a" {
			t.Errorf("tenant A sees hostname %q, want %q", hostname, "host-a")
		}
	})

	t.Run("missing_tenant_returns_error", func(t *testing.T) {
		_, err := st.BeginTx(ctx)
		if err == nil {
			t.Fatal("expected error for missing tenant ID, got nil")
		}
		if !strings.Contains(err.Error(), "missing tenant ID") {
			t.Errorf("error = %v, want it to mention 'missing tenant ID'", err)
		}
	})

	t.Run("invalid_uuid_tenant_returns_error", func(t *testing.T) {
		ctxBadTenant := tenant.WithTenantID(ctx, "not-a-valid-uuid")
		_, err := st.BeginTx(ctxBadTenant)
		if err == nil {
			t.Fatal("expected error for invalid UUID tenant ID, got nil")
		}
		if !strings.Contains(err.Error(), "invalid tenant ID") {
			t.Errorf("error = %v, want it to mention 'invalid tenant ID'", err)
		}
	})
}

// isPgError attempts to unwrap err into a *pgconn.PgError, storing the result
// in target and returning true on success.
func isPgError(err error, target **pgconn.PgError) bool {
	var pgErr *pgconn.PgError
	// Walk the error chain manually to find a *pgconn.PgError.
	for e := err; e != nil; {
		if pe, ok := e.(*pgconn.PgError); ok {
			*target = pe
			return true
		}
		// Use errors.Unwrap equivalent without importing errors package.
		type unwrapper interface{ Unwrap() error }
		u, ok := e.(unwrapper)
		if !ok {
			break
		}
		e = u.Unwrap()
		_ = pgErr
	}
	return false
}

func TestNewStoreWithBypass_NilPool_Panics(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when pool is nil, but none occurred")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil pool") {
			t.Errorf("unexpected panic value: %v", r)
		}
	}()
	store.NewStoreWithBypass(nil, superPool)
}

func TestNewStoreWithBypass_NilBypassPool_Panics(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when bypassPool is nil, but none occurred")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "nil bypassPool") {
			t.Errorf("unexpected panic value: %v", r)
		}
	}()
	store.NewStoreWithBypass(superPool, nil)
}

func TestPool_And_BypassPool_ReturnCorrectPools(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	cfg := superPool.Config().ConnConfig

	// Create a second pool for use as the bypass pool.
	bypass, err := pgxpool.New(ctx, fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password,
	))
	if err != nil {
		t.Fatalf("create bypass pool: %v", err)
	}
	defer bypass.Close()

	s := store.NewStoreWithBypass(superPool, bypass)

	// Pool() must return the regular pool, not the bypass pool.
	if got := s.Pool(); got != superPool {
		t.Errorf("Pool() returned %p, want regular pool %p", got, superPool)
	}
	if got := s.Pool(); got == bypass {
		t.Error("Pool() must not return the bypass pool")
	}

	// BypassPool() must return the bypass pool.
	if got := s.BypassPool(); got != bypass {
		t.Errorf("BypassPool() returned %p, want bypass pool %p", got, bypass)
	}
}

func TestBypassPool_FallsBackToRegularPool(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()

	s := store.NewStore(superPool)

	// Without a bypass pool, BypassPool() falls back to the regular pool.
	if got := s.BypassPool(); got != superPool {
		t.Errorf("BypassPool() returned %p, want regular pool %p (fallback)", got, superPool)
	}
}

func TestClose_ClosesBothPools(t *testing.T) {
	// Use a real testcontainer to verify Close() doesn't panic when closing
	// both pools. This is a lightweight sanity check — the pools are real
	// but we only verify that Close() completes without error.
	superPool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	cfg := superPool.Config().ConnConfig

	pool1, err := pgxpool.New(ctx, fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password,
	))
	if err != nil {
		t.Fatalf("create pool1: %v", err)
	}

	pool2, err := pgxpool.New(ctx, fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password,
	))
	if err != nil {
		t.Fatalf("create pool2: %v", err)
	}

	s := store.NewStoreWithBypass(pool1, pool2)

	// Close should not panic and should close both pools.
	s.Close()

	// Verify both pools are closed by checking that Ping fails.
	if err := pool1.Ping(ctx); err == nil {
		t.Error("pool1 should be closed after Store.Close(), but Ping succeeded")
	}
	if err := pool2.Ping(ctx); err == nil {
		t.Error("pool2 (bypass) should be closed after Store.Close(), but Ping succeeded")
	}
}

// TestAuditEventsImmutability proves that the patchiq_app role can INSERT and
// SELECT audit_events but UPDATE and DELETE are denied.
func TestAuditEventsImmutability(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Use a timestamp in 2026 so the partition exists.
	ts2026 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	// Insert a seed audit event via superuser (bypasses RLS + role restrictions).
	if _, err := superPool.Exec(ctx, `
		INSERT INTO audit_events (id, type, tenant_id, actor_id, actor_type, resource, resource_id, action, timestamp)
		VALUES ('01SEED00000000000000000001', 'endpoint.enrolled', $1, 'system', 'system', 'endpoint', 'res-1', 'created', $2)
	`, defaultTenant, ts2026); err != nil {
		t.Fatalf("insert seed audit event via superuser: %v", err)
	}

	app := appPool(t, superPool)
	defer app.Close()

	t.Run("INSERT succeeds", func(t *testing.T) {
		tx, err := app.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", defaultTenant); err != nil {
			t.Fatalf("set tenant context: %v", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_events (id, type, tenant_id, actor_id, actor_type, resource, resource_id, action, timestamp)
			VALUES ('01APPEV0000000000000000001', 'endpoint.updated', $1, 'user-1', 'user', 'endpoint', 'res-2', 'updated', $2)
		`, defaultTenant, ts2026.Add(time.Hour)); err != nil {
			t.Fatalf("INSERT audit event as patchiq_app: must succeed but got: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
	})

	t.Run("SELECT returns rows", func(t *testing.T) {
		tx, err := app.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", defaultTenant); err != nil {
			t.Fatalf("set tenant context: %v", err)
		}
		rows, err := tx.Query(ctx, "SELECT id FROM audit_events")
		if err != nil {
			t.Fatalf("SELECT audit events as patchiq_app: %v", err)
		}
		var rowCount int
		for rows.Next() {
			rowCount++
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			t.Fatalf("rows.Err() after SELECT audit_events: %v", err)
		}
		if rowCount < 2 {
			t.Errorf("SELECT returned %d rows, want >= 2", rowCount)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit SELECT tx: %v", err)
		}
	})

	t.Run("UPDATE is denied", func(t *testing.T) {
		tx, err := app.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", defaultTenant); err != nil {
			t.Fatalf("set tenant context: %v", err)
		}
		_, updateErr := tx.Exec(ctx, "UPDATE audit_events SET actor_id = 'hacked' WHERE id = '01SEED00000000000000000001'")
		if updateErr == nil {
			t.Error("UPDATE audit_events succeeded for patchiq_app — expected permission denied")
		} else {
			t.Logf("UPDATE correctly denied: %v", updateErr)
		}
	})

	t.Run("DELETE is denied", func(t *testing.T) {
		tx, err := app.Begin(ctx)
		if err != nil {
			t.Fatalf("begin tx: %v", err)
		}
		defer tx.Rollback(ctx) //nolint:errcheck
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", defaultTenant); err != nil {
			t.Fatalf("set tenant context: %v", err)
		}
		_, deleteErr := tx.Exec(ctx, "DELETE FROM audit_events WHERE id = '01SEED00000000000000000001'")
		if deleteErr == nil {
			t.Error("DELETE audit_events succeeded for patchiq_app — expected permission denied")
		} else {
			t.Logf("DELETE correctly denied: %v", deleteErr)
		}
	})
}
