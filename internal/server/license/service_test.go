package license

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/crypto"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	licdefs "github.com/skenzeriq/patchiq/internal/shared/license"
)

type stubEventBus struct {
	mu      sync.Mutex
	emitted []string
}

func (s *stubEventBus) Emit(_ context.Context, event domain.DomainEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitted = append(s.emitted, event.Type)
	return nil
}
func (s *stubEventBus) Subscribe(_ string, _ domain.EventHandler) error { return nil }
func (s *stubEventBus) Close() error                                    { return nil }

func writeTestLicense(t *testing.T, dir string, lic licdefs.License, priv *rsa.PrivateKey) string {
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
	data, err := json.MarshalIndent(signed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.piq-license")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestServiceLoadFromFile(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	entFeatures, _ := licdefs.TierTemplate(licdefs.TierEnterprise)
	lic := licdefs.License{
		LicenseID: "ent-001",
		Customer:  licdefs.Customer{ID: "c1", Name: "Acme", ContactEmail: "a@b.com"},
		Tier:      licdefs.TierEnterprise,
		Features:  entFeatures,
		IssuedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}

	dir := t.TempDir()
	path := writeTestLicense(t, dir, lic, priv)

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)

	if err := svc.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if !svc.HasFeature("sso_saml") {
		t.Error("expected HasFeature(sso_saml) = true for enterprise license")
	}
}

func TestServiceHasFeatureNoLicense(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)

	if svc.HasFeature("sso_saml") {
		t.Error("expected HasFeature(sso_saml) = false when no license loaded")
	}
}

func TestServiceStatus(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	proFeatures, _ := licdefs.TierTemplate(licdefs.TierProfessional)
	lic := licdefs.License{
		LicenseID:       "pro-001",
		Customer:        licdefs.Customer{ID: "c1", Name: "ProCo", ContactEmail: "p@b.com"},
		Tier:            licdefs.TierProfessional,
		Features:        proFeatures,
		IssuedAt:        time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:       time.Now().Add(60 * 24 * time.Hour),
		GracePeriodDays: 30,
	}

	dir := t.TempDir()
	path := writeTestLicense(t, dir, lic, priv)

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)

	if err := svc.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	status := svc.Status()
	if status.Tier != licdefs.TierProfessional {
		t.Errorf("tier = %q, want %q", status.Tier, licdefs.TierProfessional)
	}
	if status.DaysRemaining <= 0 {
		t.Errorf("days_remaining = %d, want > 0", status.DaysRemaining)
	}
	if status.CustomerName != "ProCo" {
		t.Errorf("customer_name = %q, want %q", status.CustomerName, "ProCo")
	}
	if status.EndpointUsage.Limit != 1000 {
		t.Errorf("endpoint_usage.limit = %d, want 1000", status.EndpointUsage.Limit)
	}
}

func TestServiceCheckEndpointLimit(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	commFeatures, _ := licdefs.TierTemplate(licdefs.TierCommunity)
	lic := licdefs.License{
		LicenseID: "comm-001",
		Customer:  licdefs.Customer{ID: "c1", Name: "SmallCo", ContactEmail: "s@b.com"},
		Tier:      licdefs.TierCommunity,
		Features:  commFeatures,
		IssuedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}

	dir := t.TempDir()
	path := writeTestLicense(t, dir, lic, priv)

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)

	if err := svc.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if err := svc.CheckEndpointLimit(20); err != nil {
		t.Errorf("expected 20 endpoints OK, got: %v", err)
	}
	if err := svc.CheckEndpointLimit(30); err == nil {
		t.Error("expected error for 30 endpoints with 25 limit, got nil")
	}
}

func TestServiceCheckEndpointLimitUnlimited(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	mspFeatures, _ := licdefs.TierTemplate(licdefs.TierMSP)
	lic := licdefs.License{
		LicenseID: "msp-001",
		Customer:  licdefs.Customer{ID: "c1", Name: "MSPCo", ContactEmail: "m@b.com"},
		Tier:      licdefs.TierMSP,
		Features:  mspFeatures,
		IssuedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}

	dir := t.TempDir()
	path := writeTestLicense(t, dir, lic, priv)

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)

	if err := svc.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if err := svc.CheckEndpointLimit(999999); err != nil {
		t.Errorf("expected unlimited endpoints OK, got: %v", err)
	}
}

func TestServiceCurrentTier(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, nil)

	// No license loaded => community
	if tier := svc.CurrentTier(); tier != licdefs.TierCommunity {
		t.Errorf("no license: tier = %q, want %q", tier, licdefs.TierCommunity)
	}

	// Load enterprise license
	entFeatures, _ := licdefs.TierTemplate(licdefs.TierEnterprise)
	lic := licdefs.License{
		LicenseID: "ent-002",
		Customer:  licdefs.Customer{ID: "c1", Name: "BigCo", ContactEmail: "b@b.com"},
		Tier:      licdefs.TierEnterprise,
		Features:  entFeatures,
		IssuedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}

	dir := t.TempDir()
	path := writeTestLicense(t, dir, lic, priv)

	if err := svc.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if tier := svc.CurrentTier(); tier != licdefs.TierEnterprise {
		t.Errorf("loaded enterprise: tier = %q, want %q", tier, licdefs.TierEnterprise)
	}
}

func TestServiceEmitsLoadedEvent(t *testing.T) {
	priv, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	bus := &stubEventBus{}
	entFeatures, _ := licdefs.TierTemplate(licdefs.TierEnterprise)
	lic := licdefs.License{
		LicenseID: "ent-003",
		Customer:  licdefs.Customer{ID: "c1", Name: "EventCo", ContactEmail: "e@b.com"},
		Tier:      licdefs.TierEnterprise,
		Features:  entFeatures,
		IssuedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
	}

	dir := t.TempDir()
	path := writeTestLicense(t, dir, lic, priv)

	v := NewValidator(&priv.PublicKey)
	svc := NewService(v, bus)

	if err := svc.LoadFromFile(path); err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	if !slices.Contains(bus.emitted, "license.loaded") {
		t.Errorf("expected license.loaded event, got: %v", bus.emitted)
	}
}
