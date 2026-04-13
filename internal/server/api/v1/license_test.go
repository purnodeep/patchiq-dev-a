package v1_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/skenzeriq/patchiq/internal/server/api/v1"
	serverlicense "github.com/skenzeriq/patchiq/internal/server/license"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

type stubEndpointCounter struct {
	count int
	err   error
}

func (s *stubEndpointCounter) CountAllEndpoints() (int, error) {
	return s.count, s.err
}

func newTestLicenseService(t *testing.T, tier string) *serverlicense.Service {
	t.Helper()
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := licdefs.TierTemplate(tier)
	if err != nil {
		t.Fatal(err)
	}
	lic := licdefs.License{
		LicenseID:       "LIC-TEST",
		Tier:            tier,
		Features:        tmpl,
		IssuedAt:        time.Now().Add(-24 * time.Hour),
		ExpiresAt:       time.Now().Add(365 * 24 * time.Hour),
		GracePeriodDays: 30,
	}
	canonical, err := json.Marshal(lic)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := crypto.SignPayload(priv, canonical)
	if err != nil {
		t.Fatal(err)
	}
	signed := licdefs.SignedLicense{
		License:   lic,
		Signature: base64.StdEncoding.EncodeToString(sig),
	}
	data, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "test.piq-license")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	v := serverlicense.NewValidator(&priv.PublicKey)
	svc := serverlicense.NewService(v, nil)
	if err := svc.LoadFromFile(path); err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestLicenseStatusHandler(t *testing.T) {
	svc := newTestLicenseService(t, licdefs.TierEnterprise)
	counter := &stubEndpointCounter{count: 342}
	h := v1.NewLicenseHandler(svc, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/license/status", nil)
	rec := httptest.NewRecorder()

	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var status licdefs.LicenseStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if status.Tier != licdefs.TierEnterprise {
		t.Errorf("expected tier %q, got %q", licdefs.TierEnterprise, status.Tier)
	}
	if status.EndpointUsage.Current != 342 {
		t.Errorf("expected endpoint_usage.current=342, got %d", status.EndpointUsage.Current)
	}
	if status.EndpointUsage.Limit != 10000 {
		t.Errorf("expected endpoint_usage.limit=10000, got %d", status.EndpointUsage.Limit)
	}
}

func TestLicenseStatusNoLicense(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	v := serverlicense.NewValidator(&priv.PublicKey)
	svc := serverlicense.NewService(v, nil)
	// No license loaded — should fall back to community defaults.

	h := v1.NewLicenseHandler(svc, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/license/status", nil)
	rec := httptest.NewRecorder()

	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var status licdefs.LicenseStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if status.Tier != licdefs.TierCommunity {
		t.Errorf("expected tier %q, got %q", licdefs.TierCommunity, status.Tier)
	}
}

func TestLicenseStatusCounterError(t *testing.T) {
	svc := newTestLicenseService(t, licdefs.TierEnterprise)
	counter := &stubEndpointCounter{err: errors.New("db connection failed")}
	h := v1.NewLicenseHandler(svc, counter)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/license/status", nil)
	rec := httptest.NewRecorder()

	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var status licdefs.LicenseStatus
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if status.EndpointUsage.Current != 0 {
		t.Errorf("expected endpoint_usage.current=0 on error, got %d", status.EndpointUsage.Current)
	}
}
