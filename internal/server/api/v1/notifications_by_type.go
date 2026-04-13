package v1

import (
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

// NotificationByTypeHandler serves notification channel endpoints keyed by channel_type.
type NotificationByTypeHandler struct {
	q         NotificationQuerier
	cryptoKey []byte
	eventBus  domain.EventBus
	sender    notify.Sender
}

// NewNotificationByTypeHandler creates a NotificationByTypeHandler.
func NewNotificationByTypeHandler(q NotificationQuerier, cryptoKey []byte, eventBus domain.EventBus, sender notify.Sender) *NotificationByTypeHandler {
	if q == nil {
		panic("notifications_by_type: NewNotificationByTypeHandler called with nil querier")
	}
	if len(cryptoKey) == 0 {
		panic("notifications_by_type: NewNotificationByTypeHandler called with empty cryptoKey")
	}
	if eventBus == nil {
		panic("notifications_by_type: NewNotificationByTypeHandler called with nil eventBus")
	}
	if sender == nil {
		panic("notifications_by_type: NewNotificationByTypeHandler called with nil sender")
	}
	return &NotificationByTypeHandler{q: q, cryptoKey: cryptoKey, eventBus: eventBus, sender: sender}
}

// byTypeResponse is the API response for notification channels looked up by type, including test tracking fields.
type byTypeResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ChannelType    string  `json:"channel_type"`
	Enabled        bool    `json:"enabled"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	LastTestedAt   *string `json:"last_tested_at,omitempty"`
	LastTestStatus *string `json:"last_test_status,omitempty"`
}

func toByTypeResponse(ch sqlcgen.NotificationChannel) byTypeResponse {
	resp := byTypeResponse{
		ID:          uuidToString(ch.ID),
		Name:        ch.Name,
		ChannelType: ch.ChannelType,
		Enabled:     ch.Enabled,
		CreatedAt:   ch.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:   ch.UpdatedAt.Time.Format(time.RFC3339),
	}
	if ch.LastTestedAt.Valid {
		s := ch.LastTestedAt.Time.Format(time.RFC3339)
		resp.LastTestedAt = &s
	}
	if ch.LastTestStatus.Valid {
		resp.LastTestStatus = &ch.LastTestStatus.String
	}
	return resp
}

// GetByType handles GET /api/v1/notifications/channels/by-type/{type}.
func (h *NotificationByTypeHandler) GetByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	channelType := chi.URLParam(r, "type")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ch, err := h.q.GetNotificationChannelByType(ctx, sqlcgen.GetNotificationChannelByTypeParams{
		TenantID:    tid,
		ChannelType: channelType,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found for type: "+channelType)
			return
		}
		slog.ErrorContext(ctx, "get notification channel by type", "channel_type", channelType, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification channel by type")
		return
	}

	WriteJSON(w, http.StatusOK, toByTypeResponse(ch))
}

type upsertByTypeRequest struct {
	Name    string          `json:"name"`
	Config  json.RawMessage `json:"config"`
	Enabled bool            `json:"enabled"`
}

// UpdateByType handles PUT /api/v1/notifications/channels/by-type/{type}.
func (h *NotificationByTypeHandler) UpdateByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	channelType := chi.URLParam(r, "type")

	var body upsertByTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}
	if body.Name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required")
		return
	}
	if len(body.Config) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "config is required")
		return
	}

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	encrypted, err := crypto.Encrypt(h.cryptoKey, body.Config)
	if err != nil {
		slog.ErrorContext(ctx, "encrypt channel config", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to encrypt channel config")
		return
	}

	// Try to find existing channel by type
	existing, err := h.q.GetNotificationChannelByType(ctx, sqlcgen.GetNotificationChannelByTypeParams{
		TenantID:    tid,
		ChannelType: channelType,
	})
	if err != nil && !isNotFound(err) {
		slog.ErrorContext(ctx, "lookup channel by type", "channel_type", channelType, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to lookup notification channel")
		return
	}

	if isNotFound(err) {
		// Create new
		ch, createErr := h.q.CreateNotificationChannel(ctx, sqlcgen.CreateNotificationChannelParams{
			TenantID:        tid,
			Name:            body.Name,
			ChannelType:     channelType,
			ConfigEncrypted: encrypted,
			Enabled:         body.Enabled,
		})
		if createErr != nil {
			slog.ErrorContext(ctx, "create notification channel by type", "channel_type", channelType, "tenant_id", tenantID, "error", createErr)
			WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create notification channel")
			return
		}
		emitEvent(ctx, h.eventBus, events.ChannelCreated, "channel", uuidToString(ch.ID), tenantID, map[string]string{
			"name":         ch.Name,
			"channel_type": ch.ChannelType,
		})
		WriteJSON(w, http.StatusOK, toByTypeResponse(ch))
		return
	}

	// Update existing
	ch, err := h.q.UpdateNotificationChannel(ctx, sqlcgen.UpdateNotificationChannelParams{
		ID:              existing.ID,
		TenantID:        tid,
		Name:            body.Name,
		ChannelType:     channelType,
		ConfigEncrypted: encrypted,
		Enabled:         body.Enabled,
	})
	if err != nil {
		slog.ErrorContext(ctx, "update notification channel by type", "channel_type", channelType, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update notification channel")
		return
	}

	emitEvent(ctx, h.eventBus, events.ChannelUpdated, "channel", uuidToString(ch.ID), tenantID, map[string]string{
		"name":         ch.Name,
		"channel_type": ch.ChannelType,
	})
	WriteJSON(w, http.StatusOK, toByTypeResponse(ch))
}

// TestByType handles POST /api/v1/notifications/channels/by-type/{type}/test.
func (h *NotificationByTypeHandler) TestByType(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	channelType := chi.URLParam(r, "type")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ch, err := h.q.GetNotificationChannelByType(ctx, sqlcgen.GetNotificationChannelByTypeParams{
		TenantID:    tid,
		ChannelType: channelType,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification channel not found for type: "+channelType)
			return
		}
		slog.ErrorContext(ctx, "get channel for test by type", "channel_type", channelType, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get notification channel")
		return
	}

	plaintext, err := crypto.Decrypt(h.cryptoKey, ch.ConfigEncrypted)
	if err != nil {
		slog.ErrorContext(ctx, "decrypt channel config for test", "channel_type", channelType, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to decrypt channel config")
		return
	}

	now := pgtype.Timestamptz{Time: time.Now(), Valid: true}
	sendErr := h.sender.Send(ctx, string(plaintext), "PatchIQ test notification")

	status := "success"
	if sendErr != nil {
		status = "failed"
	}

	// Best-effort update of test result
	updateErr := h.q.UpdateNotificationChannelTestResult(ctx, sqlcgen.UpdateNotificationChannelTestResultParams{
		ID:             ch.ID,
		TenantID:       tid,
		LastTestedAt:   now,
		LastTestStatus: pgtype.Text{String: status, Valid: true},
	})
	if updateErr != nil {
		slog.ErrorContext(ctx, "update notification channel test result", "channel_type", channelType, "tenant_id", tenantID, "error", updateErr)
	}

	emitEvent(ctx, h.eventBus, events.ChannelTested, "channel", uuidToString(ch.ID), tenantID, map[string]string{
		"channel_type": channelType,
		"status":       status,
	})

	if sendErr != nil {
		slog.WarnContext(ctx, "test notification send failed", "channel_type", channelType, "tenant_id", tenantID, "error", sendErr)
		WriteJSON(w, http.StatusOK, map[string]any{"success": false, "error": fmt.Sprintf("test notification failed: %s", sendErr.Error())})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}
