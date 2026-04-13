package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type fakeScanLogWriter struct {
	mu      sync.Mutex
	entries []string
}

func (f *fakeScanLogWriter) WriteLog(_ context.Context, level, message, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, level+":"+message)
	return nil
}

type fakeScanTrigger struct {
	called     bool
	moduleName string
	err        error
}

func (f *fakeScanTrigger) CollectNow(_ context.Context, name string) error {
	f.called = true
	f.moduleName = name
	return f.err
}

func TestScanHandler_Trigger_Success(t *testing.T) {
	t.Parallel()

	lw := &fakeScanLogWriter{}
	trigger := &fakeScanTrigger{}
	h := NewScanHandler(lw, trigger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan", nil)
	rec := httptest.NewRecorder()

	h.Trigger(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !trigger.called {
		t.Error("expected trigger.CollectNow to be called")
	}
	if trigger.moduleName != "inventory" {
		t.Errorf("expected module name %q, got %q", "inventory", trigger.moduleName)
	}
	if !strings.Contains(rec.Body.String(), "scan_completed") {
		t.Errorf("expected body to contain scan_completed, got %s", rec.Body.String())
	}
}

func TestScanHandler_Trigger_Failure(t *testing.T) {
	t.Parallel()

	lw := &fakeScanLogWriter{}
	trigger := &fakeScanTrigger{err: errors.New("boom")}
	h := NewScanHandler(lw, trigger)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan", nil)
	rec := httptest.NewRecorder()

	h.Trigger(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d: %s", rec.Code, rec.Body.String())
	}
	if !trigger.called {
		t.Error("expected trigger.CollectNow to be called")
	}
	if !strings.Contains(rec.Body.String(), "SCAN_FAILED") {
		t.Errorf("expected body to contain SCAN_FAILED, got %s", rec.Body.String())
	}
}

func TestScanHandler_Trigger_Unavailable(t *testing.T) {
	t.Parallel()

	lw := &fakeScanLogWriter{}
	h := NewScanHandler(lw, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/scan", nil)
	rec := httptest.NewRecorder()

	h.Trigger(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "SCAN_UNAVAILABLE") {
		t.Errorf("expected body to contain SCAN_UNAVAILABLE, got %s", rec.Body.String())
	}
}
