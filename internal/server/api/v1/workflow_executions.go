package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/workflow"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// WorkflowExecutionQuerier defines the sqlc queries needed by WorkflowExecutionHandler.
type WorkflowExecutionQuerier interface {
	GetWorkflowByID(ctx context.Context, arg sqlcgen.GetWorkflowByIDParams) (sqlcgen.Workflow, error)
	GetPublishedVersion(ctx context.Context, arg sqlcgen.GetPublishedVersionParams) (sqlcgen.WorkflowVersion, error)
	CreateWorkflowExecution(ctx context.Context, arg sqlcgen.CreateWorkflowExecutionParams) (sqlcgen.WorkflowExecution, error)
	GetWorkflowExecution(ctx context.Context, arg sqlcgen.GetWorkflowExecutionParams) (sqlcgen.WorkflowExecution, error)
	ListWorkflowExecutions(ctx context.Context, arg sqlcgen.ListWorkflowExecutionsParams) ([]sqlcgen.WorkflowExecution, error)
	CountWorkflowExecutions(ctx context.Context, arg sqlcgen.CountWorkflowExecutionsParams) (int64, error)
	ListNodeExecutions(ctx context.Context, arg sqlcgen.ListNodeExecutionsParams) ([]sqlcgen.WorkflowNodeExecution, error)
	UpdateWorkflowExecutionStatus(ctx context.Context, arg sqlcgen.UpdateWorkflowExecutionStatusParams) (sqlcgen.WorkflowExecution, error)
	GetPendingApprovalByExecution(ctx context.Context, arg sqlcgen.GetPendingApprovalByExecutionParams) (sqlcgen.ApprovalRequest, error)
	UpdateApprovalRequest(ctx context.Context, arg sqlcgen.UpdateApprovalRequestParams) (sqlcgen.ApprovalRequest, error)
}

// WorkflowExecutionHandler serves workflow execution REST API endpoints.
type WorkflowExecutionHandler struct {
	q           WorkflowExecutionQuerier
	txb         TxBeginner
	eventBus    domain.EventBus
	jobInserter CVEMatchInserter
}

// NewWorkflowExecutionHandler creates a WorkflowExecutionHandler.
func NewWorkflowExecutionHandler(q WorkflowExecutionQuerier, txb TxBeginner, eventBus domain.EventBus, jobInserter CVEMatchInserter) *WorkflowExecutionHandler {
	if q == nil {
		panic("workflow_executions: NewWorkflowExecutionHandler called with nil querier")
	}
	if txb == nil {
		panic("workflow_executions: NewWorkflowExecutionHandler called with nil txBeginner")
	}
	if eventBus == nil {
		panic("workflow_executions: NewWorkflowExecutionHandler called with nil eventBus")
	}
	return &WorkflowExecutionHandler{
		q:           q,
		txb:         txb,
		eventBus:    eventBus,
		jobInserter: jobInserter,
	}
}

// executionResponse is the JSON response for a single execution.
type executionResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	WorkflowID string `json:"workflow_id"`
	VersionID  string `json:"version_id"`
}

// executionDetailResponse combines execution info with node executions.
type executionDetailResponse struct {
	sqlcgen.WorkflowExecution
	NodeExecutions []sqlcgen.WorkflowNodeExecution `json:"node_executions"`
}

// approvalBody is the optional JSON body for approve/reject endpoints.
type approvalBody struct {
	Comment string `json:"comment"`
}

// Execute handles POST /api/v1/workflows/{id}/execute.
func (h *WorkflowExecutionHandler) Execute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	wfID, tid, ok := parseExecWorkflowParams(w, r)
	if !ok {
		return
	}

	// Verify workflow exists.
	_, err := h.q.GetWorkflowByID(ctx, sqlcgen.GetWorkflowByIDParams{ID: wfID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
			return
		}
		slog.ErrorContext(ctx, "get workflow for execution", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow")
		return
	}

	// Verify published version exists.
	ver, err := h.q.GetPublishedVersion(ctx, sqlcgen.GetPublishedVersionParams{WorkflowID: wfID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusBadRequest, "NO_PUBLISHED_VERSION", "workflow has no published version; publish a version before executing")
			return
		}
		slog.ErrorContext(ctx, "get published version for execution", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get published version")
		return
	}

	// Determine triggering user.
	var triggeredByUserID pgtype.UUID
	triggeredBy := "manual"
	if uid, ok := user.UserIDFromContext(ctx); ok && uid != "" {
		parsed, parseErr := scanUUID(uid)
		if parseErr == nil {
			triggeredByUserID = parsed
		}
	}

	exec, err := h.q.CreateWorkflowExecution(ctx, sqlcgen.CreateWorkflowExecutionParams{
		TenantID:          tid,
		WorkflowID:        wfID,
		VersionID:         ver.ID,
		Status:            "pending",
		TriggeredBy:       triggeredBy,
		TriggeredByUserID: triggeredByUserID,
		Context:           []byte("{}"),
	})
	if err != nil {
		slog.ErrorContext(ctx, "create workflow execution", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create workflow execution")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowExecutionStarted, "workflow_execution", uuidToString(exec.ID), tenantID, exec)

	// Enqueue the River job to execute the workflow DAG.
	if h.jobInserter != nil {
		_, insertErr := h.jobInserter.Insert(ctx, workflow.WorkflowExecuteJobArgs{
			ExecutionID: uuidToString(exec.ID),
			TenantID:    tenantID,
		}, nil)
		if insertErr != nil {
			slog.ErrorContext(ctx, "enqueue workflow execute job",
				"execution_id", uuidToString(exec.ID), "tenant_id", tenantID, "error", insertErr)
		} else {
			slog.InfoContext(ctx, "workflow execute job enqueued",
				"execution_id", uuidToString(exec.ID), "tenant_id", tenantID)
		}
	}

	WriteJSON(w, http.StatusCreated, executionResponse{
		ID:         uuidToString(exec.ID),
		Status:     exec.Status,
		WorkflowID: uuidToString(exec.WorkflowID),
		VersionID:  uuidToString(exec.VersionID),
	})
}

// List handles GET /api/v1/workflows/{id}/executions.
func (h *WorkflowExecutionHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	wfID, tid, ok := parseExecWorkflowParams(w, r)
	if !ok {
		return
	}

	limit := ParseLimit(r.URL.Query().Get("limit"))
	statusFilter := r.URL.Query().Get("status")

	executions, err := h.q.ListWorkflowExecutions(ctx, sqlcgen.ListWorkflowExecutionsParams{
		WorkflowID:   wfID,
		TenantID:     tid,
		StatusFilter: statusFilter,
		PageLimit:    limit,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list workflow executions", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list workflow executions")
		return
	}

	total, err := h.q.CountWorkflowExecutions(ctx, sqlcgen.CountWorkflowExecutionsParams{
		WorkflowID:   wfID,
		TenantID:     tid,
		StatusFilter: statusFilter,
	})
	if err != nil {
		slog.ErrorContext(ctx, "count workflow executions", "workflow_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count workflow executions")
		return
	}

	var nextCursor string
	if len(executions) == int(limit) {
		last := executions[len(executions)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, uuidToString(last.ID))
	}

	WriteList(w, executions, nextCursor, total)
}

// Get handles GET /api/v1/workflows/{id}/executions/{execId}.
func (h *WorkflowExecutionHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	_, tid, ok := parseExecWorkflowParams(w, r)
	if !ok {
		return
	}

	execID, err := scanUUID(chi.URLParam(r, "execId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid execution ID: not a valid UUID")
		return
	}

	exec, err := h.q.GetWorkflowExecution(ctx, sqlcgen.GetWorkflowExecutionParams{ID: execID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow execution not found")
			return
		}
		slog.ErrorContext(ctx, "get workflow execution", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow execution")
		return
	}

	nodeExecs, err := h.q.ListNodeExecutions(ctx, sqlcgen.ListNodeExecutionsParams{
		ExecutionID: execID,
		TenantID:    tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list node executions", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list node executions")
		return
	}

	WriteJSON(w, http.StatusOK, executionDetailResponse{
		WorkflowExecution: exec,
		NodeExecutions:    nodeExecs,
	})
}

// Cancel handles POST /api/v1/workflows/{id}/executions/{execId}/cancel.
func (h *WorkflowExecutionHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	_, tid, ok := parseExecWorkflowParams(w, r)
	if !ok {
		return
	}

	execID, err := scanUUID(chi.URLParam(r, "execId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid execution ID: not a valid UUID")
		return
	}

	exec, err := h.q.GetWorkflowExecution(ctx, sqlcgen.GetWorkflowExecutionParams{ID: execID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow execution not found")
			return
		}
		slog.ErrorContext(ctx, "get execution for cancel", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow execution")
		return
	}

	if exec.Status != "running" && exec.Status != "paused" && exec.Status != "pending" {
		WriteError(w, http.StatusConflict, "INVALID_STATUS", "execution cannot be cancelled in status: "+exec.Status)
		return
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	updated, err := h.q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
		ID:            execID,
		TenantID:      tid,
		Status:        "cancelled",
		CurrentNodeID: exec.CurrentNodeID,
		Context:       exec.Context,
		ErrorMessage:  "cancelled by user",
		StartedAt:     exec.StartedAt,
		CompletedAt:   now,
	})
	if err != nil {
		slog.ErrorContext(ctx, "cancel workflow execution", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to cancel workflow execution")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowExecutionCancelled, "workflow_execution", uuidToString(execID), tenantID, updated)
	WriteJSON(w, http.StatusOK, executionResponse{
		ID:         uuidToString(updated.ID),
		Status:     updated.Status,
		WorkflowID: uuidToString(updated.WorkflowID),
		VersionID:  uuidToString(updated.VersionID),
	})
}

// Approve handles POST /api/v1/workflows/{id}/executions/{execId}/approve.
func (h *WorkflowExecutionHandler) Approve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	_, tid, ok := parseExecWorkflowParams(w, r)
	if !ok {
		return
	}

	execID, err := scanUUID(chi.URLParam(r, "execId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid execution ID: not a valid UUID")
		return
	}

	exec, err := h.q.GetWorkflowExecution(ctx, sqlcgen.GetWorkflowExecutionParams{ID: execID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow execution not found")
			return
		}
		slog.ErrorContext(ctx, "get execution for approve", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow execution")
		return
	}

	if exec.Status != "paused" {
		WriteError(w, http.StatusConflict, "INVALID_STATUS", "execution must be paused to approve; current status: "+exec.Status)
		return
	}

	approval, err := h.q.GetPendingApprovalByExecution(ctx, sqlcgen.GetPendingApprovalByExecutionParams{
		ExecutionID: execID,
		TenantID:    tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "no pending approval request found for this execution")
			return
		}
		slog.ErrorContext(ctx, "get pending approval", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get pending approval request")
		return
	}

	var body approvalBody
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	actorID := resolveActorUUID(ctx)

	_, err = h.q.UpdateApprovalRequest(ctx, sqlcgen.UpdateApprovalRequestParams{
		ID:       approval.ID,
		TenantID: tid,
		Status:   "approved",
		ActedBy:  actorID,
		ActedAt:  now,
		Comment:  body.Comment,
	})
	if err != nil {
		slog.ErrorContext(ctx, "update approval request", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update approval request")
		return
	}

	updated, err := h.q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
		ID:            execID,
		TenantID:      tid,
		Status:        "running",
		CurrentNodeID: exec.CurrentNodeID,
		Context:       exec.Context,
		ErrorMessage:  exec.ErrorMessage,
		StartedAt:     exec.StartedAt,
		CompletedAt:   exec.CompletedAt,
	})
	if err != nil {
		slog.ErrorContext(ctx, "resume workflow execution", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to resume workflow execution")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowExecutionResumed, "workflow_execution", uuidToString(execID), tenantID, updated)

	// Enqueue a River job to resume the workflow execution from the paused node.
	if h.jobInserter != nil {
		_, insertErr := h.jobInserter.Insert(ctx, workflow.WorkflowExecuteJobArgs{
			ExecutionID: uuidToString(execID),
			TenantID:    tenantID,
		}, nil)
		if insertErr != nil {
			slog.ErrorContext(ctx, "enqueue workflow resume job",
				"execution_id", uuidToString(execID), "tenant_id", tenantID, "error", insertErr)
		} else {
			slog.InfoContext(ctx, "workflow resume job enqueued",
				"execution_id", uuidToString(execID), "tenant_id", tenantID)
		}
	}

	WriteJSON(w, http.StatusOK, executionResponse{
		ID:         uuidToString(updated.ID),
		Status:     updated.Status,
		WorkflowID: uuidToString(updated.WorkflowID),
		VersionID:  uuidToString(updated.VersionID),
	})
}

// Reject handles POST /api/v1/workflows/{id}/executions/{execId}/reject.
func (h *WorkflowExecutionHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	_, tid, ok := parseExecWorkflowParams(w, r)
	if !ok {
		return
	}

	execID, err := scanUUID(chi.URLParam(r, "execId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid execution ID: not a valid UUID")
		return
	}

	exec, err := h.q.GetWorkflowExecution(ctx, sqlcgen.GetWorkflowExecutionParams{ID: execID, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "workflow execution not found")
			return
		}
		slog.ErrorContext(ctx, "get execution for reject", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get workflow execution")
		return
	}

	if exec.Status != "paused" {
		WriteError(w, http.StatusConflict, "INVALID_STATUS", "execution must be paused to reject; current status: "+exec.Status)
		return
	}

	approval, err := h.q.GetPendingApprovalByExecution(ctx, sqlcgen.GetPendingApprovalByExecutionParams{
		ExecutionID: execID,
		TenantID:    tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "no pending approval request found for this execution")
			return
		}
		slog.ErrorContext(ctx, "get pending approval for reject", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get pending approval request")
		return
	}

	var body approvalBody
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	actorID := resolveActorUUID(ctx)

	_, err = h.q.UpdateApprovalRequest(ctx, sqlcgen.UpdateApprovalRequestParams{
		ID:       approval.ID,
		TenantID: tid,
		Status:   "rejected",
		ActedBy:  actorID,
		ActedAt:  now,
		Comment:  body.Comment,
	})
	if err != nil {
		slog.ErrorContext(ctx, "update approval request for reject", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update approval request")
		return
	}

	updated, err := h.q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
		ID:            execID,
		TenantID:      tid,
		Status:        "failed",
		CurrentNodeID: exec.CurrentNodeID,
		Context:       exec.Context,
		ErrorMessage:  "approval rejected: " + body.Comment,
		StartedAt:     exec.StartedAt,
		CompletedAt:   now,
	})
	if err != nil {
		slog.ErrorContext(ctx, "fail workflow execution after reject", "execution_id", chi.URLParam(r, "execId"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update workflow execution")
		return
	}

	emitEvent(ctx, h.eventBus, events.WorkflowExecutionFailed, "workflow_execution", uuidToString(execID), tenantID, updated)
	WriteJSON(w, http.StatusOK, executionResponse{
		ID:         uuidToString(updated.ID),
		Status:     updated.Status,
		WorkflowID: uuidToString(updated.WorkflowID),
		VersionID:  uuidToString(updated.VersionID),
	})
}

// parseExecWorkflowParams extracts and validates the workflow ID and tenant ID from the request.
func parseExecWorkflowParams(w http.ResponseWriter, r *http.Request) (wfID, tid pgtype.UUID, ok bool) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var err error
	wfID, err = scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid workflow ID: not a valid UUID")
		return wfID, tid, false
	}
	tid, err = scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return wfID, tid, false
	}
	return wfID, tid, true
}

// resolveActorUUID extracts the user ID from context and parses it as a UUID.
// Returns an invalid pgtype.UUID if no user ID is present or it's not a valid UUID.
func resolveActorUUID(ctx context.Context) pgtype.UUID {
	if uid, ok := user.UserIDFromContext(ctx); ok && uid != "" {
		parsed, err := scanUUID(uid)
		if err == nil {
			return parsed
		}
	}
	return pgtype.UUID{}
}
