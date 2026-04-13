package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// AuthResult holds the result of a successful authentication against Zitadel.
type AuthResult struct {
	SessionID    string
	SessionToken string
}

// ZitadelClient communicates with Zitadel's v2 APIs for user authentication
// and management. All calls use a service account PAT for authorization.
type ZitadelClient struct {
	baseURL      string
	httpClient   *http.Client
	pat          string // Personal Access Token for patchiq-service
	clientID     string
	clientSecret string
}

// NewZitadelClient creates a new client that talks to Zitadel's v2 APIs.
// The baseURL should be the full scheme+host (e.g. "http://localhost:8085").
// The pat is a Personal Access Token for the patchiq-service service account.
func NewZitadelClient(baseURL string, pat string) *ZitadelClient {
	return &ZitadelClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		pat: pat,
	}
}

// SetOIDCCredentials configures the OIDC client credentials used for token
// exchange. These must be set before calling ExchangeToken.
func (c *ZitadelClient) SetOIDCCredentials(clientID, clientSecret string) {
	c.clientID = clientID
	c.clientSecret = clientSecret
}

// Authenticate verifies email+password via POST /v2/sessions.
// Returns the session ID and session token on success.
func (c *ZitadelClient) Authenticate(ctx context.Context, email, password string) (*AuthResult, error) {
	reqBody := map[string]any{
		"checks": map[string]any{
			"user": map[string]any{
				"loginName": email,
			},
			"password": map[string]any{
				"password": password,
			},
		},
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/v2/sessions", reqBody)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	var resp struct {
		SessionID    string `json:"sessionId"`
		SessionToken string `json:"sessionToken"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("authenticate: decode response: %w", err)
	}

	return &AuthResult{
		SessionID:    resp.SessionID,
		SessionToken: resp.SessionToken,
	}, nil
}

// CreateUser creates a human user in Zitadel via POST /v2/users/human.
// The email is marked as pre-verified (invite-based flow).
// Returns the Zitadel user ID on success.
func (c *ZitadelClient) CreateUser(ctx context.Context, email, name, password string) (string, error) {
	reqBody := map[string]any{
		"profile": map[string]any{
			"givenName":   name,
			"familyName":  name,
			"displayName": name,
		},
		"email": map[string]any{
			"email":      email,
			"isVerified": true,
		},
		"password": map[string]any{
			"password":       password,
			"changeRequired": false,
		},
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/v2/users/human", reqBody)
	if err != nil {
		return "", fmt.Errorf("create user: %w", err)
	}

	var resp struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("create user: decode response: %w", err)
	}

	return resp.UserID, nil
}

// ResetPassword triggers a password reset email via Zitadel.
// It first searches for the user by email, then calls the password_reset
// endpoint. Returns nil even if the email doesn't exist (prevents enumeration).
func (c *ZitadelClient) ResetPassword(ctx context.Context, email string) error {
	userID, err := c.searchUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	if userID == "" {
		// User not found — return nil to prevent email enumeration.
		slog.InfoContext(ctx, "zitadel: password reset requested for unknown email",
			"email", email,
		)
		return nil
	}

	path := fmt.Sprintf("/v2/users/%s/password_reset", userID)
	reqBody := map[string]any{
		"sendLink": map[string]any{},
	}

	if _, err := c.doJSON(ctx, http.MethodPost, path, reqBody); err != nil {
		return fmt.Errorf("reset password: trigger reset for user %s: %w", userID, err)
	}

	slog.InfoContext(ctx, "zitadel: password reset triggered",
		"user_id", userID,
	)
	return nil
}

// SessionInfo holds user info extracted from a Zitadel session.
type SessionInfo struct {
	UserID      string
	LoginName   string
	DisplayName string
	OrgID       string
}

// ExchangeToken fetches user info from the Zitadel session and returns it.
// The login handler uses this info to mint a locally-signed JWT.
func (c *ZitadelClient) ExchangeToken(ctx context.Context, sessionID string) (string, error) {
	info, err := c.GetSessionInfo(ctx, sessionID)
	if err != nil {
		return "", err
	}
	// Return the user ID as a simple token — the login handler will use
	// GetSessionInfo directly for the full user info.
	return info.UserID, nil
}

// GetSessionInfo fetches session details from Zitadel and extracts user info.
func (c *ZitadelClient) GetSessionInfo(ctx context.Context, sessionID string) (*SessionInfo, error) {
	path := fmt.Sprintf("/v2/sessions/%s", sessionID)
	body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get session info: %w", err)
	}

	var resp struct {
		Session struct {
			Factors struct {
				User struct {
					ID             string `json:"id"`
					LoginName      string `json:"loginName"`
					DisplayName    string `json:"displayName"`
					OrganizationID string `json:"organizationId"`
				} `json:"user"`
			} `json:"factors"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("get session info: decode: %w", err)
	}

	user := resp.Session.Factors.User
	if user.ID == "" {
		return nil, fmt.Errorf("get session info: no user ID in session")
	}

	return &SessionInfo{
		UserID:      user.ID,
		LoginName:   user.LoginName,
		DisplayName: user.DisplayName,
		OrgID:       user.OrganizationID,
	}, nil
}

// searchUserByEmail searches for a user by email via POST /v2/users.
// Returns the user ID if found, or empty string if not found.
func (c *ZitadelClient) searchUserByEmail(ctx context.Context, email string) (string, error) {
	reqBody := map[string]any{
		"queries": []map[string]any{
			{
				"emailQuery": map[string]any{
					"emailAddress": email,
					"method":       "TEXT_QUERY_METHOD_EQUALS",
				},
			},
		},
		"limit": 1,
	}

	body, err := c.doJSON(ctx, http.MethodPost, "/v2/users", reqBody)
	if err != nil {
		return "", fmt.Errorf("search user by email: %w", err)
	}

	var resp struct {
		Result []struct {
			UserID string `json:"userId"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("search user by email: decode response: %w", err)
	}

	if len(resp.Result) == 0 {
		return "", nil
	}

	return resp.Result[0].UserID, nil
}

// doJSON sends a JSON request to Zitadel and returns the response body.
// It uses the service account PAT for authorization.
func (c *ZitadelClient) doJSON(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.pat)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read response from %s %s: %w", method, path, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Try to extract Zitadel error message.
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("zitadel %s %s returned %d: %s", method, path, resp.StatusCode, errResp.Message)
		}
		return nil, fmt.Errorf("zitadel %s %s returned %d: %s", method, path, resp.StatusCode, string(body))
	}

	return body, nil
}
