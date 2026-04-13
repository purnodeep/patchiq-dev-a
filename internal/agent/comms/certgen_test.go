package comms_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestLoadOrGenerateCert_GeneratesOnFirstRun(t *testing.T) {
	dir := t.TempDir()

	cert, err := comms.LoadOrGenerateCert(dir, slog.Default())
	if err != nil {
		t.Fatalf("LoadOrGenerateCert: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Fatal("expected at least one certificate in chain")
	}

	certPath := filepath.Join(dir, "certs", "cert.pem")
	keyPath := filepath.Join(dir, "certs", "key.pem")

	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("cert.pem not found: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("key.pem not found: %v", err)
	}

	certInfo, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("stat cert.pem: %v", err)
	}
	if perm := certInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("cert.pem permissions = %o, want 0600", perm)
	}

	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key.pem: %v", err)
	}
	if perm := keyInfo.Mode().Perm(); perm != 0o600 {
		t.Errorf("key.pem permissions = %o, want 0600", perm)
	}
}

func TestLoadOrGenerateCert_LoadsExisting(t *testing.T) {
	dir := t.TempDir()

	first, err := comms.LoadOrGenerateCert(dir, slog.Default())
	if err != nil {
		t.Fatalf("first call: %v", err)
	}

	second, err := comms.LoadOrGenerateCert(dir, slog.Default())
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if !bytes.Equal(first.Certificate[0], second.Certificate[0]) {
		t.Error("second call returned different certificate; expected same")
	}
}

func TestLoadOrGenerateCert_ValidCertProperties(t *testing.T) {
	dir := t.TempDir()

	tlsCert, err := comms.LoadOrGenerateCert(dir, slog.Default())
	if err != nil {
		t.Fatalf("LoadOrGenerateCert: %v", err)
	}

	parsed, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	if parsed.IsCA {
		t.Error("certificate should not be CA")
	}

	if parsed.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("certificate missing DigitalSignature key usage")
	}

	if _, ok := parsed.PublicKey.(*ecdsa.PublicKey); !ok {
		t.Errorf("expected ECDSA public key, got %T", parsed.PublicKey)
	}
}

func TestSaveServerCert_OverwritesCertFile(t *testing.T) {
	dir := t.TempDir()

	// Generate initial cert.
	_, err := comms.LoadOrGenerateCert(dir, slog.Default())
	if err != nil {
		t.Fatalf("LoadOrGenerateCert: %v", err)
	}

	certPath := filepath.Join(dir, "certs", "cert.pem")
	original, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read original cert: %v", err)
	}

	// Create a different PEM to overwrite with.
	newPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("fake-cert-data")})

	if err := comms.SaveServerCert(dir, newPEM); err != nil {
		t.Fatalf("SaveServerCert: %v", err)
	}

	updated, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read updated cert: %v", err)
	}

	if bytes.Equal(original, updated) {
		t.Error("cert.pem was not overwritten")
	}
}
