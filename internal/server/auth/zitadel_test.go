package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZitadelClient_Authenticate(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
		wantUserID string
	}{
		{
			name: "success returns session token and user info",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v2/sessions", r.URL.Path)
				assert.Equal(t, "Bearer test-pat", r.Header.Get("Authorization"))

				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

				checks, ok := body["checks"].(map[string]any)
				require.True(t, ok, "checks should be an object")
				user, ok := checks["user"].(map[string]any)
				require.True(t, ok, "checks.user should be an object")
				assert.Equal(t, "alice@example.com", user["loginName"])
				pw, ok := checks["password"].(map[string]any)
				require.True(t, ok, "checks.password should be an object")
				assert.Equal(t, "correct-password", pw["password"])

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"sessionId":    "sess-123",
					"sessionToken": "tok-abc",
					"details": map[string]any{
						"resourceOwner": "org-456",
					},
				})
			},
			wantUserID: "",
			wantErr:    false,
		},
		{
			name: "invalid credentials returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    7,
					"message": "invalid credentials",
				})
			},
			wantErr:    true,
			errContain: "invalid credentials",
		},
		{
			name: "network error returns wrapped error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Close connection abruptly to simulate network error.
				hj, ok := w.(http.Hijacker)
				if ok {
					conn, _, _ := hj.Hijack()
					conn.Close()
					return
				}
				// Fallback: just return 500
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:    true,
			errContain: "authenticate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewZitadelClient(srv.URL, "test-pat")

			result, err := client.Authenticate(context.Background(), "alice@example.com", "correct-password")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "sess-123", result.SessionID)
			assert.Equal(t, "tok-abc", result.SessionToken)
		})
	}
}

func TestZitadelClient_CreateUser(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
		wantUserID string
	}{
		{
			name: "success returns user ID",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v2/users/human", r.URL.Path)
				assert.Equal(t, "Bearer test-pat", r.Header.Get("Authorization"))

				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

				profile, ok := body["profile"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "Alice Smith", profile["displayName"])
				assert.Equal(t, "Alice Smith", profile["givenName"])

				email, ok := body["email"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "alice@example.com", email["email"])
				assert.Equal(t, true, email["isVerified"])

				pw, ok := body["password"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "Str0ngP@ss!", pw["password"])

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"userId": "user-789",
				})
			},
			wantUserID: "user-789",
		},
		{
			name: "duplicate email returns conflict error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    6,
					"message": "user already exists",
				})
			},
			wantErr:    true,
			errContain: "user already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewZitadelClient(srv.URL, "test-pat")

			userID, err := client.CreateUser(context.Background(), "alice@example.com", "Alice Smith", "Str0ngP@ss!")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantUserID, userID)
		})
	}
}

func TestZitadelClient_ResetPassword(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
	}{
		{
			name: "success triggers password reset",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/users":
					// User search
					assert.Equal(t, http.MethodPost, r.Method)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"result": []map[string]any{
							{"userId": "user-111"},
						},
					})
				case "/v2/users/user-111/password_reset":
					// Password reset
					assert.Equal(t, http.MethodPost, r.Method)
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]any{})
				default:
					t.Fatalf("unexpected request to %s", r.URL.Path)
				}
			},
			wantErr: false,
		},
		{
			name: "user not found returns nil (prevents enumeration)",
			handler: func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/users":
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"result": []map[string]any{},
					})
				default:
					t.Fatalf("unexpected request to %s — should not reach password_reset for unknown user", r.URL.Path)
				}
			},
			wantErr: false,
		},
		{
			name: "search API error returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "internal error",
				})
			},
			wantErr:    true,
			errContain: "search user by email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewZitadelClient(srv.URL, "test-pat")

			err := client.ResetPassword(context.Background(), "alice@example.com")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestZitadelClient_GetSessionInfo(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantErr    bool
		errContain string
		wantUserID string
	}{
		{
			name: "success returns session info",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/v2/sessions/sess-123", r.URL.Path)

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"session": map[string]any{
						"factors": map[string]any{
							"user": map[string]any{
								"id":             "user-456",
								"loginName":      "admin@test.local",
								"displayName":    "Admin User",
								"organizationId": "org-789",
							},
						},
					},
				})
			},
			wantUserID: "user-456",
		},
		{
			name: "session not found returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":    5,
					"message": "session not found",
				})
			},
			wantErr:    true,
			errContain: "get session info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			client := NewZitadelClient(srv.URL, "test-pat")

			info, err := client.GetSessionInfo(context.Background(), "sess-123")
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContain)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantUserID, info.UserID)
			assert.Equal(t, "admin@test.local", info.LoginName)
			assert.Equal(t, "org-789", info.OrgID)
		})
	}
}
