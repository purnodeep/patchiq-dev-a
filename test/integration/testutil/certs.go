//go:build integration

package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TLSBundle holds paths to all generated cert/key files and the parsed CA
// certificate and key for signing additional certificates if needed.
type TLSBundle struct {
	CACertPath     string
	ServerCertPath string
	ServerKeyPath  string
	AgentCertPath  string
	AgentKeyPath   string
	CACert         *x509.Certificate
	CAKey          *ecdsa.PrivateKey
}

// ServerTLSConfig returns a *tls.Config suitable for a gRPC server that
// requires and verifies mTLS client certificates signed by the ephemeral CA.
func (b *TLSBundle) ServerTLSConfig() (*tls.Config, error) {
	serverCert, err := tls.LoadX509KeyPair(b.ServerCertPath, b.ServerKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load server key pair: %w", err)
	}

	caPool := x509.NewCertPool()
	caPEM, err := os.ReadFile(b.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("append CA cert to pool: invalid PEM")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// AgentTLSConfig returns a *tls.Config suitable for a gRPC agent client that
// presents its client certificate and trusts the ephemeral CA.
func (b *TLSBundle) AgentTLSConfig() (*tls.Config, error) {
	agentCert, err := tls.LoadX509KeyPair(b.AgentCertPath, b.AgentKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load agent key pair: %w", err)
	}

	caPool := x509.NewCertPool()
	caPEM, err := os.ReadFile(b.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("append CA cert to pool: invalid PEM")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{agentCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// GenerateTLSBundle creates an ephemeral CA, server certificate, and agent
// client certificate. All certificates are valid for 1 hour. Files are written
// to a temporary directory that is automatically cleaned up when the test ends.
func GenerateTLSBundle(t *testing.T) *TLSBundle {
	t.Helper()

	dir := t.TempDir()

	// --- CA ---
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}

	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber:          randomSerial(t),
		Subject:               pkix.Name{CommonName: "PatchIQ Test CA"},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}

	caCertPath := filepath.Join(dir, "ca-cert.pem")
	writePEM(t, caCertPath, "CERTIFICATE", caCertDER)

	// --- Server cert ---
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: randomSerial(t),
		Subject:      pkix.Name{CommonName: "PatchIQ Test Server"},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("0.0.0.0"),
		},
		DNSNames: []string{
			"localhost",
			"host.docker.internal",
		},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create server certificate: %v", err)
	}

	serverCertPath := filepath.Join(dir, "server-cert.pem")
	serverKeyPath := filepath.Join(dir, "server-key.pem")
	writePEM(t, serverCertPath, "CERTIFICATE", serverCertDER)
	writeECKey(t, serverKeyPath, serverKey)

	// --- Agent client cert ---
	agentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate agent key: %v", err)
	}

	agentTemplate := &x509.Certificate{
		SerialNumber: randomSerial(t),
		Subject:      pkix.Name{CommonName: "PatchIQ Test Agent"},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(1 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	agentCertDER, err := x509.CreateCertificate(rand.Reader, agentTemplate, caCert, &agentKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create agent certificate: %v", err)
	}

	agentCertPath := filepath.Join(dir, "agent-cert.pem")
	agentKeyPath := filepath.Join(dir, "agent-key.pem")
	writePEM(t, agentCertPath, "CERTIFICATE", agentCertDER)
	writeECKey(t, agentKeyPath, agentKey)

	return &TLSBundle{
		CACertPath:     caCertPath,
		ServerCertPath: serverCertPath,
		ServerKeyPath:  serverKeyPath,
		AgentCertPath:  agentCertPath,
		AgentKeyPath:   agentKeyPath,
		CACert:         caCert,
		CAKey:          caKey,
	}
}

// randomSerial generates a cryptographically random 128-bit serial number.
func randomSerial(t *testing.T) *big.Int {
	t.Helper()
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		t.Fatalf("generate random serial: %v", err)
	}
	return serial
}

// writePEM writes a single PEM block to the given path with 0600 permissions.
func writePEM(t *testing.T, path, blockType string, derBytes []byte) {
	t.Helper()
	data := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: derBytes})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write PEM file %s: %v", path, err)
	}
}

// writeECKey marshals an ECDSA private key and writes it as PEM.
func writeECKey(t *testing.T, path string, key *ecdsa.PrivateKey) {
	t.Helper()
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal EC private key: %v", err)
	}
	writePEM(t, path, "EC PRIVATE KEY", der)
}
