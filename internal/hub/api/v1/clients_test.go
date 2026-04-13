package v1_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClientQuerier implements v1.ClientQuerier for testing.
type mockClientQuerier struct {
	createFn              func(ctx context.Context, arg sqlcgen.CreateClientParams) (sqlcgen.Client, error)
	getByIDFn             func(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error)
	getByBootstrapTokenFn func(ctx context.Context, token string) (sqlcgen.Client, error)
	getByAPIKeyHashFn     func(ctx context.Context, hash pgtype.Text) (sqlcgen.Client, error)
	listFn                func(ctx context.Context, arg sqlcgen.ListClientsParams) ([]sqlcgen.Client, error)
	countFn               func(ctx context.Context, status pgtype.Text) (int64, error)
	countPendingFn        func(ctx context.Context) (int64, error)
	updateFn              func(ctx context.Context, arg sqlcgen.UpdateClientParams) (sqlcgen.Client, error)
	approveFn             func(ctx context.Context, arg sqlcgen.ApproveClientParams) (sqlcgen.Client, error)
	declineFn             func(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error)
	suspendFn             func(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error)
	deleteFn              func(ctx context.Context, id pgtype.UUID) error
	updateSyncTimeFn      func(ctx context.Context, arg sqlcgen.UpdateClientSyncTimeParams) error
	listSyncHistoryFn     func(ctx context.Context, arg sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error)
	countSyncHistoryFn    func(ctx context.Context, arg sqlcgen.CountClientSyncHistoryParams) (int64, error)
	getEndpointTrendFn    func(ctx context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error)
}

func (m *mockClientQuerier) CreateClient(ctx context.Context, arg sqlcgen.CreateClientParams) (sqlcgen.Client, error) {
	if m.createFn != nil {
		return m.createFn(ctx, arg)
	}
	return sqlcgen.Client{ID: testUUID(1), Hostname: arg.Hostname, Status: "pending"}, nil
}

func (m *mockClientQuerier) GetClientByID(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return sqlcgen.Client{}, nil
}

func (m *mockClientQuerier) GetClientByBootstrapToken(ctx context.Context, token string) (sqlcgen.Client, error) {
	if m.getByBootstrapTokenFn != nil {
		return m.getByBootstrapTokenFn(ctx, token)
	}
	return sqlcgen.Client{}, nil
}

func (m *mockClientQuerier) GetClientByAPIKeyHash(ctx context.Context, hash pgtype.Text) (sqlcgen.Client, error) {
	if m.getByAPIKeyHashFn != nil {
		return m.getByAPIKeyHashFn(ctx, hash)
	}
	return sqlcgen.Client{}, nil
}

func (m *mockClientQuerier) ListClients(ctx context.Context, arg sqlcgen.ListClientsParams) ([]sqlcgen.Client, error) {
	if m.listFn != nil {
		return m.listFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockClientQuerier) CountClients(ctx context.Context, status pgtype.Text) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, status)
	}
	return 0, nil
}

func (m *mockClientQuerier) CountPendingClients(ctx context.Context) (int64, error) {
	if m.countPendingFn != nil {
		return m.countPendingFn(ctx)
	}
	return 0, nil
}

func (m *mockClientQuerier) UpdateClient(ctx context.Context, arg sqlcgen.UpdateClientParams) (sqlcgen.Client, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, arg)
	}
	return sqlcgen.Client{}, nil
}

func (m *mockClientQuerier) ApproveClient(ctx context.Context, arg sqlcgen.ApproveClientParams) (sqlcgen.Client, error) {
	if m.approveFn != nil {
		return m.approveFn(ctx, arg)
	}
	return sqlcgen.Client{ID: arg.ID, Status: "approved", Hostname: "test-host"}, nil
}

func (m *mockClientQuerier) DeclineClient(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error) {
	if m.declineFn != nil {
		return m.declineFn(ctx, id)
	}
	return sqlcgen.Client{ID: id, Status: "declined", Hostname: "test-host"}, nil
}

func (m *mockClientQuerier) SuspendClient(ctx context.Context, id pgtype.UUID) (sqlcgen.Client, error) {
	if m.suspendFn != nil {
		return m.suspendFn(ctx, id)
	}
	return sqlcgen.Client{ID: id, Status: "suspended", Hostname: "test-host"}, nil
}

func (m *mockClientQuerier) DeleteClient(ctx context.Context, id pgtype.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockClientQuerier) UpdateClientSyncTime(ctx context.Context, arg sqlcgen.UpdateClientSyncTimeParams) error {
	if m.updateSyncTimeFn != nil {
		return m.updateSyncTimeFn(ctx, arg)
	}
	return nil
}

func (m *mockClientQuerier) ListClientSyncHistory(ctx context.Context, arg sqlcgen.ListClientSyncHistoryParams) ([]sqlcgen.ClientSyncHistory, error) {
	if m.listSyncHistoryFn != nil {
		return m.listSyncHistoryFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockClientQuerier) CountClientSyncHistory(ctx context.Context, arg sqlcgen.CountClientSyncHistoryParams) (int64, error) {
	if m.countSyncHistoryFn != nil {
		return m.countSyncHistoryFn(ctx, arg)
	}
	return 0, nil
}

func (m *mockClientQuerier) GetClientEndpointTrend(ctx context.Context, arg sqlcgen.GetClientEndpointTrendParams) ([]sqlcgen.GetClientEndpointTrendRow, error) {
	if m.getEndpointTrendFn != nil {
		return m.getEndpointTrendFn(ctx, arg)
	}
	return nil, nil
}

func TestRegister_Success(t *testing.T) {
	bus := &mockEventBus{}
	h := v1.NewClientHandler(&mockClientQuerier{}, bus, "00000000-0000-0000-0000-000000000001")

	body, _ := json.Marshal(map[string]any{
		"hostname":       "patch-mgr-01",
		"version":        "1.0.0",
		"os":             "linux",
		"endpoint_count": 50,
		"contact_email":  "admin@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/register", bytes.NewReader(body))
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "pending", resp["status"])
	assert.NotEmpty(t, resp["bootstrap_token"])
	// Bootstrap token should be 64 hex chars (32 bytes).
	token, ok := resp["bootstrap_token"].(string)
	require.True(t, ok)
	assert.Len(t, token, 64)

	// Event emitted.
	assert.Len(t, bus.emitted, 1)
}

func TestRegister_MissingHostname(t *testing.T) {
	h := v1.NewClientHandler(&mockClientQuerier{}, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	body, _ := json.Marshal(map[string]any{
		"version": "1.0.0",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/register", bytes.NewReader(body))
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "hostname")
}

func TestRegistrationStatus_Pending(t *testing.T) {
	querier := &mockClientQuerier{
		getByBootstrapTokenFn: func(_ context.Context, token string) (sqlcgen.Client, error) {
			return sqlcgen.Client{
				ID:             testUUID(1),
				Status:         "pending",
				BootstrapToken: token,
			}, nil
		},
	}
	h := v1.NewClientHandler(querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/registration-status", nil)
	req.Header.Set("X-Bootstrap-Token", "test-token-abc123")
	rec := httptest.NewRecorder()

	h.RegistrationStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "pending", resp["status"])
}

func TestRegistrationStatus_Approved(t *testing.T) {
	querier := &mockClientQuerier{
		getByBootstrapTokenFn: func(_ context.Context, _ string) (sqlcgen.Client, error) {
			return sqlcgen.Client{
				ID:         testUUID(1),
				Status:     "approved",
				ApiKeyHash: pgtype.Text{String: "$2a$10$hash", Valid: true},
			}, nil
		},
	}
	h := v1.NewClientHandler(querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/registration-status", nil)
	req.Header.Set("X-Bootstrap-Token", "test-token")
	rec := httptest.NewRecorder()

	h.RegistrationStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "approved", resp["status"])
	// API key should be nil — plaintext is only returned at approve time.
	assert.Nil(t, resp["api_key"])
}

func TestRegistrationStatus_InvalidToken(t *testing.T) {
	querier := &mockClientQuerier{
		getByBootstrapTokenFn: func(_ context.Context, _ string) (sqlcgen.Client, error) {
			return sqlcgen.Client{}, pgx.ErrNoRows
		},
	}
	h := v1.NewClientHandler(querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/registration-status", nil)
	req.Header.Set("X-Bootstrap-Token", "invalid-token")
	rec := httptest.NewRecorder()

	h.RegistrationStatus(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "client not found")
}

func TestRegistrationStatus_MissingHeader(t *testing.T) {
	h := v1.NewClientHandler(&mockClientQuerier{}, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/registration-status", nil)
	rec := httptest.NewRecorder()

	h.RegistrationStatus(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "X-Bootstrap-Token")
}

func TestListClients(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *mockClientQuerier
		wantStatus int
		wantTotal  float64
	}{
		{
			name:  "default pagination",
			query: "",
			querier: &mockClientQuerier{
				listFn: func(_ context.Context, arg sqlcgen.ListClientsParams) ([]sqlcgen.Client, error) {
					assert.Equal(t, int32(50), arg.QueryLimit)
					assert.Equal(t, int32(0), arg.QueryOffset)
					return []sqlcgen.Client{{ID: testUUID(1), Hostname: "host-1", Status: "approved"}}, nil
				},
				countFn: func(_ context.Context, _ pgtype.Text) (int64, error) {
					return 1, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  1,
		},
		{
			name:  "filter by status",
			query: "?status=pending&limit=10&offset=5",
			querier: &mockClientQuerier{
				listFn: func(_ context.Context, arg sqlcgen.ListClientsParams) ([]sqlcgen.Client, error) {
					assert.Equal(t, "pending", arg.Status.String)
					assert.Equal(t, int32(10), arg.QueryLimit)
					assert.Equal(t, int32(5), arg.QueryOffset)
					return nil, nil
				},
				countFn: func(_ context.Context, status pgtype.Text) (int64, error) {
					assert.Equal(t, "pending", status.String)
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
		{
			name:  "db error",
			query: "",
			querier: &mockClientQuerier{
				listFn: func(_ context.Context, _ sqlcgen.ListClientsParams) ([]sqlcgen.Client, error) {
					return nil, errors.New("db down")
				},
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewClientHandler(tt.querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/clients"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantTotal, body["total"])
				assert.Contains(t, body, "clients")
			}
		})
	}
}

func TestApproveClient_Success(t *testing.T) {
	bus := &mockEventBus{}
	h := v1.NewClientHandler(&mockClientQuerier{}, bus, "00000000-0000-0000-0000-000000000001")

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+id+"/approve", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Approve(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	apiKey, ok := resp["api_key"].(string)
	require.True(t, ok)
	assert.Len(t, apiKey, 64) // 32 bytes = 64 hex chars

	assert.Len(t, bus.emitted, 1)
}

func TestApproveClient_NotFound(t *testing.T) {
	querier := &mockClientQuerier{
		approveFn: func(_ context.Context, _ sqlcgen.ApproveClientParams) (sqlcgen.Client, error) {
			return sqlcgen.Client{}, pgx.ErrNoRows
		},
	}
	h := v1.NewClientHandler(querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	id := testUUIDString(99)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+id+"/approve", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Approve(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "not in pending status")
}

func TestDeclineClient_Success(t *testing.T) {
	bus := &mockEventBus{}
	h := v1.NewClientHandler(&mockClientQuerier{}, bus, "00000000-0000-0000-0000-000000000001")

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+id+"/decline", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Decline(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "declined", resp["status"])

	assert.Len(t, bus.emitted, 1)
}

func TestSuspendClient_Success(t *testing.T) {
	bus := &mockEventBus{}
	h := v1.NewClientHandler(&mockClientQuerier{}, bus, "00000000-0000-0000-0000-000000000001")

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/"+id+"/suspend", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Suspend(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "suspended", resp["status"])

	assert.Len(t, bus.emitted, 1)
}

func TestDeleteClient(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *mockClientQuerier
		bus        *mockEventBus
		wantStatus int
	}{
		{
			name:       "valid delete returns 204",
			id:         testUUIDString(1),
			querier:    &mockClientQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "bad-uuid",
			querier:    &mockClientQuerier{},
			bus:        &mockEventBus{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "db error returns 500",
			id:   testUUIDString(1),
			querier: &mockClientQuerier{
				deleteFn: func(_ context.Context, _ pgtype.UUID) error {
					return errors.New("db down")
				},
			},
			bus:        &mockEventBus{},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewClientHandler(tt.querier, tt.bus, "00000000-0000-0000-0000-000000000001")
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/clients/"+tt.id, nil)
			req = withChiURLParam(req, "id", tt.id)
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.Delete(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusNoContent {
				assert.Len(t, tt.bus.emitted, 1)
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *mockClientQuerier
		wantStatus int
	}{
		{
			name: "valid ID returns client",
			id:   testUUIDString(1),
			querier: &mockClientQuerier{
				getByIDFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Client, error) {
					return sqlcgen.Client{ID: testUUID(1), Hostname: "host-1", Status: "approved"}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID returns 400",
			id:         "not-a-uuid",
			querier:    &mockClientQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found returns 404",
			id:   testUUIDString(99),
			querier: &mockClientQuerier{
				getByIDFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Client, error) {
					return sqlcgen.Client{}, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewClientHandler(tt.querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/"+tt.id, nil)
			req = withChiURLParam(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body, "client")
			}
		})
	}
}

func TestRegister_BootstrapTokenIsHashed(t *testing.T) {
	var storedHash string
	querier := &mockClientQuerier{
		createFn: func(_ context.Context, arg sqlcgen.CreateClientParams) (sqlcgen.Client, error) {
			storedHash = arg.BootstrapToken
			return sqlcgen.Client{ID: testUUID(1), Hostname: arg.Hostname, Status: "pending"}, nil
		},
	}
	bus := &mockEventBus{}
	h := v1.NewClientHandler(querier, bus, "00000000-0000-0000-0000-000000000001")

	body, _ := json.Marshal(map[string]any{"hostname": "test-host"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/register", bytes.NewReader(body))
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	plaintextToken, ok := resp["bootstrap_token"].(string)
	require.True(t, ok, "bootstrap_token must be a string")

	// The stored value should be a SHA-256 hash, not the plaintext.
	assert.Len(t, storedHash, 64) // SHA-256 hex = 64 chars
	assert.NotEqual(t, plaintextToken, storedHash, "bootstrap token must be hashed before storage")

	// Verify the hash matches.
	expectedHash := hashForTest(plaintextToken)
	assert.Equal(t, expectedHash, storedHash)
}

func TestRegistrationStatus_UsesHashedToken(t *testing.T) {
	plaintext := "my-bootstrap-token-abc123"
	expectedHash := hashForTest(plaintext)

	querier := &mockClientQuerier{
		getByBootstrapTokenFn: func(_ context.Context, token string) (sqlcgen.Client, error) {
			// The handler should pass the hashed token, not plaintext.
			assert.Equal(t, expectedHash, token, "handler must hash the token before lookup")
			return sqlcgen.Client{ID: testUUID(1), Status: "pending"}, nil
		},
	}
	h := v1.NewClientHandler(querier, &mockEventBus{}, "00000000-0000-0000-0000-000000000001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/clients/registration-status", nil)
	req.Header.Set("X-Bootstrap-Token", plaintext)
	rec := httptest.NewRecorder()

	h.RegistrationStatus(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRegister_UsesTenantFromContext(t *testing.T) {
	var storedTenantID pgtype.UUID
	querier := &mockClientQuerier{
		createFn: func(_ context.Context, arg sqlcgen.CreateClientParams) (sqlcgen.Client, error) {
			storedTenantID = arg.TenantID
			return sqlcgen.Client{ID: testUUID(1), Hostname: arg.Hostname, Status: "pending"}, nil
		},
	}
	bus := &mockEventBus{}
	// Default tenant is different from the one in context.
	h := v1.NewClientHandler(querier, bus, "00000000-0000-0000-0000-000000000099")

	body, _ := json.Marshal(map[string]any{"hostname": "test-host"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/register", bytes.NewReader(body))
	// Set tenant context to a specific ID — handler should prefer this over default.
	ctx := tenant.WithTenantID(req.Context(), "00000000-0000-0000-0000-000000000042")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	// Verify the tenant from context was used, not the default.
	expected, _ := parseUUIDForTest("00000000-0000-0000-0000-000000000042")
	assert.Equal(t, expected, storedTenantID, "should use tenant from auth context, not default")
}

func TestRegister_FallsBackToDefaultTenant(t *testing.T) {
	var storedTenantID pgtype.UUID
	querier := &mockClientQuerier{
		createFn: func(_ context.Context, arg sqlcgen.CreateClientParams) (sqlcgen.Client, error) {
			storedTenantID = arg.TenantID
			return sqlcgen.Client{ID: testUUID(1), Hostname: arg.Hostname, Status: "pending"}, nil
		},
	}
	bus := &mockEventBus{}
	h := v1.NewClientHandler(querier, bus, "00000000-0000-0000-0000-000000000099")

	body, _ := json.Marshal(map[string]any{"hostname": "test-host"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/register", bytes.NewReader(body))
	// No tenant context — should fall back to default.
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	expected, _ := parseUUIDForTest("00000000-0000-0000-0000-000000000099")
	assert.Equal(t, expected, storedTenantID, "should fall back to configured default tenant")
}

func TestRegister_EmptyDefaultTenantID_NoContext(t *testing.T) {
	h := v1.NewClientHandler(&mockClientQuerier{}, &mockEventBus{}, "")

	body, _ := json.Marshal(map[string]any{"hostname": "test-host"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients/register", bytes.NewReader(body))
	// No tenant context — empty default should fail gracefully.
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "parse tenant ID")
}

func parseUUIDForTest(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	return u, err
}

func hashForTest(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
