package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// DefaultConfigPath returns the default config file path for the current OS.
// Exposed for callers (main.go) that need to check existence without loading.
func DefaultConfigPath() string {
	return defaultConfigPath
}

// AgentConfig holds the configuration for the PatchIQ agent.
type AgentConfig struct {
	ServerAddress string        `koanf:"server_address" yaml:"server_address"`
	ServerHTTPURL string        `koanf:"server_http_url" yaml:"server_http_url"`
	DataDir       string        `koanf:"data_dir" yaml:"data_dir"`
	LogLevel      string        `koanf:"log_level" yaml:"log_level"`
	ScanInterval  time.Duration `koanf:"scan_interval" yaml:"scan_interval"`
}

// LoadAgentConfig loads agent configuration with precedence: defaults < file < env vars.
// If configPath is empty or the file does not exist, defaults and env vars are used without error.
func LoadAgentConfig(configPath string) (AgentConfig, error) {
	k := koanf.New(".")

	// Server address default: prefer the ldflags-baked DefaultServerAddress
	// (release builds) so the daemon path respects the bake, not just the
	// install subcommand. Fall back to localhost:50051 for dev builds where
	// DefaultServerAddress is empty.
	defaultServer := DefaultServerAddress
	if defaultServer == "" {
		defaultServer = "localhost:50051"
	}

	// Set defaults.
	defaults := map[string]any{
		"server_address":  defaultServer,
		"server_http_url": "",
		"data_dir":        DefaultDataDir(),
		"log_level":       "info",
		"scan_interval":   15 * time.Minute,
	}
	for key, val := range defaults {
		k.Set(key, val) //nolint:errcheck // Set on empty koanf cannot fail.
	}

	// Load file: use provided path, or fall back to default platform path.
	if configPath == "" {
		configPath = defaultConfigPath
	}
	if _, err := os.Stat(configPath); err != nil {
		if !os.IsNotExist(err) {
			return AgentConfig{}, fmt.Errorf("load agent config: stat %s: %w", configPath, err)
		}
	} else {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return AgentConfig{}, fmt.Errorf("load agent config file %s: %w", configPath, err)
		}
	}

	// Load env vars: PATCHIQ_AGENT_SERVER_ADDRESS -> server_address.
	if err := k.Load(env.Provider("PATCHIQ_AGENT_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "PATCHIQ_AGENT_"))
	}), nil); err != nil {
		return AgentConfig{}, fmt.Errorf("load agent env config: %w", err)
	}

	var cfg AgentConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return AgentConfig{}, fmt.Errorf("unmarshal agent config: %w", err)
	}

	return cfg, nil
}
