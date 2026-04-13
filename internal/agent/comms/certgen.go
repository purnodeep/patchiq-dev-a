package comms

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LoadOrGenerateCert loads existing TLS cert from dataDir/certs/ or generates
// a self-signed ECDSA P-256 cert (1 year validity, CN=hostname).
// Files: cert.pem, key.pem with 0600 permissions.
func LoadOrGenerateCert(dataDir string, logger *slog.Logger) (tls.Certificate, error) {
	certsDir := filepath.Join(dataDir, "certs")
	certPath := filepath.Join(certsDir, "cert.pem")
	keyPath := filepath.Join(certsDir, "key.pem")

	// Try loading existing cert. Distinguish "not found" from "broken".
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err == nil {
		return cert, nil
	}
	if _, statErr := os.Stat(certPath); statErr == nil {
		// Files exist but are unusable (corrupted, mismatched key, etc.).
		logger.Error("existing cert files are corrupted or mismatched, regenerating new identity",
			"error", err, "cert_path", certPath, "key_path", keyPath)
	}

	if err := os.MkdirAll(certsDir, 0o700); err != nil {
		return tls.Certificate{}, fmt.Errorf("create certs directory: %w", err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate ecdsa key: %w", err)
	}

	serialMax := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial number: %w", err)
	}

	hostname, err := certHostname()
	if err != nil {
		logger.Warn("failed to get hostname for cert CN, using fallback", "error", err)
		hostname = "patchiq-agent"
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		IsCA:         false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("marshal ec private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return tls.Certificate{}, fmt.Errorf("write cert.pem: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return tls.Certificate{}, fmt.Errorf("write key.pem: %w", err)
	}

	cert, err = tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("load newly generated cert: %w", err)
	}
	return cert, nil
}

// certHostname returns a reliable hostname for the TLS certificate CN.
// On macOS, os.Hostname() can return an IP address when DNS reverse lookup
// fails, so scutil is tried first.
func certHostname() (string, error) {
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("scutil", "--get", "ComputerName").Output(); err == nil {
			if name := strings.TrimSpace(string(out)); name != "" {
				return name, nil
			}
		}
		if out, err := exec.Command("scutil", "--get", "LocalHostName").Output(); err == nil {
			if name := strings.TrimSpace(string(out)); name != "" {
				return name, nil
			}
		}
	}
	return os.Hostname()
}

// SaveServerCert overwrites cert.pem with server-issued certificate (M2 forward-compat).
func SaveServerCert(dataDir string, certPEM []byte) error {
	certPath := filepath.Join(dataDir, "certs", "cert.pem")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return fmt.Errorf("write server cert: %w", err)
	}
	return nil
}
