package events_test

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
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

	// Run hub store migrations.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "store", "migrations")
	if err := runMigrations(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	// Set hub_app password.
	if _, err := pool.Exec(ctx,
		fmt.Sprintf("ALTER ROLE hub_app WITH PASSWORD '%s'", hubRolePass),
	); err != nil {
		t.Fatalf("set hub_app password: %v", err)
	}

	// Seed tenants.
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

func openStdlibDB(t *testing.T, superPool *pgxpool.Pool) *sql.DB {
	t.Helper()
	cfg := superPool.Config().ConnConfig
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password,
	)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open stdlib DB: %v", err)
	}
	return db
}

func TestAuditSubscriber_PersistsEvent(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := hubAppPool(t, superPool)
	defer app.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sub := events.NewAuditSubscriber(app, logger)

	evt := domain.NewAuditEvent(
		"config.updated",
		tenantA,
		"user-1",
		domain.ActorUser,
		"hub_config",
		"cfg-1",
		"updated",
		map[string]any{"key": "retention_days"},
		domain.EventMeta{TraceID: "trace-1", RequestID: "req-1"},
	)

	if err := sub.Handle(ctx, evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Verify event in audit_events table via superuser (bypasses RLS).
	var storedType, storedActorID, storedResource string
	err := superPool.QueryRow(ctx,
		"SELECT type, actor_id, resource FROM audit_events WHERE id = $1", evt.ID,
	).Scan(&storedType, &storedActorID, &storedResource)
	if err != nil {
		t.Fatalf("query audit event: %v", err)
	}

	if storedType != "config.updated" {
		t.Errorf("type = %q, want %q", storedType, "config.updated")
	}
	if storedActorID != "user-1" {
		t.Errorf("actor_id = %q, want %q", storedActorID, "user-1")
	}
	if storedResource != "hub_config" {
		t.Errorf("resource = %q, want %q", storedResource, "hub_config")
	}
}

func TestAuditSubscriber_TenantIsolation(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := hubAppPool(t, superPool)
	defer app.Close()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	sub := events.NewAuditSubscriber(app, logger)

	evtA := domain.NewAuditEvent("config.updated", tenantA, "user-a", domain.ActorUser, "hub_config", "cfg-a", "updated", nil, domain.EventMeta{})
	evtB := domain.NewAuditEvent("config.updated", tenantB, "user-b", domain.ActorUser, "hub_config", "cfg-b", "updated", nil, domain.EventMeta{})

	if err := sub.Handle(ctx, evtA); err != nil {
		t.Fatalf("Handle tenant A: %v", err)
	}
	if err := sub.Handle(ctx, evtB); err != nil {
		t.Fatalf("Handle tenant B: %v", err)
	}

	// Query as tenant A via app role — should only see tenant A's event.
	tx, err := app.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}

	rows, err := tx.Query(ctx, "SELECT id FROM audit_events")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(): %v", err)
	}

	if len(ids) != 1 {
		t.Errorf("tenant A sees %d events, want 1", len(ids))
	}
	if len(ids) == 1 && ids[0] != evtA.ID {
		t.Errorf("tenant A sees event %s, want %s", ids[0], evtA.ID)
	}
}

func TestEmit_RejectsUnregisteredTopic(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()

	sqlDB := openStdlibDB(t, superPool)
	defer sqlDB.Close()

	wmLogger := watermill.NewStdLogger(false, false)
	pub, subFactory, err := events.NewPublisherAndSubscriberFactory(sqlDB, wmLogger)
	if err != nil {
		t.Fatalf("create publisher/subscriber factory: %v", err)
	}
	router, err := events.NewRouter(wmLogger)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	bus := events.NewWatermillEventBus(pub, subFactory, router, wmLogger)
	defer bus.Close()

	evt := domain.NewAuditEvent(
		"unknown.unregistered",
		tenantA,
		"user-1",
		domain.ActorUser,
		"unknown",
		"unk-1",
		"created",
		nil,
		domain.EventMeta{},
	)

	err = bus.Emit(context.Background(), evt)
	if err == nil {
		t.Fatal("Emit with unregistered topic should return error, got nil")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should mention 'not registered', got: %v", err)
	}
}

func TestWildcardPatternMatching(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    int
	}{
		{"global wildcard", "*", len(events.AllTopics())},
		{"tenant prefix", "tenant.*", 2},
		{"exact match", "config.updated", 1},
		{"unknown topic returns literal", "unknown.topic", 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := events.MatchingTopics(tc.pattern)
			if len(matched) != tc.want {
				t.Errorf("pattern %q matched %d topics, want %d: %v", tc.pattern, len(matched), tc.want, matched)
			}
		})
	}
}
