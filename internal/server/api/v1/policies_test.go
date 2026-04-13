package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePolicyQuerier mocks PolicyQuerier.
type fakePolicyQuerier struct {
	createResult     sqlcgen.Policy
	createErr        error
	getResult        sqlcgen.Policy
	getErr           error
	listResult       []sqlcgen.Policy
	listErr          error
	countResult      int64
	countErr         error
	updateResult     sqlcgen.Policy
	updateErr        error
	softDeleteResult sqlcgen.Policy
	softDeleteErr    error

	// New fields for extended interface.
	listWithStatsResult    []sqlcgen.ListPoliciesWithStatsRow
	listWithStatsErr       error
	countFilteredResult    int64
	countFilteredErr       error
	bulkUpdateErr          error
	bulkDeleteErr          error
	listEvalsResult        []sqlcgen.PolicyEvaluation
	listEvalsErr           error
	createEvalResult       sqlcgen.PolicyEvaluation
	createEvalErr          error
	updateEvalStatsErr     error
	countDeploymentsResult int64
	countDeploymentsErr    error
	listDeploymentsResult  []sqlcgen.Deployment
	listDeploymentsErr     error
	endpointsByIDsResult   []sqlcgen.ListEndpointsByIDsRow
	endpointsByIDsErr      error
	selectorResult         sqlcgen.PolicyTagSelector
	selectorErr            error
}

func (f *fakePolicyQuerier) CreatePolicy(_ context.Context, _ sqlcgen.CreatePolicyParams) (sqlcgen.Policy, error) {
	return f.createResult, f.createErr
}
func (f *fakePolicyQuerier) GetPolicyByID(_ context.Context, _ sqlcgen.GetPolicyByIDParams) (sqlcgen.Policy, error) {
	return f.getResult, f.getErr
}
func (f *fakePolicyQuerier) ListPolicies(_ context.Context, _ sqlcgen.ListPoliciesParams) ([]sqlcgen.Policy, error) {
	return f.listResult, f.listErr
}
func (f *fakePolicyQuerier) CountPolicies(_ context.Context, _ sqlcgen.CountPoliciesParams) (int64, error) {
	return f.countResult, f.countErr
}
func (f *fakePolicyQuerier) UpdatePolicy(_ context.Context, _ sqlcgen.UpdatePolicyParams) (sqlcgen.Policy, error) {
	return f.updateResult, f.updateErr
}
func (f *fakePolicyQuerier) SoftDeletePolicy(_ context.Context, _ sqlcgen.SoftDeletePolicyParams) (sqlcgen.Policy, error) {
	return f.softDeleteResult, f.softDeleteErr
}
func (f *fakePolicyQuerier) ListEndpointsByIDs(_ context.Context, _ sqlcgen.ListEndpointsByIDsParams) ([]sqlcgen.ListEndpointsByIDsRow, error) {
	return f.endpointsByIDsResult, f.endpointsByIDsErr
}
func (f *fakePolicyQuerier) UpsertPolicyTagSelector(_ context.Context, _ sqlcgen.UpsertPolicyTagSelectorParams) (sqlcgen.PolicyTagSelector, error) {
	return f.selectorResult, f.selectorErr
}
func (f *fakePolicyQuerier) GetPolicyTagSelector(_ context.Context, _ sqlcgen.GetPolicyTagSelectorParams) (sqlcgen.PolicyTagSelector, error) {
	return f.selectorResult, f.selectorErr
}
func (f *fakePolicyQuerier) DeletePolicyTagSelector(_ context.Context, _ sqlcgen.DeletePolicyTagSelectorParams) error {
	return nil
}
func (f *fakePolicyQuerier) ListPoliciesWithStats(_ context.Context, _ sqlcgen.ListPoliciesWithStatsParams) ([]sqlcgen.ListPoliciesWithStatsRow, error) {
	return f.listWithStatsResult, f.listWithStatsErr
}
func (f *fakePolicyQuerier) CountPoliciesFiltered(_ context.Context, _ sqlcgen.CountPoliciesFilteredParams) (int64, error) {
	return f.countFilteredResult, f.countFilteredErr
}
func (f *fakePolicyQuerier) BulkUpdatePolicyEnabled(_ context.Context, _ sqlcgen.BulkUpdatePolicyEnabledParams) error {
	return f.bulkUpdateErr
}
func (f *fakePolicyQuerier) BulkSoftDeletePolicies(_ context.Context, _ sqlcgen.BulkSoftDeletePoliciesParams) error {
	return f.bulkDeleteErr
}
func (f *fakePolicyQuerier) ListPolicyEvaluations(_ context.Context, _ sqlcgen.ListPolicyEvaluationsParams) ([]sqlcgen.PolicyEvaluation, error) {
	return f.listEvalsResult, f.listEvalsErr
}
func (f *fakePolicyQuerier) CreatePolicyEvaluation(_ context.Context, _ sqlcgen.CreatePolicyEvaluationParams) (sqlcgen.PolicyEvaluation, error) {
	return f.createEvalResult, f.createEvalErr
}
func (f *fakePolicyQuerier) UpdatePolicyEvalStats(_ context.Context, _ sqlcgen.UpdatePolicyEvalStatsParams) error {
	return f.updateEvalStatsErr
}
func (f *fakePolicyQuerier) CountDeploymentsForPolicy(_ context.Context, _ sqlcgen.CountDeploymentsForPolicyParams) (int64, error) {
	return f.countDeploymentsResult, f.countDeploymentsErr
}
func (f *fakePolicyQuerier) ListDeploymentsForPolicy(_ context.Context, _ sqlcgen.ListDeploymentsForPolicyParams) ([]sqlcgen.Deployment, error) {
	return f.listDeploymentsResult, f.listDeploymentsErr
}

func validPolicy() sqlcgen.Policy {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000077")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.Policy{
		ID:                 id,
		TenantID:           tid,
		Name:               "test-policy",
		Enabled:            true,
		Mode:               "manual",
		SelectionMode:      "all_available",
		ScheduleType:       "manual",
		DeploymentStrategy: "all_at_once",
	}
}

func validPolicyWithStatsRow() sqlcgen.ListPoliciesWithStatsRow {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000077")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.ListPoliciesWithStatsRow{
		ID:                 id,
		TenantID:           tid,
		Name:               "test-policy",
		Enabled:            true,
		Mode:               "manual",
		SelectionMode:      "all_available",
		ScheduleType:       "manual",
		DeploymentStrategy: "all_at_once",
	}
}

// noopEvaluator is a PolicyEvaluator that returns empty results.
type noopEvaluator struct{}

func (noopEvaluator) Evaluate(_ context.Context, _, _ string, _ time.Time) ([]policy.EvaluationResult, error) {
	return nil, nil
}

// policyRow implements pgx.Row, scanning values from a sqlcgen.Policy.
type policyRow struct {
	p   sqlcgen.Policy
	err error
}

func (r policyRow) Scan(dest ...any) error { //nolint:errcheck // type assertions in test mock
	if r.err != nil {
		return r.err
	}
	// Match the column order from UpdatePolicy's RETURNING clause.
	vals := []any{
		r.p.ID, r.p.TenantID, r.p.Name, r.p.Description, r.p.Enabled,
		r.p.CreatedAt, r.p.UpdatedAt, r.p.SelectionMode, r.p.MinSeverity,
		r.p.CveIds, r.p.PackageRegex, r.p.ExcludePackages, r.p.ScheduleType,
		r.p.ScheduleCron, r.p.MwStart, r.p.MwEnd, r.p.DeploymentStrategy,
		r.p.DeletedAt, r.p.SeverityFilter, r.p.Mode, r.p.LastEvaluatedAt,
		r.p.LastEvalPass, r.p.LastEvalEndpointCount, r.p.LastEvalCompliantCount,
		r.p.DeploymentConfig, r.p.PolicyType, r.p.Timezone, r.p.MwEnabled,
	}
	for i := range dest {
		if i >= len(vals) {
			break
		}
		switch d := dest[i].(type) {
		case *pgtype.UUID:
			*d = vals[i].(pgtype.UUID) //nolint:errcheck
		case *pgtype.Text:
			*d = vals[i].(pgtype.Text) //nolint:errcheck
		case *pgtype.Timestamptz:
			*d = vals[i].(pgtype.Timestamptz) //nolint:errcheck
		case *pgtype.Time:
			*d = vals[i].(pgtype.Time) //nolint:errcheck
		case *pgtype.Bool:
			*d = vals[i].(pgtype.Bool) //nolint:errcheck
		case *pgtype.Int4:
			*d = vals[i].(pgtype.Int4) //nolint:errcheck
		case *string:
			*d = vals[i].(string) //nolint:errcheck
		case *bool:
			*d = vals[i].(bool) //nolint:errcheck
		case *[]string:
			*d = vals[i].([]string) //nolint:errcheck
		case *[]byte:
			*d = vals[i].([]byte) //nolint:errcheck
		}
	}
	return nil
}

func newTestHandler(q *fakePolicyQuerier, eb *fakeEventBus) *v1.PolicyHandler {
	ft := &fakeTx{
		q: &fakeGroupQuerier{},
		queryRowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
			// Distinguish CreatePolicy (INSERT ... RETURNING) from
			// UpdatePolicy (UPDATE ... RETURNING). The handler always runs
			// these inside a tx now, so the fake tx is the only path
			// through which store errors surface in unit tests.
			if strings.Contains(sql, "INSERT INTO policies") {
				if q.createErr != nil {
					return errRow{q.createErr}
				}
				return policyRow{p: q.createResult}
			}
			if q.updateErr != nil {
				return errRow{q.updateErr}
			}
			return policyRow{p: q.updateResult}
		},
	}
	return v1.NewPolicyHandler(q, &fakeTxBeginner{tx: ft}, eb, noopEvaluator{}, nil)
}

// --- Create Tests ---

func TestPolicyHandler_Create(t *testing.T) {
	eb := &fakeEventBus{}

	tests := []struct {
		name       string
		body       any
		querier    *fakePolicyQuerier
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "valid create returns 201",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "all_available",
				"mode":           "manual",
			},
			querier:    &fakePolicyQuerier{createResult: validPolicy()},
			wantStatus: http.StatusCreated,
			wantEvent:  true,
		},
		{
			name: "missing name returns 400",
			body: map[string]any{
				"selection_mode": "all_available",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid selection_mode returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "invalid_mode",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "by_severity missing min_severity returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "by_severity",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "by_severity invalid min_severity returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "by_severity",
				"min_severity":   "CRITICAL",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "by_cve_list empty cve_ids returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "by_cve_list",
				"cve_ids":        []string{},
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "by_regex invalid regex returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "by_regex",
				"package_regex":  "[invalid",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid schedule_type returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "all_available",
				"schedule_type":  "weekly",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid deployment_strategy returns 400",
			body: map[string]any{
				"name":                "My Policy",
				"selection_mode":      "all_available",
				"deployment_strategy": "canary",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid mw_start returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "all_available",
				"mw_start":       "25:99",
				"mw_end":         "17:00",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "mw_start without mw_end returns 400",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "all_available",
				"mw_start":       "09:00",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "store error returns 500",
			body: map[string]any{
				"name":           "My Policy",
				"selection_mode": "all_available",
				"mode":           "manual",
			},
			querier:    &fakePolicyQuerier{createErr: fmt.Errorf("database connection failed")},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := newTestHandler(tt.querier, eb)
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/policies", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "policy.created", eb.events[0].Type)
			}
		})
	}
}

// --- List Tests ---

func TestPolicyHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakePolicyQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakePolicyQuerier{
				listWithStatsResult: []sqlcgen.ListPoliciesWithStatsRow{},
				countFilteredResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "returns policies with count",
			querier: &fakePolicyQuerier{
				listWithStatsResult: []sqlcgen.ListPoliciesWithStatsRow{validPolicyWithStatsRow()},
				countFilteredResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "store error returns 500",
			querier: &fakePolicyQuerier{
				listWithStatsErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
		{
			name: "count error returns 500",
			querier: &fakePolicyQuerier{
				listWithStatsResult: []sqlcgen.ListPoliciesWithStatsRow{validPolicyWithStatsRow()},
				countFilteredErr:    fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(tt.querier, &fakeEventBus{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/policies"+tt.query, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
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

func TestPolicyHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakePolicyQuerier
		wantStatus int
	}{
		{
			name: "valid ID returns 200",
			id:   "00000000-0000-0000-0000-000000000077",
			querier: &fakePolicyQuerier{
				getResult: validPolicy(),
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000077",
			querier:    &fakePolicyQuerier{getErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(tt.querier, &fakeEventBus{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/policies/"+tt.id, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Update Tests ---

func TestPolicyHandler_Update(t *testing.T) {
	eb := &fakeEventBus{}
	updated := validPolicy()
	updated.Name = "Updated Policy"

	tests := []struct {
		name       string
		id         string
		body       any
		querier    *fakePolicyQuerier
		wantStatus int
	}{
		{
			name: "valid update returns 200",
			id:   "00000000-0000-0000-0000-000000000077",
			body: map[string]any{
				"name":           "Updated Policy",
				"selection_mode": "all_available",
				"mode":           "manual",
			},
			querier:    &fakePolicyQuerier{updateResult: updated},
			wantStatus: http.StatusOK,
		},
		{
			name: "missing name returns 400",
			id:   "00000000-0000-0000-0000-000000000077",
			body: map[string]any{
				"selection_mode": "all_available",
			},
			querier:    &fakePolicyQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			id:   "00000000-0000-0000-0000-000000000077",
			body: map[string]any{
				"name":           "Updated",
				"selection_mode": "all_available",
				"mode":           "manual",
			},
			querier:    &fakePolicyQuerier{updateErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "emits policy.updated event",
			id:   "00000000-0000-0000-0000-000000000077",
			body: map[string]any{
				"name":           "Updated Policy",
				"selection_mode": "all_available",
				"mode":           "manual",
			},
			querier:    &fakePolicyQuerier{updateResult: updated},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := newTestHandler(tt.querier, eb)
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/policies/"+tt.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Update(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "policy.updated", eb.events[0].Type)
			}
		})
	}
}

// --- Delete Tests ---

func TestPolicyHandler_Delete(t *testing.T) {
	eb := &fakeEventBus{}

	tests := []struct {
		name       string
		id         string
		querier    *fakePolicyQuerier
		wantStatus int
	}{
		{
			name:       "valid delete returns 204",
			id:         "00000000-0000-0000-0000-000000000077",
			querier:    &fakePolicyQuerier{softDeleteResult: validPolicy()},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000077",
			querier:    &fakePolicyQuerier{softDeleteErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "emits policy.deleted event",
			id:         "00000000-0000-0000-0000-000000000077",
			querier:    &fakePolicyQuerier{softDeleteResult: validPolicy()},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := newTestHandler(tt.querier, eb)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/policies/"+tt.id, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "policy.deleted", eb.events[0].Type)
			}
		})
	}
}

// --- BulkAction Tests ---

func TestPolicyHandler_BulkAction_Enable(t *testing.T) {
	eb := &fakeEventBus{}
	q := &fakePolicyQuerier{}
	h := newTestHandler(q, eb)

	body := map[string]any{
		"ids":    []string{"00000000-0000-0000-0000-000000000077"},
		"action": "enable",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/bulk", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
	rec := httptest.NewRecorder()

	h.BulkAction(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["affected"])
	assert.Len(t, eb.events, 1)
	assert.Equal(t, "policy.updated", eb.events[0].Type)
}

func TestPolicyHandler_BulkAction_InvalidAction(t *testing.T) {
	q := &fakePolicyQuerier{}
	h := newTestHandler(q, &fakeEventBus{})

	body := map[string]any{
		"ids":    []string{"00000000-0000-0000-0000-000000000077"},
		"action": "invalid",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/bulk", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
	rec := httptest.NewRecorder()

	h.BulkAction(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPolicyHandler_BulkAction_EmptyIDs(t *testing.T) {
	q := &fakePolicyQuerier{}
	h := newTestHandler(q, &fakeEventBus{})

	body := map[string]any{
		"ids":    []string{},
		"action": "enable",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/bulk", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
	rec := httptest.NewRecorder()

	h.BulkAction(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Evaluate Tests ---

type fakeEvaluator struct {
	results []policy.EvaluationResult
	err     error
}

func (f *fakeEvaluator) Evaluate(_ context.Context, _, _ string, _ time.Time) ([]policy.EvaluationResult, error) {
	return f.results, f.err
}

func TestPolicyHandler_Evaluate(t *testing.T) {
	policyID := "00000000-0000-0000-0000-000000000077"

	tests := []struct {
		name       string
		id         string
		evaluator  *fakeEvaluator
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "returns results 200",
			id:   policyID,
			evaluator: &fakeEvaluator{
				results: []policy.EvaluationResult{
					{
						EndpointID:   "ep-1",
						EndpointName: "host-1",
						Patches: []policy.PatchMatch{
							{PatchID: "p-1", Name: "patch-1", Version: "1.0", Severity: "high"},
						},
					},
				},
			},
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
		{
			name: "disabled policy returns 422",
			id:   policyID,
			evaluator: &fakeEvaluator{
				err: fmt.Errorf("evaluate policy %s: %w", policyID, policy.ErrPolicyDisabled),
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "policy not found returns 404",
			id:   policyID,
			evaluator: &fakeEvaluator{
				err: fmt.Errorf("evaluate policy %s: %w", policyID, policy.ErrPolicyNotFound),
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "outside maintenance window returns 422",
			id:   policyID,
			evaluator: &fakeEvaluator{
				err: fmt.Errorf("evaluate policy %s: %w", policyID, policy.ErrOutsideMaintenanceWindow),
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal error returns 500",
			id:   policyID,
			evaluator: &fakeEvaluator{
				err: fmt.Errorf("evaluate policy %s: list endpoints: connection refused", policyID),
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			evaluator:  &fakeEvaluator{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "emits policy.evaluated event with summary",
			id:   policyID,
			evaluator: &fakeEvaluator{
				results: []policy.EvaluationResult{
					{
						EndpointID:   "ep-1",
						EndpointName: "host-1",
						Patches: []policy.PatchMatch{
							{PatchID: "p-1", Name: "patch-1", Version: "1.0", Severity: "high"},
							{PatchID: "p-2", Name: "patch-2", Version: "2.0", Severity: "medium"},
						},
					},
				},
			},
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := v1.NewPolicyHandler(&fakePolicyQuerier{}, &fakeTxBeginner{tx: &fakeTx{q: &fakeGroupQuerier{}}}, eb, tt.evaluator, nil)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/policies/"+tt.id+"/evaluate", nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001"))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Evaluate(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				// Now emits both PolicyEvaluationRecorded and PolicyEvaluated events.
				require.GreaterOrEqual(t, len(eb.events), 1)
				// Find the policy.evaluated event.
				found := false
				for _, e := range eb.events {
					if e.Type == "policy.evaluated" {
						found = true
						break
					}
				}
				assert.True(t, found, "expected policy.evaluated event")
			}
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.id, body["policy_id"])
				assert.NotNil(t, body["results"])
				summary, ok := body["summary"].(map[string]any)
				require.True(t, ok, "summary should be a map")
				assert.Greater(t, summary["total_patches"], float64(0))
				assert.Greater(t, summary["endpoint_count"], float64(0))
			}
		})
	}
}
