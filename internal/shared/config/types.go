package config

import (
	"encoding/json"
	"time"
)

// Module name constants.
const (
	ModuleScan         = "scan"
	ModuleDeploy       = "deploy"
	ModuleNotification = "notification"
	ModuleAgent        = "agent"
	ModuleDiscovery    = "discovery"
	ModuleCVE          = "cve"
)

// ScopeType identifies the hierarchy level in the DB. Migration 060
// dropped the "group" scope; only tenant, tag, and endpoint are valid
// now (enforced by the config_overrides CHECK constraint).
const (
	ScopeTenant   = "tenant"
	ScopeTag      = "tag"
	ScopeEndpoint = "endpoint"
)

// TimeWindow represents a maintenance window.
type TimeWindow struct {
	Day   string `json:"day"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// Duration wraps time.Duration with JSON marshal support.
type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = dur
	return nil
}

// ScanConfig holds scan-related settings.
type ScanConfig struct {
	Schedule      *string `json:"scan_schedule,omitempty"`
	MaxConcurrent *int    `json:"max_concurrent,omitempty"`
}

// DeployConfig holds deployment-related settings.
type DeployConfig struct {
	MaintenanceWindow *TimeWindow `json:"maintenance_window,omitempty"`
	AutoReboot        *bool       `json:"auto_reboot,omitempty"`
	RebootDelay       *Duration   `json:"reboot_delay,omitempty"`
	MaxConcurrent     *int        `json:"max_concurrent_installs,omitempty"`
	WaveStrategy      *string     `json:"wave_strategy,omitempty"`
	NotifyUser        *bool       `json:"notify_user_before_reboot,omitempty"`
	ExcludedPackages  []string    `json:"excluded_packages,omitempty"`
	PreScript         *string     `json:"pre_script,omitempty"`
	PostScript        *string     `json:"post_script,omitempty"`
	BandwidthLimit    *string     `json:"bandwidth_limit,omitempty"`
}

// NotificationConfig holds notification settings.
type NotificationConfig struct {
	EmailEnabled *bool    `json:"email_enabled,omitempty"`
	SlackEnabled *bool    `json:"slack_enabled,omitempty"`
	Channels     []string `json:"channels,omitempty"`
}

// DiscoveryConfig holds patch discovery settings.
type DiscoveryConfig struct {
	Schedule         *string            `json:"schedule,omitempty"`
	SyncIntervalMins *int               `json:"sync_interval_mins,omitempty"`
	HTTPTimeout      *int               `json:"http_timeout_secs,omitempty"`
	MaxRetries       *int               `json:"max_retries,omitempty"`
	Repositories     []RepositoryConfig `json:"repositories,omitempty"`
}

// RepositoryConfig defines a single package repository to scan.
type RepositoryConfig struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	OsFamily string `json:"os_family"`
	OsDistro string `json:"os_distribution"`
	Enabled  bool   `json:"enabled"`
}

// CVEConfig holds CVE feed ingestion settings.
type CVEConfig struct {
	Schedule         *string `json:"schedule,omitempty"`
	SyncIntervalMins *int    `json:"sync_interval_mins,omitempty"`
	HTTPTimeout      *int    `json:"http_timeout_secs,omitempty"`
	MaxRetries       *int    `json:"max_retries,omitempty"`
}

// AgentConfig holds agent behavior settings.
type AgentConfig struct {
	HeartbeatInterval *Duration `json:"heartbeat_interval,omitempty"`
	LogLevel          *string   `json:"log_level,omitempty"`
	SelfUpdateEnabled *bool     `json:"self_update_enabled,omitempty"`
}
