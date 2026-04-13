package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/skenzeriq/patchiq/internal/server/workflow"
)

// FilterDataSource queries endpoints matching filter criteria.
type FilterDataSource interface {
	FilterEndpoints(ctx context.Context, tenantID string, cfg workflow.FilterConfig) ([]string, error)
}

// FilterHandler filters endpoints based on configured criteria.
type FilterHandler struct {
	ds FilterDataSource
}

func NewFilterHandler(ds FilterDataSource) *FilterHandler {
	return &FilterHandler{ds: ds}
}

func (h *FilterHandler) Execute(ctx context.Context, exec *workflow.ExecutionContext) (*workflow.NodeResult, error) {
	var cfg workflow.FilterConfig
	if err := json.Unmarshal(exec.Node.Config, &cfg); err != nil {
		return nil, fmt.Errorf("filter handler: unmarshal config: %w", err)
	}
	endpoints, err := h.ds.FilterEndpoints(ctx, exec.TenantID, cfg)
	if err != nil {
		return nil, fmt.Errorf("filter handler: query endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		return &workflow.NodeResult{
			Status: workflow.NodeExecFailed,
			Error:  "no endpoints matched filter criteria",
			Output: map[string]any{"filtered_endpoint_count": 0},
		}, nil
	}
	return &workflow.NodeResult{
		Status: workflow.NodeExecCompleted,
		Output: map[string]any{
			"filtered_endpoints":      endpoints,
			"filtered_endpoint_count": len(endpoints),
		},
	}, nil
}
