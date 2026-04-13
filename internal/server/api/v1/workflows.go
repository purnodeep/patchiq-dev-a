package v1

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/workflow"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// WorkflowQuerier defines the sqlc queries needed by WorkflowHandler.
type WorkflowQuerier interface {
	CreateWorkflow(ctx context.Context, arg sqlcgen.CreateWorkflowParams) (sqlcgen.Workflow, error)
	GetWorkflowByID(ctx context.Context, arg sqlcgen.GetWorkflowByIDParams) (sqlcgen.Workflow, error)
	ListWorkflows(ctx context.Context, arg sqlcgen.ListWorkflowsParams) ([]sqlcgen.ListWorkflowsRow, error)
	CountWorkflows(ctx context.Context, arg sqlcgen.CountWorkflowsParams) (int64, error)
	CountWorkflowsByStatus(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.CountWorkflowsByStatusRow, error)
	UpdateWorkflow(ctx context.Context, arg sqlcgen.UpdateWorkflowParams) (sqlcgen.Workflow, error)
	SoftDeleteWorkflow(ctx context.Context, arg sqlcgen.SoftDeleteWorkflowParams) (sqlcgen.Workflow, error)
	CreateWorkflowVersion(ctx context.Context, arg sqlcgen.CreateWorkflowVersionParams) (sqlcgen.WorkflowVersion, error)
	GetLatestVersion(ctx context.Context, arg sqlcgen.GetLatestVersionParams) (sqlcgen.WorkflowVersion, error)
	ListWorkflowVersions(ctx context.Context, arg sqlcgen.ListWorkflowVersionsParams) ([]sqlcgen.WorkflowVersion, error)
	GetMaxVersionNumber(ctx context.Context, arg sqlcgen.GetMaxVersionNumberParams) (int32, error)
	ArchiveWorkflowVersion(ctx context.Context, arg sqlcgen.ArchiveWorkflowVersionParams) error
	PublishWorkflowVersion(ctx context.Context, arg sqlcgen.PublishWorkflowVersionParams) (sqlcgen.WorkflowVersion, error)
	GetPublishedVersion(ctx context.Context, arg sqlcgen.GetPublishedVersionParams) (sqlcgen.WorkflowVersion, error)
	GetDraftVersion(ctx context.Context, arg sqlcgen.GetDraftVersionParams) (sqlcgen.WorkflowVersion, error)
	CreateWorkflowNode(ctx context.Context, arg sqlcgen.CreateWorkflowNodeParams) (sqlcgen.WorkflowNode, error)
	ListWorkflowNodes(ctx context.Context, arg sqlcgen.ListWorkflowNodesParams) ([]sqlcgen.WorkflowNode, error)
	CreateWorkflowEdge(ctx context.Context, arg sqlcgen.CreateWorkflowEdgeParams) (sqlcgen.WorkflowEdge, error)
	ListWorkflowEdges(ctx context.Context, arg sqlcgen.ListWorkflowEdgesParams) ([]sqlcgen.WorkflowEdge, error)
}

// QuerierFactory creates a WorkflowQuerier from a DBTX (transaction or connection).
type QuerierFactory func(sqlcgen.DBTX) WorkflowQuerier

// WorkflowHandler serves workflow REST API endpoints.
type WorkflowHandler struct {
	q          WorkflowQuerier
	txb        TxBeginner
	eventBus   domain.EventBus
	newQuerier QuerierFactory
}

// NewWorkflowHandler creates a WorkflowHandler.
func NewWorkflowHandler(q WorkflowQuerier, txb TxBeginner, eventBus domain.EventBus) *WorkflowHandler {
	if q == nil {
		panic("workflows: NewWorkflowHandler called with nil querier")
	}
	if txb == nil {
		panic("workflows: NewWorkflowHandler called with nil txBeginner")
	}
	if eventBus == nil {
		panic("workflows: NewWorkflowHandler called with nil eventBus")
	}
	return &WorkflowHandler{
		q:        q,
		txb:      txb,
		eventBus: eventBus,
		newQuerier: func(db sqlcgen.DBTX) WorkflowQuerier {
			return sqlcgen.New(db)
		},
	}
}

// SetQuerierFactory overrides the default sqlcgen.New querier factory (for testing).
func (h *WorkflowHandler) SetQuerierFactory(f QuerierFactory) {
	h.newQuerier = f
}

// maxRequestBodySize limits the request body to 1MB to prevent OOM from large payloads.
const maxRequestBodySize = 1 << 20 // 1MB

// List handles GET /api/v1/workflows with pagination and search.
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	cursorTime, cursorID, err := DecodeCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor")
		return
	}
	limit := ParseLimit(r.URL.Query().Get("limit"))

	tid, err := scanUUID(tenantID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var cursorTS pgtype.Timestamptz
	var cursorUUID pgtype.UUID
	if !cursorTime.IsZero() {
		cursorTS = pgtype.Timestamptz{Time: cursorTime, Valid: true}
		cursorUUID, err = scanUUID(cursorID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor: cursor ID is not a valid UUID")
			return
		}
	}

	search := EscapeLikePattern(r.URL.Query().Get("search"))
	statusFilter := r.URL.Query().Get("status")

	workflows, err := h.q.ListWorkflows(ctx, sqlcgen.ListWorkflowsParams{
		TenantID:        tid,
		Search:          search,
		StatusFilter:    statusFilter,
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageLimit:       limit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list workflows", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workflows")
		return
	}

	total, err := h.q.CountWorkflows(ctx, sqlcgen.CountWorkflowsParams{
		TenantID:     tid,
		Search:       search,
		StatusFilter: statusFilter,
	})
	if err != nil {
		slog.ErrorContext(ctx, "count workflows", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count workflows")
		return
	}

	var nextCursor string
	if len(workflows) == int(limit) {
		last := workflows[len(workflows)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	WriteList(w, workflows, nextCursor, total)
}

type workflowNodeRequest struct {
	ID        string          `json:"id"`
	NodeType  string          `json:"node_type"`
	Label     string          `json:"label"`
	PositionX float64         `json:"position_x"`
	PositionY float64         `json:"position_y"`
	Config    json.RawMessage `json:"config"`
}

type workflowEdgeRequest struct {
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	Label        string `json:"label"`
}

type workflowRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Nodes       []workflowNodeRequest `json:"nodes"`
	Edges       []workflowEdgeRequest `json:"edges"`
}

type workflowResponse struct {
	ID        string `json:"id"`
	VersionID string `json:"version_id"`
	Version   int32  `json:"version"`
	Status    string `json:"status"`
}

// decodeAndValidateWorkflowBody reads, decodes, and validates a workflow request body.
// Returns the parsed body and true on success, or writes an error response and returns false.
func decodeAndValidateWorkflowBody(w http.ResponseWriter, r *http.Request) (workflowRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var body workflowRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return body, false
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return body, false
	}
	if len(body.Nodes) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "at least one node is required")
		return body, false
	}
	if msg, ok := validateNodeRequests(body.Nodes); !ok {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)
		return body, false
	}
	return body, true
}

// insertNodesAndEdges inserts nodes and edges into a workflow version within a transaction.
// Maps client-provided temp IDs to real DB UUIDs. Returns true on success, or writes an
// error response and returns false.
func insertNodesAndEdges(ctx context.Context, w http.ResponseWriter, txQ WorkflowQuerier, tid, versionID pgtype.UUID, nodes []workflowNodeRequest, edges []workflowEdgeRequest, tenantID string) bool {
	tempIDMap := make(map[string]pgtype.UUID, len(nodes))
	for _, n := range nodes {
		node, err := txQ.CreateWorkflowNode(ctx, sqlcgen.CreateWorkflowNodeParams{
			TenantID:  tid,
			VersionID: versionID,
			NodeType:  n.NodeType,
			Label:     n.Label,
			PositionX: n.PositionX,
			PositionY: n.PositionY,
			Config:    n.Config,
		})
		if err != nil {
			slog.ErrorContext(ctx, "create workflow node", "tenant_id", tenantID, "node_temp_id", n.ID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow node")
			return false
		}
		tempIDMap[n.ID] = node.ID
	}

	for _, e := range edges {
		sourceID, ok := tempIDMap[e.SourceNodeID]
		if !ok {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "edge references unknown source_node_id: "+e.SourceNodeID)
			return false
		}
		targetID, ok := tempIDMap[e.TargetNodeID]
		if !ok {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "edge references unknown target_node_id: "+e.TargetNodeID)
			return false
		}
		if _, err := txQ.CreateWorkflowEdge(ctx, sqlcgen.CreateWorkflowEdgeParams{
			TenantID:     tid,
			VersionID:    versionID,
			SourceNodeID: sourceID,
			TargetNodeID: targetID,
			Label:        e.Label,
		}); err != nil {
			slog.ErrorContext(ctx, "create workflow edge", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow edge")
			return false
		}
	}
	return true
}

// Create handles POST /api/v1/workflows.
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	body, ok := decodeAndValidateWorkflowBody(w, r)
	if !ok {
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for create workflow", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context in tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := h.newQuerier(tx)

	wf, err := txQ.CreateWorkflow(ctx, sqlcgen.CreateWorkflowParams{
		TenantID:    tid,
		Name:        body.Name,
		Description: body.Description,
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "CONFLICT", "a workflow with this name already exists")
			return
		}
		slog.ErrorContext(ctx, "create workflow", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow")
		return
	}

	ver, err := txQ.CreateWorkflowVersion(ctx, sqlcgen.CreateWorkflowVersionParams{
		TenantID:   tid,
		WorkflowID: wf.ID,
		Version:    1,
		Status:     string(workflow.StatusDraft),
	})
	if err != nil {
		slog.ErrorContext(ctx, "create workflow version", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow version")
		return
	}

	if !insertNodesAndEdges(ctx, w, txQ, tid, ver.ID, body.Nodes, body.Edges, tenantID) {
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit create workflow tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowCreated, "workflow", uuidToString(wf.ID), tenantID, wf)
	WriteJSON(w, http.StatusCreated, workflowResponse{
		ID:        uuidToString(wf.ID),
		VersionID: uuidToString(ver.ID),
		Version:   ver.Version,
		Status:    ver.Status,
	})
}

// workflowDetailResponse combines workflow info with version, nodes, and edges.
type workflowDetailResponse struct {
	sqlcgen.Workflow
	Version *sqlcgen.WorkflowVersion `json:"version,omitempty"`
	Nodes   []sqlcgen.WorkflowNode   `json:"nodes"`
	Edges   []sqlcgen.WorkflowEdge   `json:"edges"`
}

// Get handles GET /api/v1/workflows/{id}.
func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, tid, ok := parseWorkflowParams(w, r)
	if !ok {
		return
	}

	wf, err := h.q.GetWorkflowByID(ctx, sqlcgen.GetWorkflowByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
			return
		}
		slog.ErrorContext(ctx, "get workflow", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow")
		return
	}

	ver, err := h.q.GetLatestVersion(ctx, sqlcgen.GetLatestVersionParams{WorkflowID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			slog.WarnContext(ctx, "workflow has no versions",
				"workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID)
			WriteJSON(w, http.StatusOK, workflowDetailResponse{
				Workflow: wf,
				Nodes:    []sqlcgen.WorkflowNode{},
				Edges:    []sqlcgen.WorkflowEdge{},
			})
			return
		}
		slog.ErrorContext(ctx, "get latest version", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow version")
		return
	}

	nodes, err := h.q.ListWorkflowNodes(ctx, sqlcgen.ListWorkflowNodesParams{VersionID: ver.ID, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list workflow nodes", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workflow nodes")
		return
	}

	edges, err := h.q.ListWorkflowEdges(ctx, sqlcgen.ListWorkflowEdgesParams{VersionID: ver.ID, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list workflow edges", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workflow edges")
		return
	}

	WriteJSON(w, http.StatusOK, workflowDetailResponse{
		Workflow: wf,
		Version:  &ver,
		Nodes:    nodes,
		Edges:    edges,
	})
}

// Update handles PUT /api/v1/workflows/{id}.
func (h *WorkflowHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, tid, ok := parseWorkflowParams(w, r)
	if !ok {
		return
	}

	body, ok := decodeAndValidateWorkflowBody(w, r)
	if !ok {
		return
	}

	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for update workflow", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context in tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := h.newQuerier(tx)

	// Verify workflow exists inside the transaction to avoid TOCTOU race.
	_, err = txQ.GetWorkflowByID(ctx, sqlcgen.GetWorkflowByIDParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
			return
		}
		slog.ErrorContext(ctx, "get workflow for update", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow")
		return
	}

	// Get next version number.
	maxVer, err := txQ.GetMaxVersionNumber(ctx, sqlcgen.GetMaxVersionNumberParams{
		WorkflowID: id,
		TenantID:   tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "get max version number", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
		return
	}

	// Archive current draft if one exists.
	draft, err := txQ.GetDraftVersion(ctx, sqlcgen.GetDraftVersionParams{WorkflowID: id, TenantID: tid})
	if err == nil {
		if archiveErr := txQ.ArchiveWorkflowVersion(ctx, sqlcgen.ArchiveWorkflowVersionParams{
			ID: draft.ID, TenantID: tid,
		}); archiveErr != nil {
			slog.ErrorContext(ctx, "archive draft version", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", archiveErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
			return
		}
	} else if !isNotFound(err) {
		slog.ErrorContext(ctx, "get draft version for archive", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
		return
	}

	// Create new version.
	ver, err := txQ.CreateWorkflowVersion(ctx, sqlcgen.CreateWorkflowVersionParams{
		TenantID:   tid,
		WorkflowID: id,
		Version:    maxVer + 1,
		Status:     string(workflow.StatusDraft),
	})
	if err != nil {
		slog.ErrorContext(ctx, "create new workflow version", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
		return
	}

	if !insertNodesAndEdges(ctx, w, txQ, tid, ver.ID, body.Nodes, body.Edges, tenantID) {
		return
	}

	// Update workflow name/description.
	wf, err := txQ.UpdateWorkflow(ctx, sqlcgen.UpdateWorkflowParams{
		ID:          id,
		Name:        body.Name,
		Description: body.Description,
		TenantID:    tid,
	})
	if err != nil {
		if isUniqueViolation(err) {
			WriteError(w, http.StatusConflict, "CONFLICT", "a workflow with this name already exists")
			return
		}
		slog.ErrorContext(ctx, "update workflow", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit update workflow tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowUpdated, "workflow", uuidToString(wf.ID), tenantID, wf)
	WriteJSON(w, http.StatusOK, workflowResponse{
		ID:        uuidToString(wf.ID),
		VersionID: uuidToString(ver.ID),
		Version:   ver.Version,
		Status:    ver.Status,
	})
}

// Delete handles DELETE /api/v1/workflows/{id} (soft delete).
func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, tid, ok := parseWorkflowParams(w, r)
	if !ok {
		return
	}

	wf, err := h.q.SoftDeleteWorkflow(ctx, sqlcgen.SoftDeleteWorkflowParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
			return
		}
		slog.ErrorContext(ctx, "soft delete workflow", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete workflow")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowDeleted, "workflow", uuidToString(wf.ID), tenantID, wf)
	w.WriteHeader(http.StatusNoContent)
}

// Publish handles PUT /api/v1/workflows/{id}/publish.
func (h *WorkflowHandler) Publish(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, tid, ok := parseWorkflowParams(w, r)
	if !ok {
		return
	}

	// TODO(#177): Move reads inside the transaction to eliminate TOCTOU race.
	// A concurrent Update between these reads and the publish transaction could cause a
	// stale publish or confusing error. This is acceptable for now because publish is a
	// manual action and the failure mode is safe (the publish fails, no data corruption).
	draft, err := h.q.GetDraftVersion(ctx, sqlcgen.GetDraftVersionParams{WorkflowID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "no draft version found for workflow")
			return
		}
		slog.ErrorContext(ctx, "get draft version for publish", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get draft version")
		return
	}

	nodes, err := h.q.ListWorkflowNodes(ctx, sqlcgen.ListWorkflowNodesParams{VersionID: draft.ID, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list nodes for publish", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load workflow nodes for validation")
		return
	}
	edges, err := h.q.ListWorkflowEdges(ctx, sqlcgen.ListWorkflowEdgesParams{VersionID: draft.ID, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list edges for publish", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load workflow edges for validation")
		return
	}

	// Convert to workflow domain types for validation.
	domainNodes := make([]workflow.Node, len(nodes))
	for i, n := range nodes {
		domainNodes[i] = workflow.Node{
			ID:        uuidToString(n.ID),
			NodeType:  workflow.NodeType(n.NodeType),
			Label:     n.Label,
			PositionX: n.PositionX,
			PositionY: n.PositionY,
			Config:    n.Config,
		}
	}
	domainEdges := make([]workflow.Edge, len(edges))
	for i, e := range edges {
		domainEdges[i] = workflow.Edge{
			ID:           uuidToString(e.ID),
			SourceNodeID: uuidToString(e.SourceNodeID),
			TargetNodeID: uuidToString(e.TargetNodeID),
			Label:        e.Label,
		}
	}

	if validateErr := workflow.ValidateWorkflow(domainNodes, domainEdges); validateErr != nil {
		var ve *workflow.ValidationError
		if errors.As(validateErr, &ve) {
			WriteJSON(w, http.StatusBadRequest, map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "workflow validation failed",
				"details": ve.Violations,
			})
			return
		}
		slog.ErrorContext(ctx, "validate workflow", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", validateErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to validate workflow")
		return
	}

	tx, err := h.txb.Begin(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "begin tx for publish workflow", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to publish workflow")
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID); err != nil {
		slog.ErrorContext(ctx, "set tenant context in tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to set tenant context")
		return
	}

	txQ := h.newQuerier(tx)

	// Archive currently published version (ignore not-found).
	pub, err := txQ.GetPublishedVersion(ctx, sqlcgen.GetPublishedVersionParams{WorkflowID: id, TenantID: tid})
	if err == nil {
		if archiveErr := txQ.ArchiveWorkflowVersion(ctx, sqlcgen.ArchiveWorkflowVersionParams{
			ID: pub.ID, TenantID: tid,
		}); archiveErr != nil {
			slog.ErrorContext(ctx, "archive published version", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", archiveErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to publish workflow")
			return
		}
	} else if !isNotFound(err) {
		slog.ErrorContext(ctx, "get published version for archive", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to publish workflow")
		return
	}

	published, err := txQ.PublishWorkflowVersion(ctx, sqlcgen.PublishWorkflowVersionParams{
		ID: draft.ID, TenantID: tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "publish workflow version", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to publish workflow")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.ErrorContext(ctx, "commit publish workflow tx", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to publish workflow")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowPublished, "workflow", uuidToString(id), tenantID, published)
	WriteJSON(w, http.StatusOK, published)
}

// ListVersions handles GET /api/v1/workflows/{id}/versions.
func (h *WorkflowHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, tid, ok := parseWorkflowParams(w, r)
	if !ok {
		return
	}

	versions, err := h.q.ListWorkflowVersions(ctx, sqlcgen.ListWorkflowVersionsParams{WorkflowID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "list workflow versions", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workflow versions")
		return
	}

	WriteJSON(w, http.StatusOK, versions)
}

// ListTemplates handles GET /api/v1/workflow-templates.
func (h *WorkflowHandler) ListTemplates(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, workflow.AllTemplates())
}

// EscapeLikePattern escapes SQL LIKE/ILIKE wildcards in user input.
func EscapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// parseWorkflowParams extracts and validates the workflow ID and tenant ID from the request.
// Returns zero-value UUIDs and false if either is invalid (error response already written).
func parseWorkflowParams(w http.ResponseWriter, r *http.Request) (id, tid pgtype.UUID, ok bool) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var err error
	id, err = scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid workflow ID: not a valid UUID")
		return id, tid, false
	}
	tid, err = scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return id, tid, false
	}
	return id, tid, true
}

// validateNodeRequests checks node requests for empty IDs, invalid node types, and duplicates.
// Returns ("", true) if all nodes are valid, or (message, false) with the first error found.
func validateNodeRequests(nodes []workflowNodeRequest) (string, bool) {
	seen := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		if n.ID == "" {
			return "node id must not be empty", false
		}
		if !workflow.NodeType(n.NodeType).IsValid() {
			return "invalid node_type: " + n.NodeType, false
		}
		if seen[n.ID] {
			return "duplicate node id: " + n.ID, false
		}
		seen[n.ID] = true
	}
	return "", true
}
