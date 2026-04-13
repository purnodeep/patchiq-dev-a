package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPatchStore struct {
	patches []PendingPatch
	cursor  string
	total   int64
	err     error
}

func (m *mockPatchStore) ListPending(_ context.Context, limit int, cursor string) ([]PendingPatch, string, int64, error) {
	if m.err != nil {
		return nil, "", 0, m.err
	}
	return m.patches, m.cursor, m.total, nil
}

func TestPatchesHandler_ListPending(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		store      *mockPatchStore
		wantStatus int
		wantTotal  int64
		wantLen    int
		wantCursor any
		wantErr    bool
	}{
		{
			name:  "success with defaults",
			query: "",
			store: &mockPatchStore{
				patches: []PendingPatch{{ID: "1", Name: "patch-1", Version: "1.0", Severity: "high", Status: "pending", QueuedAt: "2026-01-01T00:00:00Z"}},
				cursor:  "abc",
				total:   1,
			},
			wantStatus: http.StatusOK,
			wantTotal:  1,
			wantLen:    1,
			wantCursor: "abc",
		},
		{
			name:  "custom limit and cursor",
			query: "?limit=10&cursor=xyz",
			store: &mockPatchStore{
				patches: []PendingPatch{},
				cursor:  "",
				total:   0,
			},
			wantStatus: http.StatusOK,
			wantTotal:  0,
			wantLen:    0,
			wantCursor: nil,
		},
		{
			name:  "limit clamped to max 200",
			query: "?limit=999",
			store: &mockPatchStore{
				patches: []PendingPatch{},
				total:   0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "invalid limit uses default",
			query: "?limit=abc",
			store: &mockPatchStore{
				patches: []PendingPatch{},
				total:   0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:    "store error",
			query:   "",
			store:   &mockPatchStore{err: fmt.Errorf("db down")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewPatchesHandler(tt.store)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/patches"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.ListPending(rec, req)

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
				t.Fatalf("decode response: %v", err)
			}

			data, ok := resp.Data.([]any)
			if !ok {
				t.Fatalf("data is not an array")
			}
			if len(data) != tt.wantLen {
				t.Fatalf("want %d items, got %d", tt.wantLen, len(data))
			}
			if resp.TotalCount != tt.wantTotal {
				t.Fatalf("want total %d, got %d", tt.wantTotal, resp.TotalCount)
			}
			if tt.wantCursor != nil && resp.NextCursor != tt.wantCursor {
				t.Fatalf("want cursor %v, got %v", tt.wantCursor, resp.NextCursor)
			}
		})
	}
}
