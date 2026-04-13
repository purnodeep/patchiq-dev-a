package license_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	hublicense "github.com/skenzeriq/patchiq/internal/hub/license"
	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

func TestGenerateAndSign(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	gen := hublicense.NewGenerator(key)

	params := hublicense.GenerateParams{
		LicenseID:    "lic-001",
		CustomerID:   "cust-001",
		CustomerName: "Acme Corp",
		ContactEmail: "admin@acme.com",
		Tier:         "enterprise",
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
	}

	signed, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if signed.LicenseID != "lic-001" {
		t.Errorf("license_id = %q, want %q", signed.LicenseID, "lic-001")
	}
	if signed.Customer.Name != "Acme Corp" {
		t.Errorf("customer.name = %q, want %q", signed.Customer.Name, "Acme Corp")
	}
	if signed.Tier != "enterprise" {
		t.Errorf("tier = %q, want %q", signed.Tier, "enterprise")
	}
	if signed.Features.MaxEndpoints != 10000 {
		t.Errorf("max_endpoints = %d, want 10000", signed.Features.MaxEndpoints)
	}
	if signed.Signature == "" {
		t.Error("signature is empty")
	}
	if signed.GracePeriodDays != 30 {
		t.Errorf("grace_period_days = %d, want 30", signed.GracePeriodDays)
	}
}

func TestGenerateInvalidTier(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	gen := hublicense.NewGenerator(key)

	params := hublicense.GenerateParams{
		LicenseID:    "lic-bad",
		CustomerID:   "cust-bad",
		CustomerName: "Bad Corp",
		ContactEmail: "bad@bad.com",
		Tier:         "nonexistent",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	_, err = gen.Generate(params)
	if err == nil {
		t.Fatal("expected error for invalid tier, got nil")
	}
}

func TestGeneratedLicenseIsValidJSON(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	gen := hublicense.NewGenerator(key)

	params := hublicense.GenerateParams{
		LicenseID:    "lic-json",
		CustomerID:   "cust-json",
		CustomerName: "JSON Corp",
		ContactEmail: "json@json.com",
		Tier:         "professional",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	signed, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	data, err := json.Marshal(signed)
	if err != nil {
		t.Fatalf("marshal signed license: %v", err)
	}
	if len(data) == 0 {
		t.Error("marshalled JSON is empty")
	}
}

func TestSignatureVerifiesWithPublicKey(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	gen := hublicense.NewGenerator(key)

	params := hublicense.GenerateParams{
		LicenseID:    "lic-verify",
		CustomerID:   "cust-verify",
		CustomerName: "Verify Corp",
		ContactEmail: "verify@verify.com",
		Tier:         "enterprise",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	signed, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	canonical, err := json.Marshal(signed.License)
	if err != nil {
		t.Fatalf("marshal canonical: %v", err)
	}

	sig, err := hublicense.DecodeSignature(signed.Signature)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	if err := crypto.VerifySignature(&key.PublicKey, canonical, sig); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
}

func TestSaveToFile(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	gen := hublicense.NewGenerator(key)

	params := hublicense.GenerateParams{
		LicenseID:    "lic-save",
		CustomerID:   "cust-save",
		CustomerName: "Save Corp",
		ContactEmail: "save@save.com",
		Tier:         "professional",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	signed, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "license.json")

	if err := hublicense.SaveToFile(signed, path); err != nil {
		t.Fatalf("save to file: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var loaded licdefs.SignedLicense
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal saved license: %v", err)
	}

	if loaded.LicenseID != "lic-save" {
		t.Errorf("loaded license_id = %q, want %q", loaded.LicenseID, "lic-save")
	}
	if loaded.Signature == "" {
		t.Error("loaded signature is empty")
	}
}

func TestMaxEndpointsOverride(t *testing.T) {
	key, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	gen := hublicense.NewGenerator(key)

	params := hublicense.GenerateParams{
		LicenseID:    "lic-override",
		CustomerID:   "cust-override",
		CustomerName: "Override Corp",
		ContactEmail: "override@override.com",
		Tier:         "professional",
		MaxEndpoints: 500,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	signed, err := gen.Generate(params)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if signed.Features.MaxEndpoints != 500 {
		t.Errorf("max_endpoints = %d, want 500", signed.Features.MaxEndpoints)
	}
}
