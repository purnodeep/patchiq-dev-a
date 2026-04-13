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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeScheduleQuerier mocks ScheduleQuerier.
type fakeScheduleQuerier struct {
	createResult sqlcgen.DeploymentSchedule
	createErr    error
	getResult    sqlcgen.DeploymentSchedule
	getErr       error
	listResult   []sqlcgen.DeploymentSchedule
	listErr      error
	updateResult sqlcgen.DeploymentSchedule
	updateErr    error
	deleteErr    error
}

func (f *fakeScheduleQuerier) CreateDeploymentSchedule(_ context.Context, _ sqlcgen.CreateDeploymentScheduleParams) (sqlcgen.DeploymentSchedule, error) {
	return f.createResult, f.createErr
}
func (f *fakeScheduleQuerier) GetDeploymentScheduleByID(_ context.Context, _ sqlcgen.GetDeploymentScheduleByIDParams) (sqlcgen.DeploymentSchedule, error) {
	return f.getResult, f.getErr
}
func (f *fakeScheduleQuerier) ListDeploymentSchedulesByTenant(_ context.Context, _ pgtype.UUID) ([]sqlcgen.DeploymentSchedule, error) {
	return f.listResult, f.listErr
}
func (f *fakeScheduleQuerier) UpdateDeploymentSchedule(_ context.Context, _ sqlcgen.UpdateDeploymentScheduleParams) (sqlcgen.DeploymentSchedule, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeScheduleQuerier) DeleteDeploymentSchedule(_ context.Context, _ sqlcgen.DeleteDeploymentScheduleParams) error {
	return f.deleteErr
}

func validSchedule() sqlcgen.DeploymentSchedule {
	var id, tid, pid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000070")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = pid.Scan("00000000-0000-0000-0000-000000000010")
	return sqlcgen.DeploymentSchedule{
		ID:             id,
		TenantID:       tid,
		PolicyID:       pid,
		CronExpression: "0 2 * * *",
		Enabled:        true,
		NextRunAt:      pgtype.Timestamptz{Valid: true},
		CreatedAt:      pgtype.Timestamptz{Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Valid: true},
	}
}

func newScheduleHandler(q *fakeScheduleQuerier, eb *fakeEventBus) *v1.ScheduleHandler {
	return v1.NewScheduleHandlerForTest(q, eb)
}

// --- Create Tests ---

func TestScheduleHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		querier    *fakeScheduleQuerier
		wantStatus int
	}{
		{
			name:       "missing policy_id returns 400",
			body:       map[string]string{"cron_expression": "0 2 * * *"},
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing cron_expression returns 400",
			body:       map[string]string{"policy_id": "00000000-0000-0000-0000-000000000010"},
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			body:       "not json",
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid policy_id UUID returns 400",
			body:       map[string]string{"policy_id": "not-a-uuid", "cron_expression": "0 2 * * *"},
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid cron_expression returns 400",
			body:       map[string]string{"policy_id": "00000000-0000-0000-0000-000000000010", "cron_expression": "bad cron"},
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "valid request returns 201",
			body: map[string]string{
				"policy_id":       "00000000-0000-0000-0000-000000000010",
				"cron_expression": "0 2 * * *",
			},
			querier:    &fakeScheduleQuerier{createResult: validSchedule()},
			wantStatus: http.StatusCreated,
		},
		{
			name: "db error returns 500",
			body: map[string]string{
				"policy_id":       "00000000-0000-0000-0000-000000000010",
				"cron_expression": "0 2 * * *",
			},
			querier:    &fakeScheduleQuerier{createErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newScheduleHandler(tt.querier, eb)
			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/deployment-schedules", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- List Tests ---

func TestScheduleHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeScheduleQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns list",
			querier: &fakeScheduleQuerier{
				listResult: []sqlcgen.DeploymentSchedule{validSchedule()},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "returns empty list",
			querier: &fakeScheduleQuerier{
				listResult: []sqlcgen.DeploymentSchedule{},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "db error returns 500",
			querier: &fakeScheduleQuerier{
				listErr: errors.New("db error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newScheduleHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deployment-schedules", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body []map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body, tt.wantLen)
			}
		})
	}
}

// --- Get Tests ---

func TestScheduleHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeScheduleQuerier
		wantStatus int
	}{
		{
			name:       "found returns 200",
			id:         "00000000-0000-0000-0000-000000000070",
			querier:    &fakeScheduleQuerier{getResult: validSchedule()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000070",
			querier:    &fakeScheduleQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "db error returns 500",
			id:         "00000000-0000-0000-0000-000000000070",
			querier:    &fakeScheduleQuerier{getErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newScheduleHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deployment-schedules/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Update Tests ---

func TestScheduleHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       any
		querier    *fakeScheduleQuerier
		wantStatus int
	}{
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			body:       map[string]any{},
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			id:         "00000000-0000-0000-0000-000000000070",
			body:       "not json",
			querier:    &fakeScheduleQuerier{getResult: validSchedule()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found on get returns 404",
			id:         "00000000-0000-0000-0000-000000000070",
			body:       map[string]any{},
			querier:    &fakeScheduleQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid cron expression returns 400",
			id:         "00000000-0000-0000-0000-000000000070",
			body:       map[string]any{"cron_expression": "bad cron"},
			querier:    &fakeScheduleQuerier{getResult: validSchedule()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty cron expression returns 400",
			id:         "00000000-0000-0000-0000-000000000070",
			body:       map[string]any{"cron_expression": ""},
			querier:    &fakeScheduleQuerier{getResult: validSchedule()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "valid update returns 200",
			id:   "00000000-0000-0000-0000-000000000070",
			body: map[string]any{"enabled": false},
			querier: &fakeScheduleQuerier{
				getResult:    validSchedule(),
				updateResult: validSchedule(),
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "db error on update returns 500",
			id:   "00000000-0000-0000-0000-000000000070",
			body: map[string]any{"enabled": false},
			querier: &fakeScheduleQuerier{
				getResult: validSchedule(),
				updateErr: errors.New("db error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newScheduleHandler(tt.querier, eb)
			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/deployment-schedules/"+tt.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Update(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Delete Tests ---

func TestScheduleHandler_Delete(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeScheduleQuerier
		wantStatus int
	}{
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "success returns 204",
			id:         "00000000-0000-0000-0000-000000000070",
			querier:    &fakeScheduleQuerier{},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "db error returns 500",
			id:         "00000000-0000-0000-0000-000000000070",
			querier:    &fakeScheduleQuerier{deleteErr: errors.New("db error")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newScheduleHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/deployment-schedules/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Event Emission Tests ---

func TestScheduleHandler_Create_EmitsEvent(t *testing.T) {
	eb := &fakeEventBus{}
	q := &fakeScheduleQuerier{createResult: validSchedule()}
	h := newScheduleHandler(q, eb)

	body, _ := json.Marshal(map[string]string{
		"policy_id":       "00000000-0000-0000-0000-000000000010",
		"cron_expression": "0 2 * * *",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployment-schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenantCtx(req.Context()))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	require.Len(t, eb.events, 1)
	assert.Equal(t, "schedule.created", eb.events[0].Type)
}

func TestScheduleHandler_Delete_EmitsEvent(t *testing.T) {
	eb := &fakeEventBus{}
	q := &fakeScheduleQuerier{}
	h := newScheduleHandler(q, eb)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/deployment-schedules/00000000-0000-0000-0000-000000000070", nil)
	req = req.WithContext(tenantCtx(req.Context()))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000070")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.Len(t, eb.events, 1)
	assert.Equal(t, "schedule.deleted", eb.events[0].Type)
}
