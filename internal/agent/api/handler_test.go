package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type routerMockPatchStore struct{}

func (m routerMockPatchStore) ListPending(_ context.Context, _ int, _ string) ([]PendingPatch, string, int64, error) {
	return []PendingPatch{}, "", 0, nil
}

type routerMockHistoryStore struct{}

func (m routerMockHistoryStore) ListHistory(_ context.Context, _ int, _ string, _ string) ([]HistoryEntry, string, int64, error) {
	return []HistoryEntry{}, "", 0, nil
}

func (m routerMockHistoryStore) InsertHistory(_ context.Context, _ HistoryEntry) error {
	return nil
}

type routerMockLogStore struct{}

func (m routerMockLogStore) ListLogs(_ context.Context, _ int, _ string, _ string) ([]LogEntry, string, int64, error) {
	return []LogEntry{}, "", 0, nil
}

const testAPIKey = "test-secret-key"

func newTestRouter() http.Handler {
	return NewRouter(HandlerDeps{
		Status:  StaticStatusProvider(StatusInfo{AgentID: "test-agent", Hostname: "testhost"}),
		Patches: routerMockPatchStore{},
		History: routerMockHistoryStore{},
		Logs:    routerMockLogStore{},
		APIKey:  testAPIKey,
	})
}

func newTestRouterNoAuth() http.Handler {
	return NewRouter(HandlerDeps{
		Status:  StaticStatusProvider(StatusInfo{AgentID: "test-agent", Hostname: "testhost"}),
		Patches: routerMockPatchStore{},
		History: routerMockHistoryStore{},
		Logs:    routerMockLogStore{},
	})
}

func TestHealthEndpoint(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Fatalf("expected status=healthy, got %q", body["status"])
	}
}

func TestStatusRoute(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body StatusInfo
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AgentID != "test-agent" {
		t.Fatalf("expected agent_id=test-agent, got %q", body.AgentID)
	}
}

func TestPatchesPendingRoute(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/patches/pending", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body ListResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestHistoryRoute(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/history", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body ListResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestLogsRoute(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs", nil)
	req.Header.Set("Authorization", "Bearer "+testAPIKey)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body ListResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_WrongKey(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NoKeyConfigured(t *testing.T) {
	router := newTestRouterNoAuth()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when no API key configured, got %d", rec.Code)
	}
}

func TestHealthEndpoint_NoAuthRequired(t *testing.T) {
	router := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for health without auth, got %d", rec.Code)
	}
}
