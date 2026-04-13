package workflow

import (
	"encoding/json"
	"testing"
)

func TestBuildGraph_SimpleLinear(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "filter", NodeType: NodeFilter, Config: json.RawMessage(`{"os_types":["linux"]}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "filter"},
		{ID: "e2", SourceNodeID: "filter", TargetNodeID: "complete"},
	}

	g, err := BuildGraph(nodes, edges)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}
	if g.TriggerNodeID != "trigger" {
		t.Errorf("TriggerNodeID = %q, want %q", g.TriggerNodeID, "trigger")
	}

	order := g.TopologicalOrder()
	if len(order) != 3 {
		t.Fatalf("TopologicalOrder len = %d, want 3", len(order))
	}
	if order[0] != "trigger" {
		t.Errorf("order[0] = %q, want %q", order[0], "trigger")
	}
	if order[2] != "complete" {
		t.Errorf("order[2] = %q, want %q", order[2], "complete")
	}
}

func TestBuildGraph_Decision(t *testing.T) {
	nodes := []Node{
		{ID: "trigger", NodeType: NodeTrigger, Config: json.RawMessage(`{"trigger_type":"manual"}`)},
		{ID: "decision", NodeType: NodeDecision, Config: json.RawMessage(`{"field":"count","operator":"gt","value":"5"}`)},
		{ID: "yes_node", NodeType: NodeNotification, Config: json.RawMessage(`{"channel":"slack","target":"#ops"}`)},
		{ID: "no_node", NodeType: NodeNotification, Config: json.RawMessage(`{"channel":"email","target":"admin@co.com"}`)},
		{ID: "complete", NodeType: NodeComplete, Config: json.RawMessage(`{}`)},
	}
	edges := []Edge{
		{ID: "e1", SourceNodeID: "trigger", TargetNodeID: "decision"},
		{ID: "e2", SourceNodeID: "decision", TargetNodeID: "yes_node", Label: "yes"},
		{ID: "e3", SourceNodeID: "decision", TargetNodeID: "no_node", Label: "no"},
		{ID: "e4", SourceNodeID: "yes_node", TargetNodeID: "complete"},
		{ID: "e5", SourceNodeID: "no_node", TargetNodeID: "complete"},
	}

	g, err := BuildGraph(nodes, edges)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	successors := g.Successors("decision")
	if len(successors) != 2 {
		t.Fatalf("decision successors = %d, want 2", len(successors))
	}

	yesID, noID := g.DecisionBranches("decision")
	if yesID != "yes_node" {
		t.Errorf("yes branch = %q, want %q", yesID, "yes_node")
	}
	if noID != "no_node" {
		t.Errorf("no branch = %q, want %q", noID, "no_node")
	}

	skipped := g.SkippedDescendants("decision", "no_node", "yes_node")
	if len(skipped) != 1 || skipped[0] != "no_node" {
		t.Errorf("skipped = %v, want [no_node]", skipped)
	}
}

func TestBuildGraph_NoTrigger(t *testing.T) {
	nodes := []Node{
		{ID: "filter", NodeType: NodeFilter, Config: json.RawMessage(`{"os_types":["linux"]}`)},
	}
	_, err := BuildGraph(nodes, nil)
	if err == nil {
		t.Fatal("expected error for missing trigger")
	}
}
