package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthResult holds the result of a successful authentication against Zitadel.
type AuthResult struct {
	SessionID    string
	SessionToken string
}

// SessionInfo holds user info extracted from a Zitadel session.
type SessionInfo struct {
	UserID      string
	LoginName   string
	DisplayName string
	OrgID       string
}

// ZitadelClient communicates with Zitadel's v2 APIs for user authentication.
// All calls use a service account PAT for authorization.
type ZitadelClient struct {
	baseURL    string
	httpClient *http.Client
	pat        string
}

// NewZitadelClient creates a new client that talks to Zitadel's v2 APIs.
// The baseURL should be the full scheme+host (e.g. "http://localhost:8085").
// The pat is a Personal Access Token for the patchiq-hub service account.
func NewZitadelClient(baseURL, pat string) *ZitadelClient {
	return &ZitadelClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		pat: pat,
	}
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

// doJSON sends a JSON request to Zitadel and returns the response body.
// It uses the service account PAT for authorization.
func (c *ZitadelClient) doJSON(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		jsonBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
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
