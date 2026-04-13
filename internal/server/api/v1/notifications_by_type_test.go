package v1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake Querier for by-type handler ---

type fakeNotifByTypeQuerier struct {
	fakeNotifQuerier // embed existing fake

	getByTypeResult     sqlcgen.NotificationChannel
	getByTypeErr        error
	updateTestResultErr error
}

func (f *fakeNotifByTypeQuerier) GetNotificationChannelByType(_ context.Context, _ sqlcgen.GetNotificationChannelByTypeParams) (sqlcgen.NotificationChannel, error) {
	return f.getByTypeResult, f.getByTypeErr
}

func (f *fakeNotifByTypeQuerier) UpdateNotificationChannelTestResult(_ context.Context, _ sqlcgen.UpdateNotificationChannelTestResultParams) error {
	return f.updateTestResultErr
}

func newNotifByTypeHandler(q v1.NotificationQuerier, sender notify.Sender) *v1.NotificationByTypeHandler {
	if sender == nil {
		sender = &notify.MockSender{}
	}
	return v1.NewNotificationByTypeHandler(q, testCryptoKey, &fakeEventBus{}, sender)
}

// --- TestGetChannelByType ---

func TestGetChannelByType(t *testing.T) {
	encrypted, _ := crypto.Encrypt(testCryptoKey, []byte("slack://token@channel"))
	ch := validChannel()
	ch.ConfigEncrypted = encrypted

	tests := []struct {
		name        string
		channelType string
		querier     *fakeNotifByTypeQuerier
		wantStatus  int
	}{
		{
			name:        "returns channel for matching type",
			channelType: "slack",
			querier:     &fakeNotifByTypeQuerier{getByTypeResult: ch},
			wantStatus:  http.StatusOK,
		},
		{
			name:        "returns 404 for unknown type",
			channelType: "discord",
			querier:     &fakeNotifByTypeQuerier{getByTypeErr: pgx.ErrNoRows},
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifByTypeHandler(tt.querier, nil)
			req := notifReq(http.MethodGet, "/api/v1/notifications/channels/by-type/"+tt.channelType, nil)
			req = chiCtx(req, "type", tt.channelType)
			rec := httptest.NewRecorder()

			h.GetByType(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotContains(t, body, "config_encrypted")
				assert.Equal(t, "slack", body["channel_type"])
			}
		})
	}
}

// --- TestGetChannelByType_NotFound ---

func TestGetChannelByType_NotFound(t *testing.T) {
	q := &fakeNotifByTypeQuerier{getByTypeErr: pgx.ErrNoRows}
	h := newNotifByTypeHandler(q, nil)

	req := notifReq(http.MethodGet, "/api/v1/notifications/channels/by-type/webhook", nil)
	req = chiCtx(req, "type", "webhook")
	rec := httptest.NewRecorder()

	h.GetByType(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- TestPutChannelByType ---

func TestPutChannelByType(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		body        any
		querier     *fakeNotifByTypeQuerier
		wantStatus  int
	}{
		{
			name:        "creates new channel when not found",
			channelType: "slack",
			body:        map[string]any{"name": "Slack", "config": "slack://token@channel", "enabled": true},
			querier: &fakeNotifByTypeQuerier{
				getByTypeErr: pgx.ErrNoRows,
				fakeNotifQuerier: fakeNotifQuerier{
					createChannelResult: validChannel(),
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "updates existing channel",
			channelType: "slack",
			body:        map[string]any{"name": "Updated Slack", "config": "slack://new", "enabled": true},
			querier: &fakeNotifByTypeQuerier{
				getByTypeResult: validChannel(),
				fakeNotifQuerier: fakeNotifQuerier{
					updateChannelResult: validChannel(),
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "missing config returns 400",
			channelType: "slack",
			body:        map[string]any{"name": "Slack"},
			querier:     &fakeNotifByTypeQuerier{},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "missing name returns 400",
			channelType: "slack",
			body:        map[string]any{"config": "slack://x"},
			querier:     &fakeNotifByTypeQuerier{},
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifByTypeHandler(tt.querier, nil)
			req := notifReq(http.MethodPut, "/api/v1/notifications/channels/by-type/"+tt.channelType, tt.body)
			req = chiCtx(req, "type", tt.channelType)
			rec := httptest.NewRecorder()

			h.UpdateByType(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- TestTestChannelByType ---

func TestTestChannelByType(t *testing.T) {
	encrypted, _ := crypto.Encrypt(testCryptoKey, []byte("slack://token@channel"))
	ch := validChannel()
	ch.ConfigEncrypted = encrypted

	tests := []struct {
		name        string
		channelType string
		querier     *fakeNotifByTypeQuerier
		sender      *notify.MockSender
		wantStatus  int
		wantOK      bool
	}{
		{
			name:        "successful test",
			channelType: "slack",
			querier:     &fakeNotifByTypeQuerier{getByTypeResult: ch},
			sender:      &notify.MockSender{},
			wantStatus:  http.StatusOK,
			wantOK:      true,
		},
		{
			name:        "send failure",
			channelType: "slack",
			querier:     &fakeNotifByTypeQuerier{getByTypeResult: ch},
			sender:      &notify.MockSender{Err: fmt.Errorf("connection refused")},
			wantStatus:  http.StatusOK,
			wantOK:      false,
		},
		{
			name:        "not found",
			channelType: "webhook",
			querier:     &fakeNotifByTypeQuerier{getByTypeErr: pgx.ErrNoRows},
			sender:      &notify.MockSender{},
			wantStatus:  http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifByTypeHandler(tt.querier, tt.sender)
			req := notifReq(http.MethodPost, "/api/v1/notifications/channels/by-type/"+tt.channelType+"/test", nil)
			req = chiCtx(req, "type", tt.channelType)
			rec := httptest.NewRecorder()

			h.TestByType(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantOK, body["success"])
			}
		})
	}
}
