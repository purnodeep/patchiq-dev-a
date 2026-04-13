package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/skenzeriq/patchiq/internal/agent/api"
)

var _ api.SettingsUpdater = (*SettingsStore)(nil)
var _ api.SettingsReader = (*SettingsStore)(nil)

// SettingsStore reads/writes agent settings from the agent_state table.
type SettingsStore struct {
	db *sql.DB
}

// NewSettingsStore creates a SettingsStore.
func NewSettingsStore(db *sql.DB) *SettingsStore {
	return &SettingsStore{db: db}
}

var _ api.AgentIDReader = (*SettingsStore)(nil)

// Settings key constants.
const (
	keyScanInterval          = "setting.scan_interval"
	keyLogLevel              = "setting.log_level"
	keyAutoDeploy            = "setting.auto_deploy"
	keyHeartbeatInterval     = "setting.heartbeat_interval"
	keyBandwidthLimitKbps    = "setting.bandwidth_limit_kbps"
	keyMaxConcurrentInstalls = "setting.max_concurrent_installs"
	keyProxyURL              = "setting.proxy_url"
	keyAutoRebootWindow      = "setting.auto_reboot_window"
	keyLogRetentionDays      = "setting.log_retention_days"
	keyOfflineMode           = "setting.offline_mode"
)

// UpdateSettings persists settings changes to agent_state.
func (s *SettingsStore) UpdateSettings(ctx context.Context, req api.SettingsUpdateRequest) error {
	if req.ScanInterval != nil {
		if err := s.set(ctx, keyScanInterval, *req.ScanInterval); err != nil {
			return fmt.Errorf("update setting scan_interval: %w", err)
		}
	}
	if req.LogLevel != nil {
		if err := s.set(ctx, keyLogLevel, *req.LogLevel); err != nil {
			return fmt.Errorf("update setting log_level: %w", err)
		}
	}
	if req.AutoDeploy != nil {
		if err := s.set(ctx, keyAutoDeploy, boolToStr(*req.AutoDeploy)); err != nil {
			return fmt.Errorf("update setting auto_deploy: %w", err)
		}
	}
	if req.HeartbeatInterval != nil {
		if err := s.set(ctx, keyHeartbeatInterval, *req.HeartbeatInterval); err != nil {
			return fmt.Errorf("update setting heartbeat_interval: %w", err)
		}
	}
	if req.BandwidthLimitKbps != nil {
		if err := s.set(ctx, keyBandwidthLimitKbps, strconv.Itoa(*req.BandwidthLimitKbps)); err != nil {
			return fmt.Errorf("update setting bandwidth_limit_kbps: %w", err)
		}
	}
	if req.MaxConcurrentInstalls != nil {
		if err := s.set(ctx, keyMaxConcurrentInstalls, strconv.Itoa(*req.MaxConcurrentInstalls)); err != nil {
			return fmt.Errorf("update setting max_concurrent_installs: %w", err)
		}
	}
	if req.ProxyURL != nil {
		if err := s.set(ctx, keyProxyURL, *req.ProxyURL); err != nil {
			return fmt.Errorf("update setting proxy_url: %w", err)
		}
	}
	if req.AutoRebootWindow != nil {
		if err := s.set(ctx, keyAutoRebootWindow, *req.AutoRebootWindow); err != nil {
			return fmt.Errorf("update setting auto_reboot_window: %w", err)
		}
	}
	if req.LogRetentionDays != nil {
		if err := s.set(ctx, keyLogRetentionDays, strconv.Itoa(*req.LogRetentionDays)); err != nil {
			return fmt.Errorf("update setting log_retention_days: %w", err)
		}
	}
	if req.OfflineMode != nil {
		if err := s.set(ctx, keyOfflineMode, boolToStr(*req.OfflineMode)); err != nil {
			return fmt.Errorf("update setting offline_mode: %w", err)
		}
	}
	return nil
}

// GetScanInterval reads the scan_interval override from agent_state.
// Returns empty string if not set.
func (s *SettingsStore) GetScanInterval() string {
	return s.get(keyScanInterval)
}

// GetLogLevel reads the log_level override from agent_state.
func (s *SettingsStore) GetLogLevel() string {
	return s.get(keyLogLevel)
}

// GetAutoDeploy reads the auto_deploy override from agent_state.
func (s *SettingsStore) GetAutoDeploy() *bool {
	return s.getBool(keyAutoDeploy)
}

// GetHeartbeatInterval reads the heartbeat_interval override from agent_state.
func (s *SettingsStore) GetHeartbeatInterval() string {
	return s.get(keyHeartbeatInterval)
}

// GetBandwidthLimitKbps reads the bandwidth_limit_kbps override from agent_state.
func (s *SettingsStore) GetBandwidthLimitKbps() *int {
	return s.getInt(keyBandwidthLimitKbps)
}

// GetMaxConcurrentInstalls reads the max_concurrent_installs override from agent_state.
func (s *SettingsStore) GetMaxConcurrentInstalls() *int {
	return s.getInt(keyMaxConcurrentInstalls)
}

// GetProxyURL reads the proxy_url override from agent_state.
func (s *SettingsStore) GetProxyURL() string {
	return s.get(keyProxyURL)
}

// GetAutoRebootWindow reads the auto_reboot_window override from agent_state.
func (s *SettingsStore) GetAutoRebootWindow() string {
	return s.get(keyAutoRebootWindow)
}

// GetLogRetentionDays reads the log_retention_days override from agent_state.
func (s *SettingsStore) GetLogRetentionDays() *int {
	return s.getInt(keyLogRetentionDays)
}

// GetOfflineMode reads the offline_mode override from agent_state.
func (s *SettingsStore) GetOfflineMode() *bool {
	return s.getBool(keyOfflineMode)
}

// AgentID reads the enrolled agent_id from the agent_state table.
// Returns empty string if the agent has not been enrolled yet.
func (s *SettingsStore) AgentID() string {
	return s.get("agent_id")
}

func (s *SettingsStore) set(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agent_state (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (s *SettingsStore) get(key string) string {
	var value string
	err := s.db.QueryRow(`SELECT value FROM agent_state WHERE key = ?`, key).Scan(&value)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			slog.Warn("settings store: read failed", "key", key, "error", err)
		}
		return ""
	}
	return value
}

func (s *SettingsStore) getBool(key string) *bool {
	v := s.get(key)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return nil
	}
	return &b
}

func (s *SettingsStore) getInt(key string) *int {
	v := s.get(key)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
