package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// WorkflowExecuteJobArgs are the arguments for the workflow execution job.
// This job drives the main DAG execution loop for a workflow run.
type WorkflowExecuteJobArgs struct {
	ExecutionID string `json:"execution_id"`
	TenantID    string `json:"tenant_id"`
}

func (WorkflowExecuteJobArgs) Kind() string { return "workflow_execute" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (WorkflowExecuteJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "default"}
}

// ApprovalTimeoutJobArgs are the arguments for the approval timeout job.
// Scheduled when a node enters an approval-pending state; fires if no
// approval arrives before the deadline.
type ApprovalTimeoutJobArgs struct {
	ExecutionID string `json:"execution_id"`
	NodeID      string `json:"node_id"`
	TenantID    string `json:"tenant_id"`
}

func (ApprovalTimeoutJobArgs) Kind() string { return "approval_timeout" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ApprovalTimeoutJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// GateTimeoutJobArgs are the arguments for the gate timeout job.
// Scheduled when execution reaches a gate node; fires if the gate
// condition is not satisfied before the deadline.
type GateTimeoutJobArgs struct {
	ExecutionID string `json:"execution_id"`
	NodeID      string `json:"node_id"`
	TenantID    string `json:"tenant_id"`
}

func (GateTimeoutJobArgs) Kind() string { return "gate_timeout" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (GateTimeoutJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: "critical"}
}

// WorkflowExecuteWorker processes workflow execution jobs.
type WorkflowExecuteWorker struct {
	river.WorkerDefaults[WorkflowExecuteJobArgs]
	pool     *pgxpool.Pool
	eventBus domain.EventBus
	handlers map[NodeType]NodeHandler
}

// NewWorkflowExecuteWorker creates a fully-wired workflow execution worker.
func NewWorkflowExecuteWorker(pool *pgxpool.Pool, eventBus domain.EventBus, handlers map[NodeType]NodeHandler) *WorkflowExecuteWorker {
	return &WorkflowExecuteWorker{
		pool:     pool,
		eventBus: eventBus,
		handlers: handlers,
	}
}

func (w *WorkflowExecuteWorker) Work(ctx context.Context, job *river.Job[WorkflowExecuteJobArgs]) error {
	execIDStr := job.Args.ExecutionID
	tenantIDStr := job.Args.TenantID

	slog.InfoContext(ctx, "workflow execute job: starting",
		"execution_id", execIDStr,
		"tenant_id", tenantIDStr)

	tid := pgUUID(tenantIDStr)
	execID := pgUUID(execIDStr)

	if !tid.Valid || !execID.Valid {
		return fmt.Errorf("workflow execute job: invalid execution_id=%s or tenant_id=%s", execIDStr, tenantIDStr)
	}

	// Set tenant context for RLS inside a transaction (true = transaction-local).
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("workflow execute job: begin tx: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.ErrorContext(ctx, "workflow execute job: rollback failed", "error", err)
		}
	}()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantIDStr); err != nil {
		return fmt.Errorf("workflow execute job: set tenant context: %w", err)
	}

	q := sqlcgen.New(tx)

	// Load execution record.
	exec, err := q.GetWorkflowExecution(ctx, sqlcgen.GetWorkflowExecutionParams{
		ID:       execID,
		TenantID: tid,
	})
	if err != nil {
		return fmt.Errorf("workflow execute job: get execution: %w", err)
	}

	// Update status to running.
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	_, err = q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
		ID:            execID,
		TenantID:      tid,
		Status:        "running",
		CurrentNodeID: exec.CurrentNodeID,
		Context:       exec.Context,
		ErrorMessage:  exec.ErrorMessage,
		StartedAt:     now,
		CompletedAt:   pgtype.Timestamptz{},
	})
	if err != nil {
		return fmt.Errorf("workflow execute job: update status to running: %w", err)
	}

	w.emitEvent(ctx, events.WorkflowExecutionStarted, execIDStr, tenantIDStr, map[string]any{
		"execution_id": execIDStr,
		"status":       "running",
	})

	// Load version nodes and edges.
	nodes, err := q.ListWorkflowNodes(ctx, sqlcgen.ListWorkflowNodesParams{
		VersionID: exec.VersionID,
		TenantID:  tid,
	})
	if err != nil {
		return w.failExecution(ctx, tx, q, execID, tid, exec, fmt.Sprintf("load nodes: %v", err))
	}

	edges, err := q.ListWorkflowEdges(ctx, sqlcgen.ListWorkflowEdgesParams{
		VersionID: exec.VersionID,
		TenantID:  tid,
	})
	if err != nil {
		return w.failExecution(ctx, tx, q, execID, tid, exec, fmt.Sprintf("load edges: %v", err))
	}

	// Convert DB types to workflow engine types.
	wfNodes := make([]Node, len(nodes))
	for i, n := range nodes {
		wfNodes[i] = Node{
			ID:       uuidStr(n.ID),
			NodeType: NodeType(n.NodeType),
			Label:    n.Label,
			Config:   json.RawMessage(n.Config),
		}
	}

	wfEdges := make([]Edge, len(edges))
	for i, e := range edges {
		wfEdges[i] = Edge{
			ID:           uuidStr(e.ID),
			SourceNodeID: uuidStr(e.SourceNodeID),
			TargetNodeID: uuidStr(e.TargetNodeID),
			Label:        e.Label,
		}
	}

	// Build execution context from stored context JSON.
	var execCtx map[string]any
	if len(exec.Context) > 0 {
		if err := json.Unmarshal(exec.Context, &execCtx); err != nil {
			execCtx = make(map[string]any)
		}
	} else {
		execCtx = make(map[string]any)
	}

	// After JSON round-trip, []string becomes []interface{}.
	// Convert known string-slice context keys back to []string.
	restoreStringSlice(execCtx, "filtered_endpoints")

	// Inject tenant/execution metadata into context for handlers.
	execCtx["tenant_id"] = tenantIDStr
	execCtx["execution_id"] = execIDStr
	execCtx["workflow_id"] = uuidStr(exec.WorkflowID)
	execCtx["version_id"] = uuidStr(exec.VersionID)

	// Determine if we're resuming from a paused state.
	resumeAfterNodeID := ""
	var completedNodes map[string]bool
	if exec.CurrentNodeID.Valid {
		resumeAfterNodeID = uuidStr(exec.CurrentNodeID)
		completedNodes = w.loadCompletedNodes(ctx, q, execID, tid)
		slog.InfoContext(ctx, "workflow execute job: resuming after pause",
			"execution_id", execIDStr,
			"resume_after_node_id", resumeAfterNodeID,
			"completed_count", len(completedNodes))
	}

	// Wrap execution context so handlers get tenant/execution info.
	// The executor's RunFrom injects these into ExecutionContext.
	// We need to pre-set the fields on the executor context.
	// Actually, the InMemoryExecutor doesn't set TenantID etc on ExecutionContext.
	// We need to do that ourselves. Let me patch this by wrapping handlers.
	wrappedHandlers := make(map[NodeType]NodeHandler, len(w.handlers))
	for nt, h := range w.handlers {
		wrappedHandlers[nt] = &contextInjectingHandler{
			inner:       h,
			tenantID:    tenantIDStr,
			executionID: execIDStr,
			versionID:   uuidStr(exec.VersionID),
			workflowID:  uuidStr(exec.WorkflowID),
		}
	}
	executor := NewInMemoryExecutor(wrappedHandlers)

	// Run the DAG.
	var result *ExecutionResult
	if resumeAfterNodeID != "" {
		result, err = executor.RunFrom(ctx, wfNodes, wfEdges, execCtx, resumeAfterNodeID, completedNodes)
	} else {
		result, err = executor.Run(ctx, wfNodes, wfEdges, execCtx)
	}
	if err != nil {
		return w.failExecution(ctx, tx, q, execID, tid, exec, fmt.Sprintf("executor error: %v", err))
	}

	// Persist node execution results.
	for nodeID, nodeResult := range result.NodeResults {
		w.persistNodeResult(ctx, q, execID, tid, nodeID, wfNodes, nodeResult)
	}

	// Update execution based on result status.
	ctxJSON, _ := json.Marshal(result.Context)

	switch result.Status {
	case ExecCompleted:
		completedAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err = q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
			ID:            execID,
			TenantID:      tid,
			Status:        "completed",
			CurrentNodeID: pgtype.UUID{},
			Context:       ctxJSON,
			ErrorMessage:  "",
			StartedAt:     now,
			CompletedAt:   completedAt,
		})
		if err != nil {
			slog.ErrorContext(ctx, "workflow execute job: update completed status", "error", err)
		}
		w.emitEvent(ctx, events.WorkflowExecutionCompleted, execIDStr, tenantIDStr, map[string]any{
			"execution_id": execIDStr,
			"status":       "completed",
		})
		slog.InfoContext(ctx, "workflow execution completed", "execution_id", execIDStr)

	case ExecPaused:
		pausedNodeID := pgUUID(result.PausedAtNodeID)
		_, err = q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
			ID:            execID,
			TenantID:      tid,
			Status:        "paused",
			CurrentNodeID: pausedNodeID,
			Context:       ctxJSON,
			ErrorMessage:  "",
			StartedAt:     now,
			CompletedAt:   pgtype.Timestamptz{},
		})
		if err != nil {
			slog.ErrorContext(ctx, "workflow execute job: update paused status", "error", err)
		}
		w.emitEvent(ctx, events.WorkflowExecutionPaused, execIDStr, tenantIDStr, map[string]any{
			"execution_id":   execIDStr,
			"status":         "paused",
			"paused_at_node": result.PausedAtNodeID,
		})
		slog.InfoContext(ctx, "workflow execution paused",
			"execution_id", execIDStr,
			"paused_at_node", result.PausedAtNodeID)

	case ExecFailed:
		completedAt := pgtype.Timestamptz{Time: time.Now(), Valid: true}
		_, err = q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
			ID:            execID,
			TenantID:      tid,
			Status:        "failed",
			CurrentNodeID: pgtype.UUID{},
			Context:       ctxJSON,
			ErrorMessage:  result.Error,
			StartedAt:     now,
			CompletedAt:   completedAt,
		})
		if err != nil {
			slog.ErrorContext(ctx, "workflow execute job: update failed status", "error", err)
		}
		w.emitEvent(ctx, events.WorkflowExecutionFailed, execIDStr, tenantIDStr, map[string]any{
			"execution_id": execIDStr,
			"status":       "failed",
			"error":        result.Error,
		})
		slog.WarnContext(ctx, "workflow execution failed",
			"execution_id", execIDStr,
			"error", result.Error)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("workflow execute job: commit tx: %w", err)
	}

	return nil
}

// failExecution marks the execution as failed, commits the transaction, and returns nil
// (job succeeded, execution failed).
func (w *WorkflowExecuteWorker) failExecution(ctx context.Context, tx pgx.Tx, q *sqlcgen.Queries, execID, tid pgtype.UUID, exec sqlcgen.WorkflowExecution, errMsg string) error {
	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	_, err := q.UpdateWorkflowExecutionStatus(ctx, sqlcgen.UpdateWorkflowExecutionStatusParams{
		ID:            execID,
		TenantID:      tid,
		Status:        "failed",
		CurrentNodeID: exec.CurrentNodeID,
		Context:       exec.Context,
		ErrorMessage:  errMsg,
		StartedAt:     now,
		CompletedAt:   now,
	})
	if err != nil {
		slog.ErrorContext(ctx, "workflow execute job: failed to update execution status",
			"execution_id", uuidStr(execID), "error", err)
	}
	if commitErr := tx.Commit(ctx); commitErr != nil {
		slog.ErrorContext(ctx, "workflow execute job: commit failed in failExecution",
			"execution_id", uuidStr(execID), "error", commitErr)
	}
	w.emitEvent(ctx, events.WorkflowExecutionFailed, uuidStr(execID), uuidStr(tid), map[string]any{
		"execution_id": uuidStr(execID),
		"error":        errMsg,
	})
	slog.ErrorContext(ctx, "workflow execution failed", "execution_id", uuidStr(execID), "error", errMsg)
	return nil
}

// loadCompletedNodes returns a set of node IDs that have already completed.
func (w *WorkflowExecuteWorker) loadCompletedNodes(ctx context.Context, q *sqlcgen.Queries, execID, tid pgtype.UUID) map[string]bool {
	nodeExecs, err := q.ListNodeExecutions(ctx, sqlcgen.ListNodeExecutionsParams{
		ExecutionID: execID,
		TenantID:    tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "workflow execute job: load completed nodes", "error", err)
		return nil
	}
	completed := make(map[string]bool)
	for _, ne := range nodeExecs {
		if ne.Status == "completed" || ne.Status == "skipped" {
			completed[uuidStr(ne.NodeID)] = true
		}
	}
	return completed
}

// persistNodeResult saves a node execution record to the DB.
func (w *WorkflowExecuteWorker) persistNodeResult(ctx context.Context, q *sqlcgen.Queries, execID, tid pgtype.UUID, nodeID string, nodes []Node, result *NodeResult) {
	nodeUUID := pgUUID(nodeID)
	if !nodeUUID.Valid {
		slog.ErrorContext(ctx, "workflow execute job: invalid node UUID", "node_id", nodeID)
		return
	}

	// Find node type from the nodes list.
	var nodeType string
	for _, n := range nodes {
		if n.ID == nodeID {
			nodeType = string(n.NodeType)
			break
		}
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	outputJSON, marshalErr := json.Marshal(result.Output)
	if marshalErr != nil {
		slog.ErrorContext(ctx, "workflow execute job: marshal node output",
			"node_id", nodeID, "error", marshalErr)
		outputJSON = []byte("null")
	}

	// Try to create or update the node execution record.
	_, err := q.GetNodeExecutionByNodeID(ctx, sqlcgen.GetNodeExecutionByNodeIDParams{
		ExecutionID: execID,
		NodeID:      nodeUUID,
		TenantID:    tid,
	})
	if err != nil {
		// Record doesn't exist — create it.
		_, createErr := q.CreateNodeExecution(ctx, sqlcgen.CreateNodeExecutionParams{
			TenantID:    tid,
			ExecutionID: execID,
			NodeID:      nodeUUID,
			NodeType:    nodeType,
			Status:      string(result.Status),
			StartedAt:   now,
		})
		if createErr != nil {
			slog.ErrorContext(ctx, "workflow execute job: create node execution",
				"node_id", nodeID, "error", createErr)
			return
		}
		// Now get and update with output.
		ne, getErr := q.GetNodeExecutionByNodeID(ctx, sqlcgen.GetNodeExecutionByNodeIDParams{
			ExecutionID: execID,
			NodeID:      nodeUUID,
			TenantID:    tid,
		})
		if getErr != nil {
			slog.ErrorContext(ctx, "workflow execute job: get node execution after create",
				"node_id", nodeID, "error", getErr)
			return
		}
		if _, updateErr := q.UpdateNodeExecution(ctx, sqlcgen.UpdateNodeExecutionParams{
			ID:           ne.ID,
			TenantID:     tid,
			Status:       string(result.Status),
			Output:       outputJSON,
			ErrorMessage: result.Error,
			CompletedAt:  now,
		}); updateErr != nil {
			slog.ErrorContext(ctx, "workflow execute job: update node execution after create",
				"node_id", nodeID, "execution_id", ne.ID, "error", updateErr)
		}
	} else {
		// Record exists — this is a resumed execution, update it.
		ne, getErr := q.GetNodeExecutionByNodeID(ctx, sqlcgen.GetNodeExecutionByNodeIDParams{
			ExecutionID: execID,
			NodeID:      nodeUUID,
			TenantID:    tid,
		})
		if getErr != nil {
			slog.ErrorContext(ctx, "workflow execute job: get node execution for update",
				"node_id", nodeID, "error", getErr)
			return
		}
		if _, updateErr := q.UpdateNodeExecution(ctx, sqlcgen.UpdateNodeExecutionParams{
			ID:           ne.ID,
			TenantID:     tid,
			Status:       string(result.Status),
			Output:       outputJSON,
			ErrorMessage: result.Error,
			CompletedAt:  now,
		}); updateErr != nil {
			slog.ErrorContext(ctx, "workflow execute job: update node execution",
				"node_id", nodeID, "execution_id", ne.ID, "error", updateErr)
		}
	}
}

// emitEvent publishes a domain event.
func (w *WorkflowExecuteWorker) emitEvent(ctx context.Context, eventType, resourceID, tenantID string, payload any) {
	if w.eventBus == nil {
		return
	}
	evt := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       eventType,
		TenantID:   tenantID,
		ActorID:    "system",
		ActorType:  domain.ActorSystem,
		Resource:   "workflow_execution",
		ResourceID: resourceID,
		Action:     eventType,
		Payload:    payload,
		Timestamp:  time.Now(),
	}
	if err := w.eventBus.Emit(ctx, evt); err != nil {
		slog.ErrorContext(ctx, "workflow execute job: emit event",
			"event_type", eventType, "error", err)
	}
}

// contextInjectingHandler wraps a NodeHandler to inject execution metadata.
type contextInjectingHandler struct {
	inner       NodeHandler
	tenantID    string
	executionID string
	versionID   string
	workflowID  string
}

func (h *contextInjectingHandler) Execute(ctx context.Context, exec *ExecutionContext) (*NodeResult, error) {
	exec.TenantID = h.tenantID
	exec.ExecutionID = h.executionID
	exec.VersionID = h.versionID
	exec.WorkflowID = h.workflowID
	return h.inner.Execute(ctx, exec)
}

// restoreStringSlice converts []interface{} back to []string in the context map.
// This is needed after JSON round-trip (e.g., pause/resume serialization).
func restoreStringSlice(ctx map[string]any, key string) {
	val, ok := ctx[key]
	if !ok {
		return
	}
	if _, isStrSlice := val.([]string); isStrSlice {
		return
	}
	if arr, isAnySlice := val.([]any); isAnySlice {
		strs := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				strs = append(strs, s)
			}
		}
		ctx[key] = strs
	}
}
