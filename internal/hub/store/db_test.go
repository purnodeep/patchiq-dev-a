package store_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// TestBeginTx_ValidationErrors documents the validation behaviour of BeginTx.
//
// The validation logic (missing/empty tenant ID, invalid UUID) runs before any
// database call. However, NewStore panics on a nil pool, so these paths cannot
// be exercised in a pure unit test without either a real database connection or
// refactoring Store to accept a pool interface.
//
// Full coverage of the validation + set_config flow is provided by the
// integration tests in test/integration/ which run against a real PostgreSQL
// instance (testcontainers).
//
// TODO(PIQ-14): Refactor Store to accept a pool interface so these cases can be
// unit-tested without a database.
func TestBeginTx_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr string
	}{
		{
			name:    "missing tenant ID in context",
			ctx:     context.Background(),
			wantErr: "missing tenant ID in context",
		},
		{
			name:    "invalid UUID tenant ID",
			ctx:     tenant.WithTenantID(context.Background(), "not-a-uuid"),
			wantErr: "invalid tenant ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// BeginTx validation runs before pool.Begin, but NewStore panics on
			// nil pool — we cannot reach BeginTx without a real connection.
			// Skip until Store accepts a pool interface (PIQ-14).
			t.Skip("requires database connection or Store pool interface refactor — tracked in PIQ-14")
			_ = tt.ctx // suppress unused variable warning while skipped
			_ = tt.wantErr
		})
	}
}
