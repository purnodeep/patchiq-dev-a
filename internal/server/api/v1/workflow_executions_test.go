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
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake WorkflowExecutionQuerier ---

type fakeExecQuerier struct {
	getWorkflowResult     sqlcgen.Workflow
	getWorkflowErr        error
	getPublishedResult    sqlcgen.WorkflowVersion
	getPublishedErr       error
	createExecResult      sqlcgen.WorkflowExecution
	createExecErr         error
	getExecResult         sqlcgen.WorkflowExecution
	getExecErr            error
	listExecsResult       []sqlcgen.WorkflowExecution
	listExecsErr          error
	countExecsResult      int64
	countExecsErr         error
	listNodeExecsResult   []sqlcgen.WorkflowNodeExecution
	listNodeExecsErr      error
	updateExecResult      sqlcgen.WorkflowExecution
	updateExecErr         error
	getPendingApproval    sqlcgen.ApprovalRequest
	getPendingApprovalErr error
	updateApprovalResult  sqlcgen.ApprovalRequest
	updateApprovalErr     error
}

func (f *fakeExecQuerier) GetWorkflowByID(_ context.Context, _ sqlcgen.GetWorkflowByIDParams) (sqlcgen.Workflow, error) {
	return f.getWorkflowResult, f.getWorkflowErr
}
func (f *fakeExecQuerier) GetPublishedVersion(_ context.Context, _ sqlcgen.GetPublishedVersionParams) (sqlcgen.WorkflowVersion, error) {
	return f.getPublishedResult, f.getPublishedErr
}
func (f *fakeExecQuerier) CreateWorkflowExecution(_ context.Context, _ sqlcgen.CreateWorkflowExecutionParams) (sqlcgen.WorkflowExecution, error) {
	return f.createExecResult, f.createExecErr
}
func (f *fakeExecQuerier) GetWorkflowExecution(_ context.Context, _ sqlcgen.GetWorkflowExecutionParams) (sqlcgen.WorkflowExecution, error) {
	return f.getExecResult, f.getExecErr
}
func (f *fakeExecQuerier) ListWorkflowExecutions(_ context.Context, _ sqlcgen.ListWorkflowExecutionsParams) ([]sqlcgen.WorkflowExecution, error) {
	return f.listExecsResult, f.listExecsErr
}
func (f *fakeExecQuerier) CountWorkflowExecutions(_ context.Context, _ sqlcgen.CountWorkflowExecutionsParams) (int64, error) {
	return f.countExecsResult, f.countExecsErr
}
func (f *fakeExecQuerier) ListNodeExecutions(_ context.Context, _ sqlcgen.ListNodeExecutionsParams) ([]sqlcgen.WorkflowNodeExecution, error) {
	return f.listNodeExecsResult, f.listNodeExecsErr
}
func (f *fakeExecQuerier) UpdateWorkflowExecutionStatus(_ context.Context, _ sqlcgen.UpdateWorkflowExecutionStatusParams) (sqlcgen.WorkflowExecution, error) {
	return f.updateExecResult, f.updateExecErr
}
func (f *fakeExecQuerier) GetPendingApprovalByExecution(_ context.Context, _ sqlcgen.GetPendingApprovalByExecutionParams) (sqlcgen.ApprovalRequest, error) {
	return f.getPendingApproval, f.getPendingApprovalErr
}
func (f *fakeExecQuerier) UpdateApprovalRequest(_ context.Context, _ sqlcgen.UpdateApprovalRequestParams) (sqlcgen.ApprovalRequest, error) {
	return f.updateApprovalResult, f.updateApprovalErr
}

// --- Test Helpers ---

func validExecUUIDs() (workflowID, versionID, execID, tid pgtype.UUID) {
	_ = workflowID.Scan("00000000-0000-0000-0000-000000000088")
	_ = versionID.Scan("00000000-0000-0000-0000-000000000099")
	_ = execID.Scan("00000000-0000-0000-0000-0000000000e1")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return
}

func validExecution() sqlcgen.WorkflowExecution {
	wid, vid, eid, tid := validExecUUIDs()
	return sqlcgen.WorkflowExecution{
		ID:          eid,
		TenantID:    tid,
		WorkflowID:  wid,
		VersionID:   vid,
		Status:      "pending",
		TriggeredBy: "manual",
	}
}

func newExecHandler(q *fakeExecQuerier) *v1.WorkflowExecutionHandler {
	return v1.NewWorkflowExecutionHandler(q, &fakeTxBeginner{tx: &fakeTx{q: &fakeGroupQuerier{}}}, &fakeEventBus{}, nil)
}

func newExecHandlerWithBus(q *fakeExecQuerier, eb *fakeEventBus) *v1.WorkflowExecutionHandler {
	return v1.NewWorkflowExecutionHandler(q, &fakeTxBeginner{tx: &fakeTx{q: &fakeGroupQuerier{}}}, eb, nil)
}

func execRequest(method, path string, body any, workflowID, execID string) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body) //nolint:errcheck
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", workflowID)
	if execID != "" {
		rctx.URLParams.Add("execId", execID)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = tenant.WithTenantID(ctx, testTenantID)
	ctx = user.WithUserID(ctx, "user-123")
	req = req.WithContext(ctx)
	return req
}

// --- Execute Tests ---

func TestWorkflowExecutionHandler_Execute(t *testing.T) {
	wid, vid, eid, tid := validExecUUIDs()

	tests := []struct {
		name       string
		querier    *fakeExecQuerier
		workflowID string
		wantStatus int
		wantEvent  bool
	}{
		{
			name:       "invalid workflow ID returns 400",
			querier:    &fakeExecQuerier{},
			workflowID: "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "workflow not found returns 404",
			querier: &fakeExecQuerier{
				getWorkflowErr: pgx.ErrNoRows,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "no published version returns 400",
			querier: &fakeExecQuerier{
				getWorkflowResult: sqlcgen.Workflow{ID: wid, TenantID: tid, Name: "test"},
				getPublishedErr:   pgx.ErrNoRows,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "create execution error returns 500",
			querier: &fakeExecQuerier{
				getWorkflowResult: sqlcgen.Workflow{ID: wid, TenantID: tid, Name: "test"},
				getPublishedResult: sqlcgen.WorkflowVersion{
					ID: vid, TenantID: tid, WorkflowID: wid, Version: 1, Status: "published",
				},
				createExecErr: fmt.Errorf("db error"),
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "successful execution returns 201",
			querier: &fakeExecQuerier{
				getWorkflowResult: sqlcgen.Workflow{ID: wid, TenantID: tid, Name: "test"},
				getPublishedResult: sqlcgen.WorkflowVersion{
					ID: vid, TenantID: tid, WorkflowID: wid, Version: 1, Status: "published",
				},
				createExecResult: sqlcgen.WorkflowExecution{
					ID: eid, TenantID: tid, WorkflowID: wid, VersionID: vid,
					Status: "pending", TriggeredBy: "manual",
				},
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusCreated,
			wantEvent:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newExecHandlerWithBus(tt.querier, eb)
			req := execRequest(http.MethodPost, "/api/v1/workflows/"+tt.workflowID+"/execute", nil, tt.workflowID, "")
			rec := httptest.NewRecorder()

			h.Execute(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				assert.Len(t, eb.events, 1)
			}
			if tt.wantStatus == http.StatusCreated {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotEmpty(t, body["id"])
				assert.Equal(t, "pending", body["status"])
			}
		})
	}
}

// --- List Executions Tests ---

func TestWorkflowExecutionHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeExecQuerier
		workflowID string
		wantStatus int
	}{
		{
			name:       "invalid workflow ID returns 400",
			querier:    &fakeExecQuerier{},
			workflowID: "bad-id",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty list returns 200",
			querier: &fakeExecQuerier{
				listExecsResult:  []sqlcgen.WorkflowExecution{},
				countExecsResult: 0,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusOK,
		},
		{
			name: "list error returns 500",
			querier: &fakeExecQuerier{
				listExecsErr: fmt.Errorf("db error"),
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "returns executions",
			querier: &fakeExecQuerier{
				listExecsResult:  []sqlcgen.WorkflowExecution{validExecution()},
				countExecsResult: 1,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newExecHandler(tt.querier)
			req := execRequest(http.MethodGet, "/api/v1/workflows/"+tt.workflowID+"/executions", nil, tt.workflowID, "")
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Get Execution Tests ---

func TestWorkflowExecutionHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeExecQuerier
		workflowID string
		execID     string
		wantStatus int
	}{
		{
			name:       "invalid workflow ID returns 400",
			querier:    &fakeExecQuerier{},
			workflowID: "bad-id",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid exec ID returns 400",
			querier:    &fakeExecQuerier{},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "bad-id",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "execution not found returns 404",
			querier: &fakeExecQuerier{
				getExecErr: pgx.ErrNoRows,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "successful get returns 200",
			querier: &fakeExecQuerier{
				getExecResult:       validExecution(),
				listNodeExecsResult: []sqlcgen.WorkflowNodeExecution{},
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newExecHandler(tt.querier)
			req := execRequest(http.MethodGet, "/", nil, tt.workflowID, tt.execID)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Cancel Tests ---

func TestWorkflowExecutionHandler_Cancel(t *testing.T) {
	runningExec := validExecution()
	runningExec.Status = "running"

	cancelledExec := runningExec
	cancelledExec.Status = "cancelled"

	completedExec := validExecution()
	completedExec.Status = "completed"

	tests := []struct {
		name       string
		querier    *fakeExecQuerier
		workflowID string
		execID     string
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "cancel running execution returns 200",
			querier: &fakeExecQuerier{
				getExecResult:    runningExec,
				updateExecResult: cancelledExec,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
		{
			name: "cannot cancel completed execution",
			querier: &fakeExecQuerier{
				getExecResult: completedExec,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusConflict,
		},
		{
			name: "execution not found returns 404",
			querier: &fakeExecQuerier{
				getExecErr: pgx.ErrNoRows,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newExecHandlerWithBus(tt.querier, eb)
			req := execRequest(http.MethodPost, "/", nil, tt.workflowID, tt.execID)
			rec := httptest.NewRecorder()

			h.Cancel(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				assert.Len(t, eb.events, 1)
			}
		})
	}
}

// --- Approve Tests ---

func TestWorkflowExecutionHandler_Approve(t *testing.T) {
	pausedExec := validExecution()
	pausedExec.Status = "paused"

	runningExec := pausedExec
	runningExec.Status = "running"

	_, _, eid, tid := validExecUUIDs()

	tests := []struct {
		name       string
		querier    *fakeExecQuerier
		body       any
		workflowID string
		execID     string
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "approve paused execution returns 200",
			querier: &fakeExecQuerier{
				getExecResult: pausedExec,
				getPendingApproval: sqlcgen.ApprovalRequest{
					ID: eid, TenantID: tid, ExecutionID: eid, Status: "pending",
				},
				updateApprovalResult: sqlcgen.ApprovalRequest{Status: "approved"},
				updateExecResult:     runningExec,
			},
			body:       map[string]any{"comment": "looks good"},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
		{
			name: "execution not paused returns 409",
			querier: &fakeExecQuerier{
				getExecResult: validExecution(), // status = "pending", not "paused"
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusConflict,
		},
		{
			name: "no pending approval returns 404",
			querier: &fakeExecQuerier{
				getExecResult:         pausedExec,
				getPendingApprovalErr: pgx.ErrNoRows,
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newExecHandlerWithBus(tt.querier, eb)
			req := execRequest(http.MethodPost, "/", tt.body, tt.workflowID, tt.execID)
			rec := httptest.NewRecorder()

			h.Approve(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				assert.Len(t, eb.events, 1)
			}
		})
	}
}

// --- Reject Tests ---

func TestWorkflowExecutionHandler_Reject(t *testing.T) {
	pausedExec := validExecution()
	pausedExec.Status = "paused"

	failedExec := pausedExec
	failedExec.Status = "failed"

	_, _, eid, tid := validExecUUIDs()

	tests := []struct {
		name       string
		querier    *fakeExecQuerier
		body       any
		workflowID string
		execID     string
		wantStatus int
		wantEvent  bool
	}{
		{
			name: "reject paused execution returns 200",
			querier: &fakeExecQuerier{
				getExecResult: pausedExec,
				getPendingApproval: sqlcgen.ApprovalRequest{
					ID: eid, TenantID: tid, ExecutionID: eid, Status: "pending",
				},
				updateApprovalResult: sqlcgen.ApprovalRequest{Status: "rejected"},
				updateExecResult:     failedExec,
			},
			body:       map[string]any{"comment": "not ready"},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusOK,
			wantEvent:  true,
		},
		{
			name: "execution not paused returns 409",
			querier: &fakeExecQuerier{
				getExecResult: validExecution(), // status = "pending"
			},
			workflowID: "00000000-0000-0000-0000-000000000088",
			execID:     "00000000-0000-0000-0000-0000000000e1",
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newExecHandlerWithBus(tt.querier, eb)
			req := execRequest(http.MethodPost, "/", tt.body, tt.workflowID, tt.execID)
			rec := httptest.NewRecorder()

			h.Reject(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				assert.Len(t, eb.events, 1)
			}
		})
	}
}

// --- Constructor Tests ---

func TestNewWorkflowExecutionHandler_PanicsOnNil(t *testing.T) {
	q := &fakeExecQuerier{}
	txb := &fakeTxBeginner{tx: &fakeTx{q: &fakeGroupQuerier{}}}
	eb := &fakeEventBus{}

	assert.Panics(t, func() { v1.NewWorkflowExecutionHandler(nil, txb, eb, nil) })
	assert.Panics(t, func() { v1.NewWorkflowExecutionHandler(q, nil, eb, nil) })
	assert.Panics(t, func() { v1.NewWorkflowExecutionHandler(q, txb, nil, nil) })
	assert.NotPanics(t, func() { v1.NewWorkflowExecutionHandler(q, txb, eb, nil) })
}
