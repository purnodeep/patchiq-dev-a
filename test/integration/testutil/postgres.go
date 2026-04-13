//go:build integration

// Package testutil provides shared test helpers for PatchIQ integration tests.
// It starts a PostgreSQL 16 testcontainer, runs all server migrations, creates
// the patchiq_app role connection pool, and offers seed helpers for tenants and
// enrollment tokens.
package testutil

import (
	"context"
	"fmt"
	"net"
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
	testDBName  = "patchiq_integ"
	testDBUser  = "postgres"
	testDBPass  = "postgres"
	appRolePass = "test_app_pass"
)

// TestDB holds the resources created by SetupTestDB. Callers must defer
// Cleanup() to terminate the container and close the pool.
type TestDB struct {
	// SuperPool is a pgxpool connected as the postgres superuser.
	// Use it for seeding data that bypasses RLS.
	SuperPool *pgxpool.Pool

	cleanup func()
}

// Cleanup releases the superuser pool and terminates the PostgreSQL container.
func (db *TestDB) Cleanup() {
	if db.cleanup != nil {
		db.cleanup()
	}
}

// SetupTestDB starts a PostgreSQL 16 container, applies all server migrations,
// sets a known password on the patchiq_app role, and returns a TestDB.
func SetupTestDB(t *testing.T) *TestDB {
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

	migrationsDir := findMigrationsDir(t)

	if err := runMigrations(ctx, superPool, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// ALTER ROLE ... WITH PASSWORD does not support $1 parameters.
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

	return &TestDB{
		SuperPool: superPool,
		cleanup:   cleanup,
	}
}

// AppPool returns a pgxpool connected as the patchiq_app role. The caller
// should defer pool.Close().
func AppPool(t *testing.T, superPool *pgxpool.Pool) *pgxpool.Pool {
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

// SeedTenant inserts a tenant via the superuser pool (bypassing RLS) and
// returns the tenant ID as a string UUID.
func SeedTenant(t *testing.T, ctx context.Context, superPool *pgxpool.Pool, name, slug string) string {
	t.Helper()

	var tenantID string
	if err := superPool.QueryRow(ctx,
		"INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id::text",
		name, slug,
	).Scan(&tenantID); err != nil {
		t.Fatalf("seed tenant %q: %v", name, err)
	}
	return tenantID
}

// SeedEnrollmentToken inserts an agent_registrations row for the given tenant
// and returns the registration token string. The registration is created with
// status 'pending' and no endpoint_id.
func SeedEnrollmentToken(t *testing.T, ctx context.Context, superPool *pgxpool.Pool, tenantID, token string) string {
	t.Helper()

	if _, err := superPool.Exec(ctx,
		"INSERT INTO agent_registrations (tenant_id, registration_token, status) VALUES ($1, $2, 'pending')",
		tenantID, token,
	); err != nil {
		t.Fatalf("seed enrollment token %q for tenant %s: %v", token, tenantID, err)
	}
	return token
}

// FindFreePort returns a free TCP port on localhost by briefly binding to :0
// and immediately releasing the socket.
func FindFreePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatalf("close listener for free port: %v", err)
	}
	return port
}

// findMigrationsDir resolves the path to internal/server/store/migrations/
// relative to this source file's location. The layout is:
//
//	test/integration/testutil/postgres.go   <- this file
//	internal/server/store/migrations/       <- target
//
// So we go up 3 directories from the file's directory to reach the repo root.
func findMigrationsDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: cannot determine test file path")
	}
	// filename = .../test/integration/testutil/postgres.go
	// Dir(filename) = .../test/integration/testutil
	// Up 3 levels   = .../  (repo root)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	migrationsDir := filepath.Join(repoRoot, "internal", "server", "store", "migrations")

	if _, err := os.Stat(migrationsDir); err != nil {
		t.Fatalf("migrations directory not found at %s: %v", migrationsDir, err)
	}
	return migrationsDir
}

// runMigrations executes all *.sql migration files in ascending order.
// It extracts the "-- +goose Up" section from each file and executes it
// directly via the superuser pool, avoiding goose's SQL parser which can
// break DO $$ blocks.
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

		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return fmt.Errorf("execute migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// extractUpSection returns the SQL between "-- +goose Up" and "-- +goose Down"
// (or end of file if there is no Down section).
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
