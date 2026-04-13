package workflow

import (
	"encoding/json"
	"errors"
	"testing"
)

// --- helpers ---

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func triggerNode(id string) Node {
	return Node{
		ID:       id,
		NodeType: NodeTrigger,
		Label:    "Trigger",
		Config:   mustJSON(TriggerConfig{TriggerType: "manual"}),
	}
}

func completeNode(id string) Node {
	return Node{
		ID:       id,
		NodeType: NodeComplete,
		Label:    "Complete",
		Config:   mustJSON(CompleteConfig{}),
	}
}

func filterNode(id string) Node {
	return Node{
		ID:       id,
		NodeType: NodeFilter,
		Label:    "Filter",
		Config:   mustJSON(FilterConfig{OSTypes: []string{"linux"}}),
	}
}

func decisionNode(id string) Node {
	return Node{
		ID:       id,
		NodeType: NodeDecision,
		Label:    "Decision",
		Config:   mustJSON(DecisionConfig{Field: "severity", Operator: "gt", Value: "7"}),
	}
}

func edge(src, tgt, label string) Edge {
	return Edge{
		ID:           src + "->" + tgt,
		SourceNodeID: src,
		TargetNodeID: tgt,
		Label:        label,
	}
}

func TestValidateWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		nodes          []Node
		edges          []Edge
		wantErr        bool
		wantViolations int
		wantCodes      []string
	}{
		{
			name: "valid linear workflow",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("f1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "f1", ""),
				edge("f1", "c1", ""),
			},
			wantErr: false,
		},
		{
			name: "valid branching workflow with decision",
			nodes: []Node{
				triggerNode("t1"),
				decisionNode("d1"),
				filterNode("f1"),
				filterNode("f2"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "d1", ""),
				edge("d1", "f1", "yes"),
				edge("d1", "f2", "no"),
				edge("f1", "c1", ""),
				edge("f2", "c1", ""),
			},
			wantErr: false,
		},
		{
			name: "missing trigger node",
			nodes: []Node{
				filterNode("f1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("f1", "c1", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"MISSING_TRIGGER"},
		},
		{
			name: "missing complete node",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("f1"),
			},
			edges: []Edge{
				edge("t1", "f1", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"MISSING_COMPLETE"},
		},
		{
			name: "multiple trigger nodes",
			nodes: []Node{
				triggerNode("t1"),
				triggerNode("t2"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "c1", ""),
				edge("t2", "c1", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"MULTIPLE_TRIGGERS"},
		},
		{
			name: "cycle detected",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("a"),
				filterNode("b"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "a", ""),
				edge("a", "b", ""),
				edge("b", "a", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"CYCLE_DETECTED"},
		},
		{
			name: "disconnected node unreachable from trigger",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("f1"),
				filterNode("orphan"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "f1", ""),
				edge("f1", "c1", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"UNREACHABLE_NODES"},
		},
		{
			name: "trigger with incoming edge creates cycle",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("f1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "f1", ""),
				edge("f1", "t1", ""),
				edge("f1", "c1", ""),
			},
			wantErr:   true,
			wantCodes: []string{"TRIGGER_HAS_INCOMING"},
		},
		{
			name: "complete with outgoing edge",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("f1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "f1", ""),
				edge("f1", "c1", ""),
				edge("c1", "f1", ""),
			},
			wantErr:   true,
			wantCodes: []string{"COMPLETE_HAS_OUTGOING"},
		},
		{
			name: "decision without yes/no labels",
			nodes: []Node{
				triggerNode("t1"),
				decisionNode("d1"),
				filterNode("f1"),
				filterNode("f2"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "d1", ""),
				edge("d1", "f1", "true"),
				edge("d1", "f2", "false"),
				edge("f1", "c1", ""),
				edge("f2", "c1", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"DECISION_EDGE_LABELS"},
		},
		{
			name: "decision with only 1 outgoing edge",
			nodes: []Node{
				triggerNode("t1"),
				decisionNode("d1"),
				filterNode("f1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "d1", ""),
				edge("d1", "f1", "yes"),
				edge("f1", "c1", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"DECISION_EDGE_COUNT"},
		},
		{
			name: "multiple violations reported together",
			nodes: []Node{
				filterNode("f1"),
			},
			edges:          nil,
			wantErr:        true,
			wantViolations: 2,
			wantCodes:      []string{"MISSING_TRIGGER", "MISSING_COMPLETE"},
		},
		{
			name: "multiple complete nodes",
			nodes: []Node{
				triggerNode("t1"),
				completeNode("c1"),
				completeNode("c2"),
			},
			edges: []Edge{
				edge("t1", "c1", ""),
				edge("t1", "c2", ""),
			},
			wantErr:        true,
			wantViolations: 1,
			wantCodes:      []string{"MULTIPLE_COMPLETES"},
		},
		{
			name: "duplicate node IDs",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("t1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "c1", ""),
			},
			wantErr:   true,
			wantCodes: []string{"DUPLICATE_NODE_ID"},
		},
		{
			name: "dangling edge references unknown source node",
			nodes: []Node{
				triggerNode("t1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "c1", ""),
				edge("ghost", "c1", ""),
			},
			wantErr:   true,
			wantCodes: []string{"DANGLING_EDGE"},
		},
		{
			name: "self-loop edge",
			nodes: []Node{
				triggerNode("t1"),
				filterNode("f1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "f1", ""),
				edge("f1", "f1", ""),
				edge("f1", "c1", ""),
			},
			wantErr:   true,
			wantCodes: []string{"SELF_LOOP"},
		},
		{
			name: "dangling edge references unknown target node",
			nodes: []Node{
				triggerNode("t1"),
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "c1", ""),
				edge("t1", "ghost", ""),
			},
			wantErr:   true,
			wantCodes: []string{"DANGLING_EDGE"},
		},
		{
			name: "invalid node config detected during validation",
			nodes: []Node{
				{ID: "t1", NodeType: NodeTrigger, Label: "Trigger", Config: mustJSON(TriggerConfig{TriggerType: "invalid"})},
				completeNode("c1"),
			},
			edges: []Edge{
				edge("t1", "c1", ""),
			},
			wantErr:   true,
			wantCodes: []string{"INVALID_NODE_CONFIG"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflow(tt.nodes, tt.edges)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}

			if tt.wantViolations > 0 && len(ve.Violations) != tt.wantViolations {
				t.Errorf("expected %d violation(s), got %d: %v", tt.wantViolations, len(ve.Violations), ve.Violations)
			}

			if len(tt.wantCodes) > 0 {
				gotCodes := make(map[string]bool, len(ve.Violations))
				for _, v := range ve.Violations {
					gotCodes[v.Code] = true
				}
				for _, code := range tt.wantCodes {
					if !gotCodes[code] {
						t.Errorf("expected violation code %q not found in %v", code, ve.Violations)
					}
				}
			}
		})
	}
}
