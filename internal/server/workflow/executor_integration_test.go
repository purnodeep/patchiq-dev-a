package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// --- inline handler implementations (avoid importing handlers to prevent import cycle) ---

type inlineTriggerHandler struct{}

func (h *inlineTriggerHandler) Execute(_ context.Context, exec *ExecutionContext) (*NodeResult, error) {
	var cfg TriggerConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("trigger handler: unmarshal config: %w", err)
	}
	return &NodeResult{
		Status: NodeExecCompleted,
		Output: map[string]any{
			"trigger_type": cfg.TriggerType,
			"triggered_at": time.Now().UTC().Format(time.RFC3339),
		},
	}, nil
}

type inlineFilterHandler struct {
	endpoints []string
}

func (h *inlineFilterHandler) Execute(_ context.Context, exec *ExecutionContext) (*NodeResult, error) {
	if len(h.endpoints) == 0 {
		return &NodeResult{
			Status: NodeExecFailed,
			Error:  "no endpoints matched filter criteria",
			Output: map[string]any{"filtered_endpoint_count": 0},
		}, nil
	}
	return &NodeResult{
		Status: NodeExecCompleted,
		Output: map[string]any{
			"filtered_endpoints":      h.endpoints,
			"filtered_endpoint_count": len(h.endpoints),
		},
	}, nil
}

type inlineDecisionHandler struct{}

func (h *inlineDecisionHandler) Execute(_ context.Context, exec *ExecutionContext) (*NodeResult, error) {
	var cfg DecisionConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decision handler: unmarshal config: %w", err)
	}
	fieldVal, exists := exec.Context[cfg.Field]
	if !exists {
		return &NodeResult{
			Status: NodeExecCompleted,
			Output: map[string]any{"branch": "no", "reason": "field not found"},
		}, nil
	}
	matched := fmt.Sprintf("%v", fieldVal) > cfg.Value
	if cfg.Operator == "equals" {
		matched = fmt.Sprintf("%v", fieldVal) == cfg.Value
	}
	branch := "no"
	if matched {
		branch = "yes"
	}
	return &NodeResult{
		Status: NodeExecCompleted,
		Output: map[string]any{"branch": branch},
	}, nil
}

type inlineNotificationHandler struct {
	sentCount *int
}

func (h *inlineNotificationHandler) Execute(_ context.Context, _ *ExecutionContext) (*NodeResult, error) {
	*h.sentCount++
	return &NodeResult{
		Status: NodeExecCompleted,
		Output: map[string]any{"sent": true},
	}, nil
}

type inlineCompleteHandler struct{}

func (h *inlineCompleteHandler) Execute(_ context.Context, _ *ExecutionContext) (*NodeResult, error) {
	return &NodeResult{
		Status: NodeExecCompleted,
		Output: map[string]any{"completed": true},
	}, nil
}

type inlineApprovalHandler struct{}

func (h *inlineApprovalHandler) Execute(_ context.Context, _ *ExecutionContext) (*NodeResult, error) {
	return &NodeResult{
		Status: NodeExecRunning,
		Output: map[string]any{"approval_requested": true},
		Pause:  true,
	}, nil
}

// --- integration tests ---

func TestIntegration_FullDAGExecution(t *testing.T) {
	sentCount := 0

	h := map[NodeType]NodeHandler{
		NodeTrigger:      &inlineTriggerHandler{},
		NodeFilter:       &inlineFilterHandler{endpoints: []string{"ep-1", "ep-2", "ep-3"}},
		NodeDecision:     &inlineDecisionHandler{},
		NodeNotification: &inlineNotificationHandler{sentCount: &sentCount},
		NodeComplete:     &inlineCompleteHandler{},
	}

	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: mustJSON(TriggerConfig{TriggerType: "manual"})},
		{ID: "filter", NodeType: NodeFilter, Config: mustJSON(FilterConfig{OSTypes: []string{"linux"}})},
		{ID: "decision", NodeType: NodeDecision, Config: mustJSON(DecisionConfig{Field: "filtered_endpoint_count", Operator: "gt", Value: "0"})},
		{ID: "notify-yes", NodeType: NodeNotification, Config: mustJSON(NotificationConfig{Channel: "slack", Target: "#ops"})},
		{ID: "notify-no", NodeType: NodeNotification, Config: mustJSON(NotificationConfig{Channel: "email", Target: "admin@co.com"})},
		{ID: "complete", NodeType: NodeComplete, Config: mustJSON(CompleteConfig{})},
	}

	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "filter"},
		{ID: "e2", SourceNodeID: "filter", TargetNodeID: "decision"},
		{ID: "e3", SourceNodeID: "decision", TargetNodeID: "notify-yes", Label: "yes"},
		{ID: "e4", SourceNodeID: "decision", TargetNodeID: "notify-no", Label: "no"},
		{ID: "e5", SourceNodeID: "notify-yes", TargetNodeID: "complete"},
		{ID: "e6", SourceNodeID: "notify-no", TargetNodeID: "complete"},
	}

	executor := NewInMemoryExecutor(h)
	result, err := executor.Run(context.Background(), nodes, edges, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != ExecCompleted {
		t.Errorf("status = %q, want %q (error: %s)", result.Status, ExecCompleted, result.Error)
	}

	decisionResult := result.NodeResults["decision"]
	if branch, _ := decisionResult.Output["branch"].(string); branch != "yes" {
		t.Errorf("decision branch = %q, want yes", branch)
	}

	if result.NodeResults["notify-no"].Status != NodeExecSkipped {
		t.Errorf("notify-no status = %q, want skipped", result.NodeResults["notify-no"].Status)
	}

	if result.NodeResults["notify-yes"].Status != NodeExecCompleted {
		t.Errorf("notify-yes status = %q, want completed", result.NodeResults["notify-yes"].Status)
	}

	if sentCount != 1 {
		t.Errorf("notifications sent = %d, want 1", sentCount)
	}
}

func TestIntegration_DAGWithPause(t *testing.T) {
	h := map[NodeType]NodeHandler{
		NodeTrigger:  &inlineTriggerHandler{},
		NodeApproval: &inlineApprovalHandler{},
		NodeComplete: &inlineCompleteHandler{},
	}

	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: mustJSON(TriggerConfig{TriggerType: "manual"})},
		{ID: "approval", NodeType: NodeApproval, Config: mustJSON(ApprovalConfig{ApproverRoles: []string{"admin"}, TimeoutHours: 24})},
		{ID: "complete", NodeType: NodeComplete, Config: mustJSON(CompleteConfig{})},
	}

	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "approval"},
		{ID: "e2", SourceNodeID: "approval", TargetNodeID: "complete"},
	}

	executor := NewInMemoryExecutor(h)

	// Phase 1: run until pause.
	result, err := executor.Run(context.Background(), nodes, edges, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != ExecPaused {
		t.Errorf("status = %q, want %q", result.Status, ExecPaused)
	}
	if result.PausedAtNodeID != "approval" {
		t.Errorf("paused at = %q, want approval", result.PausedAtNodeID)
	}

	// Phase 2: resume after approval.
	completedNodes := map[string]bool{"trigger": true, "approval": true}
	resumeResult, err := executor.RunFrom(
		context.Background(), nodes, edges, result.Context,
		"approval", completedNodes,
	)
	if err != nil {
		t.Fatalf("unexpected error on resume: %v", err)
	}
	if resumeResult.Status != ExecCompleted {
		t.Errorf("resumed status = %q, want %q (error: %s)", resumeResult.Status, ExecCompleted, resumeResult.Error)
	}
	if resumeResult.NodeResults["complete"].Status != NodeExecCompleted {
		t.Errorf("complete status = %q, want completed", resumeResult.NodeResults["complete"].Status)
	}
}

func TestIntegration_NoBranchExecution(t *testing.T) {
	sentCount := 0

	h := map[NodeType]NodeHandler{
		NodeTrigger:      &inlineTriggerHandler{},
		NodeFilter:       &inlineFilterHandler{endpoints: nil},
		NodeDecision:     &inlineDecisionHandler{},
		NodeNotification: &inlineNotificationHandler{sentCount: &sentCount},
		NodeComplete:     &inlineCompleteHandler{},
	}

	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: mustJSON(TriggerConfig{TriggerType: "manual"})},
		{ID: "filter", NodeType: NodeFilter, Config: mustJSON(FilterConfig{OSTypes: []string{"linux"}})},
		{ID: "decision", NodeType: NodeDecision, Config: mustJSON(DecisionConfig{Field: "filtered_endpoint_count", Operator: "gt", Value: "0"})},
		{ID: "notify-yes", NodeType: NodeNotification, Config: mustJSON(NotificationConfig{Channel: "slack", Target: "#ops"})},
		{ID: "notify-no", NodeType: NodeNotification, Config: mustJSON(NotificationConfig{Channel: "email", Target: "admin@co.com"})},
		{ID: "complete", NodeType: NodeComplete, Config: mustJSON(CompleteConfig{})},
	}

	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "filter"},
		{ID: "e2", SourceNodeID: "filter", TargetNodeID: "decision"},
		{ID: "e3", SourceNodeID: "decision", TargetNodeID: "notify-yes", Label: "yes"},
		{ID: "e4", SourceNodeID: "decision", TargetNodeID: "notify-no", Label: "no"},
		{ID: "e5", SourceNodeID: "notify-yes", TargetNodeID: "complete"},
		{ID: "e6", SourceNodeID: "notify-no", TargetNodeID: "complete"},
	}

	executor := NewInMemoryExecutor(h)
	result, err := executor.Run(context.Background(), nodes, edges, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != ExecFailed {
		t.Errorf("status = %q, want %q (error: %s)", result.Status, ExecFailed, result.Error)
	}

	if sentCount != 0 {
		t.Errorf("notifications sent = %d, want 0", sentCount)
	}
}
