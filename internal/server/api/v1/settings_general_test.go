package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake Querier ---

type fakeGenSettingsQuerier struct {
	getResult sqlcgen.GetGeneralSettingsRow
	getErr    error

	updateResult sqlcgen.UpdateGeneralSettingsRow
	updateErr    error
}

func (f *fakeGenSettingsQuerier) GetGeneralSettings(_ context.Context, _ pgtype.UUID) (sqlcgen.GetGeneralSettingsRow, error) {
	return f.getResult, f.getErr
}

func (f *fakeGenSettingsQuerier) UpdateGeneralSettings(_ context.Context, _ sqlcgen.UpdateGeneralSettingsParams) (sqlcgen.UpdateGeneralSettingsRow, error) {
	return f.updateResult, f.updateErr
}

// --- Helpers ---

const genSettingsTenantID = "00000000-0000-0000-0000-000000000001"

func genSettingsWithTenant(req *http.Request) *http.Request {
	return req.WithContext(tenant.WithTenantID(req.Context(), genSettingsTenantID))
}

func genSettingsRequest(method, url string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	return genSettingsWithTenant(req)
}

func newGenSettingsHandler(q *fakeGenSettingsQuerier) *v1.GeneralSettingsHandler {
	return v1.NewGeneralSettingsHandler(q, &fakeEventBus{})
}

// --- Tests ---

func TestGeneralSettingsGet(t *testing.T) {
	q := &fakeGenSettingsQuerier{
		getResult: sqlcgen.GetGeneralSettingsRow{
			OrgName:           "Acme Corp",
			Timezone:          "UTC",
			DateFormat:        "YYYY-MM-DD",
			ScanIntervalHours: 24,
		},
	}
	h := newGenSettingsHandler(q)

	req := genSettingsRequest(http.MethodGet, "/api/v1/settings/general", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "Acme Corp", resp["org_name"])
	assert.Equal(t, "UTC", resp["timezone"])
	assert.Equal(t, "YYYY-MM-DD", resp["date_format"])
	assert.EqualValues(t, 24, resp["scan_interval_hours"])
}

func TestGeneralSettingsUpdate(t *testing.T) {
	q := &fakeGenSettingsQuerier{
		updateResult: sqlcgen.UpdateGeneralSettingsRow{
			OrgName:           "New Corp",
			Timezone:          "America/New_York",
			DateFormat:        "MM/DD/YYYY",
			ScanIntervalHours: 12,
		},
	}
	h := newGenSettingsHandler(q)

	body := map[string]any{
		"org_name":            "New Corp",
		"timezone":            "America/New_York",
		"date_format":         "MM/DD/YYYY",
		"scan_interval_hours": 12,
	}
	req := genSettingsRequest(http.MethodPut, "/api/v1/settings/general", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "New Corp", resp["org_name"])
	assert.Equal(t, "America/New_York", resp["timezone"])
}

func TestGeneralSettingsUpdate_InvalidTimezone(t *testing.T) {
	h := newGenSettingsHandler(&fakeGenSettingsQuerier{})

	body := map[string]any{
		"org_name":            "Acme",
		"timezone":            "Not/A/Timezone",
		"date_format":         "YYYY-MM-DD",
		"scan_interval_hours": 24,
	}
	req := genSettingsRequest(http.MethodPut, "/api/v1/settings/general", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGeneralSettingsUpdate_InvalidScanInterval(t *testing.T) {
	h := newGenSettingsHandler(&fakeGenSettingsQuerier{})

	body := map[string]any{
		"org_name":            "Acme",
		"timezone":            "UTC",
		"date_format":         "YYYY-MM-DD",
		"scan_interval_hours": 3, // 3 is not a valid interval
	}
	req := genSettingsRequest(http.MethodPut, "/api/v1/settings/general", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGeneralSettingsUpdate_InvalidDateFormat(t *testing.T) {
	h := newGenSettingsHandler(&fakeGenSettingsQuerier{})

	body := map[string]any{
		"org_name":            "Acme",
		"timezone":            "UTC",
		"date_format":         "invalid-format",
		"scan_interval_hours": 24,
	}
	req := genSettingsRequest(http.MethodPut, "/api/v1/settings/general", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGeneralSettingsUpdate_EmptyOrgName(t *testing.T) {
	h := newGenSettingsHandler(&fakeGenSettingsQuerier{})

	body := map[string]any{
		"org_name":            "",
		"timezone":            "UTC",
		"date_format":         "YYYY-MM-DD",
		"scan_interval_hours": 24,
	}
	req := genSettingsRequest(http.MethodPut, "/api/v1/settings/general", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
