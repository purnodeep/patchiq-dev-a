package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// fakeWorkflowQuerier mocks WorkflowQuerier for unit tests.
type fakeWorkflowQuerier struct {
	// Workflow CRUD
	createResult     sqlcgen.Workflow
	createErr        error
	getResult        sqlcgen.Workflow
	getErr           error
	listResult       []sqlcgen.ListWorkflowsRow
	listErr          error
	countResult      int64
	countErr         error
	updateResult     sqlcgen.Workflow
	updateErr        error
	softDeleteResult sqlcgen.Workflow
	softDeleteErr    error

	// Versions
	createVersionResult sqlcgen.WorkflowVersion
	createVersionErr    error
	latestVersionResult sqlcgen.WorkflowVersion
	latestVersionErr    error
	listVersionsResult  []sqlcgen.WorkflowVersion
	listVersionsErr     error
	maxVersionResult    int32
	maxVersionErr       error
	archiveVersionErr   error
	publishResult       sqlcgen.WorkflowVersion
	publishErr          error
	publishedResult     sqlcgen.WorkflowVersion
	publishedErr        error
	draftResult         sqlcgen.WorkflowVersion
	draftErr            error

	// Nodes and Edges
	createNodeResult sqlcgen.WorkflowNode
	createNodeErr    error
	listNodesResult  []sqlcgen.WorkflowNode
	listNodesErr     error
	createEdgeResult sqlcgen.WorkflowEdge
	createEdgeErr    error
	listEdgesResult  []sqlcgen.WorkflowEdge
	listEdgesErr     error
}

func (f *fakeWorkflowQuerier) CreateWorkflow(_ context.Context, _ sqlcgen.CreateWorkflowParams) (sqlcgen.Workflow, error) {
	return f.createResult, f.createErr
}
func (f *fakeWorkflowQuerier) GetWorkflowByID(_ context.Context, _ sqlcgen.GetWorkflowByIDParams) (sqlcgen.Workflow, error) {
	return f.getResult, f.getErr
}
func (f *fakeWorkflowQuerier) ListWorkflows(_ context.Context, _ sqlcgen.ListWorkflowsParams) ([]sqlcgen.ListWorkflowsRow, error) {
	return f.listResult, f.listErr
}
func (f *fakeWorkflowQuerier) CountWorkflows(_ context.Context, _ sqlcgen.CountWorkflowsParams) (int64, error) {
	return f.countResult, f.countErr
}
func (f *fakeWorkflowQuerier) UpdateWorkflow(_ context.Context, _ sqlcgen.UpdateWorkflowParams) (sqlcgen.Workflow, error) {
	return f.updateResult, f.updateErr
}
func (f *fakeWorkflowQuerier) SoftDeleteWorkflow(_ context.Context, _ sqlcgen.SoftDeleteWorkflowParams) (sqlcgen.Workflow, error) {
	return f.softDeleteResult, f.softDeleteErr
}
func (f *fakeWorkflowQuerier) CreateWorkflowVersion(_ context.Context, _ sqlcgen.CreateWorkflowVersionParams) (sqlcgen.WorkflowVersion, error) {
	return f.createVersionResult, f.createVersionErr
}
func (f *fakeWorkflowQuerier) GetLatestVersion(_ context.Context, _ sqlcgen.GetLatestVersionParams) (sqlcgen.WorkflowVersion, error) {
	return f.latestVersionResult, f.latestVersionErr
}
func (f *fakeWorkflowQuerier) ListWorkflowVersions(_ context.Context, _ sqlcgen.ListWorkflowVersionsParams) ([]sqlcgen.WorkflowVersion, error) {
	return f.listVersionsResult, f.listVersionsErr
}
func (f *fakeWorkflowQuerier) GetMaxVersionNumber(_ context.Context, _ sqlcgen.GetMaxVersionNumberParams) (int32, error) {
	return f.maxVersionResult, f.maxVersionErr
}
func (f *fakeWorkflowQuerier) ArchiveWorkflowVersion(_ context.Context, _ sqlcgen.ArchiveWorkflowVersionParams) error {
	return f.archiveVersionErr
}
func (f *fakeWorkflowQuerier) PublishWorkflowVersion(_ context.Context, _ sqlcgen.PublishWorkflowVersionParams) (sqlcgen.WorkflowVersion, error) {
	return f.publishResult, f.publishErr
}
func (f *fakeWorkflowQuerier) GetPublishedVersion(_ context.Context, _ sqlcgen.GetPublishedVersionParams) (sqlcgen.WorkflowVersion, error) {
	return f.publishedResult, f.publishedErr
}
func (f *fakeWorkflowQuerier) GetDraftVersion(_ context.Context, _ sqlcgen.GetDraftVersionParams) (sqlcgen.WorkflowVersion, error) {
	return f.draftResult, f.draftErr
}
func (f *fakeWorkflowQuerier) CreateWorkflowNode(_ context.Context, _ sqlcgen.CreateWorkflowNodeParams) (sqlcgen.WorkflowNode, error) {
	return f.createNodeResult, f.createNodeErr
}
func (f *fakeWorkflowQuerier) ListWorkflowNodes(_ context.Context, _ sqlcgen.ListWorkflowNodesParams) ([]sqlcgen.WorkflowNode, error) {
	return f.listNodesResult, f.listNodesErr
}
func (f *fakeWorkflowQuerier) CreateWorkflowEdge(_ context.Context, _ sqlcgen.CreateWorkflowEdgeParams) (sqlcgen.WorkflowEdge, error) {
	return f.createEdgeResult, f.createEdgeErr
}
func (f *fakeWorkflowQuerier) ListWorkflowEdges(_ context.Context, _ sqlcgen.ListWorkflowEdgesParams) ([]sqlcgen.WorkflowEdge, error) {
	return f.listEdgesResult, f.listEdgesErr
}

func (f *fakeWorkflowQuerier) CountWorkflowsByStatus(_ context.Context, _ pgtype.UUID) ([]sqlcgen.CountWorkflowsByStatusRow, error) {
	return nil, nil
}

func validWorkflow() sqlcgen.Workflow {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000088")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	return sqlcgen.Workflow{
		ID:       id,
		TenantID: tid,
		Name:     "test-workflow",
	}
}

func validWorkflowRow() sqlcgen.ListWorkflowsRow {
	wf := validWorkflow()
	return sqlcgen.ListWorkflowsRow{
		ID:             wf.ID,
		TenantID:       wf.TenantID,
		Name:           wf.Name,
		Description:    wf.Description,
		CreatedAt:      wf.CreatedAt,
		UpdatedAt:      wf.UpdatedAt,
		DeletedAt:      wf.DeletedAt,
		CurrentVersion: 1,
		CurrentStatus:  "draft",
		NodeCount:      2,
	}
}

func validVersion() sqlcgen.WorkflowVersion {
	var id, tid, wid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000099")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = wid.Scan("00000000-0000-0000-0000-000000000088")
	return sqlcgen.WorkflowVersion{
		ID:         id,
		TenantID:   tid,
		WorkflowID: wid,
		Version:    1,
		Status:     "draft",
	}
}

func validNode() sqlcgen.WorkflowNode {
	var id, tid, vid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-0000000000a1")
	_ = tid.Scan("00000000-0000-0000-0000-000000000001")
	_ = vid.Scan("00000000-0000-0000-0000-000000000099")
	return sqlcgen.WorkflowNode{
		ID:        id,
		TenantID:  tid,
		VersionID: vid,
		NodeType:  "trigger",
		Label:     "Start",
		PositionX: 0,
		PositionY: 100,
		Config:    []byte(`{"trigger_type":"manual"}`),
	}
}

func newWorkflowHandler(q *fakeWorkflowQuerier) *v1.WorkflowHandler {
	h := v1.NewWorkflowHandler(q, &fakeTxBeginner{tx: &fakeTx{q: &fakeGroupQuerier{}}}, &fakeEventBus{})
	h.SetQuerierFactory(func(_ sqlcgen.DBTX) v1.WorkflowQuerier { return q })
	return h
}

func newWorkflowHandlerWithBus(q *fakeWorkflowQuerier, eb *fakeEventBus) *v1.WorkflowHandler {
	h := v1.NewWorkflowHandler(q, &fakeTxBeginner{tx: &fakeTx{q: &fakeGroupQuerier{}}}, eb)
	h.SetQuerierFactory(func(_ sqlcgen.DBTX) v1.WorkflowQuerier { return q })
	return h
}

// --- List Tests ---

func TestWorkflowHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeWorkflowQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakeWorkflowQuerier{
				listResult:  []sqlcgen.ListWorkflowsRow{},
				countResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "returns workflows with count",
			querier: &fakeWorkflowQuerier{
				listResult:  []sqlcgen.ListWorkflowsRow{validWorkflowRow()},
				countResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "store error returns 500",
			querier: &fakeWorkflowQuerier{
				listErr: fmt.Errorf("database connection failed"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newWorkflowHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
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

// --- Create Tests ---

func TestWorkflowHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		wantStatus int
		wantEvent  bool
	}{
		{
			name:       "missing name returns 400",
			body:       map[string]any{"description": "no name"},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty name returns 400",
			body:       map[string]any{"name": ""},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no nodes returns 400",
			body:       map[string]any{"name": "Test", "nodes": []any{}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON returns 400",
			body:       "not-json{{{",
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate node IDs returns 400",
			body: map[string]any{
				"name": "Test",
				"nodes": []any{
					map[string]any{"id": "n1", "node_type": "trigger"},
					map[string]any{"id": "n1", "node_type": "filter"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty node ID returns 400",
			body: map[string]any{
				"name": "Test",
				"nodes": []any{
					map[string]any{"id": "", "node_type": "trigger"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid node_type returns 400",
			body: map[string]any{
				"name": "Test",
				"nodes": []any{
					map[string]any{"id": "n1", "node_type": "banana"},
				},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newWorkflowHandlerWithBus(&fakeWorkflowQuerier{}, eb)
			var bodyBytes []byte
			if s, ok := tt.body.(string); ok {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantEvent {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "workflow.created", eb.events[0].Type)
			}
		})
	}
}

// --- Get Tests ---

func TestWorkflowHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeWorkflowQuerier
		wantStatus int
	}{
		{
			name: "valid ID returns 200 with nodes and edges",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				getResult:           validWorkflow(),
				latestVersionResult: validVersion(),
				listNodesResult:     []sqlcgen.WorkflowNode{validNode()},
				listEdgesResult:     []sqlcgen.WorkflowEdge{},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeWorkflowQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				getErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "no versions returns 200 with empty nodes/edges",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				getResult:        validWorkflow(),
				latestVersionErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newWorkflowHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.id, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotNil(t, body["nodes"])
				assert.NotNil(t, body["edges"])
			}
		})
	}
}

// --- Update Tests ---

func TestWorkflowHandler_Update(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       any
		querier    *fakeWorkflowQuerier
		wantStatus int
	}{
		{
			name:       "missing name returns 400",
			id:         "00000000-0000-0000-0000-000000000088",
			body:       map[string]any{"description": "no name", "nodes": []any{map[string]any{"id": "n1", "node_type": "trigger"}}},
			querier:    &fakeWorkflowQuerier{getResult: validWorkflow()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "no nodes returns 400",
			id:         "00000000-0000-0000-0000-000000000088",
			body:       map[string]any{"name": "Updated", "nodes": []any{}},
			querier:    &fakeWorkflowQuerier{getResult: validWorkflow()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{"name": "Updated", "nodes": []any{map[string]any{"id": "n1", "node_type": "trigger"}}},
			querier: &fakeWorkflowQuerier{
				getErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			body:       map[string]any{"name": "Updated", "nodes": []any{map[string]any{"id": "n1", "node_type": "trigger"}}},
			querier:    &fakeWorkflowQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "duplicate node IDs returns 400",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{
				"name": "Updated",
				"nodes": []any{
					map[string]any{"id": "n1", "node_type": "trigger"},
					map[string]any{"id": "n1", "node_type": "filter"},
				},
			},
			querier:    &fakeWorkflowQuerier{getResult: validWorkflow()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "empty node ID returns 400",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{
				"name": "Updated",
				"nodes": []any{
					map[string]any{"id": "", "node_type": "trigger"},
				},
			},
			querier:    &fakeWorkflowQuerier{getResult: validWorkflow()},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid node_type returns 400",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{
				"name": "Updated",
				"nodes": []any{
					map[string]any{"id": "n1", "node_type": "unknown"},
				},
			},
			querier:    &fakeWorkflowQuerier{getResult: validWorkflow()},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newWorkflowHandlerWithBus(tt.querier, eb)
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/"+tt.id, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Update(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- Delete Tests ---

func TestWorkflowHandler_Delete(t *testing.T) {
	eb := &fakeEventBus{}

	tests := []struct {
		name       string
		id         string
		querier    *fakeWorkflowQuerier
		wantStatus int
	}{
		{
			name:       "valid delete returns 204",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeWorkflowQuerier{softDeleteResult: validWorkflow()},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "not found returns 404",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				softDeleteErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeWorkflowQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb.events = nil
			h := newWorkflowHandlerWithBus(tt.querier, eb)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workflows/"+tt.id, nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Len(t, eb.events, 1)
				assert.Equal(t, "workflow.deleted", eb.events[0].Type)
			}
		})
	}
}

// --- Publish Tests ---

func TestWorkflowHandler_Publish(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeWorkflowQuerier
		wantStatus int
		wantCode   string
	}{
		{
			name: "no draft returns 404",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				draftErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeWorkflowQuerier{},
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_ID",
		},
		{
			name: "validation failure returns 400 with violations",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				draftResult: validVersion(),
				// No nodes = missing trigger + missing complete
				listNodesResult: []sqlcgen.WorkflowNode{},
				listEdgesResult: []sqlcgen.WorkflowEdge{},
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "VALIDATION_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &fakeEventBus{}
			h := newWorkflowHandlerWithBus(tt.querier, eb)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/"+tt.id+"/publish", nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Publish(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			var body map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantCode, body["code"])
		})
	}
}

// --- ListVersions Tests ---

func TestWorkflowHandler_ListVersions(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeWorkflowQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns version list",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				listVersionsResult: []sqlcgen.WorkflowVersion{validVersion()},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "empty list",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				listVersionsResult: []sqlcgen.WorkflowVersion{},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "store error returns 500",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeWorkflowQuerier{
				listVersionsErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newWorkflowHandler(tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows/"+tt.id+"/versions", nil)
			req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.ListVersions(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body []any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body, tt.wantLen)
			}
		})
	}
}

// --- ListTemplates Tests ---

func TestWorkflowHandler_ListTemplates(t *testing.T) {
	h := newWorkflowHandler(&fakeWorkflowQuerier{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflow-templates", nil)
	rec := httptest.NewRecorder()

	h.ListTemplates(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body []any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body, 3, "expected 3 templates")
}

// --- Happy-Path Tests ---

func TestWorkflowHandler_Create_HappyPath(t *testing.T) {
	eb := &fakeEventBus{}
	q := &fakeWorkflowQuerier{
		createResult:        validWorkflow(),
		createVersionResult: validVersion(),
		createNodeResult:    validNode(),
		createEdgeResult:    sqlcgen.WorkflowEdge{},
	}
	h := newWorkflowHandlerWithBus(q, eb)

	body := map[string]any{
		"name":        "My Workflow",
		"description": "A test workflow",
		"nodes": []any{
			map[string]any{"id": "n1", "node_type": "trigger", "label": "Start", "config": map[string]any{"trigger_type": "manual"}},
			map[string]any{"id": "n2", "node_type": "complete", "label": "End", "config": map[string]any{}},
		},
		"edges": []any{
			map[string]any{"source_node_id": "n1", "target_node_id": "n2"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
	assert.NotEmpty(t, resp["version_id"])
	assert.Equal(t, float64(1), resp["version"])
	assert.Equal(t, "draft", resp["status"])
	assert.Len(t, eb.events, 1)
	assert.Equal(t, "workflow.created", eb.events[0].Type)
}

func TestWorkflowHandler_Update_HappyPath(t *testing.T) {
	eb := &fakeEventBus{}
	q := &fakeWorkflowQuerier{
		getResult:           validWorkflow(),
		maxVersionResult:    1,
		draftResult:         validVersion(),
		createVersionResult: validVersion(),
		createNodeResult:    validNode(),
		createEdgeResult:    sqlcgen.WorkflowEdge{},
		updateResult:        validWorkflow(),
	}
	h := newWorkflowHandlerWithBus(q, eb)

	body := map[string]any{
		"name": "Updated Workflow",
		"nodes": []any{
			map[string]any{"id": "n1", "node_type": "trigger", "label": "Start"},
			map[string]any{"id": "n2", "node_type": "complete", "label": "End"},
		},
		"edges": []any{
			map[string]any{"source_node_id": "n1", "target_node_id": "n2"},
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/00000000-0000-0000-0000-000000000088", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000088")
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["id"])
	assert.Equal(t, "draft", resp["status"])
	assert.Len(t, eb.events, 1)
	assert.Equal(t, "workflow.updated", eb.events[0].Type)
}

func TestWorkflowHandler_Publish_HappyPath(t *testing.T) {
	eb := &fakeEventBus{}
	triggerNode := validNode()
	completeNode := validNode()
	_ = completeNode.ID.Scan("00000000-0000-0000-0000-0000000000a2")
	completeNode.NodeType = "complete"
	completeNode.Config = []byte(`{}`)

	var srcID, tgtID pgtype.UUID
	_ = srcID.Scan("00000000-0000-0000-0000-0000000000a1")
	_ = tgtID.Scan("00000000-0000-0000-0000-0000000000a2")

	publishedVersion := validVersion()
	publishedVersion.Status = "published"

	q := &fakeWorkflowQuerier{
		draftResult:     validVersion(),
		listNodesResult: []sqlcgen.WorkflowNode{triggerNode, completeNode},
		listEdgesResult: []sqlcgen.WorkflowEdge{
			{SourceNodeID: srcID, TargetNodeID: tgtID},
		},
		publishedErr:  pgx.ErrNoRows,
		publishResult: publishedVersion,
	}
	h := newWorkflowHandlerWithBus(q, eb)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/workflows/00000000-0000-0000-0000-000000000088/publish", nil)
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
	req = chiCtx(req, "id", "00000000-0000-0000-0000-000000000088")
	rec := httptest.NewRecorder()

	h.Publish(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, eb.events, 1)
	assert.Equal(t, "workflow.published", eb.events[0].Type)
}

// --- Additional Error Path Tests ---

func TestWorkflowHandler_List_CountError(t *testing.T) {
	q := &fakeWorkflowQuerier{
		listResult: []sqlcgen.ListWorkflowsRow{validWorkflowRow()},
		countErr:   fmt.Errorf("count query failed"),
	}
	h := newWorkflowHandler(q)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/workflows", nil)
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
	rec := httptest.NewRecorder()

	h.List(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"100%", `100\%`},
		{"under_score", `under\_score`},
		{`back\slash`, `back\\slash`},
		{`100%_done\path`, `100\%\_done\\path`},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := v1.EscapeLikePattern(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
