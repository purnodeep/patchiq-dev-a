package grpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func loggingUnaryInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	st, _ := status.FromError(err)
	slog.InfoContext(ctx, "grpc unary",
		"method", info.FullMethod,
		"duration_ms", time.Since(start).Milliseconds(),
		"code", st.Code().String(),
	)
	return resp, err
}

func loggingStreamInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, ss)
	st, _ := status.FromError(err)
	slog.InfoContext(ss.Context(), "grpc stream",
		"method", info.FullMethod,
		"duration_ms", time.Since(start).Milliseconds(),
		"code", st.Code().String(),
	)
	return err
}
