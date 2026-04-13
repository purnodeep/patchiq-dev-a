package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// LogWriter writes log entries to the agent log store.
type LogWriter interface {
	WriteLog(ctx context.Context, level, message, source string) error
}

// ScanTrigger starts an on-demand inventory scan.
type ScanTrigger interface {
	CollectNow(ctx context.Context, moduleName string) error
}

// ScanHandler serves POST /api/v1/scan.
type ScanHandler struct {
	logWriter LogWriter
	trigger   ScanTrigger
}

// NewScanHandler creates a ScanHandler.
func NewScanHandler(lw LogWriter, trigger ScanTrigger) *ScanHandler {
	return &ScanHandler{logWriter: lw, trigger: trigger}
}

// Trigger handles POST /api/v1/scan — runs an immediate inventory scan.
// Collection runs synchronously so the response reflects real completion;
// the agent UI can then refetch hardware/software/status endpoints and see
// fresh data. On Windows this can take 2–3 minutes due to WUA enumeration.
func (h *ScanHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now().UTC()

	slog.InfoContext(ctx, "scan triggered via API", "time", start.Format(time.RFC3339))
	if h.logWriter != nil {
		if err := h.logWriter.WriteLog(ctx, "info", "Inventory scan triggered via API", "api"); err != nil {
			slog.ErrorContext(ctx, "write scan trigger log", "error", err)
		}
	}

	if h.trigger == nil {
		WriteError(w, http.StatusServiceUnavailable, "SCAN_UNAVAILABLE", "scan trigger not configured")
		return
	}

	if err := h.trigger.CollectNow(ctx, "inventory"); err != nil {
		slog.ErrorContext(ctx, "scan collection failed", "error", err)
		if h.logWriter != nil {
			_ = h.logWriter.WriteLog(ctx, "error", "Inventory scan failed: "+err.Error(), "api")
		}
		WriteError(w, http.StatusInternalServerError, "SCAN_FAILED", "scan failed: "+err.Error())
		return
	}

	duration := time.Since(start)
	if h.logWriter != nil {
		_ = h.logWriter.WriteLog(ctx, "info", "Inventory scan completed via API in "+duration.String(), "api")
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"status":   "scan_completed",
		"duration": duration.String(),
	})
}
