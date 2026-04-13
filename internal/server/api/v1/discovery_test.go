package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

type mockEnqueuer struct {
	lastTenantID string
	lastRepoName string
	returnID     string
	returnErr    error
}

func (m *mockEnqueuer) EnqueueDiscovery(_ context.Context, tenantID, repoName string) (string, error) {
	m.lastTenantID = tenantID
	m.lastRepoName = repoName
	return m.returnID, m.returnErr
}

func TestDiscoveryHandler_Trigger_Success(t *testing.T) {
	enq := &mockEnqueuer{returnID: "42"}
	h := NewDiscoveryHandler(enq)

	body := bytes.NewBufferString(`{"repo_name":"ubuntu-22.04-security"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/discovery/trigger", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Trigger(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}

	var resp triggerResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.JobID != "42" {
		t.Errorf("job_id = %q, want %q", resp.JobID, "42")
	}
	if resp.Status != "accepted" {
		t.Errorf("status = %q, want %q", resp.Status, "accepted")
	}
	if enq.lastRepoName != "ubuntu-22.04-security" {
		t.Errorf("repo_name = %q, want %q", enq.lastRepoName, "ubuntu-22.04-security")
	}
}

func TestDiscoveryHandler_Trigger_MissingTenant(t *testing.T) {
	enq := &mockEnqueuer{returnID: "1"}
	h := NewDiscoveryHandler(enq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/discovery/trigger", nil)
	rr := httptest.NewRecorder()
	h.Trigger(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestDiscoveryHandler_Trigger_EmptyBody(t *testing.T) {
	enq := &mockEnqueuer{returnID: "99"}
	h := NewDiscoveryHandler(enq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/discovery/trigger", nil)
	ctx := tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Trigger(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusAccepted)
	}
	if enq.lastRepoName != "" {
		t.Errorf("repo_name = %q, want empty", enq.lastRepoName)
	}
}

func TestDiscoveryHandler_Trigger_EnqueueError(t *testing.T) {
	enq := &mockEnqueuer{returnErr: fmt.Errorf("queue unavailable")}
	h := NewDiscoveryHandler(enq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/discovery/trigger", nil)
	ctx := tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000001")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.Trigger(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
