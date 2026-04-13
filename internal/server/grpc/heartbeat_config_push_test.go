package grpc_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/events"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// TestCheckConfigUpdate exercises the config-push path added by PR #354 in
// heartbeat.go / checkConfigUpdate: given a config_overrides row for an
// endpoint that was updated after the endpoint's last config_pushed_at,
// the method should return a populated *pb.AgentConfig, mark config_pushed_at,
// and emit an EndpointConfigPushed domain event.
func TestCheckConfigUpdate(t *testing.T) {
	superPool, cleanup := setupPR354DB(t)
	defer cleanup()
	// checkConfigUpdate calls store.BeginTx which requires a tenant-scoped
	// context (it reads the tenant ID to set the RLS GUC). In the real server,
	// the Heartbeat RPC injects this via tenant.WithTenantID before invoking
	// checkConfigUpdate; mirror that here.
	ctx := tenant.WithTenantID(context.Background(), pr354DefTenant)

	app := pr354AppPool(t, superPool)
	defer app.Close()

	st := store.NewStoreWithBypass(app, superPool)
	bus := &capturingEventBus{}
	quietLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := servergrpc.NewAgentServiceServer(st, bus, quietLogger)

	// Tenant UUID (default tenant seeded by migration 001).
	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(pr354DefTenant); err != nil {
		t.Fatalf("parse tenant UUID: %v", err)
	}

	// Seed an endpoint row via superuser (bypasses RLS).
	var endpointIDStr string
	if err := superPool.QueryRow(ctx,
		`INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status)
		 VALUES ($1, 'pr354-host', 'linux', '22.04', 'online')
		 RETURNING id::text`,
		pr354DefTenant,
	).Scan(&endpointIDStr); err != nil {
		t.Fatalf("seed endpoint: %v", err)
	}
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		t.Fatalf("parse endpoint UUID: %v", err)
	}
	pgEndpointID := pgtype.UUID{Bytes: endpointUUID, Valid: true}

	// Seed config_overrides row: scope_type=endpoint, module=comms, valid JSONB.
	_, err = superPool.Exec(ctx,
		`INSERT INTO config_overrides (tenant_id, scope_type, scope_id, module, config, updated_by)
		 VALUES ($1, 'endpoint', $2, 'comms', $3::jsonb, $1)`,
		pr354DefTenant, endpointIDStr,
		`{"heartbeat_interval_seconds":30,"inventory_interval_seconds":3600,"max_retry_attempts":5}`,
	)
	if err != nil {
		t.Fatalf("seed config_overrides: %v", err)
	}

	agentIDStr := endpointUUID.String()

	t.Run("first call returns config and marks pushed", func(t *testing.T) {
		cfg := svc.ExportedCheckConfigUpdate(ctx, pgEndpointID, tenantUUID, pr354DefTenant, agentIDStr, quietLogger)
		if cfg == nil {
			t.Fatal("expected non-nil AgentConfig, got nil")
		}
		if cfg.HeartbeatIntervalSeconds != 30 {
			t.Errorf("HeartbeatIntervalSeconds = %d, want 30", cfg.HeartbeatIntervalSeconds)
		}
		if cfg.InventoryIntervalSeconds != 3600 {
			t.Errorf("InventoryIntervalSeconds = %d, want 3600", cfg.InventoryIntervalSeconds)
		}
		if cfg.MaxRetryAttempts != 5 {
			t.Errorf("MaxRetryAttempts = %d, want 5", cfg.MaxRetryAttempts)
		}

		// Verify DB state: config_pushed_at is now non-null.
		var pushedAtValid bool
		if err := superPool.QueryRow(ctx,
			`SELECT config_pushed_at IS NOT NULL FROM endpoints WHERE id = $1`,
			endpointIDStr,
		).Scan(&pushedAtValid); err != nil {
			t.Fatalf("query config_pushed_at: %v", err)
		}
		if !pushedAtValid {
			t.Error("config_pushed_at should be non-null after successful config push")
		}

		// Verify event emitted.
		var found bool
		for _, evt := range bus.Events() {
			if evt.Type == events.EndpointConfigPushed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected EndpointConfigPushed event, got %d events (none matching)", len(bus.Events()))
		}
	})

	t.Run("second call returns nil (no new override)", func(t *testing.T) {
		beforeCount := len(bus.Events())
		cfg := svc.ExportedCheckConfigUpdate(ctx, pgEndpointID, tenantUUID, pr354DefTenant, agentIDStr, quietLogger)
		if cfg != nil {
			t.Errorf("expected nil on second call (no newer override), got %+v", cfg)
		}
		// No new event should have been emitted.
		if got := len(bus.Events()); got != beforeCount {
			t.Errorf("event count delta = %d, want 0", got-beforeCount)
		}
	})

	t.Run("all-zero config rejected", func(t *testing.T) {
		// Create a second endpoint + all-zero config override.
		var zeroEPStr string
		if err := superPool.QueryRow(ctx,
			`INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status)
			 VALUES ($1, 'pr354-host-zero', 'linux', '22.04', 'online')
			 RETURNING id::text`,
			pr354DefTenant,
		).Scan(&zeroEPStr); err != nil {
			t.Fatalf("seed zero endpoint: %v", err)
		}
		zeroUUID, err := uuid.Parse(zeroEPStr)
		if err != nil {
			t.Fatalf("parse zero endpoint UUID: %v", err)
		}
		pgZero := pgtype.UUID{Bytes: zeroUUID, Valid: true}

		if _, err := superPool.Exec(ctx,
			`INSERT INTO config_overrides (tenant_id, scope_type, scope_id, module, config, updated_by)
			 VALUES ($1, 'endpoint', $2, 'comms', $3::jsonb, $1)`,
			pr354DefTenant, zeroEPStr,
			`{"heartbeat_interval_seconds":0,"inventory_interval_seconds":0,"max_retry_attempts":0}`,
		); err != nil {
			t.Fatalf("seed zero config_overrides: %v", err)
		}

		beforeCount := len(bus.Events())
		cfg := svc.ExportedCheckConfigUpdate(ctx, pgZero, tenantUUID, pr354DefTenant, zeroUUID.String(), quietLogger)
		if cfg != nil {
			t.Errorf("expected nil for all-zero config, got %+v", cfg)
		}
		// config_pushed_at must remain null.
		var pushedAtValid bool
		if err := superPool.QueryRow(ctx,
			`SELECT config_pushed_at IS NOT NULL FROM endpoints WHERE id = $1`,
			zeroEPStr,
		).Scan(&pushedAtValid); err != nil {
			t.Fatalf("query config_pushed_at (zero): %v", err)
		}
		if pushedAtValid {
			t.Error("config_pushed_at should remain NULL when all-zero config is rejected")
		}
		// No new event emitted for zero path.
		for _, evt := range bus.Events()[beforeCount:] {
			if evt.Type == events.EndpointConfigPushed {
				t.Errorf("EndpointConfigPushed event emitted for all-zero config: %+v", evt)
			}
		}
	})
}
