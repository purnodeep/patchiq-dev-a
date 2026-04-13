package agent_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent"
)

// mockSettingsSource implements agent.SettingsSource for testing.
type mockSettingsSource struct {
	mu                    sync.Mutex
	heartbeatInterval     string
	scanInterval          string
	offlineMode           *bool
	logRetentionDays      *int
	bandwidthLimitKbps    *int
	maxConcurrentInstalls *int
	logLevel              string
}

func (m *mockSettingsSource) GetHeartbeatInterval() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.heartbeatInterval
}
func (m *mockSettingsSource) GetScanInterval() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.scanInterval
}
func (m *mockSettingsSource) GetOfflineMode() *bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.offlineMode
}
func (m *mockSettingsSource) GetLogRetentionDays() *int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logRetentionDays
}
func (m *mockSettingsSource) GetBandwidthLimitKbps() *int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bandwidthLimitKbps
}
func (m *mockSettingsSource) GetMaxConcurrentInstalls() *int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maxConcurrentInstalls
}
func (m *mockSettingsSource) GetLogLevel() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logLevel
}

func boolPtr(b bool) *bool { return &b }
func intPtr(n int) *int    { return &n }

func TestSettingsWatcher_DefaultValues(t *testing.T) {
	t.Parallel()
	src := &mockSettingsSource{}
	w := agent.NewSettingsWatcher(src, slog.Default())

	if w.HeartbeatInterval() != agent.DefaultHeartbeatInterval {
		t.Errorf("HeartbeatInterval() = %v, want %v", w.HeartbeatInterval(), agent.DefaultHeartbeatInterval)
	}
	if w.ScanInterval() != agent.DefaultScanInterval {
		t.Errorf("ScanInterval() = %v, want %v", w.ScanInterval(), agent.DefaultScanInterval)
	}
	if w.IsOffline() {
		t.Error("IsOffline() = true, want false")
	}
	if w.LogRetentionDays() != agent.DefaultLogRetentionDays {
		t.Errorf("LogRetentionDays() = %d, want %d", w.LogRetentionDays(), agent.DefaultLogRetentionDays)
	}
	if w.BandwidthLimitKbps() != agent.DefaultBandwidthLimitKbps {
		t.Errorf("BandwidthLimitKbps() = %d, want %d", w.BandwidthLimitKbps(), agent.DefaultBandwidthLimitKbps)
	}
	if w.MaxConcurrentInstalls() != agent.DefaultMaxConcurrentInstalls {
		t.Errorf("MaxConcurrentInstalls() = %d, want %d", w.MaxConcurrentInstalls(), agent.DefaultMaxConcurrentInstalls)
	}
}

func TestSettingsWatcher_ReadsPersistedValues(t *testing.T) {
	t.Parallel()
	src := &mockSettingsSource{
		heartbeatInterval:     "10s",
		scanInterval:          "1h",
		offlineMode:           boolPtr(true),
		logRetentionDays:      intPtr(7),
		bandwidthLimitKbps:    intPtr(1024),
		maxConcurrentInstalls: intPtr(3),
	}

	w := agent.NewSettingsWatcher(src, slog.Default())

	if w.HeartbeatInterval() != 10*time.Second {
		t.Errorf("HeartbeatInterval() = %v, want 10s", w.HeartbeatInterval())
	}
	if w.ScanInterval() != 1*time.Hour {
		t.Errorf("ScanInterval() = %v, want 1h", w.ScanInterval())
	}
	if !w.IsOffline() {
		t.Error("IsOffline() = false, want true")
	}
	if w.LogRetentionDays() != 7 {
		t.Errorf("LogRetentionDays() = %d, want 7", w.LogRetentionDays())
	}
	if w.BandwidthLimitKbps() != 1024 {
		t.Errorf("BandwidthLimitKbps() = %d, want 1024", w.BandwidthLimitKbps())
	}
	if w.MaxConcurrentInstalls() != 3 {
		t.Errorf("MaxConcurrentInstalls() = %d, want 3", w.MaxConcurrentInstalls())
	}
}

func TestSettingsWatcher_InvalidDurationFallsBackToDefault(t *testing.T) {
	t.Parallel()
	src := &mockSettingsSource{
		heartbeatInterval: "not-a-duration",
		scanInterval:      "not-a-duration",
	}

	w := agent.NewSettingsWatcher(src, slog.Default())

	if w.HeartbeatInterval() != agent.DefaultHeartbeatInterval {
		t.Errorf("HeartbeatInterval() = %v, want default %v", w.HeartbeatInterval(), agent.DefaultHeartbeatInterval)
	}
	if w.ScanInterval() != agent.DefaultScanInterval {
		t.Errorf("ScanInterval() = %v, want default %v", w.ScanInterval(), agent.DefaultScanInterval)
	}
}

func TestSettingsWatcher_StartStopsOnContextCancel(t *testing.T) {
	t.Parallel()
	src := &mockSettingsSource{}
	w := agent.NewSettingsWatcher(src, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancellation")
	}
}

func TestSettingsWatcher_PropagatesChanges(t *testing.T) {
	t.Parallel()

	// Start with initial settings.
	src := &mockSettingsSource{
		heartbeatInterval:     "30s",
		scanInterval:          "15m",
		maxConcurrentInstalls: intPtr(3),
		bandwidthLimitKbps:    intPtr(500),
		logRetentionDays:      intPtr(30),
		offlineMode:           boolPtr(false),
		logLevel:              "info",
	}

	w := agent.NewSettingsWatcher(src, slog.Default())

	// Verify initial values are loaded.
	if w.HeartbeatInterval() != 30*time.Second {
		t.Fatalf("initial HeartbeatInterval() = %v, want 30s", w.HeartbeatInterval())
	}
	if w.ScanInterval() != 15*time.Minute {
		t.Fatalf("initial ScanInterval() = %v, want 15m", w.ScanInterval())
	}
	if w.MaxConcurrentInstalls() != 3 {
		t.Fatalf("initial MaxConcurrentInstalls() = %d, want 3", w.MaxConcurrentInstalls())
	}

	// Simulate settings change in the store (e.g., pushed via server command).
	src.mu.Lock()
	src.scanInterval = "5m"
	src.maxConcurrentInstalls = intPtr(1)
	src.heartbeatInterval = "10s"
	src.bandwidthLimitKbps = intPtr(1024)
	src.logRetentionDays = intPtr(7)
	src.offlineMode = boolPtr(true)
	src.logLevel = "debug"
	src.mu.Unlock()

	// Trigger a refresh (simulates what the ticker does).
	w.Refresh()

	// Verify the watcher picked up all changes.
	if w.ScanInterval() != 5*time.Minute {
		t.Errorf("after change ScanInterval() = %v, want 5m", w.ScanInterval())
	}
	if w.MaxConcurrentInstalls() != 1 {
		t.Errorf("after change MaxConcurrentInstalls() = %d, want 1", w.MaxConcurrentInstalls())
	}
	if w.HeartbeatInterval() != 10*time.Second {
		t.Errorf("after change HeartbeatInterval() = %v, want 10s", w.HeartbeatInterval())
	}
	if w.BandwidthLimitKbps() != 1024 {
		t.Errorf("after change BandwidthLimitKbps() = %d, want 1024", w.BandwidthLimitKbps())
	}
	if w.LogRetentionDays() != 7 {
		t.Errorf("after change LogRetentionDays() = %d, want 7", w.LogRetentionDays())
	}
	if !w.IsOffline() {
		t.Error("after change IsOffline() = false, want true")
	}
}

func TestSettingsWatcher_PropagatesLogLevel(t *testing.T) {
	t.Parallel()

	src := &mockSettingsSource{logLevel: "info"}
	lv := &slog.LevelVar{}

	w := agent.NewSettingsWatcher(src, slog.Default())
	w.SetLogLevelVar(lv)

	// Force an initial refresh so the LevelVar is set.
	w.Refresh()
	if lv.Level() != slog.LevelInfo {
		t.Fatalf("initial log level = %v, want INFO", lv.Level())
	}

	// Change to debug.
	src.mu.Lock()
	src.logLevel = "debug"
	src.mu.Unlock()

	w.Refresh()
	if lv.Level() != slog.LevelDebug {
		t.Errorf("after change log level = %v, want DEBUG", lv.Level())
	}

	// Change to error.
	src.mu.Lock()
	src.logLevel = "error"
	src.mu.Unlock()

	w.Refresh()
	if lv.Level() != slog.LevelError {
		t.Errorf("after change log level = %v, want ERROR", lv.Level())
	}
}

func TestSettingsWatcher_InvalidValuesFallBackToDefaults(t *testing.T) {
	t.Parallel()

	// Start with valid values.
	src := &mockSettingsSource{
		heartbeatInterval: "10s",
		scanInterval:      "5m",
	}
	w := agent.NewSettingsWatcher(src, slog.Default())

	if w.HeartbeatInterval() != 10*time.Second {
		t.Fatalf("initial HeartbeatInterval() = %v, want 10s", w.HeartbeatInterval())
	}

	// Change to invalid values — watcher should fall back to defaults.
	src.mu.Lock()
	src.heartbeatInterval = "garbage"
	src.scanInterval = "-5m"
	src.mu.Unlock()

	w.Refresh()

	if w.HeartbeatInterval() != agent.DefaultHeartbeatInterval {
		t.Errorf("invalid HeartbeatInterval() = %v, want default %v", w.HeartbeatInterval(), agent.DefaultHeartbeatInterval)
	}
	if w.ScanInterval() != agent.DefaultScanInterval {
		t.Errorf("negative ScanInterval() = %v, want default %v", w.ScanInterval(), agent.DefaultScanInterval)
	}
}

func TestSettingsWatcher_MissingKeysFallBackToDefaults(t *testing.T) {
	t.Parallel()

	// Start with all values set.
	src := &mockSettingsSource{
		heartbeatInterval:     "10s",
		scanInterval:          "5m",
		maxConcurrentInstalls: intPtr(5),
		logRetentionDays:      intPtr(14),
		bandwidthLimitKbps:    intPtr(256),
		offlineMode:           boolPtr(true),
	}
	w := agent.NewSettingsWatcher(src, slog.Default())

	// Clear all values (simulate keys being deleted from the store).
	src.mu.Lock()
	src.heartbeatInterval = ""
	src.scanInterval = ""
	src.maxConcurrentInstalls = nil
	src.logRetentionDays = nil
	src.bandwidthLimitKbps = nil
	src.offlineMode = nil
	src.mu.Unlock()

	w.Refresh()

	if w.HeartbeatInterval() != agent.DefaultHeartbeatInterval {
		t.Errorf("HeartbeatInterval() = %v, want default %v", w.HeartbeatInterval(), agent.DefaultHeartbeatInterval)
	}
	if w.ScanInterval() != agent.DefaultScanInterval {
		t.Errorf("ScanInterval() = %v, want default %v", w.ScanInterval(), agent.DefaultScanInterval)
	}
	if w.MaxConcurrentInstalls() != agent.DefaultMaxConcurrentInstalls {
		t.Errorf("MaxConcurrentInstalls() = %d, want default %d", w.MaxConcurrentInstalls(), agent.DefaultMaxConcurrentInstalls)
	}
	if w.LogRetentionDays() != agent.DefaultLogRetentionDays {
		t.Errorf("LogRetentionDays() = %d, want default %d", w.LogRetentionDays(), agent.DefaultLogRetentionDays)
	}
	if w.BandwidthLimitKbps() != agent.DefaultBandwidthLimitKbps {
		t.Errorf("BandwidthLimitKbps() = %d, want default %d", w.BandwidthLimitKbps(), agent.DefaultBandwidthLimitKbps)
	}
	if w.IsOffline() {
		t.Error("IsOffline() = true, want false (default)")
	}
}

func TestSettingsWatcher_ZeroDurationFallsBackToDefault(t *testing.T) {
	t.Parallel()

	src := &mockSettingsSource{
		heartbeatInterval: "0s",
		scanInterval:      "0s",
	}
	w := agent.NewSettingsWatcher(src, slog.Default())

	// Zero durations should fall back to defaults because the code checks d > 0.
	if w.HeartbeatInterval() != agent.DefaultHeartbeatInterval {
		t.Errorf("zero HeartbeatInterval() = %v, want default %v", w.HeartbeatInterval(), agent.DefaultHeartbeatInterval)
	}
	if w.ScanInterval() != agent.DefaultScanInterval {
		t.Errorf("zero ScanInterval() = %v, want default %v", w.ScanInterval(), agent.DefaultScanInterval)
	}
}
