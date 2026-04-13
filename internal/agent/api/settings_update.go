package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
)

// SettingsUpdateRequest represents editable settings.
type SettingsUpdateRequest struct {
	ScanInterval          *string `json:"scan_interval,omitempty"` // e.g., "1h", "6h", "24h"
	LogLevel              *string `json:"log_level,omitempty"`     // debug, info, warn, error
	AutoDeploy            *bool   `json:"auto_deploy,omitempty"`
	HeartbeatInterval     *string `json:"heartbeat_interval,omitempty"`      // e.g., "10s", "30s", "1m", "2m", "5m"
	BandwidthLimitKbps    *int    `json:"bandwidth_limit_kbps,omitempty"`    // 0=unlimited, 128, 256, 512, 1024, 2048, 5120, 10240
	MaxConcurrentInstalls *int    `json:"max_concurrent_installs,omitempty"` // 1-4
	ProxyURL              *string `json:"proxy_url,omitempty"`               // HTTP proxy URL, empty to clear
	AutoRebootWindow      *string `json:"auto_reboot_window,omitempty"`      // "HH:MM-HH:MM", empty to clear
	LogRetentionDays      *int    `json:"log_retention_days,omitempty"`      // 7, 14, 30, 60, 90
	OfflineMode           *bool   `json:"offline_mode,omitempty"`
}

// SettingsUpdater persists settings changes.
type SettingsUpdater interface {
	UpdateSettings(ctx context.Context, req SettingsUpdateRequest) error
}

var validScanIntervals = map[string]bool{
	"1h": true, "3h": true, "6h": true, "12h": true, "24h": true,
}

var validLogLevels = map[string]bool{
	"debug": true, "info": true, "warn": true, "error": true,
}

var validHeartbeatIntervals = map[string]bool{
	"10s": true, "30s": true, "1m": true, "2m": true, "5m": true,
}

const (
	minBandwidthKbps = 64
	maxBandwidthKbps = 102400 // 100 Mbps
)

var validLogRetentionDays = map[int]bool{
	7: true, 14: true, 30: true, 60: true, 90: true,
}

var rebootWindowPattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$`)

// SettingsUpdateHandler serves PUT /api/v1/settings.
type SettingsUpdateHandler struct {
	updater SettingsUpdater
}

// NewSettingsUpdateHandler creates a SettingsUpdateHandler.
func NewSettingsUpdateHandler(u SettingsUpdater) *SettingsUpdateHandler {
	return &SettingsUpdateHandler{updater: u}
}

// Update handles PUT /api/v1/settings.
func (h *SettingsUpdateHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req SettingsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON request body")
		return
	}

	if req.ScanInterval != nil {
		if !validScanIntervals[*req.ScanInterval] {
			WriteError(w, http.StatusBadRequest, "INVALID_SCAN_INTERVAL", "scan_interval must be one of: 1h, 3h, 6h, 12h, 24h")
			return
		}
	}
	if req.LogLevel != nil {
		if !validLogLevels[*req.LogLevel] {
			WriteError(w, http.StatusBadRequest, "INVALID_LOG_LEVEL", "log_level must be one of: debug, info, warn, error")
			return
		}
	}
	if req.HeartbeatInterval != nil {
		if !validHeartbeatIntervals[*req.HeartbeatInterval] {
			WriteError(w, http.StatusBadRequest, "INVALID_HEARTBEAT_INTERVAL", "heartbeat_interval must be one of: 10s, 30s, 1m, 2m, 5m")
			return
		}
	}
	if req.BandwidthLimitKbps != nil {
		v := *req.BandwidthLimitKbps
		if v != 0 && (v < minBandwidthKbps || v > maxBandwidthKbps) {
			WriteError(w, http.StatusBadRequest, "INVALID_BANDWIDTH_LIMIT",
				fmt.Sprintf("bandwidth_limit_kbps must be 0 (unlimited) or between %d and %d", minBandwidthKbps, maxBandwidthKbps))
			return
		}
	}
	if req.MaxConcurrentInstalls != nil {
		if *req.MaxConcurrentInstalls < 1 || *req.MaxConcurrentInstalls > 4 {
			WriteError(w, http.StatusBadRequest, "INVALID_MAX_CONCURRENT_INSTALLS", "max_concurrent_installs must be between 1 and 4")
			return
		}
	}
	if req.ProxyURL != nil && *req.ProxyURL != "" {
		if _, err := url.ParseRequestURI(*req.ProxyURL); err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_PROXY_URL", "proxy_url must be a valid URL")
			return
		}
	}
	if req.AutoRebootWindow != nil && *req.AutoRebootWindow != "" {
		if !rebootWindowPattern.MatchString(*req.AutoRebootWindow) {
			WriteError(w, http.StatusBadRequest, "INVALID_AUTO_REBOOT_WINDOW", "auto_reboot_window must be in HH:MM-HH:MM format (e.g., 02:00-05:00)")
			return
		}
	}
	if req.LogRetentionDays != nil {
		if !validLogRetentionDays[*req.LogRetentionDays] {
			WriteError(w, http.StatusBadRequest, "INVALID_LOG_RETENTION_DAYS", "log_retention_days must be one of: 7, 14, 30, 60, 90")
			return
		}
	}

	if err := h.updater.UpdateSettings(r.Context(), req); err != nil {
		slog.ErrorContext(r.Context(), "settings: persist failed", "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to update settings")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
