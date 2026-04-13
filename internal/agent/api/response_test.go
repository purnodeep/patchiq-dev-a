package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		value      any
		wantStatus int
		wantBody   string
	}{
		{
			name:       "200 with map",
			status:     http.StatusOK,
			value:      map[string]string{"key": "value"},
			wantStatus: http.StatusOK,
			wantBody:   `{"key":"value"}`,
		},
		{
			name:   "201 with struct",
			status: http.StatusCreated,
			value: struct {
				ID int `json:"id"`
			}{ID: 42},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":42}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			WriteJSON(w, tt.status, tt.value)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want %q", ct, "application/json")
			}

			var got, want any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.wantBody), &want); err != nil {
				t.Fatalf("unmarshal want: %v", err)
			}
			gotStr, _ := json.Marshal(got)
			wantStr, _ := json.Marshal(want)
			if string(gotStr) != string(wantStr) {
				t.Errorf("body = %s, want %s", gotStr, wantStr)
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "name is required")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if body["code"] != "INVALID_INPUT" {
		t.Errorf("code = %v, want INVALID_INPUT", body["code"])
	}
	if body["message"] != "name is required" {
		t.Errorf("message = %v, want 'name is required'", body["message"])
	}
	details, ok := body["details"].([]any)
	if !ok {
		t.Fatalf("details is not an array: %T", body["details"])
	}
	if len(details) != 0 {
		t.Errorf("details length = %d, want 0", len(details))
	}
}

func TestWriteList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		data           any
		nextCursor     string
		totalCount     int64
		wantCursorNull bool
		wantCursor     string
	}{
		{
			name:           "with cursor",
			data:           []string{"a", "b"},
			nextCursor:     "abc123",
			totalCount:     10,
			wantCursorNull: false,
			wantCursor:     "abc123",
		},
		{
			name:           "empty cursor is null",
			data:           []string{"x"},
			nextCursor:     "",
			totalCount:     1,
			wantCursorNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			WriteList(w, tt.data, tt.nextCursor, tt.totalCount)

			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}

			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if tt.wantCursorNull {
				if body["next_cursor"] != nil {
					t.Errorf("next_cursor = %v, want null", body["next_cursor"])
				}
			} else {
				if body["next_cursor"] != tt.wantCursor {
					t.Errorf("next_cursor = %v, want %q", body["next_cursor"], tt.wantCursor)
				}
			}

			tc, ok := body["total_count"].(float64)
			if !ok {
				t.Fatalf("total_count is not a number: %T", body["total_count"])
			}
			if int64(tc) != tt.totalCount {
				t.Errorf("total_count = %v, want %d", body["total_count"], tt.totalCount)
			}
		})
	}
}
