package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePinger implements v1.Pinger for testing.
type fakePinger struct {
	err error
}

func (f *fakePinger) Ping(_ context.Context) error {
	return f.err
}

func TestNewHealthHandler_panics_on_nil_pinger(t *testing.T) {
	assert.PanicsWithValue(t, "health: NewHealthHandler called with nil pinger", func() {
		v1.NewHealthHandler(nil, time.Now(), "dev", nil)
	})
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		startTime      time.Time
		wantStatus     int
		wantBodyStatus string
	}{
		{
			name:           "returns 200 with uptime and version",
			version:        "1.2.3",
			startTime:      time.Now().Add(-5 * time.Second),
			wantStatus:     http.StatusOK,
			wantBodyStatus: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewHealthHandler(&fakePinger{}, tt.startTime, tt.version, nil)
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			h.Health(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var body map[string]any
			err := json.Unmarshal(rec.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBodyStatus, body["status"])
			assert.Contains(t, body, "uptime")
			assert.Equal(t, tt.version, body["version"])
		})
	}
}

func TestReady(t *testing.T) {
	tests := []struct {
		name           string
		pinger         v1.Pinger
		extraChecks    map[string]v1.CheckFunc
		wantStatus     int
		wantBodyStatus string
		wantChecks     map[string]string
	}{
		{
			name:           "all checks pass",
			pinger:         &fakePinger{},
			wantStatus:     http.StatusOK,
			wantBodyStatus: "ready",
			wantChecks:     map[string]string{"database": "ok"},
		},
		{
			name:           "database fails",
			pinger:         &fakePinger{err: errors.New("connection refused")},
			wantStatus:     http.StatusServiceUnavailable,
			wantBodyStatus: "unavailable",
			wantChecks:     map[string]string{"database": "error"},
		},
		{
			name:   "database ok but valkey fails",
			pinger: &fakePinger{},
			extraChecks: map[string]v1.CheckFunc{
				"valkey": func(_ context.Context) error {
					return errors.New("dial tcp: connection refused")
				},
			},
			wantStatus:     http.StatusServiceUnavailable,
			wantBodyStatus: "unavailable",
			wantChecks: map[string]string{
				"database": "ok",
				"valkey":   "error",
			},
		},
		{
			name:   "all checks pass with extras",
			pinger: &fakePinger{},
			extraChecks: map[string]v1.CheckFunc{
				"valkey":    func(_ context.Context) error { return nil },
				"river":     func(_ context.Context) error { return nil },
				"watermill": func(_ context.Context) error { return nil },
			},
			wantStatus:     http.StatusOK,
			wantBodyStatus: "ready",
			wantChecks: map[string]string{
				"database":  "ok",
				"valkey":    "ok",
				"river":     "ok",
				"watermill": "ok",
			},
		},
		{
			name:   "multiple checks fail",
			pinger: &fakePinger{err: errors.New("db down")},
			extraChecks: map[string]v1.CheckFunc{
				"valkey": func(_ context.Context) error {
					return errors.New("valkey down")
				},
			},
			wantStatus:     http.StatusServiceUnavailable,
			wantBodyStatus: "unavailable",
			wantChecks: map[string]string{
				"database": "error",
				"valkey":   "error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewHealthHandler(tt.pinger, time.Now(), "dev", tt.extraChecks)
			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()

			h.Ready(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var body map[string]any
			err := json.Unmarshal(rec.Body.Bytes(), &body)
			require.NoError(t, err)
			assert.Equal(t, tt.wantBodyStatus, body["status"])

			checks, ok := body["checks"].(map[string]any)
			require.True(t, ok, "response must contain checks map")
			for k, v := range tt.wantChecks {
				assert.Equal(t, v, checks[k], "check %q", k)
			}
			assert.Len(t, checks, len(tt.wantChecks))
		})
	}
}
