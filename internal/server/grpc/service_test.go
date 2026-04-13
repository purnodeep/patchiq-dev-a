package grpc_test

import (
	"log/slog"
	"testing"

	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
)

func TestNewAgentServiceServer(t *testing.T) {
	svc := servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, slog.Default())
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewAgentServiceServer_NilStore_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil store, got none")
		}
	}()
	servergrpc.NewAgentServiceServer(nil, noopEventBus{}, slog.Default())
}

func TestNewAgentServiceServer_NilEventBus_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil eventBus, got none")
		}
	}()
	servergrpc.NewAgentServiceServer(testStore(t), nil, slog.Default())
}

func TestNewAgentServiceServer_NilLogger_Panics(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil logger, got none")
		}
	}()
	servergrpc.NewAgentServiceServer(testStore(t), noopEventBus{}, nil)
}
