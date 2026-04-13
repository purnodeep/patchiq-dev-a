package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
)

// --- Fakes for invite tests ---

type fakeInviteQuerier struct {
	createInvitationFn    func(ctx context.Context, arg sqlcgen.CreateInvitationParams) (sqlcgen.Invitation, error)
	getInvitationByCodeFn func(ctx context.Context, code pgtype.UUID) (sqlcgen.Invitation, error)
	claimInvitationFn     func(ctx context.Context, code pgtype.UUID) (sqlcgen.Invitation, error)
	listInvitationsFn     func(ctx context.Context, arg sqlcgen.ListInvitationsParams) ([]sqlcgen.Invitation, error)
	assignUserRoleFn      func(ctx context.Context, arg sqlcgen.AssignUserRoleParams) error
	getRoleByIDFn         func(ctx context.Context, arg sqlcgen.GetRoleByIDParams) (sqlcgen.Role, error)
	getTenantByIDFn       func(ctx context.Context, id pgtype.UUID) (sqlcgen.Tenant, error)
}

func (f *fakeInviteQuerier) CreateInvitation(ctx context.Context, arg sqlcgen.CreateInvitationParams) (sqlcgen.Invitation, error) {
	if f.createInvitationFn != nil {
		return f.createInvitationFn(ctx, arg)
	}
	return sqlcgen.Invitation{}, errors.New("CreateInvitation not implemented")
}

func (f *fakeInviteQuerier) GetInvitationByCode(ctx context.Context, code pgtype.UUID) (sqlcgen.Invitation, error) {
	if f.getInvitationByCodeFn != nil {
		return f.getInvitationByCodeFn(ctx, code)
	}
	return sqlcgen.Invitation{}, errors.New("GetInvitationByCode not implemented")
}

func (f *fakeInviteQuerier) ClaimInvitation(ctx context.Context, code pgtype.UUID) (sqlcgen.Invitation, error) {
	if f.claimInvitationFn != nil {
		return f.claimInvitationFn(ctx, code)
	}
	return sqlcgen.Invitation{}, errors.New("ClaimInvitation not implemented")
}

func (f *fakeInviteQuerier) ListInvitations(ctx context.Context, arg sqlcgen.ListInvitationsParams) ([]sqlcgen.Invitation, error) {
	if f.listInvitationsFn != nil {
		return f.listInvitationsFn(ctx, arg)
	}
	return nil, errors.New("ListInvitations not implemented")
}

func (f *fakeInviteQuerier) AssignUserRole(ctx context.Context, arg sqlcgen.AssignUserRoleParams) error {
	if f.assignUserRoleFn != nil {
		return f.assignUserRoleFn(ctx, arg)
	}
	return errors.New("AssignUserRole not implemented")
}

func (f *fakeInviteQuerier) GetRoleByID(ctx context.Context, arg sqlcgen.GetRoleByIDParams) (sqlcgen.Role, error) {
	if f.getRoleByIDFn != nil {
		return f.getRoleByIDFn(ctx, arg)
	}
	return sqlcgen.Role{}, errors.New("GetRoleByID not implemented")
}

func (f *fakeInviteQuerier) GetTenantByID(ctx context.Context, id pgtype.UUID) (sqlcgen.Tenant, error) {
	if f.getTenantByIDFn != nil {
		return f.getTenantByIDFn(ctx, id)
	}
	return sqlcgen.Tenant{}, errors.New("GetTenantByID not implemented")
}

type fakeZitadelForInvite struct {
	createUserFn    func(ctx context.Context, email, name, password string) (string, error)
	authenticateFn  func(ctx context.Context, email, password string) (*AuthResult, error)
	exchangeTokenFn func(ctx context.Context, sessionToken string) (string, error)
}

func (f *fakeZitadelForInvite) CreateUser(ctx context.Context, email, name, password string) (string, error) {
	if f.createUserFn != nil {
		return f.createUserFn(ctx, email, name, password)
	}
	return "", errors.New("CreateUser not implemented")
}

func (f *fakeZitadelForInvite) Authenticate(ctx context.Context, email, password string) (*AuthResult, error) {
	if f.authenticateFn != nil {
		return f.authenticateFn(ctx, email, password)
	}
	return nil, errors.New("Authenticate not implemented")
}

func (f *fakeZitadelForInvite) ExchangeToken(ctx context.Context, sessionToken string) (string, error) {
	if f.exchangeTokenFn != nil {
		return f.exchangeTokenFn(ctx, sessionToken)
	}
	return "", errors.New("ExchangeToken not implemented")
}

// --- Helpers ---

func makePgUUID(s string) pgtype.UUID {
	parsed, _ := uuid.Parse(s)
	return pgtype.UUID{Bytes: parsed, Valid: true}
}

func makeInvitation(tenantID, code, email, roleID string) sqlcgen.Invitation {
	return sqlcgen.Invitation{
		ID:        makePgUUID(uuid.New().String()),
		TenantID:  makePgUUID(tenantID),
		Code:      makePgUUID(code),
		Email:     email,
		RoleID:    makePgUUID(roleID),
		InvitedBy: "admin-user-id",
		Status:    "pending",
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

// --- Tests ---

func TestInviteHandler_CreateInvite(t *testing.T) {
	tenantID := uuid.New().String()
	roleID := uuid.New().String()
	inviteCode := uuid.New().String()

	q := &fakeInviteQuerier{
		createInvitationFn: func(_ context.Context, arg sqlcgen.CreateInvitationParams) (sqlcgen.Invitation, error) {
			return makeInvitation(tenantID, inviteCode, "newuser@acme.com", roleID), nil
		},
	}
	bus := &fakeEventBus{}

	h := NewInviteHandler(q, nil, bus, SessionConfig{}, "http://localhost:3001")

	body := fmt.Sprintf(`{"email":"newuser@acme.com","role_id":"%s"}`, roleID)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invite", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	ctx := tenant.WithTenantID(req.Context(), tenantID)
	ctx = user.WithUserID(ctx, "admin-user-id")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreateInvite(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] == nil || resp["code"] == "" {
		t.Error("response should contain invite code")
	}
	if resp["invite_url"] == nil || resp["invite_url"] == "" {
		t.Error("response should contain invite_url")
	}

	if len(bus.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.events))
	}
	if bus.events[0].Type != "invitation.created" {
		t.Errorf("expected invitation.created event, got %s", bus.events[0].Type)
	}
}

func TestInviteHandler_CreateInvite_MissingFields(t *testing.T) {
	h := NewInviteHandler(&fakeInviteQuerier{}, nil, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invite", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	ctx := tenant.WithTenantID(req.Context(), uuid.New().String())
	ctx = user.WithUserID(ctx, "admin-user-id")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	h.CreateInvite(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInviteHandler_ValidateInvite(t *testing.T) {
	tenantID := uuid.New().String()
	roleID := uuid.New().String()
	code := uuid.New().String()

	q := &fakeInviteQuerier{
		getInvitationByCodeFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Invitation, error) {
			return makeInvitation(tenantID, code, "newuser@acme.com", roleID), nil
		},
		getRoleByIDFn: func(_ context.Context, _ sqlcgen.GetRoleByIDParams) (sqlcgen.Role, error) {
			return sqlcgen.Role{Name: "Operator"}, nil
		},
		getTenantByIDFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Tenant, error) {
			return sqlcgen.Tenant{Name: "Acme Corp"}, nil
		},
	}

	h := NewInviteHandler(q, nil, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/invite/"+code, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.ValidateInvite(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["email"] != "newuser@acme.com" {
		t.Errorf("expected email newuser@acme.com, got %v", resp["email"])
	}
	if resp["tenant_name"] != "Acme Corp" {
		t.Errorf("expected tenant_name Acme Corp, got %v", resp["tenant_name"])
	}
	if resp["role_name"] != "Operator" {
		t.Errorf("expected role_name Operator, got %v", resp["role_name"])
	}
}

func TestInviteHandler_ValidateInvite_InvalidCode(t *testing.T) {
	q := &fakeInviteQuerier{
		getInvitationByCodeFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Invitation, error) {
			return sqlcgen.Invitation{}, errors.New("no rows in result set")
		},
	}

	h := NewInviteHandler(q, nil, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	code := uuid.New().String()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/invite/"+code, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", code)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.ValidateInvite(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestInviteHandler_ValidateInvite_BadUUID(t *testing.T) {
	h := NewInviteHandler(&fakeInviteQuerier{}, nil, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/invite/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("code", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.ValidateInvite(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterHandler_Success(t *testing.T) {
	tenantID := uuid.New().String()
	roleID := uuid.New().String()
	code := uuid.New().String()
	zitadelUserID := "zitadel-user-123"
	inv := makeInvitation(tenantID, code, "newuser@acme.com", roleID)

	var claimed bool
	var roleAssigned bool

	q := &fakeInviteQuerier{
		getInvitationByCodeFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Invitation, error) {
			return inv, nil
		},
		claimInvitationFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Invitation, error) {
			claimed = true
			claimedInv := inv
			claimedInv.Status = "claimed"
			claimedInv.ClaimedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
			return claimedInv, nil
		},
		assignUserRoleFn: func(_ context.Context, arg sqlcgen.AssignUserRoleParams) error {
			roleAssigned = true
			if arg.UserID != zitadelUserID {
				t.Errorf("expected user ID %s, got %s", zitadelUserID, arg.UserID)
			}
			return nil
		},
	}

	zitadel := &fakeZitadelForInvite{
		createUserFn: func(_ context.Context, email, name, password string) (string, error) {
			if email != "newuser@acme.com" {
				t.Errorf("unexpected email: %s", email)
			}
			return zitadelUserID, nil
		},
		authenticateFn: func(_ context.Context, _, _ string) (*AuthResult, error) {
			return &AuthResult{SessionID: "sess-1", SessionToken: "token-1"}, nil
		},
		exchangeTokenFn: func(_ context.Context, _ string) (string, error) {
			return "jwt-token-abc", nil
		},
	}

	bus := &fakeEventBus{}

	cfg := SessionConfig{
		CookieName:     "piq_session",
		AccessTokenTTL: 24 * time.Hour,
	}

	h := NewInviteHandler(q, zitadel, bus, cfg, "http://localhost:3001")

	body := fmt.Sprintf(`{"code":"%s","name":"Jane Doe","password":"SecureP@ss123"}`, code)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if !claimed {
		t.Error("invitation should have been claimed")
	}
	if !roleAssigned {
		t.Error("role should have been assigned")
	}

	// Verify cookie was set.
	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "piq_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected piq_session cookie to be set")
	}
	if sessionCookie.Value != "jwt-token-abc" {
		t.Errorf("expected cookie value jwt-token-abc, got %s", sessionCookie.Value)
	}

	// Verify events: invitation.claimed, user.registered, auth.login
	if len(bus.events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(bus.events))
	}
	expectedTypes := []string{"invitation.claimed", "user.registered", "auth.login"}
	for i, et := range expectedTypes {
		if bus.events[i].Type != et {
			t.Errorf("event[%d]: expected %s, got %s", i, et, bus.events[i].Type)
		}
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["user_id"] != zitadelUserID {
		t.Errorf("expected user_id %s, got %v", zitadelUserID, resp["user_id"])
	}
}

func TestRegisterHandler_ExpiredInvite(t *testing.T) {
	q := &fakeInviteQuerier{
		getInvitationByCodeFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Invitation, error) {
			return sqlcgen.Invitation{}, errors.New("no rows in result set")
		},
	}

	h := NewInviteHandler(q, &fakeZitadelForInvite{}, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	code := uuid.New().String()
	body := fmt.Sprintf(`{"code":"%s","name":"Jane Doe","password":"SecureP@ss123"}`, code)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterHandler_AlreadyClaimed(t *testing.T) {
	q := &fakeInviteQuerier{
		getInvitationByCodeFn: func(_ context.Context, _ pgtype.UUID) (sqlcgen.Invitation, error) {
			return sqlcgen.Invitation{}, errors.New("no rows in result set")
		},
	}

	h := NewInviteHandler(q, &fakeZitadelForInvite{}, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	code := uuid.New().String()
	body := fmt.Sprintf(`{"code":"%s","name":"Jane Doe","password":"SecureP@ss123"}`, code)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRegisterHandler_WeakPassword(t *testing.T) {
	h := NewInviteHandler(&fakeInviteQuerier{}, &fakeZitadelForInvite{}, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	code := uuid.New().String()
	body := fmt.Sprintf(`{"code":"%s","name":"Jane Doe","password":"short"}`, code)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if msg, _ := resp["message"].(string); !strings.Contains(msg, "8 characters") {
		t.Errorf("expected password length error, got: %s", msg)
	}
}

func TestRegisterHandler_MissingFields(t *testing.T) {
	h := NewInviteHandler(&fakeInviteQuerier{}, &fakeZitadelForInvite{}, &fakeEventBus{}, SessionConfig{}, "http://localhost:3001")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
