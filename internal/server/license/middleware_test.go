package license

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

func newTestServiceWithTier(t *testing.T, tier string) *Service {
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

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)
	if err := svc.LoadFromFile(path); err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestRequireFeatureAllowed(t *testing.T) {
	svc := newTestServiceWithTier(t, "enterprise")
	handler := RequireFeature(svc, "sso_saml")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireFeatureDenied(t *testing.T) {
	svc := newTestServiceWithTier(t, "community")
	handler := RequireFeature(svc, "sso_saml")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body["error"] != "feature_not_licensed" {
		t.Errorf("expected feature_not_licensed, got %q", body["error"])
	}
}

func TestRequireFeatureNoLicense(t *testing.T) {
	v := NewValidator(nil) // won't be used since no file loaded
	svc := NewService(v, nil)
	handler := RequireFeature(svc, "api_access")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rr.Code)
	}
}
