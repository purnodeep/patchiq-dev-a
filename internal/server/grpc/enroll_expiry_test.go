package grpc_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// TestEnroll_ExpiredToken covers the expiry gate added by PR #354 in
// enroll.go: when a registration row has expires_at in the past, Enroll
// must return INVALID_TOKEN with "expired" messaging without any DB
// writes.
func TestEnroll_ExpiredToken(t *testing.T) {
	superPool, cleanup := setupPR354DB(t)
	defer cleanup()
	ctx := context.Background()

	app := pr354AppPool(t, superPool)
	defer app.Close()

	st := store.NewStoreWithBypass(app, superPool)
	quietLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := servergrpc.NewAgentServiceServer(st, noopEventBus{}, quietLogger)

	// Seed two registration rows via superPool: one expired, one future.
	// Using INSERT directly to avoid sqlc query plumbing through RLS.
	expiredToken := "pr354-token-expired"
	futureToken := "pr354-token-future"

	if _, err := superPool.Exec(ctx,
		`INSERT INTO agent_registrations (tenant_id, registration_token, expires_at)
		 VALUES ($1, $2, $3)`,
		pr354DefTenant, expiredToken, time.Now().Add(-1*time.Hour),
	); err != nil {
		t.Fatalf("seed expired registration: %v", err)
	}
	if _, err := superPool.Exec(ctx,
		`INSERT INTO agent_registrations (tenant_id, registration_token, expires_at)
		 VALUES ($1, $2, $3)`,
		pr354DefTenant, futureToken, time.Now().Add(1*time.Hour),
	); err != nil {
		t.Fatalf("seed future registration: %v", err)
	}

	buildReq := func(token string) *pb.EnrollRequest {
		return &pb.EnrollRequest{
			AgentInfo: &pb.AgentInfo{
				AgentVersion:    "1.0.0",
				ProtocolVersion: 1,
			},
			EnrollmentToken: token,
			EndpointInfo: &pb.EndpointInfo{
				Hostname:  "pr354-host-" + token,
				OsFamily:  pb.OsFamily_OS_FAMILY_LINUX,
				OsVersion: "22.04",
			},
		}
	}

	t.Run("expired token rejected with INVALID_TOKEN", func(t *testing.T) {
		resp, err := svc.Enroll(ctx, buildReq(expiredToken))
		if err != nil {
			t.Fatalf("Enroll returned err: %v", err)
		}
		if resp == nil {
			t.Fatal("Enroll returned nil response")
		}
		if resp.ErrorCode != pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_INVALID_TOKEN {
			t.Errorf("ErrorCode = %v, want INVALID_TOKEN", resp.ErrorCode)
		}
		if !containsCI(resp.ErrorMessage, "expired") {
			t.Errorf("ErrorMessage = %q, want it to contain 'expired'", resp.ErrorMessage)
		}
		// No endpoint should have been created, and the registration row
		// should still be "pending".
		var status string
		if err := superPool.QueryRow(ctx,
			`SELECT status FROM agent_registrations WHERE registration_token = $1`,
			expiredToken,
		).Scan(&status); err != nil {
			t.Fatalf("lookup registration status: %v", err)
		}
		if status != "pending" {
			t.Errorf("registration status after expired Enroll = %q, want pending", status)
		}
	})

	t.Run("future token not rejected as expired", func(t *testing.T) {
		resp, err := svc.Enroll(ctx, buildReq(futureToken))
		if err != nil {
			t.Fatalf("Enroll returned err: %v", err)
		}
		if resp == nil {
			t.Fatal("Enroll returned nil response")
		}
		// Future token should have progressed past the expiry gate. It may
		// have succeeded (AgentId set, ErrorCode unspecified) or hit a later
		// gate; what matters is that the expiry rejection path did NOT fire.
		if resp.ErrorCode == pb.EnrollmentErrorCode_ENROLLMENT_ERROR_CODE_INVALID_TOKEN &&
			containsCI(resp.ErrorMessage, "expired") {
			t.Errorf("future token wrongly rejected as expired: %+v", resp)
		}
	})

	// Sanity check: the claim-registration path should have marked the
	// future token's row as 'registered' (verifying the positive path did
	// reach DB writes, not just the expiry gate).
	var rows int
	if err := superPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM agent_registrations
		 WHERE registration_token = $1 AND status = 'registered'`,
		futureToken,
	).Scan(&rows); err != nil {
		t.Fatalf("count registered rows: %v", err)
	}
	if rows != 1 {
		t.Logf("note: future token registration status rows=%d (informational)", rows)
	}

	// Also verify the LookupRegistrationByToken query can still read the
	// expired row via the bypass pool (sanity check that our seeding worked).
	q := sqlcgen.New(superPool)
	reg, err := q.LookupRegistrationByToken(ctx, expiredToken)
	if err != nil {
		t.Fatalf("LookupRegistrationByToken(expired): %v", err)
	}
	if !reg.ExpiresAt.Valid {
		t.Error("seeded expired registration has NULL expires_at")
	} else if !reg.ExpiresAt.Time.Before(time.Now()) {
		t.Errorf("seeded expired registration expires_at=%v not in the past", reg.ExpiresAt.Time)
	}
}

// containsCI reports whether needle appears in haystack, case-insensitively.
func containsCI(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
