package otel_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/otel"
)

func TestRequestIDContext(t *testing.T) {
	ctx := context.Background()

	_, ok := otel.RequestIDFromContext(ctx)
	if ok {
		t.Fatal("expected no request ID")
	}

	ctx = otel.WithRequestID(ctx, "req-123")
	id, ok := otel.RequestIDFromContext(ctx)
	if !ok || id != "req-123" {
		t.Fatalf("expected req-123, got %q (ok=%v)", id, ok)
	}
}
