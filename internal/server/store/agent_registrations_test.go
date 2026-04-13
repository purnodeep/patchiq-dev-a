package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// TestAgentRegistrationQueries exercises the full agent registration lifecycle
// through the sqlc-generated Queries, running against a real PostgreSQL 16
// instance with RLS and CHECK constraints active.
func TestAgentRegistrationQueries(t *testing.T) {
	superPool, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	app := appPool(t, superPool)
	defer app.Close()

	tenantUUID := pgtype.UUID{}
	if err := tenantUUID.Scan(defaultTenant); err != nil {
		t.Fatalf("parse default tenant UUID: %v", err)
	}

	// beginTx starts a transaction with tenant context set and returns the
	// transaction plus a sqlcgen.Queries bound to it.
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

	t.Run("create_and_get", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-create-get",
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}
		if reg.Status != "pending" {
			t.Errorf("status = %q, want pending", reg.Status)
		}
		if reg.EndpointID.Valid {
			t.Error("endpoint_id should be NULL for pending registration")
		}
		if reg.RegisteredAt.Valid {
			t.Error("registered_at should be NULL for pending registration")
		}

		// GetByID
		got, err := q.GetRegistrationByID(ctx, sqlcgen.GetRegistrationByIDParams{
			ID: reg.ID, TenantID: tenantUUID,
		})
		if err != nil {
			t.Fatalf("GetRegistrationByID: %v", err)
		}
		if got.RegistrationToken != "tok-create-get" {
			t.Errorf("token = %q, want tok-create-get", got.RegistrationToken)
		}

		// GetByToken
		got2, err := q.GetRegistrationByToken(ctx, sqlcgen.GetRegistrationByTokenParams{
			RegistrationToken: "tok-create-get", TenantID: tenantUUID,
		})
		if err != nil {
			t.Fatalf("GetRegistrationByToken: %v", err)
		}
		if got2.ID != reg.ID {
			t.Errorf("GetByToken returned different ID")
		}
	})

	t.Run("list_by_tenant", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		for i := 0; i < 3; i++ {
			if _, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
				TenantID:          tenantUUID,
				RegistrationToken: "tok-list-" + string(rune('a'+i)),
			}); err != nil {
				t.Fatalf("CreateRegistration #%d: %v", i, err)
			}
		}

		list, err := q.ListRegistrationsByTenant(ctx, tenantUUID)
		if err != nil {
			t.Fatalf("ListRegistrationsByTenant: %v", err)
		}
		if len(list) < 3 {
			t.Errorf("list length = %d, want >= 3", len(list))
		}
	})

	t.Run("claim_registration", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		// Seed an endpoint via superuser (bypasses RLS) so we have a valid FK.
		var endpointID pgtype.UUID
		if err := superPool.QueryRow(ctx,
			"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'claim-host', 'linux', '22.04', 'online') RETURNING id",
			defaultTenant,
		).Scan(&endpointID); err != nil {
			t.Fatalf("insert endpoint: %v", err)
		}

		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-claim",
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}

		claimed, err := q.ClaimRegistration(ctx, sqlcgen.ClaimRegistrationParams{
			ID:         reg.ID,
			TenantID:   tenantUUID,
			EndpointID: endpointID,
		})
		if err != nil {
			t.Fatalf("ClaimRegistration: %v", err)
		}
		if claimed.Status != "registered" {
			t.Errorf("status = %q, want registered", claimed.Status)
		}
		if !claimed.EndpointID.Valid {
			t.Error("endpoint_id should be set after claim")
		}
		if !claimed.RegisteredAt.Valid {
			t.Error("registered_at should be set after claim")
		}
	})

	t.Run("claim_already_claimed_returns_no_rows", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		var endpointID pgtype.UUID
		if err := superPool.QueryRow(ctx,
			"INSERT INTO endpoints (tenant_id, hostname, os_family, os_version, status) VALUES ($1, 'double-claim', 'linux', '22.04', 'online') RETURNING id",
			defaultTenant,
		).Scan(&endpointID); err != nil {
			t.Fatalf("insert endpoint: %v", err)
		}

		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-double-claim",
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}

		// First claim succeeds.
		if _, err := q.ClaimRegistration(ctx, sqlcgen.ClaimRegistrationParams{
			ID: reg.ID, TenantID: tenantUUID, EndpointID: endpointID,
		}); err != nil {
			t.Fatalf("first ClaimRegistration: %v", err)
		}

		// Second claim returns no rows (status is no longer 'pending').
		_, err = q.ClaimRegistration(ctx, sqlcgen.ClaimRegistrationParams{
			ID: reg.ID, TenantID: tenantUUID, EndpointID: endpointID,
		})
		if err == nil {
			t.Fatal("second ClaimRegistration should fail (no rows), but succeeded")
		}
	})

	t.Run("revoke_registration", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-revoke",
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}

		revoked, err := q.RevokeRegistration(ctx, sqlcgen.RevokeRegistrationParams{
			ID: reg.ID, TenantID: tenantUUID,
		})
		if err != nil {
			t.Fatalf("RevokeRegistration: %v", err)
		}
		if revoked.Status != "revoked" {
			t.Errorf("status = %q, want revoked", revoked.Status)
		}

		// Revoking again returns no rows.
		_, err = q.RevokeRegistration(ctx, sqlcgen.RevokeRegistrationParams{
			ID: reg.ID, TenantID: tenantUUID,
		})
		if err == nil {
			t.Fatal("double RevokeRegistration should fail (no rows), but succeeded")
		}
	})

	t.Run("duplicate_token_rejected", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		if _, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-unique",
		}); err != nil {
			t.Fatalf("first CreateRegistration: %v", err)
		}

		_, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-unique",
		})
		if err == nil {
			t.Fatal("duplicate token INSERT should fail due to UNIQUE constraint")
		}
		var pgErr *pgconn.PgError
		if !isPgError(err, &pgErr) || pgErr.Code != "23505" {
			t.Errorf("expected pg error 23505 (unique_violation), got: %v", err)
		}
	})

	t.Run("expires_at_set_on_create", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		// Without explicit expires_at — should get default (7 days from now).
		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-expiry-default",
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}
		if !reg.ExpiresAt.Valid {
			t.Fatal("expires_at should be set")
		}
		// Should be roughly 7 days from now (within 1 minute tolerance).
		expectedExpiry := time.Now().Add(7 * 24 * time.Hour)
		diff := reg.ExpiresAt.Time.Sub(expectedExpiry)
		if diff < -time.Minute || diff > time.Minute {
			t.Errorf("expires_at = %v, expected ~%v (diff=%v)", reg.ExpiresAt.Time, expectedExpiry, diff)
		}
	})

	t.Run("expires_at_custom_value", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		customExpiry := time.Now().Add(1 * time.Hour)
		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-expiry-custom",
			ExpiresAt:         pgtype.Timestamptz{Time: customExpiry, Valid: true},
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}
		if !reg.ExpiresAt.Valid {
			t.Fatal("expires_at should be set")
		}
		diff := reg.ExpiresAt.Time.Sub(customExpiry)
		if diff < -time.Second || diff > time.Second {
			t.Errorf("expires_at = %v, expected ~%v", reg.ExpiresAt.Time, customExpiry)
		}
	})

	t.Run("expired_token_lookup_returns_data", func(t *testing.T) {
		tx, q := beginTx(t)
		defer tx.Rollback(ctx) //nolint:errcheck

		// Create a token that's already expired.
		pastExpiry := time.Now().Add(-1 * time.Hour)
		reg, err := q.CreateRegistration(ctx, sqlcgen.CreateRegistrationParams{
			TenantID:          tenantUUID,
			RegistrationToken: "tok-expired",
			ExpiresAt:         pgtype.Timestamptz{Time: pastExpiry, Valid: true},
		})
		if err != nil {
			t.Fatalf("CreateRegistration: %v", err)
		}

		// LookupRegistrationByToken should still return the token (expiry is checked in Go, not SQL).
		looked, err := q.LookupRegistrationByToken(ctx, "tok-expired")
		if err != nil {
			t.Fatalf("LookupRegistrationByToken: %v", err)
		}
		if looked.ID != reg.ID {
			t.Error("lookup should return the same registration")
		}
		if !looked.ExpiresAt.Valid || !time.Now().After(looked.ExpiresAt.Time) {
			t.Error("token should be expired")
		}
	})
}
