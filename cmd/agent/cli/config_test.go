package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadAgentConfig_Defaults(t *testing.T) {
	cfg, err := LoadAgentConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerAddress != "localhost:50051" {
		t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "localhost:50051")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.ScanInterval != 15*time.Minute {
		t.Errorf("ScanInterval = %v, want %v", cfg.ScanInterval, 15*time.Minute)
	}
	if cfg.DataDir == "" {
		t.Error("DataDir should not be empty")
	}
}

func TestLoadAgentConfig_File(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "agent.yaml")
	content := `server_address: "10.0.0.1:8080"
data_dir: "/opt/patchiq"
log_level: "debug"
scan_interval: "5m"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadAgentConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerAddress != "10.0.0.1:8080" {
		t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "10.0.0.1:8080")
	}
	if cfg.DataDir != "/opt/patchiq" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/opt/patchiq")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.ScanInterval != 5*time.Minute {
		t.Errorf("ScanInterval = %v, want %v", cfg.ScanInterval, 5*time.Minute)
	}
}

func TestLoadAgentConfig_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "agent.yaml")
	content := `server_address: "10.0.0.1:8080"
log_level: "debug"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("PATCHIQ_AGENT_SERVER_ADDRESS", "envhost:9999")
	t.Setenv("PATCHIQ_AGENT_LOG_LEVEL", "warn")

	cfg, err := LoadAgentConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerAddress != "envhost:9999" {
		t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "envhost:9999")
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "warn")
	}
}

func TestLoadAgentConfig_MissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadAgentConfig("/nonexistent/path/agent.yaml")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}

	if cfg.ServerAddress != "localhost:50051" {
		t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, "localhost:50051")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}
