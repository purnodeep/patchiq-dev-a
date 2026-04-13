package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

const testTenantID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

// stubHubSyncQuerier is a test double for HubSyncQuerier.
type stubHubSyncQuerier struct {
	state    sqlcgen.HubSyncState
	getErr   error
	upserted sqlcgen.UpsertHubSyncStateParams
}

func (s *stubHubSyncQuerier) GetHubSyncState(_ context.Context, _ pgtype.UUID) (sqlcgen.HubSyncState, error) {
	return s.state, s.getErr
}

func (s *stubHubSyncQuerier) UpsertHubSyncState(_ context.Context, arg sqlcgen.UpsertHubSyncStateParams) (sqlcgen.HubSyncState, error) {
	s.upserted = arg
	return s.state, nil
}

// stubRiverEnqueuer is a test double for RiverEnqueuer.
type stubRiverEnqueuer struct {
	inserted bool
	lastArgs river.JobArgs
}

func (s *stubRiverEnqueuer) Insert(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	s.inserted = true
	s.lastArgs = args
	return &rivertype.JobInsertResult{}, nil
}

func TestSyncStatus_Success(t *testing.T) {
	tid, _ := scanUUID(testTenantID)
	q := &stubHubSyncQuerier{
		state: sqlcgen.HubSyncState{
			ID:        tid,
			TenantID:  tid,
			HubUrl:    "https://hub.example.com",
			Status:    "idle",
			CreatedAt: pgtype.Timestamptz{Valid: true},
			UpdatedAt: pgtype.Timestamptz{Valid: true},
		},
	}
	h := NewHubSyncAPIHandler(q, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))

	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body syncStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.HubURL != "https://hub.example.com" {
		t.Errorf("hub_url = %q, want https://hub.example.com", body.HubURL)
	}
	if body.Status != "idle" {
		t.Errorf("status = %q, want idle", body.Status)
	}
}

func TestSyncStatus_NotFound(t *testing.T) {
	q := &stubHubSyncQuerier{
		getErr: pgx.ErrNoRows,
	}
	h := NewHubSyncAPIHandler(q, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))

	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestSyncTrigger_Success(t *testing.T) {
	tid, _ := scanUUID(testTenantID)
	q := &stubHubSyncQuerier{
		state: sqlcgen.HubSyncState{
			ID:       tid,
			TenantID: tid,
			HubUrl:   "https://hub.example.com",
			Status:   "idle",
		},
	}
	enqueuer := &stubRiverEnqueuer{}
	h := NewHubSyncAPIHandler(q, enqueuer, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/trigger", nil)
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))

	rec := httptest.NewRecorder()
	h.Trigger(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	if !enqueuer.inserted {
		t.Error("expected River job to be enqueued")
	}
}

func TestSyncConfig_Update(t *testing.T) {
	tid, _ := scanUUID(testTenantID)
	q := &stubHubSyncQuerier{
		state: sqlcgen.HubSyncState{
			ID:        tid,
			TenantID:  tid,
			HubUrl:    "https://hub.example.com",
			ApiKey:    "new-key",
			Status:    "idle",
			CreatedAt: pgtype.Timestamptz{Valid: true},
			UpdatedAt: pgtype.Timestamptz{Valid: true},
		},
	}
	h := NewHubSyncAPIHandler(q, nil, nil)

	body := `{"hub_url":"https://hub.example.com","api_key":"new-key","sync_interval":7200}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sync/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))

	rec := httptest.NewRecorder()
	h.UpdateConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if q.upserted.HubUrl != "https://hub.example.com" {
		t.Errorf("upserted hub_url = %q, want https://hub.example.com", q.upserted.HubUrl)
	}
	if q.upserted.ApiKey != "new-key" {
		t.Errorf("upserted api_key = %q, want new-key", q.upserted.ApiKey)
	}
	if q.upserted.SyncInterval != 7200 {
		t.Errorf("upserted sync_interval = %d, want 7200", q.upserted.SyncInterval)
	}
}

func TestSyncConfig_MissingURL(t *testing.T) {
	h := NewHubSyncAPIHandler(&stubHubSyncQuerier{}, nil, nil)

	body := `{"api_key":"test-key"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sync/config", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(tenant.WithTenantID(req.Context(), testTenantID))

	rec := httptest.NewRecorder()
	h.UpdateConfig(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
