package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake Querier ---

type fakeRoleMappingQuerier struct {
	listResult []sqlcgen.ListRoleMappingsRow
	listErr    error

	upsertResult sqlcgen.RoleMapping
	upsertErr    error
	upsertCalls  int

	deleteErr   error
	deleteCalls int
}

func (f *fakeRoleMappingQuerier) ListRoleMappings(_ context.Context, _ pgtype.UUID) ([]sqlcgen.ListRoleMappingsRow, error) {
	return f.listResult, f.listErr
}

func (f *fakeRoleMappingQuerier) UpsertRoleMapping(_ context.Context, _ sqlcgen.UpsertRoleMappingParams) (sqlcgen.RoleMapping, error) {
	f.upsertCalls++
	return f.upsertResult, f.upsertErr
}

func (f *fakeRoleMappingQuerier) DeleteRoleMappingsByTenant(_ context.Context, _ pgtype.UUID) error {
	f.deleteCalls++
	return f.deleteErr
}

// --- Helpers ---

const rmTenantID = "00000000-0000-0000-0000-000000000001"
const rmValidRoleID = "00000000-0000-0000-0000-000000000099"

func rmWithTenant(req *http.Request) *http.Request {
	return req.WithContext(tenant.WithTenantID(req.Context(), rmTenantID))
}

func rmRequest(method, url string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	return rmWithTenant(req)
}

func newRoleMappingHandler(q *fakeRoleMappingQuerier) *v1.RoleMappingHandler {
	txQF := func(_ pgx.Tx) v1.RoleMappingQuerier { return q }
	return v1.NewRoleMappingHandler(q, &fakeTxBeginner{tx: &fakeTx{}}, &fakeEventBus{}, txQF)
}

func makePgUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	_ = u.Scan(s)
	return u
}

// --- Tests ---

func TestRoleMappingGet_Happy(t *testing.T) {
	q := &fakeRoleMappingQuerier{
		listResult: []sqlcgen.ListRoleMappingsRow{
			{ExternalRole: "admin", PatchiqRoleID: makePgUUID(rmValidRoleID), RoleName: "Administrator"},
			{ExternalRole: "viewer", PatchiqRoleID: makePgUUID(rmValidRoleID), RoleName: "Read Only"},
		},
	}
	h := newRoleMappingHandler(q)

	req := rmRequest(http.MethodGet, "/api/v1/settings/role-mapping", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	data, ok := resp["data"].([]any)
	require.True(t, ok, "data should be an array")
	assert.Len(t, data, 2)

	first, ok := data[0].(map[string]any)
	require.True(t, ok, "first entry should be an object")
	assert.Equal(t, "admin", first["external_role"])
	assert.Equal(t, "Administrator", first["role_name"])
}

func TestRoleMappingGet_DBError(t *testing.T) {
	q := &fakeRoleMappingQuerier{
		listErr: errors.New("db connection lost"),
	}
	h := newRoleMappingHandler(q)

	req := rmRequest(http.MethodGet, "/api/v1/settings/role-mapping", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRoleMappingUpdate_Happy(t *testing.T) {
	q := &fakeRoleMappingQuerier{}
	h := newRoleMappingHandler(q)

	body := map[string]any{
		"mappings": []map[string]any{
			{"external_role": "admin", "patchiq_role_id": rmValidRoleID},
			{"external_role": "viewer", "patchiq_role_id": rmValidRoleID},
		},
	}
	req := rmRequest(http.MethodPut, "/api/v1/settings/role-mapping", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 2, q.upsertCalls)
}

func TestRoleMappingUpdate_InvalidRoleID(t *testing.T) {
	q := &fakeRoleMappingQuerier{}
	h := newRoleMappingHandler(q)

	body := map[string]any{
		"mappings": []map[string]any{
			{"external_role": "admin", "patchiq_role_id": "not-a-uuid"},
		},
	}
	req := rmRequest(http.MethodPut, "/api/v1/settings/role-mapping", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Zero(t, q.upsertCalls)
}

func TestRoleMappingUpdate_UpsertFailure(t *testing.T) {
	q := &fakeRoleMappingQuerier{
		upsertErr: errors.New("constraint violation"),
	}
	h := newRoleMappingHandler(q)

	body := map[string]any{
		"mappings": []map[string]any{
			{"external_role": "admin", "patchiq_role_id": rmValidRoleID},
		},
	}
	req := rmRequest(http.MethodPut, "/api/v1/settings/role-mapping", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRoleMappingUpdate_MalformedJSON(t *testing.T) {
	h := newRoleMappingHandler(&fakeRoleMappingQuerier{})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/role-mapping", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	req = rmWithTenant(req)

	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
