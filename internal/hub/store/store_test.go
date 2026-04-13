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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDBName  = "patchiq_hub_test"
	testDBUser  = "postgres"
	testDBPass  = "postgres"
	hubRolePass = "test_hub_pass"
	tenantA     = "00000000-0000-0000-0000-000000000001"
	tenantB     = "00000000-0000-0000-0000-000000000002"
)

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

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("create pgxpool: %v", err)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "migrations")
	if err := runMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if _, err := pool.Exec(ctx,
		fmt.Sprintf("ALTER ROLE hub_app WITH PASSWORD '%s'", hubRolePass),
	); err != nil {
		t.Fatalf("set hub_app password: %v", err)
	}

	for _, tid := range []string{tenantA, tenantB} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO tenants (id, name, slug) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING",
			tid, "tenant-"+tid[:8], "t-"+tid[:8],
		); err != nil {
			t.Fatalf("seed tenant %s: %v", tid, err)
		}
	}

	cleanup := func() {
		pool.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	}
	return pool, cleanup
}

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

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		upSQL := extractGooseUp(string(content))
		if upSQL == "" {
			continue
		}
		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return fmt.Errorf("exec migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

func extractGooseUp(content string) string {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"

	_, rest, found := strings.Cut(content, upMarker)
	if !found {
		return ""
	}
	if before, _, hasDown := strings.Cut(rest, downMarker); hasDown {
		rest = before
	}
	return strings.TrimSpace(rest)
}

func hubAppPool(t *testing.T, superPool *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	cfg := superPool.Config().ConnConfig
	appConnStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=hub_app password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, hubRolePass,
	)
	pool, err := pgxpool.New(ctx, appConnStr)
	if err != nil {
		t.Fatalf("create hub_app pool: %v", err)
	}
	return pool
}

func TestMigrations_RunCleanly(t *testing.T) {
	_, cleanup := setupTestDB(t)
	defer cleanup()
}

func TestMigrations_AllTablesExist(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	expectedTables := []string{
		"tenants", "patch_catalog", "cve_feeds", "agent_binaries",
		"hub_config", "audit_events",
	}

	for _, table := range expectedTables {
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = $1)",
			table,
		).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s does not exist", table)
		}
	}
}

func TestRLS_AuditEventTenantIsolation(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := hubAppPool(t, superPool)
	defer app.Close()

	// ULID IDs must be exactly 26 characters.
	auditIDs := map[string]string{
		tenantA: "01JTESTRLSAUDITTENANTA0001", // 26 chars
		tenantB: "01JTESTRLSAUDITTENANTB0002", // 26 chars
	}
	for _, tid := range []string{tenantA, tenantB} {
		_, err := superPool.Exec(ctx,
			`INSERT INTO audit_events (id, tenant_id, type, actor_id, actor_type, resource, resource_id, action, timestamp)
			 VALUES ($1, $2, 'test.created', 'user-1', 'user', 'test', 'test-1', 'created', now())`,
			auditIDs[tid], tid,
		)
		if err != nil {
			t.Fatalf("insert audit event for %s: %v", tid, err)
		}
	}

	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	rows, err := tx.Query(ctx, "SELECT tenant_id FROM audit_events")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var tid string
		if err := rows.Scan(&tid); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if tid != tenantA {
			t.Errorf("tenant A sees event from tenant %s", tid)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(): %v", err)
	}
	if count != 1 {
		t.Errorf("tenant A sees %d events, want 1", count)
	}
}

func TestRLS_HubConfigTenantIsolation(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := hubAppPool(t, superPool)
	defer app.Close()

	for _, tid := range []string{tenantA, tenantB} {
		_, err := superPool.Exec(ctx,
			`INSERT INTO hub_config (tenant_id, key, value, updated_by)
			 VALUES ($1, 'test_key', '"test_value"', '00000000-0000-0000-0000-000000000099')`,
			tid,
		)
		if err != nil {
			t.Fatalf("insert hub_config for %s: %v", tid, err)
		}
	}

	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	var count int
	err = tx.QueryRow(ctx, "SELECT count(*) FROM hub_config").Scan(&count)
	if err != nil {
		t.Fatalf("count hub_config: %v", err)
	}
	if count != 1 {
		t.Errorf("tenant A sees %d config rows, want 1", count)
	}
}

func TestCheckConstraints_AuditActorType(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO audit_events (id, tenant_id, type, actor_id, actor_type, resource, resource_id, action, timestamp)
		 VALUES ('01JTESTVALIDACTORTYPE00001', $1, 'test.created', 'u1', 'user', 'test', 't1', 'created', now())`,
		tenantA,
	)
	if err != nil {
		t.Fatalf("valid actor_type insert failed: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO audit_events (id, tenant_id, type, actor_id, actor_type, resource, resource_id, action, timestamp)
		 VALUES ('01JTESTBADACTORTYPECHECK02', $1, 'test.created', 'u1', 'robot', 'test', 't2', 'created', now())`,
		tenantA,
	)
	if err == nil {
		t.Fatal("invalid actor_type insert should fail, got nil")
	}
}

func TestCheckConstraints_AuditIDLength(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO audit_events (id, tenant_id, type, actor_id, actor_type, resource, resource_id, action, timestamp)
		 VALUES ('short', $1, 'test.created', 'u1', 'user', 'test', 't1', 'created', now())`,
		tenantA,
	)
	if err == nil {
		t.Fatal("short ULID insert should fail, got nil")
	}
}

func TestCheckConstraints_CatalogSeverity(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO patch_catalog (name, vendor, os_family, version, severity)
		 VALUES ('test', 'vendor', 'linux', '1.0', 'invalid')`,
	)
	if err == nil {
		t.Fatal("invalid severity insert should fail, got nil")
	}
}

func TestCheckConstraints_CVESeverity(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO cve_feeds (cve_id, severity) VALUES ('CVE-2026-0001', 'invalid')`,
	)
	if err == nil {
		t.Fatal("invalid severity insert should fail, got nil")
	}
}

func TestAuditEvents_HubAppCanInsert(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := hubAppPool(t, superPool)
	defer app.Close()

	// hub_app should be able to INSERT audit events when tenant context is set.
	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO audit_events (id, tenant_id, type, actor_id, actor_type, resource, resource_id, action, timestamp)
		 VALUES ('01JTESTHUBAPPINSERTAUDIT01', $1, 'config.updated', 'user-1', 'user', 'hub_config', 'cfg-1', 'updated', now())`,
		tenantA,
	)
	if err != nil {
		t.Fatalf("hub_app INSERT into audit_events should succeed: %v", err)
	}

	// Verify the row is readable within the same transaction.
	var storedType string
	err = tx.QueryRow(ctx, "SELECT type FROM audit_events WHERE id = '01JTESTHUBAPPINSERTAUDIT01'").Scan(&storedType)
	if err != nil {
		t.Fatalf("read back audit event: %v", err)
	}
	if storedType != "config.updated" {
		t.Errorf("type = %q, want %q", storedType, "config.updated")
	}
}

func TestAuditEvents_AppendOnly(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := hubAppPool(t, superPool)
	defer app.Close()

	_, err := superPool.Exec(ctx,
		`INSERT INTO audit_events (id, tenant_id, type, actor_id, actor_type, resource, resource_id, action, timestamp)
		 VALUES ('01JTESTAPPENDONLYAUDIT0001', $1, 'test.created', 'u1', 'user', 'test', 't1', 'created', now())`,
		tenantA,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	_, err = tx.Exec(ctx,
		"UPDATE audit_events SET type = 'modified' WHERE id = '01JTESTAPPENDONLYAUDIT0001'",
	)
	if err == nil {
		t.Fatal("UPDATE on audit_events should be denied for hub_app, got nil")
	}

	_, err = tx.Exec(ctx,
		"DELETE FROM audit_events WHERE id = '01JTESTAPPENDONLYAUDIT0001'",
	)
	if err == nil {
		t.Fatal("DELETE on audit_events should be denied for hub_app, got nil")
	}
}
