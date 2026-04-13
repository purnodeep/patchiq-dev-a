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
	"github.com/skenzeriq/patchiq/internal/server/notify"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Fake Querier ---

type fakeNotifQuerier struct {
	createChannelResult sqlcgen.NotificationChannel
	createChannelErr    error

	getChannelResult sqlcgen.NotificationChannel
	getChannelErr    error

	listChannelsResult []sqlcgen.NotificationChannel
	listChannelsErr    error

	updateChannelResult sqlcgen.NotificationChannel
	updateChannelErr    error

	deleteChannelRows int64
	deleteChannelErr  error

	upsertPrefResult sqlcgen.NotificationPreference
	upsertPrefErr    error

	listPrefsResult []sqlcgen.NotificationPreference
	listPrefsErr    error

	listHistoryResult []sqlcgen.NotificationHistory
	listHistoryErr    error
	listHistoryParams sqlcgen.ListNotificationHistoryParams // captured for inspection

	countHistoryResult int64
	countHistoryErr    error

	getByTypeResult     sqlcgen.NotificationChannel
	getByTypeErr        error
	updateTestResultErr error

	historyByIDResult sqlcgen.NotificationHistory
	historyByIDErr    error

	updateHistoryStatusErr error

	digestConfigResult sqlcgen.NotificationDigestConfig
	digestConfigErr    error

	upsertDigestResult sqlcgen.NotificationDigestConfig
	upsertDigestErr    error
}

// --- Fake River Enqueuer ---

type fakeRiverEnqueuer struct {
	insertErr    error
	insertCalled bool
}

func (f *fakeRiverEnqueuer) EnqueueNotification(_ context.Context, _ notify.SendJobArgs) error {
	f.insertCalled = true
	return f.insertErr
}

func (f *fakeNotifQuerier) CreateNotificationChannel(_ context.Context, _ sqlcgen.CreateNotificationChannelParams) (sqlcgen.NotificationChannel, error) {
	return f.createChannelResult, f.createChannelErr
}

func (f *fakeNotifQuerier) GetNotificationChannel(_ context.Context, _ sqlcgen.GetNotificationChannelParams) (sqlcgen.NotificationChannel, error) {
	return f.getChannelResult, f.getChannelErr
}

func (f *fakeNotifQuerier) ListNotificationChannels(_ context.Context, _ pgtype.UUID) ([]sqlcgen.NotificationChannel, error) {
	return f.listChannelsResult, f.listChannelsErr
}

func (f *fakeNotifQuerier) UpdateNotificationChannel(_ context.Context, _ sqlcgen.UpdateNotificationChannelParams) (sqlcgen.NotificationChannel, error) {
	return f.updateChannelResult, f.updateChannelErr
}

func (f *fakeNotifQuerier) DeleteNotificationChannel(_ context.Context, _ sqlcgen.DeleteNotificationChannelParams) (int64, error) {
	return f.deleteChannelRows, f.deleteChannelErr
}

func (f *fakeNotifQuerier) UpsertNotificationPreference(_ context.Context, _ sqlcgen.UpsertNotificationPreferenceParams) (sqlcgen.NotificationPreference, error) {
	return f.upsertPrefResult, f.upsertPrefErr
}

func (f *fakeNotifQuerier) ListNotificationPreferences(_ context.Context, _ pgtype.UUID) ([]sqlcgen.NotificationPreference, error) {
	return f.listPrefsResult, f.listPrefsErr
}

func (f *fakeNotifQuerier) ListNotificationHistory(_ context.Context, arg sqlcgen.ListNotificationHistoryParams) ([]sqlcgen.NotificationHistory, error) {
	f.listHistoryParams = arg
	return f.listHistoryResult, f.listHistoryErr
}

func (f *fakeNotifQuerier) CountNotificationHistory(_ context.Context, _ sqlcgen.CountNotificationHistoryParams) (int64, error) {
	return f.countHistoryResult, f.countHistoryErr
}

func (f *fakeNotifQuerier) GetNotificationChannelByType(_ context.Context, _ sqlcgen.GetNotificationChannelByTypeParams) (sqlcgen.NotificationChannel, error) {
	return f.getByTypeResult, f.getByTypeErr
}

func (f *fakeNotifQuerier) UpdateNotificationChannelTestResult(_ context.Context, _ sqlcgen.UpdateNotificationChannelTestResultParams) error {
	return f.updateTestResultErr
}

func (f *fakeNotifQuerier) GetNotificationHistoryByID(_ context.Context, _ sqlcgen.GetNotificationHistoryByIDParams) (sqlcgen.NotificationHistory, error) {
	return f.historyByIDResult, f.historyByIDErr
}

func (f *fakeNotifQuerier) UpdateNotificationHistoryStatus(_ context.Context, _ sqlcgen.UpdateNotificationHistoryStatusParams) error {
	return f.updateHistoryStatusErr
}

func (f *fakeNotifQuerier) GetDigestConfig(_ context.Context, _ pgtype.UUID) (sqlcgen.NotificationDigestConfig, error) {
	return f.digestConfigResult, f.digestConfigErr
}

func (f *fakeNotifQuerier) UpsertDigestConfig(_ context.Context, _ sqlcgen.UpsertDigestConfigParams) (sqlcgen.NotificationDigestConfig, error) {
	return f.upsertDigestResult, f.upsertDigestErr
}

// --- Helpers ---

const testTenantID = "00000000-0000-0000-0000-000000000001"

var testCryptoKey = crypto.GenerateKey()

func validChannel() sqlcgen.NotificationChannel {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000088")
	_ = tid.Scan(testTenantID)
	return sqlcgen.NotificationChannel{
		ID:              id,
		TenantID:        tid,
		Name:            "Ops Slack",
		ChannelType:     "slack",
		ConfigEncrypted: []byte("encrypted-data"),
		Enabled:         true,
	}
}

func validPreference() sqlcgen.NotificationPreference {
	var id, tid pgtype.UUID
	_ = id.Scan("00000000-0000-0000-0000-000000000077")
	_ = tid.Scan(testTenantID)
	return sqlcgen.NotificationPreference{
		ID:             id,
		TenantID:       tid,
		UserID:         "system",
		TriggerType:    "deployment.failed",
		EmailEnabled:   true,
		SlackEnabled:   false,
		WebhookEnabled: false,
		Urgency:        "immediate",
	}
}

func validHistory() sqlcgen.NotificationHistory {
	var tid, cid pgtype.UUID
	_ = tid.Scan(testTenantID)
	_ = cid.Scan("00000000-0000-0000-0000-000000000088")
	return sqlcgen.NotificationHistory{
		ID:          "01JTEST0000000000000000001",
		TenantID:    tid,
		TriggerType: "deployment.failed",
		ChannelID:   cid,
		UserID:      "system",
		Status:      "sent",
		Payload:     []byte(`{"msg":"hello"}`),
	}
}

func newNotifHandler(q *fakeNotifQuerier) *v1.NotificationHandler {
	return v1.NewNotificationHandler(q, &fakeTxBeginner{tx: &fakeTx{}}, testCryptoKey, &notify.MockSender{}, &fakeEventBus{}, &fakeRiverEnqueuer{})
}

func notifReq(method, url string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))
	return req
}

// --- CreateChannel Tests ---

func TestNotificationHandler_CreateChannel(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		querier    *fakeNotifQuerier
		wantStatus int
		wantNoConf bool // config must not appear in response
	}{
		{
			name: "valid create returns 201",
			body: map[string]any{"name": "Ops Slack", "channel_type": "slack", "config": "slack://token@channel"},
			querier: &fakeNotifQuerier{
				createChannelResult: validChannel(),
			},
			wantStatus: http.StatusCreated,
			wantNoConf: true,
		},
		{
			name:       "missing name returns 400",
			body:       map[string]any{"channel_type": "slack", "config": "slack://x"},
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing channel_type returns 400",
			body:       map[string]any{"name": "test", "config": "slack://x"},
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing config returns 400",
			body:       map[string]any{"name": "test", "channel_type": "slack"},
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "store error returns 500",
			body: map[string]any{"name": "test", "channel_type": "slack", "config": "slack://x"},
			querier: &fakeNotifQuerier{
				createChannelErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodPost, "/api/v1/notifications/channels", tt.body)
			rec := httptest.NewRecorder()

			h.CreateChannel(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantNoConf {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotEmpty(t, body["id"])
				assert.NotContains(t, body, "config_encrypted")
				assert.NotContains(t, body, "config")
			}
		})
	}
}

// --- ListChannels Tests ---

func TestNotificationHandler_ListChannels(t *testing.T) {
	tests := []struct {
		name       string
		querier    *fakeNotifQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns empty list",
			querier: &fakeNotifQuerier{
				listChannelsResult: []sqlcgen.NotificationChannel{},
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name: "returns channels",
			querier: &fakeNotifQuerier{
				listChannelsResult: []sqlcgen.NotificationChannel{validChannel()},
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "store error returns 500",
			querier: &fakeNotifQuerier{
				listChannelsErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodGet, "/api/v1/notifications/channels", nil)
			rec := httptest.NewRecorder()

			h.ListChannels(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body []map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Len(t, body, tt.wantLen)
			}
		})
	}
}

// --- GetChannel Tests ---

func TestNotificationHandler_GetChannel(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeNotifQuerier
		wantStatus int
	}{
		{
			name:       "valid ID returns 200",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeNotifQuerier{getChannelResult: validChannel()},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeNotifQuerier{getChannelErr: pgx.ErrNoRows},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodGet, "/api/v1/notifications/channels/"+tt.id, nil)
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.GetChannel(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.NotContains(t, body, "config_encrypted")
			}
		})
	}
}

// --- DeleteChannel Tests ---

func TestNotificationHandler_DeleteChannel(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *fakeNotifQuerier
		wantStatus int
	}{
		{
			name:       "valid delete returns 204",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeNotifQuerier{deleteChannelRows: 1},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeNotifQuerier{deleteChannelRows: 0},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "bad-uuid",
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodDelete, "/api/v1/notifications/channels/"+tt.id, nil)
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.DeleteChannel(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- UpdateChannel Tests ---

func TestNotificationHandler_UpdateChannel(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       any
		querier    *fakeNotifQuerier
		wantStatus int
	}{
		{
			name: "valid update returns 200",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{"name": "Updated", "channel_type": "slack", "config": "slack://new", "enabled": true},
			querier: &fakeNotifQuerier{
				updateChannelResult: validChannel(),
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "omitting name falls back to existing name and returns 200",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{"channel_type": "slack", "config": "slack://new"},
			querier: &fakeNotifQuerier{
				getChannelResult:    validChannel(),
				updateChannelResult: validChannel(),
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not found returns 404",
			id:   "00000000-0000-0000-0000-000000000088",
			body: map[string]any{"name": "test", "channel_type": "slack", "config": "slack://x", "enabled": true},
			querier: &fakeNotifQuerier{
				updateChannelErr: pgx.ErrNoRows,
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodPut, "/api/v1/notifications/channels/"+tt.id, tt.body)
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.UpdateChannel(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- GetPreferences Tests ---

func TestNotificationHandler_GetPreferences(t *testing.T) {
	tests := []struct {
		name           string
		querier        *fakeNotifQuerier
		wantStatus     int
		wantCategories int // expected number of categories in response (-1 = skip check)
	}{
		{
			name: "returns grouped response with 4 categories",
			querier: &fakeNotifQuerier{
				listPrefsResult:    []sqlcgen.NotificationPreference{validPreference()},
				listChannelsResult: []sqlcgen.NotificationChannel{},
			},
			wantStatus:     http.StatusOK,
			wantCategories: 4,
		},
		{
			name: "returns 4 categories even with empty prefs",
			querier: &fakeNotifQuerier{
				listPrefsResult:    []sqlcgen.NotificationPreference{},
				listChannelsResult: []sqlcgen.NotificationChannel{},
			},
			wantStatus:     http.StatusOK,
			wantCategories: 4,
		},
		{
			name: "store error returns 500",
			querier: &fakeNotifQuerier{
				listPrefsErr: fmt.Errorf("database error"),
			},
			wantStatus:     http.StatusInternalServerError,
			wantCategories: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodGet, "/api/v1/notifications/preferences", nil)
			rec := httptest.NewRecorder()

			h.GetPreferences(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantCategories >= 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				cats, ok := body["categories"].([]any)
				require.True(t, ok, "expected categories array in response")
				assert.Len(t, cats, tt.wantCategories)
			}
		})
	}
}

// --- GetPreferences_GroupedResponse Test ---

func TestGetPreferences_GroupedResponse(t *testing.T) {
	q := &fakeNotifQuerier{
		listPrefsResult:    []sqlcgen.NotificationPreference{validPreference(), validPreference()},
		listChannelsResult: []sqlcgen.NotificationChannel{validChannel()},
	}
	h := newNotifHandler(q)
	req := notifReq(http.MethodGet, "/api/v1/notifications/preferences", nil)
	rec := httptest.NewRecorder()

	h.GetPreferences(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	cats, ok := body["categories"].([]any)
	require.True(t, ok)
	assert.Len(t, cats, 4)

	channels, ok := body["channels"].([]any)
	require.True(t, ok)
	assert.Len(t, channels, 3) // email, slack, webhook
}

// --- UpdatePreferences Tests ---

func TestNotificationHandler_UpdatePreferences(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		querier    *fakeNotifQuerier
		wantStatus int
	}{
		{
			name: "valid upsert returns 204",
			body: map[string]any{
				"preferences": []map[string]any{
					{
						"trigger_type":    "deployment.failed",
						"email_enabled":   true,
						"slack_enabled":   false,
						"webhook_enabled": false,
						"urgency":         "immediate",
					},
				},
			},
			querier:    &fakeNotifQuerier{upsertPrefResult: validPreference()},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "invalid trigger type returns 400",
			body: map[string]any{
				"preferences": []map[string]any{
					{"trigger_type": "bad.trigger", "email_enabled": true},
				},
			},
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid body returns 400",
			body:       "not json",
			querier:    &fakeNotifQuerier{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodPut, "/api/v1/notifications/preferences", tt.body)
			rec := httptest.NewRecorder()

			h.UpdatePreferences(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// --- ListHistory Tests ---

func TestNotificationHandler_ListHistory(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *fakeNotifQuerier
		wantStatus int
		wantLen    int
	}{
		{
			name: "returns history",
			querier: &fakeNotifQuerier{
				listHistoryResult:  []sqlcgen.NotificationHistory{validHistory()},
				countHistoryResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "returns empty list",
			querier: &fakeNotifQuerier{
				listHistoryResult:  []sqlcgen.NotificationHistory{},
				countHistoryResult: 0,
			},
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:  "with filters",
			query: "?trigger_type=deployment.failed&status=sent&limit=5",
			querier: &fakeNotifQuerier{
				listHistoryResult:  []sqlcgen.NotificationHistory{validHistory()},
				countHistoryResult: 1,
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name: "store error returns 500",
			querier: &fakeNotifQuerier{
				listHistoryErr: fmt.Errorf("database error"),
			},
			wantStatus: http.StatusInternalServerError,
			wantLen:    -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newNotifHandler(tt.querier)
			req := notifReq(http.MethodGet, "/api/v1/notifications/history"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.ListHistory(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantLen >= 0 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				data, ok := body["data"].([]any)
				require.True(t, ok)
				assert.Len(t, data, tt.wantLen)
			}
		})
	}
}

// --- TestChannel Tests ---

func TestNotificationHandler_TestChannel(t *testing.T) {
	encrypted, _ := crypto.Encrypt(testCryptoKey, []byte("slack://token@channel"))

	tests := []struct {
		name       string
		id         string
		querier    *fakeNotifQuerier
		sender     *notify.MockSender
		wantStatus int
		wantOK     bool
	}{
		{
			name: "successful test returns 200 with success true",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeNotifQuerier{
				getChannelResult: func() sqlcgen.NotificationChannel {
					ch := validChannel()
					ch.ConfigEncrypted = encrypted
					return ch
				}(),
			},
			sender:     &notify.MockSender{},
			wantStatus: http.StatusOK,
			wantOK:     true,
		},
		{
			name: "send failure returns 200 with success false",
			id:   "00000000-0000-0000-0000-000000000088",
			querier: &fakeNotifQuerier{
				getChannelResult: func() sqlcgen.NotificationChannel {
					ch := validChannel()
					ch.ConfigEncrypted = encrypted
					return ch
				}(),
			},
			sender:     &notify.MockSender{Err: fmt.Errorf("connection refused")},
			wantStatus: http.StatusOK,
			wantOK:     false,
		},
		{
			name:       "not found returns 404",
			id:         "00000000-0000-0000-0000-000000000088",
			querier:    &fakeNotifQuerier{getChannelErr: pgx.ErrNoRows},
			sender:     &notify.MockSender{},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewNotificationHandler(tt.querier, &fakeTxBeginner{tx: &fakeTx{}}, testCryptoKey, tt.sender, &fakeEventBus{}, &fakeRiverEnqueuer{})
			req := notifReq(http.MethodPost, "/api/v1/notifications/channels/"+tt.id+"/test", nil)
			req = chiCtx(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.TestChannel(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantOK, body["success"])
			}
		})
	}
}

// --- RetryNotification Tests ---

func failedHistory() sqlcgen.NotificationHistory {
	var tid, cid pgtype.UUID
	_ = tid.Scan(testTenantID)
	_ = cid.Scan("00000000-0000-0000-0000-000000000088")
	return sqlcgen.NotificationHistory{
		ID:          "01JTEST0000000000000000002",
		TenantID:    tid,
		TriggerType: "deployment.failed",
		ChannelID:   cid,
		UserID:      "system",
		Status:      "failed",
		RetryCount:  0,
		ChannelType: pgtype.Text{String: "slack", Valid: true},
		Recipient:   "#ops",
		Subject:     "Deployment Failed",
	}
}

func TestRetryNotification_HappyPath(t *testing.T) {
	encrypted, _ := crypto.Encrypt(testCryptoKey, []byte(`{"url":"slack://token@channel"}`))
	ch := validChannel()
	ch.ConfigEncrypted = encrypted

	river := &fakeRiverEnqueuer{}
	q := &fakeNotifQuerier{
		historyByIDResult: failedHistory(),
		getChannelResult:  ch,
	}
	h := v1.NewNotificationHandler(q, &fakeTxBeginner{tx: &fakeTx{}}, testCryptoKey, &notify.MockSender{}, &fakeEventBus{}, river)

	req := notifReq(http.MethodPost, "/api/v1/notifications/history/01JTEST0000000000000000002/retry", nil)
	req = chiCtx(req, "id", "01JTEST0000000000000000002")
	rec := httptest.NewRecorder()

	h.RetryNotification(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)
	assert.True(t, river.insertCalled)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "pending", body["status"])
}

func TestRetryNotification_NotFailed(t *testing.T) {
	h := newNotifHandler(&fakeNotifQuerier{
		historyByIDResult: func() sqlcgen.NotificationHistory {
			h := failedHistory()
			h.Status = "delivered"
			return h
		}(),
	})
	req := notifReq(http.MethodPost, "/api/v1/notifications/history/01JTEST0000000000000000002/retry", nil)
	req = chiCtx(req, "id", "01JTEST0000000000000000002")
	rec := httptest.NewRecorder()

	h.RetryNotification(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestRetryNotification_MaxRetries(t *testing.T) {
	h := newNotifHandler(&fakeNotifQuerier{
		historyByIDResult: func() sqlcgen.NotificationHistory {
			h := failedHistory()
			h.RetryCount = 3
			return h
		}(),
	})
	req := notifReq(http.MethodPost, "/api/v1/notifications/history/01JTEST0000000000000000002/retry", nil)
	req = chiCtx(req, "id", "01JTEST0000000000000000002")
	rec := httptest.NewRecorder()

	h.RetryNotification(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "max retries")
}

// --- GetDigestConfig Tests ---

func TestGetDigestConfig_NotFound(t *testing.T) {
	h := newNotifHandler(&fakeNotifQuerier{
		digestConfigErr: pgx.ErrNoRows,
	})
	req := notifReq(http.MethodGet, "/api/v1/notifications/digest-config", nil)
	rec := httptest.NewRecorder()

	h.GetDigestConfig(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "daily", body["frequency"])
	assert.Equal(t, "09:00", body["delivery_time"])
}

func TestGetDigestConfig_Exists(t *testing.T) {
	var tid pgtype.UUID
	_ = tid.Scan(testTenantID)
	h := newNotifHandler(&fakeNotifQuerier{
		digestConfigResult: sqlcgen.NotificationDigestConfig{
			TenantID:     tid,
			Frequency:    "weekly",
			DeliveryTime: pgtype.Time{Microseconds: 8 * 3600 * 1_000_000, Valid: true},
			Format:       "html",
		},
	})
	req := notifReq(http.MethodGet, "/api/v1/notifications/digest-config", nil)
	rec := httptest.NewRecorder()

	h.GetDigestConfig(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "weekly", body["frequency"])
	assert.Equal(t, "08:00", body["delivery_time"])
}

// --- UpdateDigestConfig Tests ---

func TestUpdateDigestConfig_HappyPath(t *testing.T) {
	var tid pgtype.UUID
	_ = tid.Scan(testTenantID)
	h := newNotifHandler(&fakeNotifQuerier{
		upsertDigestResult: sqlcgen.NotificationDigestConfig{
			TenantID:     tid,
			Frequency:    "weekly",
			DeliveryTime: pgtype.Time{Microseconds: 8 * 3600 * 1_000_000, Valid: true},
			Format:       "html",
		},
	})
	req := notifReq(http.MethodPut, "/api/v1/notifications/digest-config", map[string]any{
		"frequency":     "weekly",
		"delivery_time": "08:00",
		"format":        "html",
	})
	rec := httptest.NewRecorder()

	h.UpdateDigestConfig(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "weekly", body["frequency"])
}

func TestUpdateDigestConfig_InvalidFrequency(t *testing.T) {
	h := newNotifHandler(&fakeNotifQuerier{})
	req := notifReq(http.MethodPut, "/api/v1/notifications/digest-config", map[string]any{
		"frequency":     "monthly",
		"delivery_time": "09:00",
		"format":        "html",
	})
	rec := httptest.NewRecorder()

	h.UpdateDigestConfig(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- ListHistory ChannelType Filter Test ---

func TestListHistory_ChannelTypeFilter(t *testing.T) {
	q := &fakeNotifQuerier{
		listHistoryResult:  []sqlcgen.NotificationHistory{validHistory()},
		countHistoryResult: 1,
	}
	h := newNotifHandler(q)
	req := notifReq(http.MethodGet, "/api/v1/notifications/history?channel_type=email", nil)
	rec := httptest.NewRecorder()

	h.ListHistory(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, q.listHistoryParams.ChannelType.Valid)
	assert.Equal(t, "email", q.listHistoryParams.ChannelType.String)
}

// --- ListHistory Category Filter Test ---

func TestListHistory_CategoryFilter(t *testing.T) {
	var tid, cid pgtype.UUID
	_ = tid.Scan(testTenantID)
	_ = cid.Scan("00000000-0000-0000-0000-000000000088")

	deploymentEntry := sqlcgen.NotificationHistory{
		ID:          "01JTEST0000000000000000010",
		TenantID:    tid,
		TriggerType: "deployment.failed",
		ChannelID:   cid,
		UserID:      "system",
		Status:      "sent",
	}
	cveEntry := sqlcgen.NotificationHistory{
		ID:          "01JTEST0000000000000000011",
		TenantID:    tid,
		TriggerType: "cve.critical_discovered",
		ChannelID:   cid,
		UserID:      "system",
		Status:      "sent",
	}

	q := &fakeNotifQuerier{
		listHistoryResult:  []sqlcgen.NotificationHistory{deploymentEntry, cveEntry},
		countHistoryResult: 2,
	}
	h := newNotifHandler(q)
	req := notifReq(http.MethodGet, "/api/v1/notifications/history?category=security", nil)
	rec := httptest.NewRecorder()

	h.ListHistory(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)
	entry, ok2 := data[0].(map[string]any)
	require.True(t, ok2)
	assert.Equal(t, "cve.critical_discovered", entry["trigger_type"])
}
