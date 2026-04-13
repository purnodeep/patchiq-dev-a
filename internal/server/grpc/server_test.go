package grpc_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	patchiqv1 "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestNewGRPCServer_defaults(t *testing.T) {
	srv := servergrpc.NewGRPCServer(servergrpc.ServerConfig{})
	if srv == nil {
		t.Fatal("expected non-nil gRPC server")
	}
}

func TestNewGRPCServerWithTLS_invalidCert(t *testing.T) {
	_, err := servergrpc.NewGRPCServerWithTLS(servergrpc.ServerConfig{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	})
	if err == nil {
		t.Fatal("expected error for non-existent cert files")
	}
}

// generateCA creates a self-signed CA cert + key, writes them to dir, returns the cert pool.
func generateCA(t *testing.T, dir, prefix string) (*x509.CertPool, string, string) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: prefix + " CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caCert, err := x509.ParseCertificate(caDER)
	require.NoError(t, err)

	certPath := filepath.Join(dir, prefix+"-ca.pem")
	keyPath := filepath.Join(dir, prefix+"-ca-key.pem")
	writePEM(t, certPath, "CERTIFICATE", caDER)
	writeKeyPEM(t, keyPath, caKey)

	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	return pool, certPath, keyPath
}

// generateLeafCert creates a cert signed by the given CA, writes to dir.
func generateLeafCert(t *testing.T, dir, prefix string, caCertPath, caKeyPath string, isServer bool) (string, string) {
	t.Helper()

	caKeyPEM, err := os.ReadFile(caKeyPath)
	require.NoError(t, err)
	block, _ := pem.Decode(caKeyPEM)
	caKey, err := x509.ParseECPrivateKey(block.Bytes)
	require.NoError(t, err)

	caCertPEM, err := os.ReadFile(caCertPath)
	require.NoError(t, err)
	caBlock, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	require.NoError(t, err)

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: prefix},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	if isServer {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.DNSNames = []string{"localhost"}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	} else {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &leafKey.PublicKey, caKey)
	require.NoError(t, err)

	certPath := filepath.Join(dir, prefix+".pem")
	keyPath := filepath.Join(dir, prefix+"-key.pem")
	writePEM(t, certPath, "CERTIFICATE", leafDER)
	writeKeyPEM(t, keyPath, leafKey)
	return certPath, keyPath
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}))
}

func writeKeyPEM(t *testing.T, path string, key *ecdsa.PrivateKey) {
	t.Helper()
	der, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	writePEM(t, path, "EC PRIVATE KEY", der)
}

func TestNewGRPCServerWithTLS_RejectsUntrustedClient(t *testing.T) {
	dir := t.TempDir()

	// Server CA + server cert
	_, serverCACertPath, serverCAKeyPath := generateCA(t, dir, "server")
	serverCertPath, serverKeyPath := generateLeafCert(t, dir, "server", serverCACertPath, serverCAKeyPath, true)

	// Trusted client CA + client cert
	_, trustedCACertPath, trustedCAKeyPath := generateCA(t, dir, "trusted-client")
	trustedClientCertPath, trustedClientKeyPath := generateLeafCert(t, dir, "trusted-client", trustedCACertPath, trustedCAKeyPath, false)

	// Untrusted client CA + client cert (different CA)
	_, _, untrustedCAKeyPath := generateCA(t, dir, "untrusted-client")
	untrustedCACertPath := filepath.Join(dir, "untrusted-client-ca.pem")
	untrustedClientCertPath, untrustedClientKeyPath := generateLeafCert(t, dir, "untrusted-client", untrustedCACertPath, untrustedCAKeyPath, false)

	// Create server with trusted client CA
	srv, err := servergrpc.NewGRPCServerWithTLS(servergrpc.ServerConfig{
		CertFile: serverCertPath,
		KeyFile:  serverKeyPath,
		CAFile:   trustedCACertPath,
	})
	require.NoError(t, err)

	// Register a dummy service so we can dial
	patchiqv1.RegisterAgentServiceServer(srv, &patchiqv1.UnimplementedAgentServiceServer{})

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go srv.Serve(lis) //nolint:errcheck
	defer srv.Stop()

	// Load server CA for client to trust the server
	serverCAPEM, err := os.ReadFile(serverCACertPath)
	require.NoError(t, err)
	serverCAPool := x509.NewCertPool()
	serverCAPool.AppendCertsFromPEM(serverCAPEM)

	// Test 1: Trusted client cert should connect
	trustedCert, err := tls.LoadX509KeyPair(trustedClientCertPath, trustedClientKeyPath)
	require.NoError(t, err)
	trustedCreds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{trustedCert},
		RootCAs:      serverCAPool,
	})
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(trustedCreds))
	require.NoError(t, err)
	defer conn.Close()

	// Make an actual RPC call to verify connection works
	client := patchiqv1.NewAgentServiceClient(conn)
	_, err = client.Enroll(t.Context(), &patchiqv1.EnrollRequest{})
	// Enroll will fail with Unimplemented, but that's fine — it means TLS handshake succeeded
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unimplemented")

	// Test 2: Untrusted client cert should fail with a TLS/connection error, not Unimplemented
	untrustedCert, err := tls.LoadX509KeyPair(untrustedClientCertPath, untrustedClientKeyPath)
	require.NoError(t, err)
	untrustedCreds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{untrustedCert},
		RootCAs:      serverCAPool,
	})
	conn2, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(untrustedCreds))
	if err == nil {
		client2 := patchiqv1.NewAgentServiceClient(conn2)
		_, err = client2.Enroll(t.Context(), &patchiqv1.EnrollRequest{})
		conn2.Close()
		require.Error(t, err, "expected RPC to fail with untrusted client cert")
		require.NotContains(t, err.Error(), "Unimplemented",
			"untrusted client should be rejected at TLS level, not reach the RPC handler")
	}
}

func TestNewGRPCServerWithTLS_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	_, serverCACertPath, serverCAKeyPath := generateCA(t, dir, "server")
	serverCertPath, serverKeyPath := generateLeafCert(t, dir, "server", serverCACertPath, serverCAKeyPath, true)

	// Write garbage content to a CA file
	badCAPath := filepath.Join(dir, "bad-ca.pem")
	require.NoError(t, os.WriteFile(badCAPath, []byte("not a certificate"), 0o600))

	_, err := servergrpc.NewGRPCServerWithTLS(servergrpc.ServerConfig{
		CertFile: serverCertPath,
		KeyFile:  serverKeyPath,
		CAFile:   badCAPath,
	})
	require.Error(t, err, "expected error for invalid CA PEM content")
	require.Contains(t, err.Error(), "no valid certificates found")
}

func TestNewGRPCServerWithTLS_MissingCAFile(t *testing.T) {
	dir := t.TempDir()
	_, serverCACertPath, serverCAKeyPath := generateCA(t, dir, "server")
	serverCertPath, serverKeyPath := generateLeafCert(t, dir, "server", serverCACertPath, serverCAKeyPath, true)

	_, err := servergrpc.NewGRPCServerWithTLS(servergrpc.ServerConfig{
		CertFile: serverCertPath,
		KeyFile:  serverKeyPath,
		CAFile:   "",
	})
	require.Error(t, err, "expected error when CAFile is empty")
}
