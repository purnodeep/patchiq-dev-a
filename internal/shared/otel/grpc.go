package otel

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/stats"
)

// GRPCServerHandler returns a gRPC stats.Handler for server-side OTel tracing.
func GRPCServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler()
}

// GRPCClientHandler returns a gRPC stats.Handler for client-side OTel tracing.
func GRPCClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler()
}
