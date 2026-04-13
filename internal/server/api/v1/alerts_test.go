package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAlertQuerier mocks AlertQuerier.
type fakeAlertQuerier struct {
	listResult        []sqlcgen.Alert
	listErr           error
	countResult       int64
	countErr          error
	countUnreadResult sqlcgen.CountUnreadAlertsRow
	countUnreadErr    error

	getCreatedAtResult pgtype.Timestamptz
	getCreatedAtErr    error

	updateStatusResult sqlcgen.Alert
	updateStatusErr    error

	bulkUpdateResult int64
	bulkUpdateErr    error

	listRulesResult  []sqlcgen.AlertRule
	listRulesErr     error
	getRuleResult    sqlcgen.AlertRule
	getRuleErr       error
	createRuleResult sqlcgen.AlertRule
	createRuleErr    error
	updateRuleResult sqlcgen.AlertRule
	updateRuleErr    error
	deleteRuleResult int64
	deleteRuleErr    error
}

func (f *fakeAlertQuerier) ListAlertsFiltered(_ context.Context, _ sqlcgen.ListAlertsFilteredParams) ([]sqlcgen.Alert, error) {
	return f.listResult, f.listErr
}

func (f *fakeAlertQuerier) CountAlertsFiltered(_ context.Context, _ sqlcgen.CountAlertsFilteredParams) (int64, error) {
	return f.countResult, f.countErr
}

func (f *fakeAlertQuerier) CountUnreadAlerts(_ context.Context, _ sqlcgen.CountUnreadAlertsParams) (sqlcgen.CountUnreadAlertsRow, error) {
	return f.countUnreadResult, f.countUnreadErr
}

func (f *fakeAlertQuerier) GetAlertCreatedAt(_ context.Context, _ sqlcgen.GetAlertCreatedAtParams) (pgtype.Timestamptz, error) {
	return f.getCreatedAtResult, f.getCreatedAtErr
}

func (f *fakeAlertQuerier) UpdateAlertStatus(_ context.Context, _ sqlcgen.UpdateAlertStatusParams) (sqlcgen.Alert, error) {
	return f.updateStatusResult, f.updateStatusErr
}

func (f *fakeAlertQuerier) BulkUpdateAlertStatus(_ context.Context, _ sqlcgen.BulkUpdateAlertStatusParams) (int64, error) {
	return f.bulkUpdateResult, f.bulkUpdateErr
}

func (f *fakeAlertQuerier) ListAlertRules(_ context.Context, _ pgtype.UUID) ([]sqlcgen.AlertRule, error) {
	return f.listRulesResult, f.listRulesErr
}

func (f *fakeAlertQuerier) GetAlertRule(_ context.Context, _ sqlcgen.GetAlertRuleParams) (sqlcgen.AlertRule, error) {
	return f.getRuleResult, f.getRuleErr
}

func (f *fakeAlertQuerier) CreateAlertRule(_ context.Context, _ sqlcgen.CreateAlertRuleParams) (sqlcgen.AlertRule, error) {
	return f.createRuleResult, f.createRuleErr
}

func (f *fakeAlertQuerier) UpdateAlertRule(_ context.Context, _ sqlcgen.UpdateAlertRuleParams) (sqlcgen.AlertRule, error) {
	return f.updateRuleResult, f.updateRuleErr
}

func (f *fakeAlertQuerier) DeleteAlertRule(_ context.Context, _ sqlcgen.DeleteAlertRuleParams) (int64, error) {
	return f.deleteRuleResult, f.deleteRuleErr
}

func validAlert() sqlcgen.Alert {
	var tid pgtype.UUID
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	var ruleID pgtype.UUID
	_ = ruleID.Scan("00000000-0000-0000-0000-000000000099")
	return sqlcgen.Alert{
		ID:          "01JTEST000000000000000000",
		TenantID:    tid,
		RuleID:      ruleID,
		EventID:     "evt-001",
		Severity:    "critical",
		Category:    "security",
		Title:       "Test Alert",
		Description: "Something happened",
		Resource:    "endpoint",
		ResourceID:  "ep-123",
		Status:      "unread",
		Payload:     []byte(`{"key":"value"}`),
		CreatedAt:   pgtype.Timestamptz{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
}

func validAlertRule() sqlcgen.AlertRule {
	var tid pgtype.UUID
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	var id pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	return sqlcgen.AlertRule{
		ID:                  id,
		TenantID:            tid,
		EventType:           "endpoint.created",
		Severity:            "warning",
		Category:            "system",
		TitleTemplate:       "New endpoint: {{.Name}}",
		DescriptionTemplate: "Endpoint {{.Name}} was created",
		Enabled:             true,
		CreatedAt:           pgtype.Timestamptz{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
	}
}

func newAlertHandler(q *fakeAlertQuerier) *v1.AlertHandler {
	return v1.NewAlertHandler(q, nil, &fakeEventBus{}, nil)
}

func TestAlertHandler_List_Empty(t *testing.T) {
	q := &fakeAlertQuerier{
		listResult:  []sqlcgen.Alert{},
		countResult: 0,
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body["data"], 0)
	assert.Equal(t, float64(0), body["total_count"])
}

func TestAlertHandler_List_WithResults(t *testing.T) {
	q := &fakeAlertQuerier{
		listResult:  []sqlcgen.Alert{validAlert()},
		countResult: 1,
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?severity=critical&category=security", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body["data"], 1)
	assert.Equal(t, float64(1), body["total_count"])

	items, _ := body["data"].([]any)
	first, _ := items[0].(map[string]any)
	assert.Equal(t, "01JTEST000000000000000000", first["id"])
	assert.Equal(t, "critical", first["severity"])
	assert.Equal(t, "security", first["category"])
}

func TestAlertHandler_List_StoreError(t *testing.T) {
	q := &fakeAlertQuerier{
		listErr: fmt.Errorf("database connection failed"),
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAlertHandler_Count(t *testing.T) {
	q := &fakeAlertQuerier{
		countUnreadResult: sqlcgen.CountUnreadAlertsRow{
			CriticalUnread: 3,
			WarningUnread:  5,
			InfoUnread:     10,
			TotalUnread:    18,
		},
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/count", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(3), body["critical_unread"])
	assert.Equal(t, float64(5), body["warning_unread"])
	assert.Equal(t, float64(10), body["info_unread"])
	assert.Equal(t, float64(18), body["total_unread"])
}

func TestAlertHandler_Count_Error(t *testing.T) {
	q := &fakeAlertQuerier{
		countUnreadErr: fmt.Errorf("db error"),
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/count", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Count(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAlertHandler_UpdateStatus(t *testing.T) {
	alert := validAlert()
	alert.Status = "read"
	alert.ReadAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	q := &fakeAlertQuerier{
		getCreatedAtResult: pgtype.Timestamptz{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		updateStatusResult: alert,
	}
	h := newAlertHandler(q)
	body := `{"status":"read"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/01JTEST000000000000000000/status", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "01JTEST000000000000000000")
	rec := httptest.NewRecorder()

	h.UpdateStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "read", resp["status"])
}

func TestAlertHandler_UpdateStatus_NotFound(t *testing.T) {
	q := &fakeAlertQuerier{
		getCreatedAtErr: fmt.Errorf("no rows in result set"),
	}
	h := newAlertHandler(q)
	body := `{"status":"read"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/nonexistent/status", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "nonexistent")
	rec := httptest.NewRecorder()

	h.UpdateStatus(rec, req)

	// Should return 500 or 404 depending on error type; the mock returns generic error
	assert.True(t, rec.Code >= 400)
}

func TestAlertHandler_BulkUpdateStatus(t *testing.T) {
	q := &fakeAlertQuerier{
		bulkUpdateResult: 3,
	}
	h := newAlertHandler(q)
	body := `{"ids":["id1","id2","id3"],"status":"dismissed"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/bulk-status", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.BulkUpdateStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(3), resp["updated_count"])
}

func TestAlertHandler_BulkUpdateStatus_InvalidBody(t *testing.T) {
	q := &fakeAlertQuerier{}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/alerts/bulk-status", strings.NewReader("not json"))
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.BulkUpdateStatus(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAlertHandler_ListRules(t *testing.T) {
	q := &fakeAlertQuerier{
		listRulesResult: []sqlcgen.AlertRule{validAlertRule()},
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/alert-rules", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.ListRules(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body, 1)
	assert.Equal(t, "endpoint.created", body[0]["event_type"])
}

func TestAlertHandler_CreateRule(t *testing.T) {
	q := &fakeAlertQuerier{
		createRuleResult: validAlertRule(),
	}
	h := newAlertHandler(q)
	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"New endpoint","description_template":"Created","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-rules", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.CreateRule(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "endpoint.created", resp["event_type"])
}

func TestAlertHandler_UpdateRule(t *testing.T) {
	q := &fakeAlertQuerier{
		updateRuleResult: validAlertRule(),
	}
	h := newAlertHandler(q)
	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"Updated","description_template":"Updated desc","enabled":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/00000000-0000-0000-0000-000000000099", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.UpdateRule(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAlertHandler_DeleteRule(t *testing.T) {
	q := &fakeAlertQuerier{
		deleteRuleResult: 1,
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.DeleteRule(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// fakeBackfiller records calls to Backfill and satisfies v1.AlertBackfiller.
type fakeBackfiller struct {
	calls []events.BackfillRule
	ret   int
	err   error
}

func (f *fakeBackfiller) Backfill(_ context.Context, rule events.BackfillRule, _ time.Duration) (int, error) {
	f.calls = append(f.calls, rule)
	return f.ret, f.err
}

func TestAlertHandler_CreateRule_TriggersBackfill(t *testing.T) {
	q := &fakeAlertQuerier{createRuleResult: validAlertRule()}
	bf := &fakeBackfiller{}
	h := v1.NewAlertHandler(q, nil, &fakeEventBus{}, bf)

	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"x","description_template":"y","enabled":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-rules", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.CreateRule(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, bf.calls, 1, "expected exactly one backfill call for newly created enabled rule")
	assert.Equal(t, "endpoint.created", bf.calls[0].EventType)
}

func TestAlertHandler_CreateRule_DisabledSkipsBackfill(t *testing.T) {
	disabled := validAlertRule()
	disabled.Enabled = false
	q := &fakeAlertQuerier{createRuleResult: disabled}
	bf := &fakeBackfiller{}
	h := v1.NewAlertHandler(q, nil, &fakeEventBus{}, bf)

	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"x","description_template":"y","enabled":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-rules", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.CreateRule(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Empty(t, bf.calls, "disabled rule creation must not backfill")
}

func TestAlertHandler_UpdateRule_BackfillsOnEnableTransition(t *testing.T) {
	prev := validAlertRule()
	prev.Enabled = false
	next := validAlertRule()
	next.Enabled = true
	q := &fakeAlertQuerier{getRuleResult: prev, updateRuleResult: next}
	bf := &fakeBackfiller{}
	h := v1.NewAlertHandler(q, nil, &fakeEventBus{}, bf)

	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"x","description_template":"y","enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/00000000-0000-0000-0000-000000000099", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.UpdateRule(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, bf.calls, 1, "disabled->enabled transition must trigger backfill")
}

func TestAlertHandler_UpdateRule_NoBackfillWhenAlreadyEnabled(t *testing.T) {
	prev := validAlertRule()
	prev.Enabled = true
	next := validAlertRule()
	next.Enabled = true
	q := &fakeAlertQuerier{getRuleResult: prev, updateRuleResult: next}
	bf := &fakeBackfiller{}
	h := v1.NewAlertHandler(q, nil, &fakeEventBus{}, bf)

	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"x","description_template":"y","enabled":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/00000000-0000-0000-0000-000000000099", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.UpdateRule(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, bf.calls, "enabled->enabled edit must not re-backfill")
}

func TestAlertHandler_UpdateRule_NoBackfillOnDisable(t *testing.T) {
	prev := validAlertRule()
	prev.Enabled = true
	next := validAlertRule()
	next.Enabled = false
	q := &fakeAlertQuerier{getRuleResult: prev, updateRuleResult: next}
	bf := &fakeBackfiller{}
	h := v1.NewAlertHandler(q, nil, &fakeEventBus{}, bf)

	body := `{"event_type":"endpoint.created","severity":"warning","category":"system","title_template":"x","description_template":"y","enabled":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/00000000-0000-0000-0000-000000000099", strings.NewReader(body))
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.UpdateRule(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, bf.calls, "enabled->disabled transition must not backfill")
}

func TestAlertHandler_DeleteRule_NotFound(t *testing.T) {
	q := &fakeAlertQuerier{
		deleteRuleResult: 0,
	}
	h := newAlertHandler(q)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/alert-rules/00000000-0000-0000-0000-000000000099", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000099")
	rec := httptest.NewRecorder()

	h.DeleteRule(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
