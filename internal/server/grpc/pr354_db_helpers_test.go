package grpc_test

// Database helper for tests added in PR #354: enrollment token expiry and
// heartbeat config push. Mirrors the helpers in inventory_integration_test.go
// but without the `integration` build tag so the new TestEnroll_ExpiredToken
// and TestCheckConfigUpdate tests run under `go test ./internal/server/grpc/...`.

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
	pr354TestDBName  = "patchiq_pr354_test"
	pr354TestDBUser  = "postgres"
	pr354TestDBPass  = "postgres"
	pr354AppRolePass = "pr354_app_pass"
	pr354DefTenant   = "00000000-0000-0000-0000-000000000001"
)

// setupPR354DB starts a PostgreSQL 16 container, runs all migrations, and
// returns a superuser pool plus a cleanup function.
func setupPR354DB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(pr354TestDBName),
		postgres.WithUsername(pr354TestDBUser),
		postgres.WithPassword(pr354TestDBPass),
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

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "store", "migrations")

	if err := runPR354Migrations(ctx, superPool, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if _, err := superPool.Exec(ctx,
		fmt.Sprintf("ALTER ROLE patchiq_app WITH PASSWORD '%s'", pr354AppRolePass),
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

func runPR354Migrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
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

		upSQL, err := extractPR354UpSection(string(content))
		if err != nil {
			return fmt.Errorf("extract Up section from %s: %w", filepath.Base(f), err)
		}
		if upSQL == "" {
			continue
		}
		// pgxpool.Exec wraps the SQL in an implicit transaction. CREATE INDEX
		// CONCURRENTLY cannot run inside a transaction, so we strip the
		// CONCURRENTLY keyword for test-only migrations (matches the store
		// package's test harness).
		if strings.Contains(string(content), "-- +goose NO TRANSACTION") {
			upSQL = strings.ReplaceAll(upSQL, "CONCURRENTLY ", "")
		}
		if _, err := pool.Exec(ctx, upSQL); err != nil {
			return fmt.Errorf("execute migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

func extractPR354UpSection(content string) (string, error) {
	const upMarker = "-- +goose Up"
	const downMarker = "-- +goose Down"

	upIdx := strings.Index(content, upMarker)
	if upIdx == -1 {
		return "", fmt.Errorf("missing %q marker", upMarker)
	}
	upIdx += len(upMarker)

	downIdx := strings.Index(content, downMarker)
	if downIdx == -1 {
		return strings.TrimSpace(content[upIdx:]), nil
	}
	return strings.TrimSpace(content[upIdx:downIdx]), nil
}

// pr354AppPool returns a pgxpool connected as the patchiq_app role.
func pr354AppPool(t *testing.T, superPool *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	cfg := superPool.Config().ConnConfig
	appConnStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=patchiq_app password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, pr354AppRolePass,
	)
	pool, err := pgxpool.New(ctx, appConnStr)
	if err != nil {
		t.Fatalf("create app pool: %v", err)
	}
	return pool
}
