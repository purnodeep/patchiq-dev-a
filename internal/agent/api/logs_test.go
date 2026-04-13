package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockLogStore struct {
	entries []LogEntry
	cursor  string
	total   int64
	err     error
}

func (m *mockLogStore) ListLogs(_ context.Context, limit int, cursor string, level string) ([]LogEntry, string, int64, error) {
	if m.err != nil {
		return nil, "", 0, m.err
	}
	return m.entries, m.cursor, m.total, nil
}

func TestLogsHandler_List(t *testing.T) {
	src := "agent"
	tests := []struct {
		name       string
		query      string
		store      *mockLogStore
		wantStatus int
		wantTotal  int64
		wantLen    int
		wantErr    bool
	}{
		{
			name:  "success",
			query: "",
			store: &mockLogStore{
				entries: []LogEntry{
					{ID: "1", Level: "error", Message: "something failed", Source: &src, Timestamp: "2026-01-01T00:00:00Z"},
				},
				cursor: "next",
				total:  10,
			},
			wantStatus: http.StatusOK,
			wantTotal:  10,
			wantLen:    1,
		},
		{
			name:  "with level filter",
			query: "?level=warn&limit=25",
			store: &mockLogStore{
				entries: []LogEntry{},
				total:   0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:    "store error",
			query:   "",
			store:   &mockLogStore{err: fmt.Errorf("db error")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewLogsHandler(tt.store)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/logs"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			if tt.wantErr {
				if rec.Code != http.StatusInternalServerError {
					t.Fatalf("want status 500, got %d", rec.Code)
				}
				return
			}

			if rec.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, rec.Code)
			}

			var resp ListResponse
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode: %v", err)
			}

			data, ok := resp.Data.([]any)
			if !ok {
				t.Fatalf("Data is not []any: %T", resp.Data)
			}
			if len(data) != tt.wantLen {
				t.Fatalf("want %d items, got %d", tt.wantLen, len(data))
			}
			if resp.TotalCount != tt.wantTotal {
				t.Fatalf("want total %d, got %d", tt.wantTotal, resp.TotalCount)
			}
		})
	}
}
