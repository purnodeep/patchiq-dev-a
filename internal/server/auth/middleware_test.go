package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

func TestUserMiddleware(t *testing.T) {
	handler := auth.UserMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := user.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("user ID not in context")
		}
		_, _ = w.Write([]byte(id))
	}))

	tests := []struct {
		name       string
		userID     string
		wantStatus int
		wantErr    string
	}{
		{"valid user ID", "user-123", http.StatusOK, ""},
		{"exactly max length user ID accepted", strings.Repeat("a", 128), http.StatusOK, ""},
		{"missing header returns 400", "", http.StatusBadRequest, "missing X-User-ID header"},
		{"too long user ID returns 400", strings.Repeat("a", 129), http.StatusBadRequest, "invalid X-User-ID header"},
		{"control char tab rejected", "user\t123", http.StatusBadRequest, "invalid X-User-ID header: contains control characters"},
		{"control char null rejected", "user\x00id", http.StatusBadRequest, "invalid X-User-ID header: contains control characters"},
		{"control char DEL rejected", "user\x7fid", http.StatusBadRequest, "invalid X-User-ID header: contains control characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.userID != "" {
				req.Header.Set("X-User-ID", tt.userID)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantErr != "" {
				if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
					t.Errorf("Content-Type = %q, want application/json", ct)
				}
				var body map[string]any
				if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if body["message"] != tt.wantErr {
					t.Errorf("message = %q, want %q", body["message"], tt.wantErr)
				}
				if body["code"] != "AUTH_ERROR" {
					t.Errorf("code = %q, want AUTH_ERROR", body["code"])
				}
			}

			if tt.wantStatus == http.StatusOK && tt.wantErr == "" {
				if rec.Body.String() != tt.userID {
					t.Errorf("body = %q, want %q", rec.Body.String(), tt.userID)
				}
			}
		})
	}
}

func TestRequirePermission(t *testing.T) {
	makeCtx := func(tenantID, userID string) context.Context {
		ctx := context.Background()
		ctx = tenant.WithTenantID(ctx, tenantID)
		ctx = user.WithUserID(ctx, userID)
		return ctx
	}

	t.Run("allows when permission matches", func(t *testing.T) {
		store := &mockPermissionStore{
			permissions: []auth.Permission{
				{Resource: "endpoints", Action: "read", Scope: "*"},
			},
		}
		eval := auth.NewEvaluator(store)
		handler := auth.RequirePermission(eval, "endpoints", "read")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(makeCtx("00000000-0000-0000-0000-000000000001", "user-1"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("denies when permission missing", func(t *testing.T) {
		store := &mockPermissionStore{permissions: []auth.Permission{}}
		eval := auth.NewEvaluator(store)
		handler := auth.RequirePermission(eval, "endpoints", "delete")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not be called")
			}),
		)

		req := httptest.NewRequest(http.MethodDelete, "/", nil)
		req = req.WithContext(makeCtx("00000000-0000-0000-0000-000000000001", "user-1"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}

		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["message"] == nil || body["message"] == "" {
			t.Error("expected message in body")
		}
		if body["code"] != "AUTH_ERROR" {
			t.Errorf("code = %q, want AUTH_ERROR", body["code"])
		}
	})

	t.Run("returns 500 on store error", func(t *testing.T) {
		store := &mockPermissionStore{err: fmt.Errorf("db down")}
		eval := auth.NewEvaluator(store)
		handler := auth.RequirePermission(eval, "endpoints", "read")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not be called")
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = req.WithContext(makeCtx("00000000-0000-0000-0000-000000000001", "user-1"))
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}

		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
	})

	t.Run("returns 401 when user identity missing from context", func(t *testing.T) {
		store := &mockPermissionStore{
			permissions: []auth.Permission{
				{Resource: "endpoints", Action: "read", Scope: "*"},
			},
		}
		eval := auth.NewEvaluator(store)
		handler := auth.RequirePermission(eval, "endpoints", "read")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not be called")
			}),
		)

		// Request with no user identity in context (no UserMiddleware applied)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}

		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}

		var body map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["message"] != "missing user identity" {
			t.Errorf("message = %q, want %q", body["message"], "missing user identity")
		}
		if body["code"] != "AUTH_ERROR" {
			t.Errorf("code = %q, want AUTH_ERROR", body["code"])
		}
	})
}
