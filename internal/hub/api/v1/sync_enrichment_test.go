package v1_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncHandler_ParsesSummaryHeaders(t *testing.T) {
	clientUUID := pgtype.UUID{Valid: true}
	clientUUID.Bytes = [16]byte{0xaa, 0xbb, 0xcc, 0xdd, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}
	tenantUUID := pgtype.UUID{Valid: true}
	tenantUUID.Bytes = [16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	mock := &mockSyncQuerier{
		listUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
			return []sqlcgen.PatchCatalog{}, nil
		},
		listDeletedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
			return nil, nil
		},
		getClientFn: func(_ context.Context, _ pgtype.Text) (sqlcgen.Client, error) {
			return sqlcgen.Client{
				ID:            clientUUID,
				TenantID:      tenantUUID,
				EndpointCount: 80,
			}, nil
		},
	}

	handler := v1.NewSyncHandler(mock, "test-api-key", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("X-Endpoint-Count", "80")
	req.Header.Set("X-Os-Summary", `{"linux":50,"windows":30}`)
	req.Header.Set("X-Endpoint-Status-Summary", `{"connected":70,"disconnected":10}`)
	req.Header.Set("X-Compliance-Summary", `{"NIST 800-53":87,"PCI-DSS":92}`)

	rr := httptest.NewRecorder()
	handler.Sync(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "response body: %s", rr.Body.String())

	// Verify client summaries were updated.
	require.NotNil(t, mock.updatedSummaries, "expected UpdateClientSummaries to be called")
	assert.Equal(t, int32(80), mock.updatedSummaries.EndpointCount)
	assert.Equal(t, clientUUID, mock.updatedSummaries.ID)
	assert.JSONEq(t, `{"linux":50,"windows":30}`, string(mock.updatedSummaries.OsSummary))
	assert.JSONEq(t, `{"connected":70,"disconnected":10}`, string(mock.updatedSummaries.EndpointStatusSummary))
	assert.JSONEq(t, `{"NIST 800-53":87,"PCI-DSS":92}`, string(mock.updatedSummaries.ComplianceSummary))

	// Verify sync history was inserted.
	require.NotNil(t, mock.insertedHistory, "expected InsertClientSyncHistory to be called")
	assert.Equal(t, int32(80), mock.insertedHistory.EndpointCount)
	assert.Equal(t, "success", mock.insertedHistory.Status)
	assert.Equal(t, clientUUID, mock.insertedHistory.ClientID)
	assert.Equal(t, tenantUUID, mock.insertedHistory.TenantID)
	assert.True(t, mock.insertedHistory.StartedAt.Valid)
	assert.True(t, mock.insertedHistory.FinishedAt.Valid)
	assert.True(t, mock.insertedHistory.DurationMs.Valid)
	assert.Equal(t, int32(0), mock.insertedHistory.EntriesDelivered)
	assert.Equal(t, int32(0), mock.insertedHistory.DeletesDelivered)
}

func TestSyncHandler_WorksWithoutSummaryHeaders(t *testing.T) {
	clientUUID := pgtype.UUID{Valid: true}
	clientUUID.Bytes = [16]byte{0xaa, 0xbb, 0xcc, 0xdd, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c}
	tenantUUID := pgtype.UUID{Valid: true}
	tenantUUID.Bytes = [16]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}

	mock := &mockSyncQuerier{
		listUpdatedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]sqlcgen.PatchCatalog, error) {
			return []sqlcgen.PatchCatalog{}, nil
		},
		listDeletedFn: func(_ context.Context, _ pgtype.Timestamptz) ([]pgtype.UUID, error) {
			return nil, nil
		},
		getClientFn: func(_ context.Context, _ pgtype.Text) (sqlcgen.Client, error) {
			return sqlcgen.Client{
				ID:       clientUUID,
				TenantID: tenantUUID,
			}, nil
		},
	}

	handler := v1.NewSyncHandler(mock, "test-api-key", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync?since=2026-01-01T00:00:00Z", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")

	rr := httptest.NewRecorder()
	handler.Sync(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "response body: %s", rr.Body.String())

	// Sync should still succeed — summaries will use defaults.
	var resp map[string]any
	err := json.NewDecoder(rr.Body).Decode(&resp)
	require.NoError(t, err)
	_, ok := resp["server_time"]
	assert.True(t, ok, "expected server_time in response")

	// Client summaries should still be updated (with zero/empty defaults).
	require.NotNil(t, mock.updatedSummaries, "expected UpdateClientSummaries to be called")
	assert.Equal(t, int32(0), mock.updatedSummaries.EndpointCount)
	assert.JSONEq(t, `{}`, string(mock.updatedSummaries.OsSummary))

	// Sync history should still be inserted.
	require.NotNil(t, mock.insertedHistory, "expected InsertClientSyncHistory to be called")
	assert.Equal(t, "success", mock.insertedHistory.Status)
}
