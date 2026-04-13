package license

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

func makeSignedLicenseJSON(t *testing.T, priv *rsa.PrivateKey, lic licdefs.License) []byte {
	t.Helper()
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
	data, err := json.Marshal(signed)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func makeTestLicense(expiresAt time.Time) licdefs.License {
	return licdefs.License{
		LicenseID: "test-lic-001",
		Customer: licdefs.Customer{
			ID:           "cust-001",
			Name:         "Acme Corp",
			ContactEmail: "admin@acme.com",
		},
		Tier: licdefs.TierProfessional,
		Features: licdefs.Features{
			MaxEndpoints: 100,
			OSSupport:    []string{"windows", "linux"},
		},
		IssuedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:       expiresAt,
		GracePeriodDays: 30,
	}
}

func TestValidatorValidLicense(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	lic := makeTestLicense(time.Now().Add(90 * 24 * time.Hour)) // expires in 90 days
	data := makeSignedLicenseJSON(t, priv, lic)

	v := NewValidator(&priv.PublicKey)
	got, err := v.Validate(data)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got == nil {
		t.Fatal("expected license, got nil")
		return
	}
	if got.LicenseID != lic.LicenseID {
		t.Errorf("license ID = %q, want %q", got.LicenseID, lic.LicenseID)
	}
	if got.Tier != licdefs.TierProfessional {
		t.Errorf("tier = %q, want %q", got.Tier, licdefs.TierProfessional)
	}
}

func TestValidatorInvalidSignature(t *testing.T) {
	signingKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	differentKey, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	lic := makeTestLicense(time.Now().Add(90 * 24 * time.Hour))
	data := makeSignedLicenseJSON(t, signingKey, lic)

	v := NewValidator(&differentKey.PublicKey) // validate with different key
	got, err := v.Validate(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil license, got: %+v", got)
	}
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got: %v", err)
	}
}

func TestValidatorExpiredLicense(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	// Expired 100 days ago, grace period is 30 days => grace ended 70 days ago
	lic := makeTestLicense(now.Add(-100 * 24 * time.Hour))
	data := makeSignedLicenseJSON(t, priv, lic)

	v := NewValidator(&priv.PublicKey)
	v.WithClock(func() time.Time { return now })

	got, err := v.Validate(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil license, got: %+v", got)
	}
	if !errors.Is(err, ErrExpired) {
		t.Errorf("expected ErrExpired, got: %v", err)
	}
}

func TestValidatorGracePeriod(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	// Expired 10 days ago, grace period is 30 days => still in grace
	lic := makeTestLicense(now.Add(-10 * 24 * time.Hour))
	data := makeSignedLicenseJSON(t, priv, lic)

	v := NewValidator(&priv.PublicKey)
	v.WithClock(func() time.Time { return now })

	got, err := v.Validate(data)
	if err == nil {
		t.Fatal("expected ErrInGracePeriod error, got nil")
	}
	if !errors.Is(err, ErrInGracePeriod) {
		t.Errorf("expected ErrInGracePeriod, got: %v", err)
	}
	if got == nil {
		t.Fatal("expected license to be returned during grace period, got nil")
		return
	}
	if got.LicenseID != lic.LicenseID {
		t.Errorf("license ID = %q, want %q", got.LicenseID, lic.LicenseID)
	}
}

func TestValidatorClockDriftTolerance(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	// Expired 1 day ago — within 48h clock drift tolerance
	lic := makeTestLicense(now.Add(-1 * 24 * time.Hour))
	data := makeSignedLicenseJSON(t, priv, lic)

	v := NewValidator(&priv.PublicKey)
	v.WithClock(func() time.Time { return now })

	got, err := v.Validate(data)
	if err != nil {
		t.Fatalf("expected no error (within clock drift tolerance), got: %v", err)
	}
	if got == nil {
		t.Fatal("expected license, got nil")
		return
	}
	if got.LicenseID != lic.LicenseID {
		t.Errorf("license ID = %q, want %q", got.LicenseID, lic.LicenseID)
	}
}
