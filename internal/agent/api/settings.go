package api

import "net/http"

// SettingsInfo holds agent configuration returned by GET /api/v1/settings.
// Values are read-only; most come from the agent config file.
type SettingsInfo struct {
	AgentID               string `json:"agent_id"`
	AgentVersion          string `json:"agent_version"`
	ConfigFile            string `json:"config_file"`
	DataDir               string `json:"data_dir"`
	LogFile               string `json:"log_file"`
	DBPath                string `json:"db_path"`
	ServerURL             string `json:"server_url"`
	HTTPAddr              string `json:"http_addr"`
	ScanInterval          string `json:"scan_interval"`
	ScanTimeout           string `json:"scan_timeout"`
	LogLevel              string `json:"log_level"`
	AutoDeploy            bool   `json:"auto_deploy"`
	HeartbeatInterval     string `json:"heartbeat_interval"`
	BandwidthLimitKbps    int    `json:"bandwidth_limit_kbps"`
	MaxConcurrentInstalls int    `json:"max_concurrent_installs"`
	ProxyURL              string `json:"proxy_url"`
	AutoRebootWindow      string `json:"auto_reboot_window"`
	LogRetentionDays      int    `json:"log_retention_days"`
	OfflineMode           bool   `json:"offline_mode"`
}

// SettingsProvider retrieves agent configuration.
type SettingsProvider interface {
	Settings() SettingsInfo
}

type staticSettingsProvider struct{ info SettingsInfo }

func (p staticSettingsProvider) Settings() SettingsInfo { return p.info }

// StaticSettingsProvider returns a SettingsProvider that always returns the given info.
func StaticSettingsProvider(info SettingsInfo) SettingsProvider {
	return staticSettingsProvider{info: info}
}

// SettingsReader reads persisted setting overrides.
type SettingsReader interface {
	GetScanInterval() string
	GetLogLevel() string
	GetAutoDeploy() *bool
	GetHeartbeatInterval() string
	GetBandwidthLimitKbps() *int
	GetMaxConcurrentInstalls() *int
	GetProxyURL() string
	GetAutoRebootWindow() string
	GetLogRetentionDays() *int
	GetOfflineMode() *bool
}

// AgentIDReader reads the agent_id from persistent state (e.g., agent_state table).
type AgentIDReader interface {
	AgentID() string
}

// DynamicSettingsProvider merges base config with persisted overrides.
type DynamicSettingsProvider struct {
	base          SettingsInfo
	reader        SettingsReader
	agentIDReader AgentIDReader
}

// NewDynamicSettingsProvider creates a provider that reads persisted overrides on each call.
func NewDynamicSettingsProvider(base SettingsInfo, reader SettingsReader) *DynamicSettingsProvider {
	return &DynamicSettingsProvider{base: base, reader: reader}
}

// SetAgentIDReader sets the source for reading the enrolled agent_id.
// Must be called before Settings() is invoked.
func (p *DynamicSettingsProvider) SetAgentIDReader(r AgentIDReader) {
	p.agentIDReader = r
}

// Settings returns the current settings, merging base config with any persisted overrides.
func (p *DynamicSettingsProvider) Settings() SettingsInfo {
	info := p.base

	// Read agent_id from persistent state (populated after enrollment).
	if p.agentIDReader != nil {
		if id := p.agentIDReader.AgentID(); id != "" {
			info.AgentID = id
		}
	}

	if v := p.reader.GetScanInterval(); v != "" {
		info.ScanInterval = v
	}
	if v := p.reader.GetLogLevel(); v != "" {
		info.LogLevel = v
	}
	if v := p.reader.GetAutoDeploy(); v != nil {
		info.AutoDeploy = *v
	}
	if v := p.reader.GetHeartbeatInterval(); v != "" {
		info.HeartbeatInterval = v
	}
	if v := p.reader.GetBandwidthLimitKbps(); v != nil {
		info.BandwidthLimitKbps = *v
	}
	if v := p.reader.GetMaxConcurrentInstalls(); v != nil {
		info.MaxConcurrentInstalls = *v
	}
	if v := p.reader.GetProxyURL(); v != "" {
		info.ProxyURL = v
	}
	if v := p.reader.GetAutoRebootWindow(); v != "" {
		info.AutoRebootWindow = v
	}
	if v := p.reader.GetLogRetentionDays(); v != nil {
		info.LogRetentionDays = *v
	}
	if v := p.reader.GetOfflineMode(); v != nil {
		info.OfflineMode = *v
	}

	return info
}

// SettingsHandler serves GET /api/v1/settings.
type SettingsHandler struct{ provider SettingsProvider }

// NewSettingsHandler creates a SettingsHandler.
func NewSettingsHandler(p SettingsProvider) *SettingsHandler { return &SettingsHandler{provider: p} }

// Get handles GET /api/v1/settings.
func (h *SettingsHandler) Get(w http.ResponseWriter, _ *http.Request) {
	WriteJSON(w, http.StatusOK, h.provider.Settings())
}
