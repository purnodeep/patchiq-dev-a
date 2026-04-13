package otel_test

import (
	"context"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/otel"
	otelapi "go.opentelemetry.io/otel"
)

func TestInitNoop(t *testing.T) {
	ctx := context.Background()
	shutdown, err := otel.Init(ctx, otel.Config{
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("Init with empty endpoint should not error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("shutdown func should not be nil")
	}
	if err := shutdown(ctx); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := shutdown(ctx); err != nil {
		t.Fatalf("second shutdown should be idempotent: %v", err)
	}
}

func TestInitWithEndpoint(t *testing.T) {
	ctx := context.Background()
	shutdown, err := otel.Init(ctx, otel.Config{
		ServiceName:    "test-service",
		ServiceVersion: "0.1.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Insecure:       true,
	})
	if err != nil {
		t.Fatalf("Init should not error: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
	}()

	tracer := otelapi.GetTracerProvider().Tracer("test")
	_, span := tracer.Start(ctx, "test-span")
	sc := span.SpanContext()
	if !sc.TraceID().IsValid() {
		t.Fatal("expected valid trace ID from real tracer provider")
	}
	span.End()
}
