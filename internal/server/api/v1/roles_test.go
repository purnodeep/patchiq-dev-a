package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTenantID is declared in notifications_test.go (same package).

// --- Role test fakes ---

type fakeRoleEventBus struct {
	events []domain.DomainEvent
}

func (f *fakeRoleEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	f.events = append(f.events, event)
	return nil
}
func (f *fakeRoleEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (f *fakeRoleEventBus) Close() error                                    { return nil }

type fakeRoleQuerier struct {
	listResult      []sqlcgen.ListRolesWithCountRow
	listErr         error
	countResult     int64
	countErr        error
	getResult       sqlcgen.Role
	getErr          error
	createResult    sqlcgen.Role
	createErr       error
	updateResult    sqlcgen.Role
	updateErr       error
	deleteResult    int64
	deleteErr       error
	listPermsResult []sqlcgen.RolePermission
	listPermsErr    error
	deletePermsErr  error
	createPermErr   error
}

func (f *fakeRoleQuerier) ListRolesWithCount(_ context.Context, _ sqlcgen.ListRolesWithCountParams) ([]sqlcgen.ListRolesWithCountRow, error) {
	return f.listResult, f.listErr
}
func (f *fakeRoleQuerier) CountRoles(_ context.Context, _ sqlcgen.CountRolesParams) (int64, error) {
	return f.countResult, f.countErr
}
func (f *fakeRoleQuerier) GetRoleByID(_ context.Context, _ sqlcgen.GetRoleByIDParams) (sqlcgen.Role, error) {
	return f.getResult, f.getErr
}
func (f *fakeRoleQuerier) CreateRole(_ context.Context, _ sqlcgen.CreateRoleParams) (sqlcgen.Role, error) {
	return f.createResult, f.createErr
}
func (f *fakeRoleQuerier) UpdateRole(_ context.Context, _ sqlcgen.UpdateRoleParams) (sqlcgen.Role, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeRoleQuerier) DeleteRole(_ context.Context, _ sqlcgen.DeleteRoleParams) (int64, error) {
	return f.deleteResult, f.deleteErr
}
func (f *fakeRoleQuerier) ListRolePermissions(_ context.Context, _ sqlcgen.ListRolePermissionsParams) ([]sqlcgen.RolePermission, error) {
	return f.listPermsResult, f.listPermsErr
}
func (f *fakeRoleQuerier) DeleteRolePermissions(_ context.Context, _ sqlcgen.DeleteRolePermissionsParams) error {
	return f.deletePermsErr
}
func (f *fakeRoleQuerier) CreateRolePermission(_ context.Context, _ sqlcgen.CreateRolePermissionParams) error {
	return f.createPermErr
}

type fakeRoleTx struct {
	q *fakeRoleQuerier
}

func (f *fakeRoleTx) Begin(_ context.Context) (pgx.Tx, error) { return f, nil }
func (f *fakeRoleTx) Commit(_ context.Context) error          { return nil }
func (f *fakeRoleTx) Rollback(_ context.Context) error        { return nil }
func (f *fakeRoleTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeRoleTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeRoleTx) LargeObjects() pgx.LargeObjects                             { return pgx.LargeObjects{} }
func (f *fakeRoleTx) Prepare(_ context.Context, _ string, _ string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f *fakeRoleTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (f *fakeRoleTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { return nil, nil }
func (f *fakeRoleTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row        { return nil }
func (f *fakeRoleTx) Conn() *pgx.Conn                                               { return nil }

type fakeRoleTxBeginner struct {
	tx  *fakeRoleTx
	err error
}

func (f *fakeRoleTxBeginner) Begin(_ context.Context) (pgx.Tx, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tx, nil
}

// fakeRoleTxQF returns a TxQuerierFactory that ignores the tx and returns
// the querier from the fakeRoleTx. This lets unit tests control the
// transactional querier's behaviour without a real database.
func fakeRoleTxQF(q v1.RoleQuerier) v1.TxQuerierFactory {
	return func(_ pgx.Tx) v1.RoleQuerier { return q }
}

// roleCtx injects chi URL params into the request context.
func roleCtx(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func validRole() sqlcgen.Role {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000088")
	_ = tid.Scan(testTenantID)
	return sqlcgen.Role{
		ID:       id,
		TenantID: tid,
		Name:     "test-role",
	}
}

func validRoleRow() sqlcgen.ListRolesWithCountRow {
	r := validRole()
	return sqlcgen.ListRolesWithCountRow{
		ID:              r.ID,
		TenantID:        r.TenantID,
		Name:            r.Name,
		Description:     r.Description,
		ParentRoleID:    r.ParentRoleID,
		IsSystem:        r.IsSystem,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
		PermissionCount: 5,
	}
}

// --- Get Tests ---

func TestRoleHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeRoleQuerier
		wantStatus int
	}{
		{
			name:       "valid ID returns 200",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeRoleQuerier{getResult: validRole()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeRoleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeRoleQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewRoleHandler(tt.querier, &fakeRoleTxBeginner{tx: &fakeRoleTx{q: tt.querier}}, &fakeRoleEventBus{}, fakeRoleTxQF(tt.querier))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/"+tt.id, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = roleCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Update Tests ---

func TestRoleHandler_Update(t *testing.T) {
	eb := &fakeRoleEventBus{}
	updated := validRole()
	updated.Name = "updated-role"

	tests := []struct {
		name       string
		id         string
		body       any
		querier    *fakeRoleQuerier
		txbErr     error
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "valid update returns 200",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{
				"name": "updated-role",
				"permissions": []map[string]string{
					{"resource": "endpoints", "action": "write", "scope": "all"},
				},
			},
			querier:    &fakeRoleQuerier{updateResult: updated},
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
		{
			name:       "missing name returns 400",
			id:         "00000000-0000-0000-0000-000000000088",
			body:       map[string]any{"description": "no name"},
			querier:    &fakeRoleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "system role returns not found (404)",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{"name": "try-update-system"},
			querier: &fakeRoleQuerier{
				updateErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "delete permissions failure returns 500",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{"name": "updated", "permissions": []map[string]string{
				{"resource": "endpoints", "action": "read", "scope": "*"},
			}},
			querier:    &fakeRoleQuerier{updateResult: validRole(), deletePermsErr: fmt.Errorf("db error")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "duplicate role name on update returns 409",
			id:         "00000000-0000-0000-0000-000000000088",
			body:       map[string]any{"name": "existing-role"},
			querier:    &fakeRoleQuerier{updateErr: &pgconn.PgError{Code: "23505"}},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid parent role on update returns 400",
			id:         "00000000-0000-0000-0000-000000000088",
			body:       map[string]any{"name": "test-role", "parent_role_id": "00000000-0000-0000-0000-000000000099"},
			querier:    &fakeRoleQuerier{updateErr: &pgconn.PgError{Code: "23503"}},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := v1.NewRoleHandler(tt.querier, &fakeRoleTxBeginner{err: tt.txbErr, tx: &fakeRoleTx{q: tt.querier}}, eb, fakeRoleTxQF(tt.querier))
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/roles/"+tt.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = roleCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Update(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				require.Len(t, eb.events, 1)
				assert.Equal(t, "role.updated", eb.events[0].Type)
			}
		})
	}
}

// --- Create Tests ---

func TestRoleHandler_Create(t *testing.T) {
	eb := &fakeRoleEventBus{}

	tests := []struct {
		name       string
		body       any
		querier    *fakeRoleQuerier
		txbErr     error
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "valid create returns 201",
			body: map[string]any{
				"name":        "custom-role",
				"description": "A custom role",
				"permissions": []map[string]string{
					{"resource": "endpoints", "action": "read", "scope": "own"},
				},
			},
			querier:    &fakeRoleQuerier{createResult: validRole()},
			wantStatus: http.StatusCreated,
			wantEvent:  true,
		},
		{
			name:       "missing name returns 400",
			body:       map[string]any{"description": "no name"},
			querier:    &fakeRoleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body returns 400",
			body:       "not json",
			querier:    &fakeRoleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "tx begin failure returns 500",
			body:       map[string]any{"name": "test-role", "permissions": []map[string]string{}},
			querier:    &fakeRoleQuerier{},
			txbErr:     fmt.Errorf("connection refused"),
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "create permission failure returns 500",
			body: map[string]any{
				"name": "test-role",
				"permissions": []map[string]string{
					{"resource": "endpoints", "action": "read", "scope": "*"},
				},
			},
			querier:    &fakeRoleQuerier{createResult: validRole(), createPermErr: fmt.Errorf("db error")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "duplicate role name returns 409",
			body:       map[string]any{"name": "existing-role"},
			querier:    &fakeRoleQuerier{createErr: &pgconn.PgError{Code: "23505"}},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid parent role returns 400",
			body:       map[string]any{"name": "test-role", "parent_role_id": "00000000-0000-0000-0000-000000000099"},
			querier:    &fakeRoleQuerier{createErr: &pgconn.PgError{Code: "23503"}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty permission resource returns 400",
			body: map[string]any{
				"name": "test-role",
				"permissions": []map[string]string{
					{"resource": "", "action": "read", "scope": "*"},
				},
			},
			querier:    &fakeRoleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := v1.NewRoleHandler(tt.querier, &fakeRoleTxBeginner{err: tt.txbErr, tx: &fakeRoleTx{q: tt.querier}}, eb, fakeRoleTxQF(tt.querier))
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				require.Len(t, eb.events, 1)
				assert.Equal(t, "role.created", eb.events[0].Type)
			}
		})
	}
}

// --- GetPermissions Tests ---

func TestRoleHandler_GetPermissions(t *testing.T) {
	var roleID, tid pgtype.UUID
	_ = roleID.Scan("00000000-0000-0000-0000-000000000088")
	_ = tid.Scan(testTenantID)

	tests := []struct {
		name       string
		id         string
		querier    *fakeRoleQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns permissions",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeRoleQuerier{
				getResult: validRole(),
				listPermsResult: []sqlcgen.RolePermission{
					{ID: roleID, TenantID: tid, RoleID: roleID, Resource: "endpoints", Action: "read", Scope: "own"},
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "role not found returns 404",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeRoleQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewRoleHandler(tt.querier, &fakeRoleTxBeginner{tx: &fakeRoleTx{q: tt.querier}}, &fakeRoleEventBus{}, fakeRoleTxQF(tt.querier))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/"+tt.id+"/permissions", nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = roleCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.GetPermissions(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body []any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body, tt.wantLen)
			}
		})
	}
}

// --- Delete Tests ---

func TestRoleHandler_Delete(t *testing.T) {
	eb := &fakeRoleEventBus{}

	tests := []struct {
		name       string
		id         string
		querier    *fakeRoleQuerier
		wantStatus int
		wantEvent  bool
	}{
		{
			name:       "valid delete returns 204",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeRoleQuerier{deleteResult: 1},
			wantStatus: http.StatusNoContent,
			wantEvent:  true,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeRoleQuerier{deleteResult: 0, getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "system role returns 403",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeRoleQuerier{
				deleteResult: 0,
				getResult:    sqlcgen.Role{ID: validRole().ID, TenantID: validRole().TenantID, Name: "admin", IsSystem: true},
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeRoleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := v1.NewRoleHandler(tt.querier, &fakeRoleTxBeginner{tx: &fakeRoleTx{q: tt.querier}}, eb, fakeRoleTxQF(tt.querier))
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/"+tt.id, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = roleCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				require.Len(t, eb.events, 1)
				assert.Equal(t, "role.deleted", eb.events[0].Type)
			}
		})
	}
}

// --- List Tests ---

func TestRoleHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakeRoleQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns roles with count",
			querier: &fakeRoleQuerier{
				listResult:  []sqlcgen.ListRolesWithCountRow{validRoleRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "returns empty list",
			querier: &fakeRoleQuerier{
				listResult:  []sqlcgen.ListRolesWithCountRow{},
				countResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "store error returns 500",
			querier: &fakeRoleQuerier{
				listErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewRoleHandler(tt.querier, &fakeRoleTxBeginner{tx: &fakeRoleTx{q: tt.querier}}, &fakeRoleEventBus{}, fakeRoleTxQF(tt.querier))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/roles"+tt.query, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body["data"], tt.wantLen)
			}
		})
	}
}
