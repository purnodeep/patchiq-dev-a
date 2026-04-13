package v1_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	v1 "github.com/skenzeriq/patchiq/internal/hub/api/v1"
	"github.com/skenzeriq/patchiq/internal/hub/store/sqlcgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLicenseQuerier implements v1.LicenseQuerier for testing.
type mockLicenseQuerier struct {
	createFn                 func(ctx context.Context, arg sqlcgen.CreateLicenseParams) (sqlcgen.License, error)
	getByIDFn                func(ctx context.Context, id pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error)
	listFn                   func(ctx context.Context, arg sqlcgen.ListLicensesParams) ([]sqlcgen.ListLicensesRow, error)
	countFn                  func(ctx context.Context, arg sqlcgen.CountLicensesParams) (int64, error)
	revokeFn                 func(ctx context.Context, id pgtype.UUID) (sqlcgen.License, error)
	assignFn                 func(ctx context.Context, arg sqlcgen.AssignLicenseToClientParams) (sqlcgen.License, error)
	renewFn                  func(ctx context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error)
	usageHistoryFn           func(ctx context.Context, arg sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error)
	listAuditByResourceIDFn  func(ctx context.Context, arg sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error)
	countAuditByResourceIDFn func(ctx context.Context, arg sqlcgen.CountAuditEventsByResourceIDParams) (int64, error)
}

func (m *mockLicenseQuerier) CreateLicense(ctx context.Context, arg sqlcgen.CreateLicenseParams) (sqlcgen.License, error) {
	if m.createFn != nil {
		return m.createFn(ctx, arg)
	}
	return sqlcgen.License{ID: testUUID(1), Tier: arg.Tier, CustomerName: arg.CustomerName}, nil
}

func (m *mockLicenseQuerier) GetLicenseByID(ctx context.Context, id pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return sqlcgen.GetLicenseByIDRow{}, nil
}

func (m *mockLicenseQuerier) ListLicenses(ctx context.Context, arg sqlcgen.ListLicensesParams) ([]sqlcgen.ListLicensesRow, error) {
	if m.listFn != nil {
		return m.listFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockLicenseQuerier) CountLicenses(ctx context.Context, arg sqlcgen.CountLicensesParams) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, arg)
	}
	return 0, nil
}

func (m *mockLicenseQuerier) RevokeLicense(ctx context.Context, id pgtype.UUID) (sqlcgen.License, error) {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, id)
	}
	return sqlcgen.License{ID: id, Tier: "professional", CustomerName: "Test Co"}, nil
}

func (m *mockLicenseQuerier) AssignLicenseToClient(ctx context.Context, arg sqlcgen.AssignLicenseToClientParams) (sqlcgen.License, error) {
	if m.assignFn != nil {
		return m.assignFn(ctx, arg)
	}
	return sqlcgen.License{ID: arg.ID, ClientID: arg.ClientID}, nil
}

func (m *mockLicenseQuerier) RenewLicense(ctx context.Context, arg sqlcgen.RenewLicenseParams) (sqlcgen.License, error) {
	if m.renewFn != nil {
		return m.renewFn(ctx, arg)
	}
	return sqlcgen.License{ID: arg.ID, Tier: "professional", CustomerName: "Test Co"}, nil
}

func (m *mockLicenseQuerier) GetLicenseUsageHistory(ctx context.Context, arg sqlcgen.GetLicenseUsageHistoryParams) ([]sqlcgen.GetLicenseUsageHistoryRow, error) {
	if m.usageHistoryFn != nil {
		return m.usageHistoryFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockLicenseQuerier) ListAuditEventsByResourceID(ctx context.Context, arg sqlcgen.ListAuditEventsByResourceIDParams) ([]sqlcgen.AuditEvent, error) {
	if m.listAuditByResourceIDFn != nil {
		return m.listAuditByResourceIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockLicenseQuerier) CountAuditEventsByResourceID(ctx context.Context, arg sqlcgen.CountAuditEventsByResourceIDParams) (int64, error) {
	if m.countAuditByResourceIDFn != nil {
		return m.countAuditByResourceIDFn(ctx, arg)
	}
	return 0, nil
}

func TestCreateLicense_Success(t *testing.T) {
	bus := &mockEventBus{}
	h := v1.NewLicenseHandler(&mockLicenseQuerier{}, bus)

	body, _ := json.Marshal(map[string]any{
		"customer_name":  "Acme Corp",
		"customer_email": "admin@acme.com",
		"tier":           "professional",
		"max_endpoints":  100,
		"expires_at":     "2027-01-01T00:00:00Z",
		"notes":          "Initial license",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses", bytes.NewReader(body))
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp, "license")

	assert.Len(t, bus.emitted, 1)
}

func TestCreateLicense_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name:    "missing customer_name",
			body:    map[string]any{"tier": "professional", "max_endpoints": 100, "expires_at": "2027-01-01T00:00:00Z"},
			wantErr: "customer_name",
		},
		{
			name:    "missing tier",
			body:    map[string]any{"customer_name": "Acme", "max_endpoints": 100, "expires_at": "2027-01-01T00:00:00Z"},
			wantErr: "tier",
		},
		{
			name:    "invalid tier",
			body:    map[string]any{"customer_name": "Acme", "tier": "invalid", "max_endpoints": 100, "expires_at": "2027-01-01T00:00:00Z"},
			wantErr: "tier must be one of",
		},
		{
			name:    "missing max_endpoints",
			body:    map[string]any{"customer_name": "Acme", "tier": "professional", "expires_at": "2027-01-01T00:00:00Z"},
			wantErr: "max_endpoints",
		},
		{
			name:    "missing expires_at",
			body:    map[string]any{"customer_name": "Acme", "tier": "professional", "max_endpoints": 100},
			wantErr: "expires_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewLicenseHandler(&mockLicenseQuerier{}, &mockEventBus{})
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses", bytes.NewReader(body))
			req = withTenantCtx(req)
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantErr)
		})
	}
}

func TestListLicenses(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		querier    *mockLicenseQuerier
		wantStatus int
		wantTotal  float64
	}{
		{
			name:  "default pagination",
			query: "",
			querier: &mockLicenseQuerier{
				listFn: func(_ context.Context, arg sqlcgen.ListLicensesParams) ([]sqlcgen.ListLicensesRow, error) {
					assert.Equal(t, int32(50), arg.QueryLimit)
					return []sqlcgen.ListLicensesRow{{ID: testUUID(1), Tier: "professional"}}, nil
				},
				countFn: func(_ context.Context, _ sqlcgen.CountLicensesParams) (int64, error) {
					return 1, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  1,
		},
		{
			name:  "with tier and status filters",
			query: "?tier=enterprise&status=active",
			querier: &mockLicenseQuerier{
				listFn: func(_ context.Context, arg sqlcgen.ListLicensesParams) ([]sqlcgen.ListLicensesRow, error) {
					assert.Equal(t, "enterprise", arg.Tier.String)
					assert.Equal(t, "active", arg.StatusFilter.String)
					return nil, nil
				},
				countFn: func(_ context.Context, arg sqlcgen.CountLicensesParams) (int64, error) {
					assert.Equal(t, "enterprise", arg.Tier.String)
					assert.Equal(t, "active", arg.StatusFilter.String)
					return 0, nil
				},
			},
			wantStatus: http.StatusOK,
			wantTotal:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewLicenseHandler(tt.querier, &mockEventBus{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses"+tt.query, nil)
			rec := httptest.NewRecorder()

			h.List(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Equal(t, tt.wantTotal, body["total"])
				assert.Contains(t, body, "licenses")
			}
		})
	}
}

func TestGetLicense(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		querier    *mockLicenseQuerier
		wantStatus int
	}{
		{
			name: "valid ID",
			id:   testUUIDString(1),
			querier: &mockLicenseQuerier{
				getByIDFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
					return sqlcgen.GetLicenseByIDRow{ID: testUUID(1), Tier: "professional"}, nil
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid UUID",
			id:         "bad-uuid",
			querier:    &mockLicenseQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "not found",
			id:   testUUIDString(99),
			querier: &mockLicenseQuerier{
				getByIDFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.GetLicenseByIDRow, error) {
					return sqlcgen.GetLicenseByIDRow{}, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewLicenseHandler(tt.querier, &mockEventBus{})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+tt.id, nil)
			req = withChiURLParam(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Get(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body, "license")
			}
		})
	}
}

func TestRevokeLicense_Success(t *testing.T) {
	bus := &mockEventBus{}
	h := v1.NewLicenseHandler(&mockLicenseQuerier{}, bus)

	id := testUUIDString(1)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/"+id+"/revoke", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Revoke(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "revoked", resp["status"])

	assert.Len(t, bus.emitted, 1)
}

func TestRevokeLicense_NotFound(t *testing.T) {
	querier := &mockLicenseQuerier{
		revokeFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.License, error) {
			return sqlcgen.License{}, pgx.ErrNoRows
		},
	}
	h := v1.NewLicenseHandler(querier, &mockEventBus{})

	id := testUUIDString(99)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/"+id+"/revoke", nil)
	req = withChiURLParam(req, "id", id)
	req = withTenantCtx(req)
	rec := httptest.NewRecorder()

	h.Revoke(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "already revoked")
}

func TestAssignLicense(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		body       map[string]any
		querier    *mockLicenseQuerier
		wantStatus int
	}{
		{
			name:       "valid assignment",
			id:         testUUIDString(1),
			body:       map[string]any{"client_id": testUUIDString(2)},
			querier:    &mockLicenseQuerier{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing client_id",
			id:         testUUIDString(1),
			body:       map[string]any{},
			querier:    &mockLicenseQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid license UUID",
			id:         "bad-uuid",
			body:       map[string]any{"client_id": testUUIDString(2)},
			querier:    &mockLicenseQuerier{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "license not found",
			id:   testUUIDString(99),
			body: map[string]any{"client_id": testUUIDString(2)},
			querier: &mockLicenseQuerier{
				assignFn: func(_ context.Context, _ sqlcgen.AssignLicenseToClientParams) (sqlcgen.License, error) {
					return sqlcgen.License{}, pgx.ErrNoRows
				},
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := v1.NewLicenseHandler(tt.querier, &mockEventBus{})
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/licenses/"+tt.id+"/assign", bytes.NewReader(body))
			req = withChiURLParam(req, "id", tt.id)
			rec := httptest.NewRecorder()

			h.Assign(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				var body map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
				assert.Contains(t, body, "license")
			}
		})
	}
}
