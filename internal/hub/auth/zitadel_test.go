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
	}{
		{
			name: "success returns session ID and token",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/v2/sessions", r.URL.Path)
				assert.Equal(t, "Bearer test-pat", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

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
				})
			},
			wantErr: false,
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
			name: "server error returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"message": "internal server error",
				})
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

func TestZitadelClient_GetSessionInfo(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantErr     bool
		errContain  string
		wantUserID  string
		wantLogin   string
		wantDisplay string
		wantOrgID   string
	}{
		{
			name: "success returns all user fields",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/v2/sessions/sess-123", r.URL.Path)
				assert.Equal(t, "Bearer test-pat", r.Header.Get("Authorization"))

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"session": map[string]any{
						"factors": map[string]any{
							"user": map[string]any{
								"id":             "user-456",
								"loginName":      "alice@example.com",
								"displayName":    "Alice Smith",
								"organizationId": "org-789",
							},
						},
					},
				})
			},
			wantUserID:  "user-456",
			wantLogin:   "alice@example.com",
			wantDisplay: "Alice Smith",
			wantOrgID:   "org-789",
		},
		{
			name: "missing user ID in session returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"session": map[string]any{
						"factors": map[string]any{
							"user": map[string]any{
								"loginName": "alice@example.com",
								// no id field
							},
						},
					},
				})
			},
			wantErr:    true,
			errContain: "no user ID in session",
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
			assert.Equal(t, tt.wantLogin, info.LoginName)
			assert.Equal(t, tt.wantDisplay, info.DisplayName)
			assert.Equal(t, tt.wantOrgID, info.OrgID)
		})
	}
}
