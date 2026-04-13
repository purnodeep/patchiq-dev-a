package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// IAMSettingsQuerier is the DB interface required by IAMSettingsHandler.
type IAMSettingsQuerier interface {
	GetIAMSettings(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetIAMSettingsRow, error)
	UpsertIAMSettings(ctx context.Context, arg sqlcgen.UpsertIAMSettingsParams) (sqlcgen.IamSetting, error)
	UpdateIAMTestResult(ctx context.Context, arg sqlcgen.UpdateIAMTestResultParams) error
	ListRoleMappings(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.ListRoleMappingsRow, error)
}

// IAMSettingsHandler handles GET/PUT /api/v1/settings/iam.
type IAMSettingsHandler struct {
	q         IAMSettingsQuerier
	cryptoKey []byte
	eventBus  domain.EventBus
}

// NewIAMSettingsHandler creates a new IAM settings handler.
func NewIAMSettingsHandler(q IAMSettingsQuerier, cryptoKey []byte, eventBus domain.EventBus) *IAMSettingsHandler {
	if q == nil {
		panic("settings_iam: NewIAMSettingsHandler called with nil querier")
	}
	if len(cryptoKey) == 0 {
		panic("settings_iam: NewIAMSettingsHandler called with empty cryptoKey")
	}
	if eventBus == nil {
		panic("settings_iam: NewIAMSettingsHandler called with nil eventBus")
	}
	return &IAMSettingsHandler{q: q, cryptoKey: cryptoKey, eventBus: eventBus}
}

// iamSettingsResponse is the API response for IAM settings.
type iamSettingsResponse struct {
	ZitadelOrgID     string            `json:"zitadel_org_id"`
	SsoURL           string            `json:"sso_url"`
	ClientID         string            `json:"client_id"`
	UserSyncEnabled  bool              `json:"user_sync_enabled"`
	UserSyncInterval int32             `json:"user_sync_interval_minutes"`
	ConnectionStatus string            `json:"connection_status"`
	LastTestedAt     *time.Time        `json:"last_tested_at,omitempty"`
	RoleMappings     []roleMappingItem `json:"role_mappings"`
}

type roleMappingItem struct {
	ExternalRole  string `json:"external_role"`
	PatchIQRoleID string `json:"patchiq_role_id"`
	RoleName      string `json:"role_name"`
}

// Get returns the current IAM settings for the tenant.
func (h *IAMSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())
	slog.InfoContext(r.Context(), "iam settings get", "tenant_id", tid)

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "iam settings get: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.GetIAMSettings(r.Context(), pgTID)
	if err != nil {
		if isNotFound(err) {
			WriteJSON(w, http.StatusOK, iamSettingsResponse{
				ConnectionStatus: "unknown",
				RoleMappings:     []roleMappingItem{},
			})
			return
		}
		slog.ErrorContext(r.Context(), "iam settings get: query failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve IAM settings")
		return
	}

	// Decrypt and optionally mask client ID
	clientID := ""
	if len(row.ClientIDEncrypted) > 0 {
		decrypted, decErr := crypto.Decrypt(h.cryptoKey, row.ClientIDEncrypted)
		if decErr != nil {
			slog.WarnContext(r.Context(), "iam settings get: decrypt client_id failed (key rotation?), returning masked placeholder", "error", decErr)
			clientID = "re-enter client ID (encryption key changed)"
		} else {
			clientID = string(decrypted)
		}
	}

	reveal := r.URL.Query().Get("reveal") == "true"
	if !reveal && len(clientID) > 12 {
		clientID = clientID[:12] + "\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022"
	}

	// Connection status
	connStatus := "unknown"
	if row.LastTestStatus.Valid {
		switch row.LastTestStatus.String {
		case "success":
			connStatus = "connected"
		case "failure":
			connStatus = "error"
		default:
			connStatus = row.LastTestStatus.String
		}
	}

	var lastTestedAt *time.Time
	if row.LastTestedAt.Valid {
		t := row.LastTestedAt.Time
		lastTestedAt = &t
	}

	// Role mappings
	mappings, mappErr := h.q.ListRoleMappings(r.Context(), pgTID)
	if mappErr != nil {
		slog.ErrorContext(r.Context(), "iam settings get: list role mappings failed", "error", mappErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve role mappings")
		return
	}
	rmItems := make([]roleMappingItem, 0, len(mappings))
	for _, m := range mappings {
		rmItems = append(rmItems, roleMappingItem{
			ExternalRole:  m.ExternalRole,
			PatchIQRoleID: uuidToString(m.PatchiqRoleID),
			RoleName:      m.RoleName,
		})
	}

	WriteJSON(w, http.StatusOK, iamSettingsResponse{
		ZitadelOrgID:     row.ZitadelOrgID,
		SsoURL:           row.SsoUrl,
		ClientID:         clientID,
		UserSyncEnabled:  row.UserSyncEnabled,
		UserSyncInterval: row.UserSyncInterval,
		ConnectionStatus: connStatus,
		LastTestedAt:     lastTestedAt,
		RoleMappings:     rmItems,
	})
}

// iamUpdateRequest is the request body for updating IAM settings.
type iamUpdateRequest struct {
	ZitadelOrgID     string `json:"zitadel_org_id"`
	SsoURL           string `json:"sso_url"`
	ClientID         string `json:"client_id"`
	UserSyncEnabled  bool   `json:"user_sync_enabled"`
	UserSyncInterval int32  `json:"user_sync_interval_minutes"`
}

// Update updates the IAM settings for the tenant.
func (h *IAMSettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())

	var req iamUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "iam settings update: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	if req.UserSyncInterval < 1 || req.UserSyncInterval > 1440 {
		WriteError(w, http.StatusBadRequest, "INVALID_SYNC_INTERVAL", "user_sync_interval_minutes must be between 1 and 1440")
		return
	}

	if req.SsoURL != "" {
		if err := validateSsoURL(req.SsoURL); err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_SSO_URL", err.Error())
			return
		}
	}

	var encryptedClientID []byte
	if req.ClientID != "" {
		encryptedClientID, err = crypto.Encrypt(h.cryptoKey, []byte(req.ClientID))
		if err != nil {
			slog.ErrorContext(r.Context(), "iam settings update: encrypt client_id failed", "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt client ID")
			return
		}
	}

	result, err := h.q.UpsertIAMSettings(r.Context(), sqlcgen.UpsertIAMSettingsParams{
		TenantID:          pgTID,
		ZitadelOrgID:      req.ZitadelOrgID,
		UserSyncEnabled:   req.UserSyncEnabled,
		UserSyncInterval:  req.UserSyncInterval,
		SsoUrl:            req.SsoURL,
		ClientIDEncrypted: encryptedClientID,
	})
	if err != nil {
		slog.ErrorContext(r.Context(), "iam settings update: upsert failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update IAM settings")
		return
	}

	emitEvent(r.Context(), h.eventBus, events.SettingsIAMUpdated, "settings", tid, tid, map[string]any{
		"zitadel_org_id":    req.ZitadelOrgID,
		"sso_url":           req.SsoURL,
		"user_sync_enabled": req.UserSyncEnabled,
	})

	slog.InfoContext(r.Context(), "iam settings updated", "tenant_id", tid)
	WriteJSON(w, http.StatusOK, map[string]any{
		"zitadel_org_id":             result.ZitadelOrgID,
		"sso_url":                    result.SsoUrl,
		"user_sync_enabled":          result.UserSyncEnabled,
		"user_sync_interval_minutes": result.UserSyncInterval,
	})
}

// TestConnection tests the SSO/OIDC connection by fetching the discovery document.
func (h *IAMSettingsHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	tid := tenant.MustTenantID(r.Context())

	pgTID, err := scanUUID(tid)
	if err != nil {
		slog.ErrorContext(r.Context(), "iam test connection: invalid tenant id in context", "tenant_id", tid, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	row, err := h.q.GetIAMSettings(r.Context(), pgTID)
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusBadRequest, "NO_SSO_URL", "SSO URL not configured")
			return
		}
		slog.ErrorContext(r.Context(), "iam test connection: get settings failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve IAM settings")
		return
	}

	if row.SsoUrl == "" {
		WriteError(w, http.StatusBadRequest, "NO_SSO_URL", "SSO URL not configured")
		return
	}

	start := time.Now()
	discoveryURL := row.SsoUrl + "/.well-known/openid-configuration"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(discoveryURL)
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		slog.ErrorContext(r.Context(), "iam test connection: request failed", "url", discoveryURL, "error", err)
		h.saveTestResult(r.Context(), pgTID, "failure")
		emitEvent(r.Context(), h.eventBus, events.SettingsIAMConnectionTested, "settings", tid, tid, map[string]any{"success": false})
		WriteJSON(w, http.StatusOK, map[string]any{
			"success":    false,
			"latency_ms": latencyMs,
			"error":      "connection failed: could not reach SSO endpoint",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		slog.ErrorContext(r.Context(), "iam test connection: read response failed", "url", discoveryURL, "error", err)
		h.saveTestResult(r.Context(), pgTID, "failure")
		emitEvent(r.Context(), h.eventBus, events.SettingsIAMConnectionTested, "settings", tid, tid, map[string]any{"success": false})
		WriteJSON(w, http.StatusOK, map[string]any{
			"success":    false,
			"latency_ms": latencyMs,
			"error":      "connection failed: could not read SSO response",
		})
		return
	}

	var discovery map[string]any
	if err := json.Unmarshal(body, &discovery); err != nil {
		h.saveTestResult(r.Context(), pgTID, "failure")
		emitEvent(r.Context(), h.eventBus, events.SettingsIAMConnectionTested, "settings", tid, tid, map[string]any{"success": false})
		WriteJSON(w, http.StatusOK, map[string]any{
			"success":    false,
			"latency_ms": latencyMs,
			"error":      "invalid OIDC discovery response",
		})
		return
	}

	// Validate required fields
	for _, field := range []string{"issuer", "authorization_endpoint", "token_endpoint"} {
		if _, ok := discovery[field]; !ok {
			h.saveTestResult(r.Context(), pgTID, "failure")
			emitEvent(r.Context(), h.eventBus, events.SettingsIAMConnectionTested, "settings", tid, tid, map[string]any{"success": false})
			WriteJSON(w, http.StatusOK, map[string]any{
				"success":    false,
				"latency_ms": latencyMs,
				"error":      fmt.Sprintf("missing required field: %s", field),
			})
			return
		}
	}

	h.saveTestResult(r.Context(), pgTID, "success")
	emitEvent(r.Context(), h.eventBus, events.SettingsIAMConnectionTested, "settings", tid, tid, map[string]any{"success": true, "latency_ms": latencyMs})

	WriteJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"latency_ms": latencyMs,
	})
}

// validateSsoURL ensures the SSO URL is a valid HTTPS URL pointing to an external host.
func validateSsoURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("sso_url is not a valid URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("sso_url must use https scheme")
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("sso_url must include a hostname")
	}
	// Block internal/loopback addresses
	blocked := []string{"localhost", "127.0.0.1", "::1", "0.0.0.0"}
	lower := strings.ToLower(host)
	for _, b := range blocked {
		if lower == b {
			return fmt.Errorf("sso_url must not point to a local address")
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("sso_url must not point to a private or link-local address")
		}
	}
	return nil
}

func (h *IAMSettingsHandler) saveTestResult(ctx context.Context, tenantID pgtype.UUID, status string) {
	if err := h.q.UpdateIAMTestResult(ctx, sqlcgen.UpdateIAMTestResultParams{
		TenantID:       tenantID,
		LastTestStatus: pgtype.Text{String: status, Valid: true},
		LastTestedAt:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}); err != nil {
		slog.ErrorContext(ctx, "iam test connection: save result failed", "error", err)
	}
}
