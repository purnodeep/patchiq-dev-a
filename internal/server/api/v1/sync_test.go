package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

type spyEventBus struct {
	emitted []domain.DomainEvent
}

func (s *spyEventBus) Emit(_ context.Context, evt domain.DomainEvent) error {
	s.emitted = append(s.emitted, evt)
	return nil
}

func (s *spyEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (s *spyEventBus) Close() error                                    { return nil }

func TestHubSyncHandler_TriggerSync(t *testing.T) {
	const tenantID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	tests := []struct {
		name       string
		hubHandler http.HandlerFunc
		wantStatus int
		wantError  bool
	}{
		{
			name: "successful sync from Hub",
			hubHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer test-key" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"entries":     []json.RawMessage{json.RawMessage(`{"id":"1"}`), json.RawMessage(`{"id":"2"}`)},
					"deleted_ids": []string{"old-1"},
					"server_time": "2026-03-04T00:00:00Z",
				})
			},
			wantStatus: http.StatusOK,
			wantError:  false,
		},
		{
			name:       "Hub unreachable",
			hubHandler: nil, // we'll use an invalid URL
			wantStatus: http.StatusBadGateway,
			wantError:  true,
		},
		{
			name: "Hub returns invalid JSON",
			hubHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, "not json{{{")
			},
			wantStatus: http.StatusBadGateway,
			wantError:  true,
		},
		{
			name: "Hub returns 401 bad API key",
			hubHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			},
			wantStatus: http.StatusBadGateway,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hubURL string
			if tt.hubHandler != nil {
				hub := httptest.NewServer(tt.hubHandler)
				defer hub.Close()
				hubURL = hub.URL
			} else {
				hubURL = "http://127.0.0.1:1" // unreachable
			}

			bus := &spyEventBus{}
			h := NewHubSyncHandler(hubURL, "test-key", bus, nil)
			h.client = &http.Client{}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/sync/hub?since=1970-01-01T00:00:00Z", nil)
			ctx := tenant.WithTenantID(req.Context(), tenantID)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			h.TriggerSync(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var body map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if tt.wantError {
				if _, ok := body["error"]; !ok {
					t.Fatal("expected error field in response")
				}
				return
			}

			// Successful case assertions
			if body["synced"] != float64(2) {
				t.Errorf("synced = %v, want 2", body["synced"])
			}
			if body["deleted"] != float64(1) {
				t.Errorf("deleted = %v, want 1", body["deleted"])
			}
			if body["server_time"] != "2026-03-04T00:00:00Z" {
				t.Errorf("server_time = %v, want 2026-03-04T00:00:00Z", body["server_time"])
			}

			// Verify event was emitted
			if len(bus.emitted) != 1 {
				t.Fatalf("emitted %d events, want 1", len(bus.emitted))
			}
			evt := bus.emitted[0]
			if evt.Type != "catalog.synced" {
				t.Errorf("event type = %q, want catalog.synced", evt.Type)
			}
			if evt.TenantID != tenantID {
				t.Errorf("event tenant = %q, want %q", evt.TenantID, tenantID)
			}
		})
	}
}
