package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockHistoryStore struct {
	entries []HistoryEntry
	cursor  string
	total   int64
	err     error
}

func (m *mockHistoryStore) ListHistory(_ context.Context, limit int, cursor string, dateRange string) ([]HistoryEntry, string, int64, error) {
	if m.err != nil {
		return nil, "", 0, m.err
	}
	return m.entries, m.cursor, m.total, nil
}

func (m *mockHistoryStore) InsertHistory(_ context.Context, _ HistoryEntry) error {
	return m.err
}

func TestHistoryHandler_List(t *testing.T) {
	errMsg := "timeout"
	tests := []struct {
		name       string
		query      string
		store      *mockHistoryStore
		wantStatus int
		wantTotal  int64
		wantLen    int
		wantErr    bool
	}{
		{
			name:  "success",
			query: "",
			store: &mockHistoryStore{
				entries: []HistoryEntry{
					{ID: "1", PatchName: "p1", PatchVersion: "1.0", Action: "install", Result: "failure", ErrorMessage: &errMsg, CompletedAt: "2026-01-01T00:00:00Z"},
				},
				cursor: "",
				total:  1,
			},
			wantStatus: http.StatusOK,
			wantTotal:  1,
			wantLen:    1,
		},
		{
			name:  "custom limit",
			query: "?limit=5&cursor=abc",
			store: &mockHistoryStore{
				entries: []HistoryEntry{},
				total:   0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:    "store error",
			query:   "",
			store:   &mockHistoryStore{err: fmt.Errorf("db error")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHistoryHandler(tt.store)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/history"+tt.query, nil)
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
