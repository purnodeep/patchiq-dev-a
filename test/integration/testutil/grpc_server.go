//go:build integration

package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// TestGRPCServer holds a running gRPC server and its metadata for integration tests.
type TestGRPCServer struct {
	Server   *grpc.Server
	Addr     string
	EventBus *CapturingEventBus
}

// StartGRPCServer starts a gRPC server with mTLS on a random port.
// The server is automatically stopped via t.Cleanup.
func StartGRPCServer(t *testing.T, st *store.Store, tlsBundle *TLSBundle) *TestGRPCServer {
	t.Helper()
	return startGRPCServer(t, st, tlsBundle, 0)
}

// StartGRPCServerOnPort starts a gRPC server with mTLS on the specified port.
// Use this when the port must be known before the server starts (e.g., offline tests
// that pre-register the port in agent config).
func StartGRPCServerOnPort(t *testing.T, st *store.Store, tlsBundle *TLSBundle, port int) *TestGRPCServer {
	t.Helper()
	return startGRPCServer(t, st, tlsBundle, port)
}

// StartInsecureGRPCServer starts a gRPC server without TLS on a random port.
// Use this when the agent does not yet support mTLS (PIQ-116).
func StartInsecureGRPCServer(t *testing.T, st *store.Store) *TestGRPCServer {
	t.Helper()
	return startGRPCServer(t, st, nil, 0)
}

// StartInsecureGRPCServerOnPort starts a gRPC server without TLS on the specified port.
func StartInsecureGRPCServerOnPort(t *testing.T, st *store.Store, port int) *TestGRPCServer {
	t.Helper()
	return startGRPCServer(t, st, nil, port)
}

func startGRPCServer(t *testing.T, st *store.Store, tlsBundle *TLSBundle, port int) *TestGRPCServer {
	t.Helper()

	var opts []grpc.ServerOption
	if tlsBundle != nil {
		tlsConfig, err := tlsBundle.ServerTLSConfig()
		if err != nil {
			t.Fatalf("grpc_server: server TLS config: %v", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	grpcServer := grpc.NewServer(opts...)

	eventBus := &CapturingEventBus{}
	logger := slog.New(slog.NewTextHandler(&testLogWriter{t: t}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	agentSvc := servergrpc.NewAgentServiceServer(st, eventBus, logger)
	pb.RegisterAgentServiceServer(grpcServer, agentSvc)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("grpc_server: listen on %s: %v", addr, err)
	}

	go func() {
		// Serve blocks until GracefulStop is called in t.Cleanup, which
		// closes the listener and causes Serve to return.
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("grpc_server: serve exited: %v", err)
		}
	}()

	t.Cleanup(func() {
		grpcServer.GracefulStop()
	})

	return &TestGRPCServer{
		Server:   grpcServer,
		Addr:     lis.Addr().String(),
		EventBus: eventBus,
	}
}

// testLogWriter adapts testing.T to io.Writer so slog output appears in test logs.
type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (int, error) {
	w.t.Helper()
	w.t.Log(string(p))
	return len(p), nil
}

// ---------------------------------------------------------------------------
// CapturingEventBus
// ---------------------------------------------------------------------------

// CapturingEventBus records all emitted domain events for test assertions.
// It implements [domain.EventBus].
type CapturingEventBus struct {
	mu     sync.Mutex
	events []domain.DomainEvent
}

var _ domain.EventBus = (*CapturingEventBus)(nil)

// Emit records the event.
func (c *CapturingEventBus) Emit(_ context.Context, evt domain.DomainEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evt)
	return nil
}

// Subscribe is a no-op for the capturing bus.
func (c *CapturingEventBus) Subscribe(string, domain.EventHandler) error { return nil }

// Close is a no-op for the capturing bus.
func (c *CapturingEventBus) Close() error { return nil }

// Events returns a copy of all recorded events.
func (c *CapturingEventBus) Events() []domain.DomainEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	copied := make([]domain.DomainEvent, len(c.events))
	copy(copied, c.events)
	return copied
}

// HasEventType reports whether any recorded event has the given type.
func (c *CapturingEventBus) HasEventType(eventType string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, evt := range c.events {
		if evt.Type == eventType {
			return true
		}
	}
	return false
}

// EventTypes returns a slice of all recorded event types, in emission order.
func (c *CapturingEventBus) EventTypes() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	types := make([]string, len(c.events))
	for i, evt := range c.events {
		types[i] = evt.Type
	}
	return types
}
