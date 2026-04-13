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
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDeploymentQuerier mocks DeploymentQuerier.
type fakeDeploymentQuerier struct {
	// EvalQuerier
	getPolicyResult          sqlcgen.Policy
	getPolicyErr             error
	listEndpointsByIDsResult []sqlcgen.ListEndpointsByIDsRow
	listEndpointsByIDsErr    error
	listPatchesResult        []sqlcgen.Patch
	listPatchesErr           error

	// CancelQuerier
	setDeploymentCancelledResult sqlcgen.Deployment
	setDeploymentCancelledErr    error
	cancelDeploymentTargetsErr   error
	cancelCommandsErr            error

	// CRUD
	createDeploymentResult       sqlcgen.Deployment
	createDeploymentErr          error
	createDeploymentWaveResult   sqlcgen.DeploymentWave
	createDeploymentWaveErr      error
	createDeploymentTargetResult sqlcgen.DeploymentTarget
	createDeploymentTargetErr    error
	getDeploymentResult          sqlcgen.Deployment
	getDeploymentErr             error
	listDeploymentsResult        []sqlcgen.Deployment
	listDeploymentsErr           error
	countDeploymentsResult       int64
	countDeploymentsErr          error
	listTargetsResult            []sqlcgen.DeploymentTarget
	listTargetsErr               error
	setTotalTargetsResult        sqlcgen.Deployment
	setTotalTargetsErr           error

	// Wave-aware CRUD
	createDeploymentWithWaveConfigResult sqlcgen.Deployment
	createDeploymentWithWaveConfigErr    error
	createDeploymentWaveWithConfigResult sqlcgen.DeploymentWave
	createDeploymentWaveWithConfigErr    error
	createDeploymentTargetWithWaveResult sqlcgen.DeploymentTarget
	createDeploymentTargetWithWaveErr    error
	setDeploymentWaveTargetCountResult   sqlcgen.DeploymentWave
	setDeploymentWaveTargetCountErr      error
	listDeploymentWavesResult            []sqlcgen.DeploymentWave
	listDeploymentWavesErr               error
}

func (f *fakeDeploymentQuerier) GetPolicyByID(_ context.Context, _ sqlcgen.GetPolicyByIDParams) (sqlcgen.Policy, error) {
	return f.getPolicyResult, f.getPolicyErr
}
func (f *fakeDeploymentQuerier) ListEndpointsByIDs(_ context.Context, _ sqlcgen.ListEndpointsByIDsParams) ([]sqlcgen.ListEndpointsByIDsRow, error) {
	return f.listEndpointsByIDsResult, f.listEndpointsByIDsErr
}
func (f *fakeDeploymentQuerier) ListPatchesForPolicyFilters(_ context.Context, _ sqlcgen.ListPatchesForPolicyFiltersParams) ([]sqlcgen.Patch, error) {
	return f.listPatchesResult, f.listPatchesErr
}
func (f *fakeDeploymentQuerier) SetDeploymentCancelled(_ context.Context, _ sqlcgen.SetDeploymentCancelledParams) (sqlcgen.Deployment, error) {
	return f.setDeploymentCancelledResult, f.setDeploymentCancelledErr
}
func (f *fakeDeploymentQuerier) CancelDeploymentTargets(_ context.Context, _ sqlcgen.CancelDeploymentTargetsParams) error {
	return f.cancelDeploymentTargetsErr
}
func (f *fakeDeploymentQuerier) CancelCommandsByDeployment(_ context.Context, _ sqlcgen.CancelCommandsByDeploymentParams) error {
	return f.cancelCommandsErr
}
func (f *fakeDeploymentQuerier) CreateDeployment(_ context.Context, _ sqlcgen.CreateDeploymentParams) (sqlcgen.Deployment, error) {
	return f.createDeploymentResult, f.createDeploymentErr
}
func (f *fakeDeploymentQuerier) CreateDeploymentWave(_ context.Context, _ sqlcgen.CreateDeploymentWaveParams) (sqlcgen.DeploymentWave, error) {
	return f.createDeploymentWaveResult, f.createDeploymentWaveErr
}
func (f *fakeDeploymentQuerier) CreateDeploymentTarget(_ context.Context, _ sqlcgen.CreateDeploymentTargetParams) (sqlcgen.DeploymentTarget, error) {
	return f.createDeploymentTargetResult, f.createDeploymentTargetErr
}
func (f *fakeDeploymentQuerier) GetDeploymentByID(_ context.Context, _ sqlcgen.GetDeploymentByIDParams) (sqlcgen.Deployment, error) {
	return f.getDeploymentResult, f.getDeploymentErr
}
func (f *fakeDeploymentQuerier) ListDeploymentsByTenantFiltered(_ context.Context, _ sqlcgen.ListDeploymentsByTenantFilteredParams) ([]sqlcgen.Deployment, error) {
	return f.listDeploymentsResult, f.listDeploymentsErr
}
func (f *fakeDeploymentQuerier) CountDeploymentsByTenantFiltered(_ context.Context, _ sqlcgen.CountDeploymentsByTenantFilteredParams) (int64, error) {
	return f.countDeploymentsResult, f.countDeploymentsErr
}
func (f *fakeDeploymentQuerier) ListDeploymentTargets(_ context.Context, _ sqlcgen.ListDeploymentTargetsParams) ([]sqlcgen.DeploymentTarget, error) {
	return f.listTargetsResult, f.listTargetsErr
}
func (f *fakeDeploymentQuerier) SetDeploymentTotalTargets(_ context.Context, _ sqlcgen.SetDeploymentTotalTargetsParams) (sqlcgen.Deployment, error) {
	return f.setTotalTargetsResult, f.setTotalTargetsErr
}
func (f *fakeDeploymentQuerier) CreateDeploymentWithWaveConfig(_ context.Context, _ sqlcgen.CreateDeploymentWithWaveConfigParams) (sqlcgen.Deployment, error) {
	return f.createDeploymentWithWaveConfigResult, f.createDeploymentWithWaveConfigErr
}
func (f *fakeDeploymentQuerier) CreateDeploymentWaveWithConfig(_ context.Context, _ sqlcgen.CreateDeploymentWaveWithConfigParams) (sqlcgen.DeploymentWave, error) {
	return f.createDeploymentWaveWithConfigResult, f.createDeploymentWaveWithConfigErr
}
func (f *fakeDeploymentQuerier) CreateDeploymentWithOrchestration(_ context.Context, _ sqlcgen.CreateDeploymentWithOrchestrationParams) (sqlcgen.Deployment, error) {
	return f.createDeploymentResult, f.createDeploymentErr
}
func (f *fakeDeploymentQuerier) CreateDeploymentTargetWithWave(_ context.Context, _ sqlcgen.CreateDeploymentTargetWithWaveParams) (sqlcgen.DeploymentTarget, error) {
	return f.createDeploymentTargetWithWaveResult, f.createDeploymentTargetWithWaveErr
}
func (f *fakeDeploymentQuerier) SetDeploymentWaveTargetCount(_ context.Context, _ sqlcgen.SetDeploymentWaveTargetCountParams) (sqlcgen.DeploymentWave, error) {
	return f.setDeploymentWaveTargetCountResult, f.setDeploymentWaveTargetCountErr
}
func (f *fakeDeploymentQuerier) ListDeploymentWaves(_ context.Context, _ sqlcgen.ListDeploymentWavesParams) ([]sqlcgen.DeploymentWave, error) {
	return f.listDeploymentWavesResult, f.listDeploymentWavesErr
}
func (f *fakeDeploymentQuerier) ListDeploymentTargetsWithHostname(_ context.Context, _ sqlcgen.ListDeploymentTargetsWithHostnameParams) ([]sqlcgen.ListDeploymentTargetsWithHostnameRow, error) {
	return nil, nil
}
func (f *fakeDeploymentQuerier) ListDeploymentTargetsByWave(_ context.Context, _ sqlcgen.ListDeploymentTargetsByWaveParams) ([]sqlcgen.ListDeploymentTargetsByWaveRow, error) {
	return nil, nil
}
func (f *fakeDeploymentQuerier) RetryFailedTargets(_ context.Context, _ sqlcgen.RetryFailedTargetsParams) (int64, error) {
	return 0, nil
}
func (f *fakeDeploymentQuerier) CountDeploymentsByStatus(_ context.Context, _ pgtype.UUID) ([]sqlcgen.CountDeploymentsByStatusRow, error) {
	return nil, nil
}
func (f *fakeDeploymentQuerier) ListDeploymentPatchSummary(_ context.Context, _ sqlcgen.ListDeploymentPatchSummaryParams) ([]sqlcgen.ListDeploymentPatchSummaryRow, error) {
	return nil, nil
}
func (f *fakeDeploymentQuerier) SetDeploymentRetrying(_ context.Context, _ sqlcgen.SetDeploymentRetryingParams) (sqlcgen.Deployment, error) {
	return sqlcgen.Deployment{}, nil
}

func validDeployment() sqlcgen.Deployment {
	var id, tid, pid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000050")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = pid.Scan("00000000-0000-0000-0000-000000000010")
	return sqlcgen.Deployment{
		ID:       id,
		TenantID: tid,
		PolicyID: pid,
		Status:   "created",
	}
}

func newDeploymentHandler(q *fakeDeploymentQuerier, eb *fakeEventBus) *v1.DeploymentHandler {
	// For unit tests, pass nil pool and riverClient since Create tests
	// that exercise the transaction path need integration tests.
	return v1.NewDeploymentHandlerForTest(q, eb, deployment.NewEvaluator(nil), deployment.NewStateMachine())
}

// --- Create Tests ---

func TestDeploymentHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		querier    *fakeDeploymentQuerier
		wantStatus int
	}{
		// NOTE: The happy-path Create test (201) requires a real pgxpool.Pool and
		// River client for the transactional flow. It is covered by integration tests.
		// Unit tests here validate request validation paths only.
		{
			name:       "missing policy_id returns 400",
			body:       map[string]string{},
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			body:       "not json",
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid policy_id UUID returns 400",
			body:       map[string]string{"policy_id": "not-a-uuid"},
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "wave percentages not summing to 100 returns 400",
			body: map[string]any{
				"policy_id": "00000000-0000-0000-0000-000000000010",
				"wave_config": []map[string]any{
					{"percentage": 30, "success_threshold": 0.9, "error_rate_max": 0.1, "delay_minutes": 5},
					{"percentage": 30, "success_threshold": 0.9, "error_rate_max": 0.1, "delay_minutes": 0},
				},
			},
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newDeploymentHandler(tt.querier, eb)
			var bodyBytes []byte
			switch v := tt.body.(type) {
			case string:
				bodyBytes = []byte(v)
			default:
				bodyBytes, _ = json.Marshal(v)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenantCtx(req.Context()))
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- List Tests ---

func TestDeploymentHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakeDeploymentQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns list",
			querier: &fakeDeploymentQuerier{
				listDeploymentsResult:  []sqlcgen.Deployment{validDeployment()},
				countDeploymentsResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "returns empty list",
			querier: &fakeDeploymentQuerier{
				listDeploymentsResult:  []sqlcgen.Deployment{},
				countDeploymentsResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "invalid cursor returns 400",
			query:      "?cursor=bad-cursor",
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newDeploymentHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments"+tt.query, nil)
			req = req.WithContext(tenantCtx(req.Context()))
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

// --- Get Tests ---

func TestDeploymentHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeDeploymentQuerier
		wantStatus int
	}{
		{
			name: "found returns 200",
			id:   "00000000-0000-0000-0000-000000000050",
			querier: &fakeDeploymentQuerier{
				getDeploymentResult: validDeployment(),
				listTargetsResult:   []sqlcgen.DeploymentTarget{},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000050",
			querier:    &fakeDeploymentQuerier{getDeploymentErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newDeploymentHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/"+tt.id, nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Cancel Tests ---

func TestDeploymentHandler_Cancel(t *testing.T) {
	t.Run("invalid UUID returns 400", func(t *testing.T) {
		eb := &fakeEventBus{}
		h := newDeploymentHandler(&fakeDeploymentQuerier{}, eb)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/not-a-uuid/cancel", nil)
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", "not-a-uuid")
		rec := httptest.NewRecorder()

		h.Cancel(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		eb := &fakeEventBus{}
		q := &fakeDeploymentQuerier{
			setDeploymentCancelledErr: pgx.ErrNoRows,
		}

		cancelTxFactory := func(_ context.Context, _ string) (deployment.CancelQuerier, func() error, func() error, error) {
			noop := func() error { return nil }
			return q, noop, noop, nil
		}

		h := v1.NewDeploymentHandlerWithCancelTxForTest(q, cancelTxFactory, eb, deployment.NewEvaluator(nil), deployment.NewStateMachine())
		depID := "00000000-0000-0000-0000-000000000050"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+depID+"/cancel", nil)
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", depID)
		rec := httptest.NewRecorder()

		h.Cancel(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Empty(t, eb.events, "expected no events on not-found")
	})

	t.Run("commit failure returns 500 and emits zero events", func(t *testing.T) {
		eb := &fakeEventBus{}
		q := &fakeDeploymentQuerier{
			setDeploymentCancelledResult: validDeployment(),
		}
		q.setDeploymentCancelledResult.Status = "cancelled"

		commitErr := errors.New("commit failed")
		cancelTxFactory := func(_ context.Context, _ string) (deployment.CancelQuerier, func() error, func() error, error) {
			noop := func() error { return nil }
			commit := func() error { return commitErr }
			return q, commit, noop, nil
		}

		h := v1.NewDeploymentHandlerWithCancelTxForTest(q, cancelTxFactory, eb, deployment.NewEvaluator(nil), deployment.NewStateMachine())
		depID := "00000000-0000-0000-0000-000000000050"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+depID+"/cancel", nil)
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", depID)
		rec := httptest.NewRecorder()

		h.Cancel(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Empty(t, eb.events, "expected no events when commit fails")
	})

	t.Run("success emits cancellation events", func(t *testing.T) {
		eb := &fakeEventBus{}
		q := &fakeDeploymentQuerier{
			setDeploymentCancelledResult: validDeployment(),
		}
		q.setDeploymentCancelledResult.Status = "cancelled"

		cancelTxFactory := func(_ context.Context, _ string) (deployment.CancelQuerier, func() error, func() error, error) {
			noop := func() error { return nil }
			return q, noop, noop, nil
		}

		h := v1.NewDeploymentHandlerWithCancelTxForTest(q, cancelTxFactory, eb, deployment.NewEvaluator(nil), deployment.NewStateMachine())
		depID := "00000000-0000-0000-0000-000000000050"
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/"+depID+"/cancel", nil)
		req = req.WithContext(tenantCtx(req.Context()))
		req = chiCtx(req, "id", depID)
		rec := httptest.NewRecorder()

		h.Cancel(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		require.NotEmpty(t, eb.events, "expected cancellation events to be emitted")

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "cancelled", body["status"])
	})
}

// --- GetWaves Tests ---

func TestDeploymentHandler_GetWaves(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeDeploymentQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns waves list",
			id:   "00000000-0000-0000-0000-000000000050",
			querier: &fakeDeploymentQuerier{
				listDeploymentWavesResult: []sqlcgen.DeploymentWave{
					validDeploymentWave(1),
				},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "returns empty waves list",
			id:   "00000000-0000-0000-0000-000000000050",
			querier: &fakeDeploymentQuerier{
				listDeploymentWavesResult: []sqlcgen.DeploymentWave{},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeDeploymentQuerier{},
			wantStatus: http.StatusBadRequest,
			wantLen:    -1,
		},
		{
			name: "db error returns 500",
			id:   "00000000-0000-0000-0000-000000000050",
			querier: &fakeDeploymentQuerier{
				listDeploymentWavesErr: errors.New("db error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newDeploymentHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/"+tt.id+"/waves", nil)
			req = req.WithContext(tenantCtx(req.Context()))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.GetWaves(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body []map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body, tt.wantLen)
			}
		})
	}
}

func validDeploymentWave(waveNumber int32) sqlcgen.DeploymentWave {
	var id, tid, did pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000060")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = did.Scan("00000000-0000-0000-0000-000000000050")
	return sqlcgen.DeploymentWave{
		ID:           id,
		TenantID:     tid,
		DeploymentID: did,
		WaveNumber:   waveNumber,
		Status:       "pending",
		Percentage:   100,
		TargetCount:  5,
	}
}
