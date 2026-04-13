package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake IAM Querier ---

type fakeIAMQuerier struct {
	getResult    sqlcgen.GetIAMSettingsRow
	getErr       error
	upsertResult sqlcgen.IamSetting
	upsertErr    error
	updateErr    error
	roleMappings []sqlcgen.ListRoleMappingsRow
	roleMappErr  error

	// capture
	lastUpsert  sqlcgen.UpsertIAMSettingsParams
	lastTestRes sqlcgen.UpdateIAMTestResultParams
}

func (f *fakeIAMQuerier) GetIAMSettings(_ context.Context, _ pgtype.UUID) (sqlcgen.GetIAMSettingsRow, error) {
	return f.getResult, f.getErr
}

func (f *fakeIAMQuerier) UpsertIAMSettings(_ context.Context, arg sqlcgen.UpsertIAMSettingsParams) (sqlcgen.IamSetting, error) {
	f.lastUpsert = arg
	return f.upsertResult, f.upsertErr
}

func (f *fakeIAMQuerier) UpdateIAMTestResult(_ context.Context, arg sqlcgen.UpdateIAMTestResultParams) error {
	f.lastTestRes = arg
	return f.updateErr
}

func (f *fakeIAMQuerier) ListRoleMappings(_ context.Context, _ pgtype.UUID) ([]sqlcgen.ListRoleMappingsRow, error) {
	return f.roleMappings, f.roleMappErr
}

// --- Helpers ---

const iamTenantID = "00000000-0000-0000-0000-000000000001"

func iamWithTenant(req *http.Request) *http.Request {
	return req.WithContext(tenant.WithTenantID(req.Context(), iamTenantID))
}

func iamRequest(method, url string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	return iamWithTenant(req)
}

func newTestIAMHandler(q v1.IAMSettingsQuerier) *v1.IAMSettingsHandler {
	key := crypto.GenerateKey()
	return v1.NewIAMSettingsHandler(q, key, &fakeEventBus{})
}

// --- Tests ---

func TestIAMSettingsGet(t *testing.T) {
	key := crypto.GenerateKey()
	clientID := "patchiq-pm-c4f9a2b1e8d3-secret"
	encrypted, err := crypto.Encrypt(key, []byte(clientID))
	require.NoError(t, err)

	q := &fakeIAMQuerier{
		getResult: sqlcgen.GetIAMSettingsRow{
			ZitadelOrgID:      "org-acme-123",
			SsoUrl:            "http://localhost:8085",
			ClientIDEncrypted: encrypted,
			UserSyncEnabled:   true,
			UserSyncInterval:  15,
			LastTestStatus:    pgtype.Text{String: "success", Valid: true},
		},
		roleMappings: []sqlcgen.ListRoleMappingsRow{
			{ExternalRole: "admin", RoleName: "Administrator"},
		},
	}

	h := v1.NewIAMSettingsHandler(q, key, &fakeEventBus{})

	req := iamRequest(http.MethodGet, "/api/v1/settings/iam", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "org-acme-123", resp["zitadel_org_id"])
	assert.Equal(t, "http://localhost:8085", resp["sso_url"])
	// Client ID should be masked: first 12 chars + "••••••••"
	maskedClientID, _ := resp["client_id"].(string)
	assert.Equal(t, "patchiq-pm-c••••••••", maskedClientID)
	assert.Equal(t, "connected", resp["connection_status"])
}

func TestIAMSettingsGet_WithReveal(t *testing.T) {
	key := crypto.GenerateKey()
	clientID := "patchiq-pm-c4f9a2b1e8d3-secret"
	encrypted, err := crypto.Encrypt(key, []byte(clientID))
	require.NoError(t, err)

	q := &fakeIAMQuerier{
		getResult: sqlcgen.GetIAMSettingsRow{
			SsoUrl:            "http://localhost:8085",
			ClientIDEncrypted: encrypted,
		},
	}

	h := v1.NewIAMSettingsHandler(q, key, &fakeEventBus{})

	req := iamRequest(http.MethodGet, "/api/v1/settings/iam?reveal=true", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, clientID, resp["client_id"])
}

func TestIAMSettingsGet_NotFound(t *testing.T) {
	q := &fakeIAMQuerier{
		getErr: pgx.ErrNoRows,
	}

	h := newTestIAMHandler(q)

	req := iamRequest(http.MethodGet, "/api/v1/settings/iam", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "", resp["sso_url"])
	assert.Equal(t, "unknown", resp["connection_status"])
}

func TestIAMSettingsUpdate(t *testing.T) {
	q := &fakeIAMQuerier{
		upsertResult: sqlcgen.IamSetting{
			ZitadelOrgID:    "org-acme-123",
			SsoUrl:          "https://auth.example.com",
			UserSyncEnabled: true,
		},
	}

	key := crypto.GenerateKey()
	h := v1.NewIAMSettingsHandler(q, key, &fakeEventBus{})

	body := map[string]any{
		"zitadel_org_id":             "org-acme-123",
		"sso_url":                    "https://auth.example.com",
		"client_id":                  "patchiq-pm-secret-123",
		"user_sync_enabled":          true,
		"user_sync_interval_minutes": 15,
	}
	req := iamRequest(http.MethodPut, "/api/v1/settings/iam", body)
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Verify encrypted client_id was passed
	assert.NotNil(t, q.lastUpsert.ClientIDEncrypted)
	assert.Equal(t, "org-acme-123", q.lastUpsert.ZitadelOrgID)
	assert.Equal(t, "https://auth.example.com", q.lastUpsert.SsoUrl)

	// Verify we can decrypt the stored value
	decrypted, err := crypto.Decrypt(key, q.lastUpsert.ClientIDEncrypted)
	require.NoError(t, err)
	assert.Equal(t, "patchiq-pm-secret-123", string(decrypted))
}

func TestIAMTestConnection_Success(t *testing.T) {
	// Mock OIDC discovery endpoint
	oidcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 oidcServerURL(r),
			"authorization_endpoint": oidcServerURL(r) + "/authorize",
			"token_endpoint":         oidcServerURL(r) + "/token",
		})
	}))
	defer oidcServer.Close()

	q := &fakeIAMQuerier{
		getResult: sqlcgen.GetIAMSettingsRow{
			SsoUrl: oidcServer.URL,
		},
	}

	h := newTestIAMHandler(q)

	req := iamRequest(http.MethodPost, "/api/v1/settings/iam/test", nil)
	rec := httptest.NewRecorder()
	h.TestConnection(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, true, resp["success"])
	assert.Contains(t, resp, "latency_ms")
	// Verify test result was saved
	assert.Equal(t, "success", q.lastTestRes.LastTestStatus.String)
}

func TestIAMTestConnection_Failure(t *testing.T) {
	q := &fakeIAMQuerier{
		getResult: sqlcgen.GetIAMSettingsRow{
			SsoUrl: "http://127.0.0.1:1", // unreachable port
		},
	}

	h := newTestIAMHandler(q)

	req := iamRequest(http.MethodPost, "/api/v1/settings/iam/test", nil)
	rec := httptest.NewRecorder()
	h.TestConnection(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, false, resp["success"])
	assert.Contains(t, resp, "error")
}

func TestIAMSettingsUpdate_SSOURLValidation(t *testing.T) {
	tests := []struct {
		name   string
		ssoURL string
		want   int
	}{
		{"rejects HTTP scheme", "http://auth.example.com", http.StatusBadRequest},
		{"rejects localhost", "https://localhost/auth", http.StatusBadRequest},
		{"rejects 127.0.0.1", "https://127.0.0.1/auth", http.StatusBadRequest},
		{"rejects ::1", "https://[::1]/auth", http.StatusBadRequest},
		{"rejects private 10.x", "https://10.0.0.1/auth", http.StatusBadRequest},
		{"rejects private 172.16.x", "https://172.16.0.1/auth", http.StatusBadRequest},
		{"rejects private 192.168.x", "https://192.168.1.1/auth", http.StatusBadRequest},
		{"rejects link-local", "https://169.254.169.254/auth", http.StatusBadRequest},
		{"rejects 0.0.0.0", "https://0.0.0.0/auth", http.StatusBadRequest},
		{"accepts valid HTTPS URL", "https://auth.example.com", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeIAMQuerier{}
			key := crypto.GenerateKey()
			h := v1.NewIAMSettingsHandler(q, key, &fakeEventBus{})

			body := map[string]any{
				"sso_url":                    tt.ssoURL,
				"zitadel_org_id":             "org-1",
				"client_id":                  "secret",
				"user_sync_enabled":          false,
				"user_sync_interval_minutes": 60,
			}
			req := iamRequest(http.MethodPut, "/api/v1/settings/iam", body)
			rec := httptest.NewRecorder()
			h.Update(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestIAMSettingsUpdate_SyncIntervalBounds(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		want     int
	}{
		{"rejects 0", 0, http.StatusBadRequest},
		{"rejects negative", -1, http.StatusBadRequest},
		{"rejects above 1440", 1441, http.StatusBadRequest},
		{"accepts 1", 1, http.StatusOK},
		{"accepts 1440", 1440, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeIAMQuerier{}
			key := crypto.GenerateKey()
			h := v1.NewIAMSettingsHandler(q, key, &fakeEventBus{})

			body := map[string]any{
				"sso_url":                    "https://auth.example.com",
				"zitadel_org_id":             "org-1",
				"client_id":                  "secret",
				"user_sync_enabled":          true,
				"user_sync_interval_minutes": tt.interval,
			}
			req := iamRequest(http.MethodPut, "/api/v1/settings/iam", body)
			rec := httptest.NewRecorder()
			h.Update(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

// oidcServerURL extracts the base URL from the request (for the mock server).
func oidcServerURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
