package agent

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Default setting values match the current hardcoded behavior.
const (
	DefaultHeartbeatInterval     = 30 * time.Second
	DefaultScanInterval          = 6 * time.Hour
	DefaultLogRetentionDays      = 30
	DefaultBandwidthLimitKbps    = 0
	DefaultMaxConcurrentInstalls = 1
)

// settingsWatcherRefreshInterval controls how often the watcher re-reads settings from SQLite.
const settingsWatcherRefreshInterval = 30 * time.Second

// SettingsSource reads persisted agent settings. Implemented by store.SettingsStore.
// Defined here to avoid an import cycle (agent -> store -> api -> agent).
type SettingsSource interface {
	GetHeartbeatInterval() string
	GetScanInterval() string
	GetOfflineMode() *bool
	GetLogRetentionDays() *int
	GetBandwidthLimitKbps() *int
	GetMaxConcurrentInstalls() *int
	GetLogLevel() string
}

// watcherCache holds the parsed, in-memory copy of agent settings.
type watcherCache struct {
	heartbeatInterval     time.Duration
	scanInterval          time.Duration
	offlineMode           bool
	logRetentionDays      int
	bandwidthLimitKbps    int
	maxConcurrentInstalls int
	logLevel              slog.Level
}

// SettingsWatcher periodically reads agent settings from the SettingsStore
// and caches them in memory so hot-path consumers avoid SQLite reads.
type SettingsWatcher struct {
	store       SettingsSource
	mu          sync.RWMutex
	cache       watcherCache
	logger      *slog.Logger
	logLevelVar *slog.LevelVar // if set, updated when log_level setting changes
}

// NewSettingsWatcher creates a watcher with sensible defaults.
func NewSettingsWatcher(s SettingsSource, logger *slog.Logger) *SettingsWatcher {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	w := &SettingsWatcher{
		store:  s,
		logger: logger,
		cache: watcherCache{
			heartbeatInterval:     DefaultHeartbeatInterval,
			scanInterval:          DefaultScanInterval,
			logRetentionDays:      DefaultLogRetentionDays,
			bandwidthLimitKbps:    DefaultBandwidthLimitKbps,
			maxConcurrentInstalls: DefaultMaxConcurrentInstalls,
		},
	}
	// Load once synchronously so callers see persisted values immediately.
	w.refresh()
	return w
}

// SetLogLevelVar sets the slog.LevelVar that will be updated when the
// log_level setting changes. Must be called before Start.
func (w *SettingsWatcher) SetLogLevelVar(lv *slog.LevelVar) {
	w.logLevelVar = lv
}

// Start runs the periodic refresh loop. Blocks until ctx is cancelled.
func (w *SettingsWatcher) Start(ctx context.Context) {
	w.logger.InfoContext(ctx, "settings watcher started", "refresh_interval", settingsWatcherRefreshInterval)
	ticker := time.NewTicker(settingsWatcherRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.InfoContext(ctx, "settings watcher stopped")
			return
		case <-ticker.C:
			w.refresh()
		}
	}
}

// refresh reads all settings from the store and updates the cache.
func (w *SettingsWatcher) refresh() {
	var c watcherCache

	// Heartbeat interval
	if v := w.store.GetHeartbeatInterval(); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.heartbeatInterval = d
		} else {
			w.logger.Warn("settings watcher: invalid heartbeat_interval, using default", "value", v)
			c.heartbeatInterval = DefaultHeartbeatInterval
		}
	} else {
		c.heartbeatInterval = DefaultHeartbeatInterval
	}

	// Scan interval
	if v := w.store.GetScanInterval(); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			c.scanInterval = d
		} else {
			w.logger.Warn("settings watcher: invalid scan_interval, using default", "value", v)
			c.scanInterval = DefaultScanInterval
		}
	} else {
		c.scanInterval = DefaultScanInterval
	}

	// Offline mode
	if v := w.store.GetOfflineMode(); v != nil {
		c.offlineMode = *v
	}

	// Log retention days
	if v := w.store.GetLogRetentionDays(); v != nil {
		c.logRetentionDays = *v
	} else {
		c.logRetentionDays = DefaultLogRetentionDays
	}

	// Bandwidth limit
	if v := w.store.GetBandwidthLimitKbps(); v != nil {
		c.bandwidthLimitKbps = *v
	} else {
		c.bandwidthLimitKbps = DefaultBandwidthLimitKbps
	}

	// Max concurrent installs
	if v := w.store.GetMaxConcurrentInstalls(); v != nil {
		c.maxConcurrentInstalls = *v
	} else {
		c.maxConcurrentInstalls = DefaultMaxConcurrentInstalls
	}

	// Log level
	if v := w.store.GetLogLevel(); v != "" {
		c.logLevel = parseSettingsLogLevel(v)
	} else {
		c.logLevel = slog.LevelInfo
	}

	w.mu.Lock()
	prev := w.cache
	w.cache = c
	w.mu.Unlock()

	// Update the slog.LevelVar if log level changed.
	if w.logLevelVar != nil && prev.logLevel != c.logLevel {
		w.logLevelVar.Set(c.logLevel)
	}

	// Log changes at info level so operators can see when settings take effect.
	if prev.heartbeatInterval != c.heartbeatInterval {
		w.logger.Info("settings watcher: heartbeat_interval changed", "old", prev.heartbeatInterval, "new", c.heartbeatInterval)
	}
	if prev.scanInterval != c.scanInterval {
		w.logger.Info("settings watcher: scan_interval changed", "old", prev.scanInterval, "new", c.scanInterval)
	}
	if prev.offlineMode != c.offlineMode {
		w.logger.Info("settings watcher: offline_mode changed", "old", prev.offlineMode, "new", c.offlineMode)
	}
	if prev.logLevel != c.logLevel {
		w.logger.Info("settings watcher: log_level changed", "old", prev.logLevel, "new", c.logLevel)
	}
	if prev.bandwidthLimitKbps != c.bandwidthLimitKbps {
		w.logger.Info("settings watcher: bandwidth_limit_kbps changed", "old", prev.bandwidthLimitKbps, "new", c.bandwidthLimitKbps)
	}
	if prev.maxConcurrentInstalls != c.maxConcurrentInstalls {
		w.logger.Info("settings watcher: max_concurrent_installs changed", "old", prev.maxConcurrentInstalls, "new", c.maxConcurrentInstalls)
	}
}

// HeartbeatInterval returns the current heartbeat interval.
func (w *SettingsWatcher) HeartbeatInterval() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cache.heartbeatInterval
}

// ScanInterval returns the current scan/collection interval.
func (w *SettingsWatcher) ScanInterval() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cache.scanInterval
}

// IsOffline returns whether offline mode is enabled.
func (w *SettingsWatcher) IsOffline() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cache.offlineMode
}

// LogRetentionDays returns the configured log retention in days.
func (w *SettingsWatcher) LogRetentionDays() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cache.logRetentionDays
}

// BandwidthLimitKbps returns the download bandwidth limit in kbps (0=unlimited).
func (w *SettingsWatcher) BandwidthLimitKbps() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cache.bandwidthLimitKbps
}

// MaxConcurrentInstalls returns the max number of concurrent patch installs.
func (w *SettingsWatcher) MaxConcurrentInstalls() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cache.maxConcurrentInstalls
}

// parseSettingsLogLevel converts a log level string to slog.Level.
func parseSettingsLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
