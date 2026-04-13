package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// NotificationQuerier defines the sqlc queries needed by NotificationHandler.
type NotificationQuerier interface {
	CreateNotificationChannel(ctx context.Context, arg sqlcgen.CreateNotificationChannelParams) (sqlcgen.NotificationChannel, error)
	GetNotificationChannel(ctx context.Context, arg sqlcgen.GetNotificationChannelParams) (sqlcgen.NotificationChannel, error)
	ListNotificationChannels(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, arg sqlcgen.UpdateNotificationChannelParams) (sqlcgen.NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, arg sqlcgen.DeleteNotificationChannelParams) (int64, error)
	UpsertNotificationPreference(ctx context.Context, arg sqlcgen.UpsertNotificationPreferenceParams) (sqlcgen.NotificationPreference, error)
	ListNotificationPreferences(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.NotificationPreference, error)
	ListNotificationHistory(ctx context.Context, arg sqlcgen.ListNotificationHistoryParams) ([]sqlcgen.NotificationHistory, error)
	CountNotificationHistory(ctx context.Context, arg sqlcgen.CountNotificationHistoryParams) (int64, error)
	GetNotificationChannelByType(ctx context.Context, arg sqlcgen.GetNotificationChannelByTypeParams) (sqlcgen.NotificationChannel, error)
	UpdateNotificationChannelTestResult(ctx context.Context, arg sqlcgen.UpdateNotificationChannelTestResultParams) error
	GetNotificationHistoryByID(ctx context.Context, arg sqlcgen.GetNotificationHistoryByIDParams) (sqlcgen.NotificationHistory, error)
	UpdateNotificationHistoryStatus(ctx context.Context, arg sqlcgen.UpdateNotificationHistoryStatusParams) error
	GetDigestConfig(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.NotificationDigestConfig, error)
	UpsertDigestConfig(ctx context.Context, arg sqlcgen.UpsertDigestConfigParams) (sqlcgen.NotificationDigestConfig, error)
}

// NotificationEnqueuer enqueues notification send jobs.
type NotificationEnqueuer interface {
	EnqueueNotification(ctx context.Context, args notify.SendJobArgs) error
}

// NotificationHandler serves notification REST API endpoints.
type NotificationHandler struct {
	q         NotificationQuerier
	pool      TxBeginner
	cryptoKey []byte
	sender    notify.Sender
	eventBus  domain.EventBus
	river     NotificationEnqueuer
}

// NewNotificationHandler creates a NotificationHandler.
func NewNotificationHandler(q NotificationQuerier, pool TxBeginner, cryptoKey []byte, sender notify.Sender, eventBus domain.EventBus, river NotificationEnqueuer) *NotificationHandler {
	if q == nil {
		panic("notifications: NewNotificationHandler called with nil querier")
	}
	if pool == nil {
		panic("notifications: NewNotificationHandler called with nil pool")
	}
	if len(cryptoKey) == 0 {
		panic("notifications: NewNotificationHandler called with empty cryptoKey")
	}
	if sender == nil {
		panic("notifications: NewNotificationHandler called with nil sender")
	}
	if eventBus == nil {
		panic("notifications: NewNotificationHandler called with nil eventBus")
	}
	if river == nil {
		panic("notifications: NewNotificationHandler called with nil river enqueuer")
	}
	return &NotificationHandler{q: q, pool: pool, cryptoKey: cryptoKey, sender: sender, eventBus: eventBus, river: river}
}

// --- Response Types ---

type channelResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type preferencesCategoryResponse struct {
	ID          string                    `json:"id"`
	Label       string                    `json:"label"`
	Description string                    `json:"description"`
	Events      []preferenceEventResponse `json:"events"`
}

type preferenceEventResponse struct {
	TriggerType    string `json:"trigger_type"`
	Label          string `json:"label"`
	EmailEnabled   bool   `json:"email_enabled"`
	SlackEnabled   bool   `json:"slack_enabled"`
	WebhookEnabled bool   `json:"webhook_enabled"`
	Urgency        string `json:"urgency"`
}

type channelStatusResponse struct {
	Type       string `json:"type"`
	Configured bool   `json:"configured"`
	ChannelID  string `json:"channel_id,omitempty"`
}

type preferencesResponse struct {
	Categories []preferencesCategoryResponse `json:"categories"`
	Channels   []channelStatusResponse       `json:"channels"`
}

type historyResponse struct {
	ID           string `json:"id"`
	TriggerType  string `json:"trigger_type"`
	Category     string `json:"category"`
	ChannelID    string `json:"channel_id,omitempty"`
	ChannelType  string `json:"channel_type,omitempty"`
	Recipient    string `json:"recipient,omitempty"`
	Subject      string `json:"subject,omitempty"`
	UserID       string `json:"user_id"`
	Status       string `json:"status"`
	Payload      any    `json:"payload,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	RetryCount   int32  `json:"retry_count"`
	CreatedAt    string `json:"created_at"`
}

type digestConfigResponse struct {
	Frequency    string `json:"frequency"`
	DeliveryTime string `json:"delivery_time"` // "HH:MM" UTC
	Format       string `json:"format"`
}

func toChannelResponse(ch sqlcgen.NotificationChannel) channelResponse {
	return channelResponse{
		ID:          uuidToString(ch.ID),
		Name:        ch.Name,
		ChannelType: ch.ChannelType,
		Enabled:     ch.Enabled,
		CreatedAt:   ch.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:   ch.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func toHistoryResponse(h sqlcgen.NotificationHistory) historyResponse {
	resp := historyResponse{
		ID:          h.ID,
		TriggerType: h.TriggerType,
		Category:    notify.CategoryForTrigger(h.TriggerType),
		ChannelID:   uuidToString(h.ChannelID),
		UserID:      h.UserID,
		Status:      h.Status,
		RetryCount:  h.RetryCount,
		Recipient:   h.Recipient,
		Subject:     h.Subject,
		CreatedAt:   h.CreatedAt.Time.Format(time.RFC3339),
	}
	if h.ChannelType.Valid {
		resp.ChannelType = h.ChannelType.String
	}
	if h.ErrorMessage.Valid {
		resp.ErrorMessage = h.ErrorMessage.String
	}
	if len(h.Payload) > 0 {
		var payload any
		if err := json.Unmarshal(h.Payload, &payload); err == nil {
			resp.Payload = payload
		}
	}
	return resp
}

// --- Channel CRUD ---

type createChannelRequest struct {
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Config      string `json:"config"`
}

// CreateChannel handles POST /api/v1/notifications/channels.
func (h *NotificationHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body createChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if body.ChannelType == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel_type is required")
		return
	}
	if body.Config == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "config is required")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	encrypted, err := crypto.Encrypt(h.cryptoKey, []byte(body.Config))
	if err != nil {
		slog.ErrorContext(ctx, "encrypt channel config", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt channel config")
		return
	}

	ch, err := h.q.CreateNotificationChannel(ctx, sqlcgen.CreateNotificationChannelParams{
		TenantID:        tid,
		Name:            body.Name,
		ChannelType:     body.ChannelType,
		ConfigEncrypted: encrypted,
		Enabled:         true,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create notification channel", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create notification channel")
		return
	}

	emitEvent(ctx, h.eventBus, events.ChannelCreated, "channel", uuidToString(ch.ID), tenantID, map[string]string{
		"name":         ch.Name,
		"channel_type": ch.ChannelType,
	})
	WriteJSON(w, http.StatusCreated, toChannelResponse(ch))
}

// ListChannels handles GET /api/v1/notifications/channels.
func (h *NotificationHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	channels, err := h.q.ListNotificationChannels(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list notification channels", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notification channels")
		return
	}

	resp := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		resp = append(resp, toChannelResponse(ch))
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetChannel handles GET /api/v1/notifications/channels/{id}.
func (h *NotificationHandler) GetChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid channel ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ch, err := h.q.GetNotificationChannel(ctx, sqlcgen.GetNotificationChannelParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found")
			return
		}
		slog.ErrorContext(ctx, "get notification channel", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification channel")
		return
	}

	WriteJSON(w, http.StatusOK, toChannelResponse(ch))
}

type updateChannelRequest struct {
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Config      string `json:"config"`
	Enabled     *bool  `json:"enabled"`
}

// UpdateChannel handles PUT /api/v1/notifications/channels/{id}.
func (h *NotificationHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid channel ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	// Fetch existing channel to support partial updates.
	existing, err := h.q.GetNotificationChannel(ctx, sqlcgen.GetNotificationChannelParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found")
			return
		}
		slog.ErrorContext(ctx, "get notification channel for update", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification channel")
		return
	}

	var body updateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	// Fall back to existing values for unset fields.
	name := body.Name
	if name == "" {
		name = existing.Name
	}
	channelType := body.ChannelType
	if channelType == "" {
		channelType = existing.ChannelType
	}
	enabled := existing.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	// Re-encrypt config only if provided, otherwise keep existing.
	configEncrypted := existing.ConfigEncrypted
	if body.Config != "" {
		configEncrypted, err = crypto.Encrypt(h.cryptoKey, []byte(body.Config))
		if err != nil {
			slog.ErrorContext(ctx, "encrypt channel config", "tenant_id", tenantID, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt channel config")
			return
		}
	}

	ch, err := h.q.UpdateNotificationChannel(ctx, sqlcgen.UpdateNotificationChannelParams{
		ID:              id,
		TenantID:        tid,
		Name:            name,
		ChannelType:     channelType,
		ConfigEncrypted: configEncrypted,
		Enabled:         enabled,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found")
			return
		}
		slog.ErrorContext(ctx, "update notification channel", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update notification channel")
		return
	}

	emitEvent(ctx, h.eventBus, events.ChannelUpdated, "channel", uuidToString(ch.ID), tenantID, map[string]string{
		"name":         ch.Name,
		"channel_type": ch.ChannelType,
	})
	WriteJSON(w, http.StatusOK, toChannelResponse(ch))
}

// DeleteChannel handles DELETE /api/v1/notifications/channels/{id}.
func (h *NotificationHandler) DeleteChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid channel ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rows, err := h.q.DeleteNotificationChannel(ctx, sqlcgen.DeleteNotificationChannelParams{ID: id, TenantID: tid})
	if err != nil {
		slog.ErrorContext(ctx, "delete notification channel", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete notification channel")
		return
	}
	if rows == 0 {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found")
		return
	}

	emitEvent(ctx, h.eventBus, events.ChannelDeleted, "channel", uuidToString(id), tenantID, nil)
	w.WriteHeader(http.StatusNoContent)
}

// TestChannel handles POST /api/v1/notifications/channels/{id}/test.
func (h *NotificationHandler) TestChannel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	id, err := scanUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid channel ID: not a valid UUID")
		return
	}
	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ch, err := h.q.GetNotificationChannel(ctx, sqlcgen.GetNotificationChannelParams{ID: id, TenantID: tid})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found")
			return
		}
		slog.ErrorContext(ctx, "get channel for test", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification channel")
		return
	}

	plaintext, err := crypto.Decrypt(h.cryptoKey, ch.ConfigEncrypted)
	if err != nil {
		slog.WarnContext(ctx, "decrypt channel config for test", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   "Channel configuration could not be decrypted. Please re-save the channel configuration and try again.",
		})
		return
	}

	if err := h.sender.Send(ctx, string(plaintext), "PatchIQ test notification"); err != nil {
		slog.WarnContext(ctx, "test notification send failed", "channel_id", chi.URLParam(r, "id"), "tenant_id", tenantID, "error", err)
		WriteJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error()})
		return
	}

	emitEvent(ctx, h.eventBus, events.ChannelTested, "channel", uuidToString(id), tenantID, nil)

	WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

// --- Preferences ---

var triggerLabels = map[string]string{
	notify.TriggerDeploymentStarted:        "Deployment Started",
	notify.TriggerDeploymentCompleted:      "Deployment Completed",
	notify.TriggerDeploymentFailed:         "Deployment Failed",
	notify.TriggerDeploymentRollback:       "Rollback Initiated",
	notify.TriggerComplianceEvalComplete:   "Framework Evaluation Complete",
	notify.TriggerComplianceControlFailed:  "Control Failed",
	notify.TriggerComplianceSLAApproaching: "SLA Approaching (72h)",
	notify.TriggerComplianceSLAOverdue:     "SLA Overdue",
	notify.TriggerCVECriticalDiscovered:    "Critical CVE Published (CVSS >= 9.0)",
	notify.TriggerCVEExploitDetected:       "Exploit Detected in Wild",
	notify.TriggerCVEKEVAdded:              "KEV Entry Added",
	notify.TriggerCVEPatchAvailable:        "Patch Available for Affected CVE",
	notify.TriggerAgentOffline:             "Agent Offline (> 30 min)",
	notify.TriggerSystemHubSyncFailed:      "Hub Sync Failed",
	notify.TriggerSystemLicenseExpiring:    "License Expiring (30-day warning)",
	notify.TriggerSystemScanCompleted:      "Scan Completed",
}

var categoryOrder = []struct {
	id    string
	label string
	desc  string
}{
	{"deployments", "Deployments", "Patch deployment lifecycle events"},
	{"compliance", "Compliance", "Compliance framework evaluation events"},
	{"security", "Security", "CVE and vulnerability events"},
	{"system", "System", "Agent and platform health events"},
}

// GetPreferences handles GET /api/v1/notifications/preferences.
func (h *NotificationHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	prefs, err := h.q.ListNotificationPreferences(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get preferences: list", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notification preferences")
		return
	}

	prefByTrigger := make(map[string]sqlcgen.NotificationPreference, len(prefs))
	for _, p := range prefs {
		prefByTrigger[p.TriggerType] = p
	}

	channels, err := h.q.ListNotificationChannels(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "get preferences: list channels", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notification channels")
		return
	}

	channelsByType := make(map[string]sqlcgen.NotificationChannel)
	for _, ch := range channels {
		if ch.Enabled {
			channelsByType[ch.ChannelType] = ch
		}
	}

	categories := make([]preferencesCategoryResponse, 0, len(categoryOrder))
	for _, cat := range categoryOrder {
		triggers := notify.TriggerCategories[cat.id]
		evs := make([]preferenceEventResponse, 0, len(triggers))
		for _, triggerType := range triggers {
			p, ok := prefByTrigger[triggerType]
			ev := preferenceEventResponse{
				TriggerType: triggerType,
				Label:       triggerLabels[triggerType],
				Urgency:     notify.DefaultUrgency[triggerType],
			}
			if ok {
				ev.EmailEnabled = p.EmailEnabled
				ev.SlackEnabled = p.SlackEnabled
				ev.WebhookEnabled = p.WebhookEnabled
				ev.Urgency = p.Urgency
			}
			evs = append(evs, ev)
		}
		categories = append(categories, preferencesCategoryResponse{
			ID:          cat.id,
			Label:       cat.label,
			Description: cat.desc,
			Events:      evs,
		})
	}

	chanStatuses := make([]channelStatusResponse, 0, 3)
	for _, ct := range []string{"email", "slack", "webhook"} {
		cs := channelStatusResponse{Type: ct}
		if ch, ok := channelsByType[ct]; ok {
			cs.Configured = true
			cs.ChannelID = uuidToString(ch.ID)
		}
		chanStatuses = append(chanStatuses, cs)
	}

	WriteJSON(w, http.StatusOK, preferencesResponse{
		Categories: categories,
		Channels:   chanStatuses,
	})
}

// UpdatePreferences handles PUT /api/v1/notifications/preferences.
func (h *NotificationHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body struct {
		Preferences []struct {
			TriggerType    string `json:"trigger_type"`
			EmailEnabled   bool   `json:"email_enabled"`
			SlackEnabled   bool   `json:"slack_enabled"`
			WebhookEnabled bool   `json:"webhook_enabled"`
			Urgency        string `json:"urgency"`
		} `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	for _, p := range body.Preferences {
		if !notify.IsValidTrigger(p.TriggerType) {
			WriteError(w, http.StatusBadRequest, "INVALID_TRIGGER", fmt.Sprintf("invalid trigger type: %s", p.TriggerType))
			return
		}
		urgency := p.Urgency
		if urgency != "immediate" && urgency != "digest" {
			urgency = notify.DefaultUrgency[p.TriggerType]
		}
		if _, err := h.q.UpsertNotificationPreference(ctx, sqlcgen.UpsertNotificationPreferenceParams{
			TenantID:       tid,
			UserID:         "system",
			TriggerType:    p.TriggerType,
			EmailEnabled:   p.EmailEnabled,
			SlackEnabled:   p.SlackEnabled,
			WebhookEnabled: p.WebhookEnabled,
			Urgency:        urgency,
		}); err != nil {
			slog.ErrorContext(ctx, "update preferences: upsert", "trigger", p.TriggerType, "error", err)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to upsert notification preference")
			return
		}
	}

	emitEvent(ctx, h.eventBus, events.NotificationPreferencesUpdated, "notification_preferences", tenantID, tenantID, map[string]any{
		"count": len(body.Preferences),
	})

	w.WriteHeader(http.StatusNoContent)
}

// --- History ---

// ListHistory handles GET /api/v1/notifications/history.
func (h *NotificationHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	limit := ParseLimit(r.URL.Query().Get("limit"))
	cursor := r.URL.Query().Get("cursor")
	triggerType := r.URL.Query().Get("trigger_type")
	status := r.URL.Query().Get("status")
	channelType := r.URL.Query().Get("channel_type")
	category := r.URL.Query().Get("category")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	var cursorID pgtype.Text
	if cursor != "" {
		cursorID = pgtype.Text{String: cursor, Valid: true}
	}

	var triggerFilter pgtype.Text
	if triggerType != "" {
		triggerFilter = pgtype.Text{String: triggerType, Valid: true}
	}

	var statusFilter pgtype.Text
	if status != "" {
		statusFilter = pgtype.Text{String: status, Valid: true}
	}

	var channelTypeFilter pgtype.Text
	if channelType != "" {
		channelTypeFilter = pgtype.Text{String: channelType, Valid: true}
	}

	var fromDate pgtype.Timestamptz
	if from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "invalid from date: expected RFC3339 format")
			return
		}
		fromDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	var toDate pgtype.Timestamptz
	if to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PARAM", "invalid to date: expected RFC3339 format")
			return
		}
		toDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	history, err := h.q.ListNotificationHistory(ctx, sqlcgen.ListNotificationHistoryParams{
		TenantID:    tid,
		Limit:       limit,
		TriggerType: triggerFilter,
		Status:      statusFilter,
		ChannelType: channelTypeFilter,
		FromDate:    fromDate,
		ToDate:      toDate,
		CursorID:    cursorID,
	})
	if err != nil {
		slog.ErrorContext(ctx, "list notification history", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list notification history")
		return
	}

	total, err := h.q.CountNotificationHistory(ctx, sqlcgen.CountNotificationHistoryParams{
		TenantID:    tid,
		TriggerType: triggerFilter,
		Status:      statusFilter,
		ChannelType: channelTypeFilter,
		FromDate:    fromDate,
		ToDate:      toDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "count notification history", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count notification history")
		return
	}

	if category != "" {
		categoryTriggers := notify.TriggerCategories[category]
		if len(categoryTriggers) > 0 {
			allowed := make(map[string]bool, len(categoryTriggers))
			for _, t := range categoryTriggers {
				allowed[t] = true
			}
			filtered := history[:0]
			for _, e := range history {
				if allowed[e.TriggerType] {
					filtered = append(filtered, e)
				}
			}
			history = filtered
		}
	}

	resp := make([]historyResponse, 0, len(history))
	for _, h := range history {
		resp = append(resp, toHistoryResponse(h))
	}

	var nextCursor string
	if len(history) == int(limit) {
		nextCursor = history[len(history)-1].ID
	}

	WriteList(w, resp, nextCursor, total)
}

// --- Retry ---

const maxRetryCount = 3

// RetryNotification handles POST /api/v1/notifications/history/{id}/retry.
func (h *NotificationHandler) RetryNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	id := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	entry, err := h.q.GetNotificationHistoryByID(ctx, sqlcgen.GetNotificationHistoryByIDParams{
		ID:       id,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification not found")
			return
		}
		slog.ErrorContext(ctx, "retry notification: get history", "id", id, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification history")
		return
	}

	if entry.Status != "failed" {
		http.Error(w, fmt.Sprintf("cannot retry notification with status %q: only failed notifications can be retried", entry.Status), http.StatusUnprocessableEntity)
		return
	}

	if entry.RetryCount >= maxRetryCount {
		http.Error(w, fmt.Sprintf("max retries (%d) exceeded", maxRetryCount), http.StatusUnprocessableEntity)
		return
	}

	if !entry.ChannelID.Valid {
		http.Error(w, "notification has no associated channel", http.StatusUnprocessableEntity)
		return
	}

	ch, err := h.q.GetNotificationChannel(ctx, sqlcgen.GetNotificationChannelParams{
		ID:       entry.ChannelID,
		TenantID: tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "retry notification: get channel", "channel_id", entry.ChannelID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification channel")
		return
	}

	configJSON, err := crypto.Decrypt(h.cryptoKey, ch.ConfigEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "retry notification: decrypt channel config", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to decrypt channel config")
		return
	}

	var cfg map[string]string
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		slog.ErrorContext(ctx, "retry notification: parse channel config", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to parse channel config")
		return
	}

	shoutrrrURL, ok := cfg["url"]
	if !ok {
		WriteError(w, http.StatusInternalServerError, "CONFIG_ERROR", "channel has no Shoutrrr URL configured")
		return
	}

	if err := h.river.EnqueueNotification(ctx, notify.SendJobArgs{
		TenantID:    tenantID,
		UserID:      entry.UserID,
		TriggerType: entry.TriggerType,
		ChannelID:   uuidToString(entry.ChannelID),
		ChannelType: ch.ChannelType,
		Recipient:   entry.Recipient,
		Subject:     entry.Subject,
		ShoutrrrURL: shoutrrrURL,
		Message:     notify.FormatMessage(entry.TriggerType, nil),
	}); err != nil {
		slog.ErrorContext(ctx, "retry notification: enqueue", "id", id, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to enqueue retry job")
		return
	}

	if err := h.q.UpdateNotificationHistoryStatus(ctx, sqlcgen.UpdateNotificationHistoryStatusParams{
		ID:       id,
		TenantID: tid,
		Status:   "pending",
	}); err != nil {
		slog.ErrorContext(ctx, "retry notification: update status", "id", id, "error", err)
		// Don't fail — job is already enqueued
	}

	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "pending"})
}

// --- Digest Config ---

// GetDigestConfig handles GET /api/v1/notifications/digest-config.
func (h *NotificationHandler) GetDigestConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	cfg, err := h.q.GetDigestConfig(ctx, tid)
	if err != nil {
		if isNotFound(err) {
			WriteJSON(w, http.StatusOK, digestConfigResponse{
				Frequency:    "daily",
				DeliveryTime: "09:00",
				Format:       "html",
			})
			return
		}
		slog.ErrorContext(ctx, "get digest config", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get digest config")
		return
	}

	totalSecs := cfg.DeliveryTime.Microseconds / 1_000_000
	deliveryTimeStr := fmt.Sprintf("%02d:%02d", totalSecs/3600, (totalSecs%3600)/60)

	WriteJSON(w, http.StatusOK, digestConfigResponse{
		Frequency:    cfg.Frequency,
		DeliveryTime: deliveryTimeStr,
		Format:       cfg.Format,
	})
}

// UpdateDigestConfig handles PUT /api/v1/notifications/digest-config.
func (h *NotificationHandler) UpdateDigestConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	var body struct {
		Frequency    string `json:"frequency"`
		DeliveryTime string `json:"delivery_time"`
		Format       string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Frequency != "daily" && body.Frequency != "weekly" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "frequency must be daily or weekly")
		return
	}
	if body.Format != "html" && body.Format != "plaintext" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "format must be html or plaintext")
		return
	}

	t, err := time.Parse("15:04", body.DeliveryTime)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "delivery_time must be HH:MM format")
		return
	}
	microseconds := int64(t.Hour())*3600*1_000_000 + int64(t.Minute())*60*1_000_000
	deliveryTime := pgtype.Time{Microseconds: microseconds, Valid: true}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	cfg, err := h.q.UpsertDigestConfig(ctx, sqlcgen.UpsertDigestConfigParams{
		TenantID:     tid,
		Frequency:    body.Frequency,
		DeliveryTime: deliveryTime,
		Format:       body.Format,
	})
	if err != nil {
		slog.ErrorContext(ctx, "update digest config", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update digest config")
		return
	}

	emitEvent(ctx, h.eventBus, events.DigestConfigUpdated, "notification_digest_config", tenantID, tenantID, map[string]any{
		"frequency": cfg.Frequency,
		"format":    cfg.Format,
	})

	WriteJSON(w, http.StatusOK, digestConfigResponse{
		Frequency:    cfg.Frequency,
		DeliveryTime: body.DeliveryTime,
		Format:       cfg.Format,
	})
}

// TestDigest handles POST /api/v1/notifications/digest/test.
func (h *NotificationHandler) TestDigest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	slog.InfoContext(ctx, "test digest requested")
	WriteJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
