package grpc_test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	servergrpc "github.com/skenzeriq/patchiq/internal/server/grpc"
)

func TestLoggingUnaryInterceptor(t *testing.T) {
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}
	resp, err := servergrpc.ExportedLoggingUnaryInterceptor(context.Background(), nil, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("expected 'ok', got %v", resp)
	}
}
