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
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeUserRoleQuerier implements v1.UserRoleQuerier for unit tests.
type fakeUserRoleQuerier struct {
	assignErr  error
	revokeRows int64
	revokeErr  error
	listResult []sqlcgen.Role
	listErr    error
}

func (f *fakeUserRoleQuerier) AssignUserRole(_ context.Context, _ sqlcgen.AssignUserRoleParams) error {
	return f.assignErr
}

func (f *fakeUserRoleQuerier) RevokeUserRole(_ context.Context, _ sqlcgen.RevokeUserRoleParams) (int64, error) {
	return f.revokeRows, f.revokeErr
}

func (f *fakeUserRoleQuerier) ListUserRoles(_ context.Context, _ sqlcgen.ListUserRolesParams) ([]sqlcgen.Role, error) {
	return f.listResult, f.listErr
}

const (
	testUserID  = "00000000-0000-0000-0000-000000000099"
	testRoleID  = "00000000-0000-0000-0000-000000000042"
	testActorID = "00000000-0000-0000-0000-000000000077"
)

func userRoleCtx() context.Context {
	ctx := context.Background()
	ctx = tenant.WithTenantID(ctx, testTenantID)
	ctx = user.WithUserID(ctx, testActorID)
	return ctx
}

func withChiUserID(ctx context.Context, userID string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

func withChiUserAndRoleID(ctx context.Context, userID, roleID string) context.Context {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", userID)
	rctx.URLParams.Add("roleId", roleID)
	return context.WithValue(ctx, chi.RouteCtxKey, rctx)
}

func TestUserRoleHandler_Assign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       any
		querier    *fakeUserRoleQuerier
		wantStatus int
		wantEvent  bool
	}{
		{
			name:       "valid assign returns 200 and emits event",
			body:       map[string]string{"role_id": testRoleID},
			querier:    &fakeUserRoleQuerier{},
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
		{
			name:       "missing role_id returns 400",
			body:       map[string]string{},
			querier:    &fakeUserRoleQuerier{},
			wantStatus: http.StatusBadRequest,
			wantEvent:  false,
		},
		{
			name:       "invalid role_id returns 400",
			body:       map[string]string{"role_id": "not-a-uuid"},
			querier:    &fakeUserRoleQuerier{},
			wantStatus: http.StatusBadRequest,
			wantEvent:  false,
		},
		{
			name:       "store error returns 500",
			body:       map[string]string{"role_id": testRoleID},
			querier:    &fakeUserRoleQuerier{assignErr: fmt.Errorf("db down")},
			wantStatus: http.StatusInternalServerError,
			wantEvent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			eb := &fakeEventBus{}
			h := v1.NewUserRoleHandler(tt.querier, eb)

			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users/"+testUserID+"/roles", bytes.NewReader(bodyBytes))
			ctx := withChiUserID(userRoleCtx(), testUserID)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.Assign(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantEvent {
				require.Len(t, eb.events, 1)
				assert.Equal(t, events.UserRoleAssigned, eb.events[0].Type)
			} else {
				assert.Empty(t, eb.events)
			}
		})
	}
}

func TestUserRoleHandler_Revoke(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		querier    *fakeUserRoleQuerier
		roleID     string
		wantStatus int
		wantEvent  bool
	}{
		{
			name:       "valid revoke returns 204 and emits event",
			querier:    &fakeUserRoleQuerier{revokeRows: 1},
			roleID:     testRoleID,
			wantStatus: http.StatusNoContent,
			wantEvent:  true,
		},
		{
			name:       "not found returns 404",
			querier:    &fakeUserRoleQuerier{revokeRows: 0},
			roleID:     testRoleID,
			wantStatus: http.StatusNotFound,
			wantEvent:  false,
		},
		{
			name:       "invalid roleId returns 400",
			querier:    &fakeUserRoleQuerier{},
			roleID:     "not-a-uuid",
			wantStatus: http.StatusBadRequest,
			wantEvent:  false,
		},
		{
			name:       "store error returns 500",
			querier:    &fakeUserRoleQuerier{revokeErr: fmt.Errorf("db down")},
			roleID:     testRoleID,
			wantStatus: http.StatusInternalServerError,
			wantEvent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			eb := &fakeEventBus{}
			h := v1.NewUserRoleHandler(tt.querier, eb)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+testUserID+"/roles/"+tt.roleID, nil)
			ctx := withChiUserAndRoleID(userRoleCtx(), testUserID, tt.roleID)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.Revoke(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantEvent {
				require.Len(t, eb.events, 1)
				assert.Equal(t, events.UserRoleRevoked, eb.events[0].Type)
			} else {
				assert.Empty(t, eb.events)
			}
		})
	}
}

func TestUserRoleHandler_List(t *testing.T) {
	t.Parallel()

	roleUUID := pgtype.UUID{Bytes: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x42}, Valid: true}

	tests := []struct {
		name       string
		querier    *fakeUserRoleQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns user roles",
			querier: &fakeUserRoleQuerier{
				listResult: []sqlcgen.Role{
					{ID: roleUUID, Name: "admin"},
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "store error returns 500",
			querier:    &fakeUserRoleQuerier{listErr: fmt.Errorf("db down")},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			eb := &fakeEventBus{}
			h := v1.NewUserRoleHandler(tt.querier, eb)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+testUserID+"/roles", nil)
			ctx := withChiUserID(userRoleCtx(), testUserID)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantLen >= 0 {
				var roles []json.RawMessage
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &roles))
				assert.Len(t, roles, tt.wantLen)
			}
		})
	}
}
