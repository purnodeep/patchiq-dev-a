package workflow_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

func TestAllTemplatesAreValid(t *testing.T) {
	templates := workflow.AllTemplates()
	if len(templates) != 3 {
		t.Fatalf("expected 3 templates, got %d", len(templates))
	}

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			if tmpl.ID == "" {
				t.Error("template ID is empty")
			}
			if tmpl.Name == "" {
				t.Error("template Name is empty")
			}
			if tmpl.Description == "" {
				t.Error("template Description is empty")
			}
			if len(tmpl.Nodes) == 0 {
				t.Error("template has no nodes")
			}
			if len(tmpl.Edges) == 0 {
				t.Error("template has no edges")
			}

			if err := workflow.ValidateWorkflow(tmpl.Nodes, tmpl.Edges); err != nil {
				t.Errorf("template %q failed DAG validation: %v", tmpl.Name, err)
			}

			for _, node := range tmpl.Nodes {
				if err := workflow.ValidateNodeConfig(node.NodeType, node.Config); err != nil {
					t.Errorf("template %q node %q (%s) config invalid: %v", tmpl.Name, node.ID, node.NodeType, err)
				}
			}
		})
	}
}

func TestTemplateIDs(t *testing.T) {
	templates := workflow.AllTemplates()
	expectedIDs := []string{"critical-fast-track", "standard-approval-flow", "canary-deployment"}
	for i, tmpl := range templates {
		if tmpl.ID != expectedIDs[i] {
			t.Errorf("template[%d] ID = %q, want %q", i, tmpl.ID, expectedIDs[i])
		}
	}
}
