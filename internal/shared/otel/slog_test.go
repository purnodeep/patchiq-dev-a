package otel_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestHandlerInjectsTraceID(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test-op")
	defer span.End()

	logger.InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if _, ok := entry["trace_id"]; !ok {
		t.Fatal("expected trace_id in log output")
	}
	if _, ok := entry["span_id"]; !ok {
		t.Fatal("expected span_id in log output")
	}
}

func TestHandlerInjectsTenantID(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	ctx := tenant.WithTenantID(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	logger.InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["tenant_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("expected tenant_id, got %v", entry["tenant_id"])
	}
}

func TestHandlerInjectsRequestID(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	ctx := piqotel.WithRequestID(context.Background(), "req-abc-123")
	logger.InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["request_id"] != "req-abc-123" {
		t.Fatalf("expected request_id, got %v", entry["request_id"])
	}
}

func TestHandlerInjectsUserID(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	ctx := user.WithUserID(context.Background(), "user-42")
	logger.InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["user_id"] != "user-42" {
		t.Fatalf("expected user_id, got %v", entry["user_id"])
	}
}

func TestHandlerOmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	for _, key := range []string{"trace_id", "span_id", "tenant_id", "request_id", "user_id"} {
		if _, ok := entry[key]; ok {
			t.Fatalf("expected %s to be absent from log output", key)
		}
	}
}

func TestHandlerWithAttrsPreservesInjection(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	derived := slog.New(h.WithAttrs([]slog.Attr{slog.String("component", "api")}))

	ctx := tenant.WithTenantID(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	derived.InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["tenant_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("expected tenant_id after WithAttrs")
	}
	if entry["component"] != "api" {
		t.Fatal("expected component attr after WithAttrs")
	}
}

func TestHandlerWithGroupPreservesInjection(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	grouped := slog.New(h.WithGroup("request"))

	ctx := tenant.WithTenantID(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	grouped.InfoContext(ctx, "hello", "method", "GET")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	// tenant_id should be at top level (injected before group)
	if entry["tenant_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatal("expected tenant_id at top level after WithGroup")
	}

	// Verify record attrs are nested under the group
	reqGroup, ok := entry["request"].(map[string]any)
	if !ok {
		t.Fatal("expected 'request' group in log output")
	}
	if reqGroup["method"] != "GET" {
		t.Fatalf("expected method=GET inside request group, got %v", reqGroup["method"])
	}
}

func TestHandlerInjectsAllFieldsSimultaneously(t *testing.T) {
	var buf bytes.Buffer
	h := piqotel.NewHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(h)

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test-op")
	defer span.End()

	ctx = tenant.WithTenantID(ctx, "550e8400-e29b-41d4-a716-446655440000")
	ctx = piqotel.WithRequestID(ctx, "req-xyz")
	ctx = user.WithUserID(ctx, "user-99")

	logger.InfoContext(ctx, "all fields")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}

	expected := []string{"trace_id", "span_id", "tenant_id", "request_id", "user_id"}
	for _, key := range expected {
		if _, ok := entry[key]; !ok {
			t.Errorf("expected %s in log output", key)
		}
	}
	if entry["tenant_id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("wrong tenant_id: %v", entry["tenant_id"])
	}
	if entry["request_id"] != "req-xyz" {
		t.Errorf("wrong request_id: %v", entry["request_id"])
	}
	if entry["user_id"] != "user-99" {
		t.Errorf("wrong user_id: %v", entry["user_id"])
	}
}
