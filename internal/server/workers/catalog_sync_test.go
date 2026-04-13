package workers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// stubCatalogSyncStore is a test double implementing CatalogSyncStore.
type stubCatalogSyncStore struct {
	state       sqlcgen.HubSyncState
	getErr      error
	started     bool
	completed   bool
	failed      bool
	failedMsg   string
	completedAt sqlcgen.UpdateHubSyncCompletedParams
}

func (s *stubCatalogSyncStore) GetHubSyncState(_ context.Context, _ pgtype.UUID) (sqlcgen.HubSyncState, error) {
	return s.state, s.getErr
}

func (s *stubCatalogSyncStore) UpdateHubSyncStarted(_ context.Context, _ pgtype.UUID) error {
	s.started = true
	return nil
}

func (s *stubCatalogSyncStore) UpdateHubSyncCompleted(_ context.Context, arg sqlcgen.UpdateHubSyncCompletedParams) error {
	s.completed = true
	s.completedAt = arg
	return nil
}

func (s *stubCatalogSyncStore) UpdateHubSyncFailed(_ context.Context, arg sqlcgen.UpdateHubSyncFailedParams) error {
	s.failed = true
	s.failedMsg = arg.ErrorMessage.String
	return nil
}

func (s *stubCatalogSyncStore) ListAllHubSyncStates(_ context.Context) ([]sqlcgen.HubSyncState, error) {
	return []sqlcgen.HubSyncState{s.state}, nil
}

func (s *stubCatalogSyncStore) GetEndpointOsSummary(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetEndpointOsSummaryRow, error) {
	return []sqlcgen.GetEndpointOsSummaryRow{}, nil
}

func (s *stubCatalogSyncStore) GetEndpointStatusSummary(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetEndpointStatusSummaryRow, error) {
	return []sqlcgen.GetEndpointStatusSummaryRow{}, nil
}

func (s *stubCatalogSyncStore) GetFrameworkScoreSummary(_ context.Context, _ pgtype.UUID) ([]sqlcgen.GetFrameworkScoreSummaryRow, error) {
	return []sqlcgen.GetFrameworkScoreSummaryRow{}, nil
}

func (s *stubCatalogSyncStore) UpdateHubCVESyncCompleted(_ context.Context, _ pgtype.UUID) error {
	return nil
}

func (s *stubCatalogSyncStore) UpdateHubCVESyncFailed(_ context.Context, _ sqlcgen.UpdateHubCVESyncFailedParams) error {
	return nil
}

// stubEventBus captures emitted events for test assertions.
type stubEventBus struct {
	emitted []domain.DomainEvent
}

func (s *stubEventBus) Emit(_ context.Context, evt domain.DomainEvent) error {
	s.emitted = append(s.emitted, evt)
	return nil
}

func (s *stubEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (s *stubEventBus) Close() error                                    { return nil }

func newTestTenantID() pgtype.UUID {
	return pgtype.UUID{
		Bytes: [16]byte{0xaa, 0xaa, 0xaa, 0xaa, 0xbb, 0xbb, 0xcc, 0xcc, 0xdd, 0xdd, 0xee, 0xee, 0xee, 0xee, 0xee, 0xee},
		Valid: true,
	}
}

func TestCatalogSync_Success(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entries": []json.RawMessage{
				json.RawMessage(`{"name":"patch-1","version":"1.0"}`),
				json.RawMessage(`{"name":"patch-2","version":"2.0"}`),
			},
			"deleted_ids": []string{},
			"server_time": "2026-03-08T00:00:00Z",
		})
	}))
	defer hub.Close()

	tenantID := newTestTenantID()
	store := &stubCatalogSyncStore{
		state: sqlcgen.HubSyncState{
			TenantID:     tenantID,
			HubUrl:       hub.URL,
			ApiKey:       "test-api-key",
			SyncInterval: 3600,
			Status:       "idle",
		},
	}
	bus := &stubEventBus{}
	worker := NewCatalogSyncWorker(store, nil, bus)

	err := worker.SyncForTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("SyncForTenant() error = %v", err)
	}

	if !store.started {
		t.Error("expected UpdateHubSyncStarted to be called")
	}
	if !store.completed {
		t.Error("expected UpdateHubSyncCompleted to be called")
	}
	if store.completedAt.EntryCount != 2 {
		t.Errorf("entry_count = %d, want 2", store.completedAt.EntryCount)
	}
	if store.failed {
		t.Error("expected UpdateHubSyncFailed NOT to be called")
	}

	// Verify events: sync_started + catalog.synced
	if len(bus.emitted) != 2 {
		t.Fatalf("emitted %d events, want 2", len(bus.emitted))
	}
	if bus.emitted[0].Type != "catalog.sync_started" {
		t.Errorf("event[0].Type = %q, want catalog.sync_started", bus.emitted[0].Type)
	}
	if bus.emitted[1].Type != "catalog.synced" {
		t.Errorf("event[1].Type = %q, want catalog.synced", bus.emitted[1].Type)
	}
}

func TestCatalogSync_HubUnreachable(t *testing.T) {
	tenantID := newTestTenantID()
	store := &stubCatalogSyncStore{
		state: sqlcgen.HubSyncState{
			TenantID:     tenantID,
			HubUrl:       "http://127.0.0.1:1", // unreachable
			ApiKey:       "test-api-key",
			SyncInterval: 3600,
			Status:       "idle",
		},
	}
	bus := &stubEventBus{}
	worker := NewCatalogSyncWorker(store, nil, bus)
	worker.client = &http.Client{Timeout: 1 * time.Second}

	err := worker.SyncForTenant(context.Background(), tenantID)
	if err == nil {
		t.Fatal("expected error for unreachable Hub")
	}

	if !store.started {
		t.Error("expected UpdateHubSyncStarted to be called")
	}
	if !store.failed {
		t.Error("expected UpdateHubSyncFailed to be called")
	}
	if store.completed {
		t.Error("expected UpdateHubSyncCompleted NOT to be called")
	}

	// Verify events: sync_started + sync_failed
	if len(bus.emitted) != 2 {
		t.Fatalf("emitted %d events, want 2", len(bus.emitted))
	}
	if bus.emitted[1].Type != "catalog.sync_failed" {
		t.Errorf("event[1].Type = %q, want catalog.sync_failed", bus.emitted[1].Type)
	}
}

func TestCatalogSync_ParsesInstallerMetadata(t *testing.T) {
	entry := `{
		"name": "KB5034765",
		"vendor": "microsoft",
		"os_family": "windows",
		"version": "10.0.22621.3155",
		"severity": "critical",
		"description": "Cumulative Update",
		"installer_type": "wua",
		"silent_args": "",
		"checksum_sha256": "",
		"binary_ref": "",
		"source_url": "https://support.microsoft.com/kb/5034765",
		"product": "Windows 11",
		"os_package_name": ""
	}`

	var ce catalogEntry
	err := json.Unmarshal([]byte(entry), &ce)
	require.NoError(t, err)
	assert.Equal(t, "wua", ce.InstallerType)
	assert.Equal(t, "", ce.SilentArgs)
}

func TestCatalogSync_ParsesReleaseDate(t *testing.T) {
	entryJSON := `{
		"name": "KB5034441",
		"vendor": "microsoft",
		"os_family": "windows",
		"os_package_name": "KB5034441",
		"installer_type": "wua",
		"silent_args": "",
		"release_date": "2024-02-13T00:00:00Z"
	}`

	var ce catalogEntry
	err := json.Unmarshal([]byte(entryJSON), &ce)
	require.NoError(t, err)
	assert.Equal(t, "wua", ce.InstallerType)
	assert.Equal(t, "", ce.SilentArgs)
	assert.Equal(t, "KB5034441", ce.OsPackageName)
	require.NotNil(t, ce.ReleaseDate)
	assert.Equal(t, 2024, ce.ReleaseDate.Year())
	assert.Equal(t, 2, int(ce.ReleaseDate.Month()))
	assert.Equal(t, 13, ce.ReleaseDate.Day())

	// Verify resolvePackageName uses OsPackageName
	assert.Equal(t, "KB5034441", resolvePackageName(ce))

	// Verify derefTime
	assert.Equal(t, *ce.ReleaseDate, derefTime(ce.ReleaseDate))
}

func TestResolvePackageName_Fallbacks(t *testing.T) {
	tests := []struct {
		name     string
		entry    catalogEntry
		expected string
	}{
		{
			name:     "uses os_package_name when set",
			entry:    catalogEntry{Name: "patch", Product: "Product", OsPackageName: "pkg-name"},
			expected: "pkg-name",
		},
		{
			name:     "falls back to product when os_package_name empty",
			entry:    catalogEntry{Name: "patch", Product: "Product", OsPackageName: ""},
			expected: "Product",
		},
		{
			name:     "falls back to name when both empty",
			entry:    catalogEntry{Name: "KB5034441", Product: "", OsPackageName: ""},
			expected: "KB5034441",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, resolvePackageName(tc.entry))
		})
	}
}

func TestCatalogSync_EmptyResponse(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"entries":     []json.RawMessage{},
			"deleted_ids": []string{},
			"server_time": "2026-03-08T00:00:00Z",
		})
	}))
	defer hub.Close()

	tenantID := newTestTenantID()
	store := &stubCatalogSyncStore{
		state: sqlcgen.HubSyncState{
			TenantID:     tenantID,
			HubUrl:       hub.URL,
			ApiKey:       "test-api-key",
			SyncInterval: 3600,
			Status:       "idle",
		},
	}
	bus := &stubEventBus{}
	worker := NewCatalogSyncWorker(store, nil, bus)

	err := worker.SyncForTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("SyncForTenant() error = %v", err)
	}

	if !store.completed {
		t.Error("expected UpdateHubSyncCompleted to be called")
	}
	if store.completedAt.EntryCount != 0 {
		t.Errorf("entry_count = %d, want 0", store.completedAt.EntryCount)
	}
	if store.failed {
		t.Error("expected UpdateHubSyncFailed NOT to be called")
	}
}
