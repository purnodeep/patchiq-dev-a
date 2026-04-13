package otel_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	otelapi "go.opentelemetry.io/otel"
)

func TestHTTPMiddlewareCreatesSpan(t *testing.T) {
	// Set up a real tracer provider so otelhttp creates real spans
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	prev := otelapi.GetTracerProvider()
	otelapi.SetTracerProvider(tp)
	t.Cleanup(func() { otelapi.SetTracerProvider(prev) })

	var spanValid bool
	handler := piqotel.HTTPMiddleware("test-service")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		spanValid = span.SpanContext().IsValid()
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !spanValid {
		t.Error("expected valid span context inside handler")
	}
}
