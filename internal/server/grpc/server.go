package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
)

// ServerConfig holds gRPC server configuration.
type ServerConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

func sharedServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.MaxConcurrentStreams(500),
		grpc.StatsHandler(piqotel.GRPCServerHandler()),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             15 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(loggingUnaryInterceptor),
		grpc.ChainStreamInterceptor(loggingStreamInterceptor),
	}
}

// NewGRPCServer creates a gRPC server with keepalive, OTel tracing, and logging interceptors.
func NewGRPCServer(cfg ServerConfig) *grpc.Server {
	return grpc.NewServer(sharedServerOptions()...)
}

// NewGRPCServerWithTLS creates a gRPC server with mTLS (TLS 1.3+, client cert required and verified against CA),
// keepalive, OTel tracing, and logging interceptors.
func NewGRPCServerWithTLS(cfg ServerConfig) (*grpc.Server, error) {
	if cfg.CAFile == "" {
		return nil, fmt.Errorf("create mTLS server: CAFile is required")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server TLS cert: %w", err)
	}

	caPEM, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read client CA file: %w", err)
	}

	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse client CA file: no valid certificates found in %s", cfg.CAFile)
	}

	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAs,
		MinVersion:   tls.VersionTLS13,
	}
	opts := append([]grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}, sharedServerOptions()...)
	return grpc.NewServer(opts...), nil
}
