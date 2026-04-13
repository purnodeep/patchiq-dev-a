package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatusHandler_Get(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		info     StatusInfo
		wantCode int
	}{
		{
			name: "all fields populated",
			info: StatusInfo{
				AgentID:          "agent-123",
				Hostname:         "web-01",
				OSFamily:         "linux",
				OSVersion:        "Ubuntu 24.04",
				AgentVersion:     "0.1.0",
				EnrollmentStatus: "enrolled",
				ServerURL:        "https://pm.example.com",
				LastHeartbeat:    &now,
				UptimeSeconds:    3600,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "null last_heartbeat",
			info: StatusInfo{
				AgentID:          "agent-456",
				Hostname:         "db-01",
				OSFamily:         "windows",
				OSVersion:        "11",
				AgentVersion:     "0.1.0",
				EnrollmentStatus: "pending",
				ServerURL:        "https://pm.example.com",
				LastHeartbeat:    nil,
				UptimeSeconds:    0,
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewStatusHandler(StaticStatusProvider(tt.info))
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)

			handler.Get(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status code: got %d, want %d", rec.Code, tt.wantCode)
			}

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Fatalf("content-type: got %q, want %q", ct, "application/json")
			}

			var got map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if got["agent_id"] != tt.info.AgentID {
				t.Errorf("agent_id: got %v, want %v", got["agent_id"], tt.info.AgentID)
			}
			if got["hostname"] != tt.info.Hostname {
				t.Errorf("hostname: got %v, want %v", got["hostname"], tt.info.Hostname)
			}
			if got["os_family"] != tt.info.OSFamily {
				t.Errorf("os_family: got %v, want %v", got["os_family"], tt.info.OSFamily)
			}
			if got["os_version"] != tt.info.OSVersion {
				t.Errorf("os_version: got %v, want %v", got["os_version"], tt.info.OSVersion)
			}
			if got["agent_version"] != tt.info.AgentVersion {
				t.Errorf("agent_version: got %v, want %v", got["agent_version"], tt.info.AgentVersion)
			}
			if got["enrollment_status"] != tt.info.EnrollmentStatus {
				t.Errorf("enrollment_status: got %v, want %v", got["enrollment_status"], tt.info.EnrollmentStatus)
			}
			if got["server_url"] != tt.info.ServerURL {
				t.Errorf("server_url: got %v, want %v", got["server_url"], tt.info.ServerURL)
			}

			// Check uptime_seconds as float64 (JSON number).
			wantUptime := float64(tt.info.UptimeSeconds)
			if got["uptime_seconds"] != wantUptime {
				t.Errorf("uptime_seconds: got %v, want %v", got["uptime_seconds"], wantUptime)
			}

			// Check last_heartbeat: null or RFC3339 string.
			if tt.info.LastHeartbeat == nil {
				if got["last_heartbeat"] != nil {
					t.Errorf("last_heartbeat: got %v, want nil", got["last_heartbeat"])
				}
			} else {
				want := tt.info.LastHeartbeat.Format(time.RFC3339)
				if got["last_heartbeat"] != want {
					t.Errorf("last_heartbeat: got %v, want %v", got["last_heartbeat"], want)
				}
			}
		})
	}
}
