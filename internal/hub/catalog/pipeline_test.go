package catalog

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skenzeriq/patchiq/internal/hub/events"
	"github.com/skenzeriq/patchiq/internal/hub/feeds"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// --- Mocks ---

type mockPipelineStore struct {
	source    sqlcgen.FeedSource
	syncState sqlcgen.FeedSyncState

	upsertedEntries []sqlcgen.UpsertCatalogEntryFromFeedParams
	createdCVEs     []sqlcgen.CreateCVEFeedParams
	upsertedCVEs    []sqlcgen.UpsertCVEFeedParams
	linkedCVEs      []sqlcgen.LinkCatalogCVEParams

	syncStartCalled   bool
	syncSuccessParams sqlcgen.UpdateFeedSyncStateSuccessParams
	syncSuccessCalled bool
	syncErrorParams   sqlcgen.UpdateFeedSyncStateErrorParams
	syncErrorCalled   bool

	// Hook overrides for customizable behavior.
	getCVEFeedByCVEIDFn func(ctx context.Context, cveID string) (sqlcgen.CVEFeed, error)
}

func (m *mockPipelineStore) GetFeedSourceByName(_ context.Context, _ string) (sqlcgen.FeedSource, error) {
	return m.source, nil
}

func (m *mockPipelineStore) GetFeedSyncState(_ context.Context, _ pgtype.UUID) (sqlcgen.FeedSyncState, error) {
	return m.syncState, nil
}

func (m *mockPipelineStore) UpdateFeedSyncStateStart(_ context.Context, _ pgtype.UUID) error {
	m.syncStartCalled = true
	return nil
}

func (m *mockPipelineStore) UpdateFeedSyncStateSuccess(_ context.Context, arg sqlcgen.UpdateFeedSyncStateSuccessParams) error {
	m.syncSuccessCalled = true
	m.syncSuccessParams = arg
	return nil
}

func (m *mockPipelineStore) UpdateFeedSyncStateError(_ context.Context, arg sqlcgen.UpdateFeedSyncStateErrorParams) error {
	m.syncErrorCalled = true
	m.syncErrorParams = arg
	return nil
}

func (m *mockPipelineStore) UpsertCatalogEntryFromFeed(_ context.Context, arg sqlcgen.UpsertCatalogEntryFromFeedParams) (sqlcgen.PatchCatalog, error) {
	m.upsertedEntries = append(m.upsertedEntries, arg)
	return sqlcgen.PatchCatalog{
		ID:   testUUID(1),
		Name: arg.Name,
	}, nil
}

func (m *mockPipelineStore) GetCVEFeedByCVEID(ctx context.Context, cveID string) (sqlcgen.CVEFeed, error) {
	if m.getCVEFeedByCVEIDFn != nil {
		return m.getCVEFeedByCVEIDFn(ctx, cveID)
	}
	return sqlcgen.CVEFeed{ID: testUUID(100), CveID: cveID}, nil
}

func (m *mockPipelineStore) CreateCVEFeed(_ context.Context, arg sqlcgen.CreateCVEFeedParams) (sqlcgen.CVEFeed, error) {
	m.createdCVEs = append(m.createdCVEs, arg)
	return sqlcgen.CVEFeed{ID: testUUID(200), CveID: arg.CveID}, nil
}

func (m *mockPipelineStore) UpsertCVEFeed(_ context.Context, arg sqlcgen.UpsertCVEFeedParams) (sqlcgen.CVEFeed, error) {
	m.upsertedCVEs = append(m.upsertedCVEs, arg)
	return sqlcgen.CVEFeed{ID: testUUID(200), CveID: arg.CveID}, nil
}

func (m *mockPipelineStore) LinkCatalogCVE(_ context.Context, arg sqlcgen.LinkCatalogCVEParams) error {
	m.linkedCVEs = append(m.linkedCVEs, arg)
	return nil
}

func (m *mockPipelineStore) CreateBinaryFetchState(_ context.Context, _ sqlcgen.CreateBinaryFetchStateParams) (sqlcgen.BinaryFetchState, error) {
	return sqlcgen.BinaryFetchState{}, nil
}

func (m *mockPipelineStore) GetPackageAlias(_ context.Context, _ sqlcgen.GetPackageAliasParams) (sqlcgen.PackageAlias, error) {
	return sqlcgen.PackageAlias{}, fmt.Errorf("no alias found")
}

type mockEventBus struct {
	events []domain.DomainEvent
}

func (m *mockEventBus) Emit(_ context.Context, evt domain.DomainEvent) error {
	m.events = append(m.events, evt)
	return nil
}

func (m *mockEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (m *mockEventBus) Close() error                                    { return nil }

type mockFeed struct {
	name    string
	entries []feeds.RawEntry
	cursor  string
	err     error
}

func (m *mockFeed) Name() string { return m.name }
func (m *mockFeed) Fetch(_ context.Context, _ string) ([]feeds.RawEntry, string, error) {
	return m.entries, m.cursor, m.err
}

// testUUID creates a deterministic pgtype.UUID from an int for testing.
func testUUID(n byte) pgtype.UUID {
	var id [16]byte
	id[15] = n
	return pgtype.UUID{Bytes: id, Valid: true}
}

func newTestStore() *mockPipelineStore {
	return &mockPipelineStore{
		source: sqlcgen.FeedSource{
			ID:                  testUUID(10),
			Name:                "nvd",
			Enabled:             true,
			SyncIntervalSeconds: 3600,
		},
		syncState: sqlcgen.FeedSyncState{
			FeedSourceID: testUUID(10),
			Cursor:       "prev-cursor",
			Status:       "idle",
		},
	}
}

// --- Tests ---

func TestPipeline_Sync_creates_new_catalog_entries(t *testing.T) {
	store := newTestStore()
	bus := &mockEventBus{}
	feed := &mockFeed{
		name: "nvd",
		entries: []feeds.RawEntry{
			{
				Name:        "KB5001234",
				Vendor:      "microsoft",
				OSFamily:    "windows",
				Version:     "1.0.0",
				Severity:    "high",
				ReleaseDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
				Summary:     "Security update",
				SourceURL:   "https://example.com/kb5001234",
				CVEs:        []string{"CVE-2026-0001"},
			},
		},
		cursor: "next-cursor-1",
	}

	p := NewPipeline(store, bus, nil)
	err := p.Sync(context.Background(), feed)

	require.NoError(t, err)
	require.Len(t, store.upsertedEntries, 1)
	assert.Equal(t, "KB5001234", store.upsertedEntries[0].Name)
	assert.Equal(t, "microsoft", store.upsertedEntries[0].Vendor)
	assert.Equal(t, "windows", store.upsertedEntries[0].OsFamily)
	assert.Equal(t, testUUID(10), store.upsertedEntries[0].FeedSourceID)
	require.Len(t, store.linkedCVEs, 1)

	// Should emit catalog.created and feed.sync_completed events.
	var catalogCreated, syncCompleted bool
	for _, evt := range bus.events {
		switch evt.Type {
		case events.CatalogCreated:
			catalogCreated = true
		case events.FeedSyncCompleted:
			syncCompleted = true
		}
	}
	assert.True(t, catalogCreated, "expected catalog.created event")
	assert.True(t, syncCompleted, "expected feed.sync_completed event")
}

func TestPipeline_Sync_skips_invalid_entries_and_continues(t *testing.T) {
	store := newTestStore()
	bus := &mockEventBus{}
	feed := &mockFeed{
		name: "nvd",
		entries: []feeds.RawEntry{
			{
				// Invalid: missing Name.
				Vendor:   "microsoft",
				OSFamily: "windows",
			},
			{
				Name:     "KB5005678",
				Vendor:   "microsoft",
				OSFamily: "windows",
				Severity: "medium",
			},
		},
		cursor: "next-cursor-2",
	}

	p := NewPipeline(store, bus, nil)
	err := p.Sync(context.Background(), feed)

	require.NoError(t, err)
	require.Len(t, store.upsertedEntries, 1, "only valid entry should be upserted")
	assert.Equal(t, "KB5005678", store.upsertedEntries[0].Name)

	// Success should still be recorded with count=1.
	assert.True(t, store.syncSuccessCalled)
	assert.Equal(t, int64(1), store.syncSuccessParams.EntriesIngested)
}

func TestPipeline_Sync_updates_sync_state_on_success(t *testing.T) {
	store := newTestStore()
	bus := &mockEventBus{}
	feed := &mockFeed{
		name: "nvd",
		entries: []feeds.RawEntry{
			{
				Name:     "KB5009999",
				Vendor:   "microsoft",
				OSFamily: "windows",
				Severity: "low",
			},
			{
				Name:     "KB5009998",
				Vendor:   "microsoft",
				OSFamily: "windows",
				Severity: "high",
			},
		},
		cursor: "cursor-after",
	}

	p := NewPipeline(store, bus, nil)
	err := p.Sync(context.Background(), feed)

	require.NoError(t, err)
	assert.True(t, store.syncStartCalled)
	assert.True(t, store.syncSuccessCalled)
	assert.Equal(t, testUUID(10), store.syncSuccessParams.FeedSourceID)
	assert.Equal(t, "cursor-after", store.syncSuccessParams.Cursor)
	assert.Equal(t, int64(2), store.syncSuccessParams.EntriesIngested)
	assert.True(t, store.syncSuccessParams.NextSyncAt.Valid, "next_sync_at should be set")
}

func TestPipeline_Sync_records_error_on_fetch_failure(t *testing.T) {
	store := newTestStore()
	bus := &mockEventBus{}
	feed := &mockFeed{
		name: "nvd",
		err:  fmt.Errorf("network timeout"),
	}

	p := NewPipeline(store, bus, nil)
	err := p.Sync(context.Background(), feed)

	require.Error(t, err)
	assert.True(t, errors.Is(err, feed.err) || assert.Contains(t, err.Error(), "network timeout"))
	assert.True(t, store.syncErrorCalled)
	assert.Equal(t, testUUID(10), store.syncErrorParams.FeedSourceID)
	assert.Contains(t, store.syncErrorParams.LastError.String, "network timeout")

	// Should emit feed.sync_failed event.
	var syncFailed bool
	for _, evt := range bus.events {
		if evt.Type == events.FeedSyncFailed {
			syncFailed = true
		}
	}
	assert.True(t, syncFailed, "expected feed.sync_failed event")
}

func TestResolveOsPackageName(t *testing.T) {
	tests := []struct {
		name     string
		entry    feeds.RawEntry
		expected string
	}{
		{
			name:     "KB-prefixed entry uses Name",
			entry:    feeds.RawEntry{Name: "KB5034441", Product: "Windows 11 Version 23H2"},
			expected: "KB5034441",
		},
		{
			name:     "KB-prefixed entry with empty product uses Name",
			entry:    feeds.RawEntry{Name: "KB5001234", Product: ""},
			expected: "KB5001234",
		},
		{
			name:     "non-KB entry with product uses Product",
			entry:    feeds.RawEntry{Name: "curl-7.88.1", Product: "curl"},
			expected: "curl",
		},
		{
			name:     "non-KB entry with empty product falls back to Name",
			entry:    feeds.RawEntry{Name: "openssl", Product: ""},
			expected: "openssl",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveOsPackageName(&tc.entry)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestPipeline_Sync_MSRC_entry_sets_os_package_name_to_KB(t *testing.T) {
	store := newTestStore()
	bus := &mockEventBus{}
	feed := &mockFeed{
		name: "msrc",
		entries: []feeds.RawEntry{
			{
				Name:     "KB5034441",
				Vendor:   "microsoft",
				OSFamily: "windows",
				Severity: "high",
				Product:  "Windows 11 Version 23H2",
			},
		},
		cursor: "msrc-cursor",
	}

	p := NewPipeline(store, bus, nil)
	err := p.Sync(context.Background(), feed)

	require.NoError(t, err)
	require.Len(t, store.upsertedEntries, 1)
	assert.Equal(t, "KB5034441", store.upsertedEntries[0].OsPackageName,
		"MSRC KB entries must use KB name as os_package_name for agent CVE matching")
	assert.Equal(t, "Windows 11 Version 23H2", store.upsertedEntries[0].Product,
		"Product field should remain unchanged")
}

func TestPipeline_Sync_handles_CVE_upsert_for_all_CVEs(t *testing.T) {
	store := newTestStore()
	bus := &mockEventBus{}
	feed := &mockFeed{
		name: "nvd",
		entries: []feeds.RawEntry{
			{
				Name:     "KB6001111",
				Vendor:   "microsoft",
				OSFamily: "windows",
				Severity: "critical",
				CVEs:     []string{"CVE-2026-1111", "CVE-2026-2222"},
			},
		},
		cursor: "cursor-cve",
	}

	p := NewPipeline(store, bus, nil)
	err := p.Sync(context.Background(), feed)

	require.NoError(t, err)
	// Both CVEs should be upserted.
	require.Len(t, store.upsertedCVEs, 2)
	assert.Equal(t, "CVE-2026-1111", store.upsertedCVEs[0].CveID)
	assert.Equal(t, "CVE-2026-2222", store.upsertedCVEs[1].CveID)
	assert.Equal(t, "nvd", store.upsertedCVEs[0].Source)

	// Both CVEs should be linked.
	require.Len(t, store.linkedCVEs, 2)
}
