package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSettingsQuerier struct {
	listFn   func(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.HubConfig, error)
	getFn    func(ctx context.Context, arg sqlcgen.GetHubConfigParams) (sqlcgen.HubConfig, error)
	upsertFn func(ctx context.Context, arg sqlcgen.UpsertHubConfigParams) (sqlcgen.HubConfig, error)
}

func (m *mockSettingsQuerier) ListHubConfigByTenant(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.HubConfig, error) {
	if m.listFn != nil {
		return m.listFn(ctx, tenantID)
	}
	return nil, nil
}

func (m *mockSettingsQuerier) GetHubConfig(ctx context.Context, arg sqlcgen.GetHubConfigParams) (sqlcgen.HubConfig, error) {
	if m.getFn != nil {
		return m.getFn(ctx, arg)
	}
	return sqlcgen.HubConfig{}, pgx.ErrNoRows
}

func (m *mockSettingsQuerier) UpsertHubConfig(ctx context.Context, arg sqlcgen.UpsertHubConfigParams) (sqlcgen.HubConfig, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, arg)
	}
	return sqlcgen.HubConfig{}, nil
}

// mockEventBus, withTenantCtx, and testTenantID are defined in catalog_test.go.

func TestListSettings_Success(t *testing.T) {
	querier := &mockSettingsQuerier{
		listFn: func(_ context.Context, _ pgtype.UUID) ([]sqlcgen.HubConfig, error) {
			return []sqlcgen.HubConfig{
				{Key: "hub.name", Value: []byte(`"PatchIQ Hub"`)},
				{Key: "hub.timezone", Value: []byte(`"UTC"`)},
			}, nil
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	req := withTenantCtx(httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, `"PatchIQ Hub"`, string(body["hub.name"]))
	assert.Equal(t, `"UTC"`, string(body["hub.timezone"]))
}

func TestListSettings_Empty(t *testing.T) {
	querier := &mockSettingsQuerier{
		listFn: func(_ context.Context, _ pgtype.UUID) ([]sqlcgen.HubConfig, error) {
			return []sqlcgen.HubConfig{}, nil
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	req := withTenantCtx(httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "{}\n", rec.Body.String())
}

func TestListSettings_DBError(t *testing.T) {
	querier := &mockSettingsQuerier{
		listFn: func(_ context.Context, _ pgtype.UUID) ([]sqlcgen.HubConfig, error) {
			return nil, errors.New("db down")
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	req := withTenantCtx(httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetSetting_Success(t *testing.T) {
	querier := &mockSettingsQuerier{
		getFn: func(_ context.Context, arg sqlcgen.GetHubConfigParams) (sqlcgen.HubConfig, error) {
			return sqlcgen.HubConfig{Key: arg.Key, Value: []byte(`"PatchIQ Hub"`)}, nil
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	r := chi.NewRouter()
	r.Get("/api/v1/settings/{key}", h.Get)

	req := withTenantCtx(httptest.NewRequest(http.MethodGet, "/api/v1/settings/hub.name", nil))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "hub.name", body["key"])
	assert.Equal(t, "PatchIQ Hub", body["value"])
}

func TestGetSetting_NotFound(t *testing.T) {
	querier := &mockSettingsQuerier{
		getFn: func(_ context.Context, _ sqlcgen.GetHubConfigParams) (sqlcgen.HubConfig, error) {
			return sqlcgen.HubConfig{}, pgx.ErrNoRows
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	r := chi.NewRouter()
	r.Get("/api/v1/settings/{key}", h.Get)

	req := withTenantCtx(httptest.NewRequest(http.MethodGet, "/api/v1/settings/nonexistent", nil))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUpsertSetting_Success(t *testing.T) {
	bus := &mockEventBus{}
	querier := &mockSettingsQuerier{
		upsertFn: func(_ context.Context, arg sqlcgen.UpsertHubConfigParams) (sqlcgen.HubConfig, error) {
			return sqlcgen.HubConfig{Key: arg.Key, Value: arg.Value}, nil
		},
	}
	h := v1.NewSettingsHandler(querier, bus)

	body := `{"key":"hub.name","value":"New Hub Name"}`
	req := withTenantCtx(httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "hub.name", resp["key"])

	// Verify event was emitted.
	require.Len(t, bus.emitted, 1)
	assert.Equal(t, "config.updated", bus.emitted[0].Type)
}

func TestUpsertSetting_InvalidBody(t *testing.T) {
	h := v1.NewSettingsHandler(&mockSettingsQuerier{}, &mockEventBus{})

	req := withTenantCtx(httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString("invalid")))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpsertSetting_MissingKey(t *testing.T) {
	h := v1.NewSettingsHandler(&mockSettingsQuerier{}, &mockEventBus{})

	body := `{"value":"something"}`
	req := withTenantCtx(httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpsertSetting_MissingValue(t *testing.T) {
	h := v1.NewSettingsHandler(&mockSettingsQuerier{}, &mockEventBus{})

	body := `{"key":"hub.name"}`
	req := withTenantCtx(httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpsertSetting_DBError(t *testing.T) {
	querier := &mockSettingsQuerier{
		upsertFn: func(_ context.Context, _ sqlcgen.UpsertHubConfigParams) (sqlcgen.HubConfig, error) {
			return sqlcgen.HubConfig{}, errors.New("db down")
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	body := `{"key":"hub.name","value":"Test"}`
	req := withTenantCtx(httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()

	h.Upsert(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "upsert setting")
}

func TestGetSetting_DBError(t *testing.T) {
	querier := &mockSettingsQuerier{
		getFn: func(_ context.Context, _ sqlcgen.GetHubConfigParams) (sqlcgen.HubConfig, error) {
			return sqlcgen.HubConfig{}, errors.New("db down")
		},
	}
	h := v1.NewSettingsHandler(querier, &mockEventBus{})

	r := chi.NewRouter()
	r.Get("/api/v1/settings/{key}", h.Get)

	req := withTenantCtx(httptest.NewRequest(http.MethodGet, "/api/v1/settings/hub.name", nil))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "get setting")
}
