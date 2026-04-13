package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTx implements pgx.Tx for testing. It delegates GetUserPermissions calls
// to the embedded fakePermQuerier via sqlcgen.New(fakeTx).
type fakeTx struct {
	pgx.Tx // embed to satisfy interface; only Query/Rollback/Commit used
	rows   []sqlcgen.GetUserPermissionsRow
	err    error
}

func (f *fakeTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeRows{rows: f.rows}, nil
}

func (f *fakeTx) Rollback(_ context.Context) error { return nil }
func (f *fakeTx) Commit(_ context.Context) error   { return nil }

// fakeTxBeginner implements auth.TxBeginner for testing.
type fakeTxBeginner struct {
	tx     *fakeTx
	begErr error
}

func (f *fakeTxBeginner) BeginTx(_ context.Context) (pgx.Tx, error) {
	if f.begErr != nil {
		return nil, f.begErr
	}
	return f.tx, nil
}

// fakeRows implements pgx.Rows to return pre-configured permission rows.
type fakeRows struct {
	rows []sqlcgen.GetUserPermissionsRow
	idx  int
}

func (r *fakeRows) Next() bool {
	if r.idx < len(r.rows) {
		r.idx++
		return true
	}
	return false
}

func (r *fakeRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	if len(dest) >= 3 {
		//nolint:errcheck // test helper — types are known
		*(dest[0].(*string)) = row.Resource
		//nolint:errcheck // test helper — types are known
		*(dest[1].(*string)) = row.Action
		//nolint:errcheck // test helper — types are known
		*(dest[2].(*string)) = row.Scope
	}
	return nil
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

func tenantCtx(tenantID string) context.Context {
	return tenant.WithTenantID(context.Background(), tenantID)
}

func TestSQLPermissionStore_GetUserPermissions(t *testing.T) {
	tests := []struct {
		name     string
		rows     []sqlcgen.GetUserPermissionsRow
		wantLen  int
		wantPerm auth.Permission
	}{
		{
			name: "converts rows to permissions",
			rows: []sqlcgen.GetUserPermissionsRow{
				{Resource: "endpoints", Action: "read", Scope: "*"},
				{Resource: "deployments", Action: "create", Scope: "group:prod"},
			},
			wantLen:  2,
			wantPerm: auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"},
		},
		{
			name:    "empty rows returns empty slice",
			rows:    []sqlcgen.GetUserPermissionsRow{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txb := &fakeTxBeginner{tx: &fakeTx{rows: tt.rows}}
			store := auth.NewSQLPermissionStore(txb)
			perms, err := store.GetUserPermissions(tenantCtx("00000000-0000-0000-0000-000000000001"), "00000000-0000-0000-0000-000000000001", "user-1")
			require.NoError(t, err)
			assert.Len(t, perms, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantPerm, perms[0])
			}
		})
	}
}

func TestSQLPermissionStore_InvalidTenantID(t *testing.T) {
	txb := &fakeTxBeginner{tx: &fakeTx{}}
	store := auth.NewSQLPermissionStore(txb)
	_, err := store.GetUserPermissions(context.Background(), "not-a-uuid", "user-1")
	require.Error(t, err)
}

func TestSQLPermissionStore_QuerierError(t *testing.T) {
	txb := &fakeTxBeginner{tx: &fakeTx{err: errors.New("db down")}}
	store := auth.NewSQLPermissionStore(txb)
	_, err := store.GetUserPermissions(tenantCtx("00000000-0000-0000-0000-000000000001"), "00000000-0000-0000-0000-000000000001", "user-1")
	require.Error(t, err)
}

func TestSQLPermissionStore_BeginTxError(t *testing.T) {
	txb := &fakeTxBeginner{begErr: errors.New("pool exhausted")}
	store := auth.NewSQLPermissionStore(txb)
	_, err := store.GetUserPermissions(tenantCtx("00000000-0000-0000-0000-000000000001"), "00000000-0000-0000-0000-000000000001", "user-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin tx")
}
