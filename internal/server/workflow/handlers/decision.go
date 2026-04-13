package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// DecisionHandler evaluates a condition against the execution context
// and returns a "yes" or "no" branch in the output.
type DecisionHandler struct{}

// NewDecisionHandler creates a new DecisionHandler.
func NewDecisionHandler() *DecisionHandler { return &DecisionHandler{} }

// Execute evaluates the decision condition and returns the branch result.
func (h *DecisionHandler) Execute(_ context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.DecisionConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("decision handler: unmarshal config: %w", err)
	}

	fieldVal, exists := exec.Context[cfg.Field]
	if !exists {
		return &workflow.NodeResult{
			Status: workflow.NodeExecCompleted,
			Output: map[string]any{"branch": "no", "reason": "field not found in context"},
		}, nil
	}

	matched := evaluateCondition(fieldVal, cfg.Operator, cfg.Value)
	branch := "no"
	if matched {
		branch = "yes"
	}

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"branch":      branch,
			"field":       cfg.Field,
			"field_value": fmt.Sprintf("%v", fieldVal),
			"operator":    cfg.Operator,
			"compare_to":  cfg.Value,
		},
	}, nil
}

func evaluateCondition(fieldVal any, operator, value string) bool {
	fieldStr := fmt.Sprintf("%v", fieldVal)

	switch operator {
	case "equals":
		return fieldStr == value
	case "not_equals":
		return fieldStr != value
	case "in":
		for _, p := range strings.Split(value, ",") {
			if strings.TrimSpace(p) == fieldStr {
				return true
			}
		}
		return false
	case "gt":
		return compareNumeric(fieldVal, value) > 0
	case "lt":
		return compareNumeric(fieldVal, value) < 0
	default:
		return false
	}
}

func compareNumeric(fieldVal any, value string) int {
	var a float64
	switch v := fieldVal.(type) {
	case int:
		a = float64(v)
	case int64:
		a = float64(v)
	case float64:
		a = v
	default:
		var err error
		a, err = strconv.ParseFloat(fmt.Sprintf("%v", fieldVal), 64)
		if err != nil {
			return 0
		}
	}

	b, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}
