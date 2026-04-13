package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// JobEnqueuer abstracts River job insertion for discovery.
type JobEnqueuer interface {
	EnqueueDiscovery(ctx context.Context, tenantID, repoName string) (string, error)
}

// DiscoveryHandler handles discovery-related HTTP endpoints.
type DiscoveryHandler struct {
	enqueuer JobEnqueuer
}

// NewDiscoveryHandler creates a DiscoveryHandler with the given job enqueuer.
func NewDiscoveryHandler(enqueuer JobEnqueuer) *DiscoveryHandler {
	return &DiscoveryHandler{enqueuer: enqueuer}
}

type triggerRequest struct {
	RepoName string `json:"repo_name"`
}

type triggerResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// Trigger handles POST /api/v1/admin/discovery/trigger.
// It enqueues a patch discovery job and returns 202 Accepted.
func (h *DiscoveryHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, ok := tenant.TenantIDFromContext(ctx)
	if !ok {
		http.Error(w, `{"error":"missing tenant ID"}`, http.StatusBadRequest)
		return
	}

	var req triggerRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
	}

	jobID, err := h.enqueuer.EnqueueDiscovery(ctx, tenantID, req.RepoName)
	if err != nil {
		slog.ErrorContext(ctx, "discovery trigger: enqueue failed", "error", err)
		http.Error(w, `{"error":"failed to enqueue discovery job"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(triggerResponse{JobID: jobID, Status: "accepted"}); err != nil {
		slog.ErrorContext(ctx, "discovery trigger: encode response failed", "error", err)
	}
}
