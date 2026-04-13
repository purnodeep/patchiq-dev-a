package workflow

import (
	"context"
	"encoding/json"
	"testing"
)

type stubHandler struct {
	output map[string]any
}

func (s *stubHandler) Execute(_ context.Context, _ *ExecutionContext) (*NodeResult, error) {
	return &NodeResult{
		Status: NodeExecCompleted,
		Output: s.output,
	}, nil
}

type pauseHandler struct{}

func (p *pauseHandler) Execute(_ context.Context, _ *ExecutionContext) (*NodeResult, error) {
	return &NodeResult{
		Status: NodeExecRunning,
		Output: map[string]any{"paused": true},
		Pause:  true,
	}, nil
}

type failNodeHandler struct {
	err string
}

func (f *failNodeHandler) Execute(_ context.Context, _ *ExecutionContext) (*NodeResult, error) {
	return &NodeResult{
		Status: NodeExecFailed,
		Error:  f.err,
	}, nil
}

func TestExecutor_LinearWorkflow(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "filter", NodeType: NodeFilter, Config: json.RawMessage(`{"os_types":["linux"]}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "filter"},
		{ID: "e2", SourceNodeID: "filter", TargetNodeID: "complete"},
	}

	handlers := map[NodeType]NodeHandler{
		NodeTrigger:  &stubHandler{output: map[string]any{"trigger_type": "manual"}},
		NodeFilter:   &stubHandler{output: map[string]any{"filtered_endpoint_count": 10}},
		NodeComplete: &stubHandler{},
	}

	exec := NewInMemoryExecutor(handlers)
	result, err := exec.Run(context.Background(), nodes, edges, map[string]any{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Status != ExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, ExecCompleted)
	}
	if len(result.NodeResults) != 3 {
		t.Errorf("node results = %d, want 3", len(result.NodeResults))
	}
}

func TestExecutor_PauseAtApproval(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "approval", NodeType: NodeApproval, Config: json.RawMessage(`{"approver_roles":["admin"],"timeout_hours":24}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "approval"},
		{ID: "e2", SourceNodeID: "approval", TargetNodeID: "complete"},
	}

	handlers := map[NodeType]NodeHandler{
		NodeTrigger:  &stubHandler{},
		NodeApproval: &pauseHandler{},
		NodeComplete: &stubHandler{},
	}

	exec := NewInMemoryExecutor(handlers)
	result, err := exec.Run(context.Background(), nodes, edges, map[string]any{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Status != ExecPaused {
		t.Errorf("status = %q, want %q", result.Status, ExecPaused)
	}
	if result.PausedAtNodeID != "approval" {
		t.Errorf("paused at = %q, want %q", result.PausedAtNodeID, "approval")
	}
	if len(result.NodeResults) != 2 {
		t.Errorf("node results = %d, want 2", len(result.NodeResults))
	}
}

func TestExecutor_DecisionBranching(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "decision", NodeType: NodeDecision, Config: json.RawMessage(`{"field":"count","operator":"gt","value":"5"}`)},
		{ID: "yes_path", NodeType: NodeNotification, Config: json.RawMessage(`{"channel":"slack","target":"#ops"}`)},
		{ID: "no_path", NodeType: NodeNotification, Config: json.RawMessage(`{"channel":"email","target":"admin@co.com"}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "decision"},
		{ID: "e2", SourceNodeID: "decision", TargetNodeID: "yes_path", Label: "yes"},
		{ID: "e3", SourceNodeID: "decision", TargetNodeID: "no_path", Label: "no"},
		{ID: "e4", SourceNodeID: "yes_path", TargetNodeID: "complete"},
		{ID: "e5", SourceNodeID: "no_path", TargetNodeID: "complete"},
	}

	decisionHandler := &stubHandler{output: map[string]any{"branch": "yes"}}
	handlers := map[NodeType]NodeHandler{
		NodeTrigger:      &stubHandler{},
		NodeDecision:     decisionHandler,
		NodeNotification: &stubHandler{},
		NodeComplete:     &stubHandler{},
	}

	exec := NewInMemoryExecutor(handlers)
	result, err := exec.Run(context.Background(), nodes, edges, map[string]any{"count": 10})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Status != ExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, ExecCompleted)
	}
	noResult, ok := result.NodeResults["no_path"]
	if !ok {
		t.Fatal("no_path result not found")
	}
	if noResult.Status != NodeExecSkipped {
		t.Errorf("no_path status = %q, want %q", noResult.Status, NodeExecSkipped)
	}
}

func TestExecutor_NodeFailure(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "filter", NodeType: NodeFilter, Config: json.RawMessage(`{"os_types":["linux"]}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "filter"},
		{ID: "e2", SourceNodeID: "filter", TargetNodeID: "complete"},
	}

	handlers := map[NodeType]NodeHandler{
		NodeTrigger:  &stubHandler{},
		NodeFilter:   &failNodeHandler{err: "no endpoints matched"},
		NodeComplete: &stubHandler{},
	}

	exec := NewInMemoryExecutor(handlers)
	result, err := exec.Run(context.Background(), nodes, edges, map[string]any{})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Status != ExecFailed {
		t.Errorf("status = %q, want %q", result.Status, ExecFailed)
	}
	if result.Error == "" {
		t.Error("expected non-empty error")
	}
}

func TestExecutor_ResumeAfterPause(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "approval", NodeType: NodeApproval, Config: json.RawMessage(`{"approver_roles":["admin"],"timeout_hours":24}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "approval"},
		{ID: "e2", SourceNodeID: "approval", TargetNodeID: "complete"},
	}

	handlers := map[NodeType]NodeHandler{
		NodeTrigger:  &stubHandler{},
		NodeApproval: &stubHandler{}, // On resume, approval acts as completed
		NodeComplete: &stubHandler{},
	}

	exec := NewInMemoryExecutor(handlers)
	completed := map[string]bool{"trigger": true, "approval": true}
	result, err := exec.RunFrom(context.Background(), nodes, edges, map[string]any{}, "approval", completed)
	if err != nil {
		t.Fatalf("RunFrom failed: %v", err)
	}
	if result.Status != ExecCompleted {
		t.Errorf("status = %q, want %q", result.Status, ExecCompleted)
	}
}
