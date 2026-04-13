package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// TagResolver resolves tags for a set of endpoints.
type TagResolver interface {
	ResolveTags(ctx context.Context, tenantID string, endpointIDs []string) (map[string][]string, error)
}

// TagGateHandler evaluates a tag expression against workflow context endpoints.
// Endpoints that match the expression pass through; non-matching are filtered out.
type TagGateHandler struct {
	resolver TagResolver
}

// NewTagGateHandler creates a new TagGateHandler.
func NewTagGateHandler(resolver TagResolver) *TagGateHandler {
	return &TagGateHandler{resolver: resolver}
}

// Execute evaluates the tag expression against the filtered endpoints.
// Passes if at least one endpoint matches, fails if none match.
func (h *TagGateHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.TagGateConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("tag gate handler: unmarshal config: %w", err)
	}

	endpoints, _ := exec.Context["filtered_endpoints"].([]string)
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints available for tag gate evaluation",
			Output: map[string]any{
				"tag_expression": cfg.TagExpression,
				"matched_count":  0,
			},
		}, nil
	}

	tagMap, err := h.resolver.ResolveTags(ctx, exec.TenantID, endpoints)
	if err != nil {
		return nil, fmt.Errorf("tag gate handler: resolve tags: %w", err)
	}

	var matched []string
	for _, epID := range endpoints {
		tags := tagMap[epID]
		if matchesTagExpression(tags, cfg.TagExpression) {
			matched = append(matched, epID)
		}
	}

	if len(matched) == 0 {
		slog.InfoContext(ctx, "tag gate handler: no endpoints matched expression",
			"node_id", exec.Node.ID, "tag_expression", cfg.TagExpression,
			"total_endpoints", len(endpoints))
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  fmt.Sprintf("no endpoints matched tag expression %q", cfg.TagExpression),
			Output: map[string]any{
				"tag_expression": cfg.TagExpression,
				"matched_count":  0,
				"total_count":    len(endpoints),
			},
		}, nil
	}

	// Update filtered_endpoints in context to only include matched ones.
	exec.Context["filtered_endpoints"] = matched

	slog.InfoContext(ctx, "tag gate handler: endpoints matched",
		"node_id", exec.Node.ID, "tag_expression", cfg.TagExpression,
		"matched_count", len(matched), "total_count", len(endpoints))

	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"tag_expression": cfg.TagExpression,
			"matched_count":  len(matched),
			"total_count":    len(endpoints),
		},
	}, nil
}

// matchesTagExpression checks if the endpoint's tags contain the expression tag.
// This is a simple contains-check; complex boolean expressions can be added in M3.
func matchesTagExpression(tags []string, expression string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, expression) {
			return true
		}
	}
	return false
}
