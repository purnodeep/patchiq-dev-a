package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		envVars map[string]string
		check   func(t *testing.T, cfg *config.ServerConfig)
		wantErr bool
	}{
		{
			name:  "defaults from file",
			setup: func(t *testing.T) string { return "../../../configs/server.yaml" },
			envVars: map[string]string{
				"PATCHIQ_DATABASE_URL":           "postgres://localhost/test",
				"PATCHIQ_WATERMILL_POSTGRES_URL": "postgres://localhost/test",
				"PATCHIQ_VALKEY_URL":             "localhost:6379",
			},
			check: func(t *testing.T, cfg *config.ServerConfig) {
				assert.Equal(t, 8080, cfg.Server.HTTP.Port)
				assert.Equal(t, 50051, cfg.Server.GRPC.Port)
				assert.False(t, cfg.Server.GRPC.Reflection)
				assert.Equal(t, 200, cfg.Database.MaxConns)
				assert.Equal(t, "development", cfg.Env)
			},
		},
		{
			name:  "env overrides",
			setup: func(t *testing.T) string { return "../../../configs/server.yaml" },
			envVars: map[string]string{
				"PATCHIQ_SERVER_HTTP_PORT":       "9090",
				"PATCHIQ_DATABASE_MAX_CONNS":     "50",
				"PATCHIQ_DATABASE_URL":           "postgres://localhost/test",
				"PATCHIQ_WATERMILL_POSTGRES_URL": "postgres://localhost/test",
				"PATCHIQ_VALKEY_URL":             "localhost:6379",
			},
			check: func(t *testing.T, cfg *config.ServerConfig) {
				assert.Equal(t, 9090, cfg.Server.HTTP.Port)
				assert.Equal(t, 50, cfg.Database.MaxConns)
			},
		},
		{
			name:    "missing file returns error",
			setup:   func(t *testing.T) string { return "/nonexistent/path.yaml" },
			wantErr: true,
		},
		{
			name: "minimal file with required fields",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "minimal.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			check: func(t *testing.T, cfg *config.ServerConfig) {
				assert.Equal(t, "test", cfg.Env)
			},
		},
		{
			name: "validation fails on zero http port",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "bad.yaml")
				content := "env: test\ndatabase:\n  url: postgres://localhost/test\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on missing database url",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "nodb.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n  grpc:\n    port: 50051\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "port 65535 is valid for both http and grpc",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "maxport.yaml")
				content := "env: test\nserver:\n  http:\n    port: 65535\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 65534\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			check: func(t *testing.T, cfg *config.ServerConfig) {
				assert.Equal(t, 65535, cfg.Server.HTTP.Port)
				assert.Equal(t, 65534, cfg.Server.GRPC.Port)
			},
		},
		{
			name: "port 65536 is invalid",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "overport.yaml")
				content := "env: test\nserver:\n  http:\n    port: 65536\n  grpc:\n    port: 50051\ndatabase:\n  url: postgres://localhost/test\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "grpc port 0 is invalid",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zerogrpc.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n  grpc:\n    port: 0\ndatabase:\n  url: postgres://localhost/test\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "same port for http and grpc is invalid",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "sameport.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n  grpc:\n    port: 8080\ndatabase:\n  url: postgres://localhost/test\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on missing watermill postgres_url",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "no-watermill.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on missing valkey url",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "no-valkey.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on zero read_timeout",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zero-read.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 0s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on zero write_timeout",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zero-write.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 0s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on zero idle_timeout",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zero-idle.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 0s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on zero shutdown_timeout",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zero-shutdown.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 0s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on invalid log level",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "bad-log.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\nlog:\n  level: verbose\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "valid log level debug is accepted",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "debug-log.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\nlog:\n  level: debug\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			check: func(t *testing.T, cfg *config.ServerConfig) {
				assert.Equal(t, "debug", cfg.Log.Level)
			},
		},
		{
			name: "empty log level is accepted",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "no-log.yaml")
				content := "env: test\nserver:\n  http:\n    port: 8080\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  grpc:\n    port: 50051\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/test\nwatermill:\n  postgres_url: postgres://localhost/test\nvalkey:\n  url: localhost:6379\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			check: func(t *testing.T, cfg *config.ServerConfig) {
				assert.Equal(t, "", cfg.Log.Level)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			path := tt.setup(t)
			cfg, err := config.Load(path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadHub(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		envVars map[string]string
		check   func(t *testing.T, cfg *config.HubConfig)
		wantErr bool
	}{
		{
			name:  "defaults from file",
			setup: func(t *testing.T) string { return "../../../configs/hub.yaml" },
			envVars: map[string]string{
				"PATCHIQ_HUB_DATABASE_URL":           "postgres://localhost/hubtest",
				"PATCHIQ_HUB_WATERMILL_POSTGRES_URL": "postgres://localhost/hubtest",
			},
			check: func(t *testing.T, cfg *config.HubConfig) {
				assert.Equal(t, 8082, cfg.Hub.HTTP.Port)
				assert.Equal(t, 200, cfg.Database.MaxConns)
				assert.Equal(t, "development", cfg.Env)
			},
		},
		{
			name:  "env overrides with PATCHIQ_HUB_ prefix",
			setup: func(t *testing.T) string { return "../../../configs/hub.yaml" },
			envVars: map[string]string{
				"PATCHIQ_HUB_HUB_HTTP_PORT":          "9092",
				"PATCHIQ_HUB_DATABASE_MAX_CONNS":     "50",
				"PATCHIQ_HUB_DATABASE_URL":           "postgres://localhost/hubtest",
				"PATCHIQ_HUB_WATERMILL_POSTGRES_URL": "postgres://localhost/hubtest",
			},
			check: func(t *testing.T, cfg *config.HubConfig) {
				assert.Equal(t, 9092, cfg.Hub.HTTP.Port)
				assert.Equal(t, 50, cfg.Database.MaxConns)
			},
		},
		{
			name:    "missing file returns error",
			setup:   func(t *testing.T) string { return "/nonexistent/hub.yaml" },
			wantErr: true,
		},
		{
			name: "validation fails on zero http port",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "bad-hub.yaml")
				content := "env: test\ndatabase:\n  url: postgres://localhost/hub\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on missing database url",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "nodb-hub.yaml")
				content := "env: test\nhub:\n  http:\n    port: 8082\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on zero shutdown timeout",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zero-shutdown.yaml")
				content := "env: test\nhub:\n  http:\n    port: 8082\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  shutdown_timeout: 0s\ndatabase:\n  url: postgres://localhost/hub\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on zero read timeout",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "zero-read.yaml")
				content := "env: test\nhub:\n  http:\n    port: 8082\n    read_timeout: 0s\n    write_timeout: 30s\n    idle_timeout: 120s\n  shutdown_timeout: 15s\ndatabase:\n  url: postgres://localhost/hub\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
		{
			name: "validation fails on missing watermill postgres_url",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "no-watermill-hub.yaml")
				content := "env: test\nhub:\n  http:\n    port: 8082\n    read_timeout: 30s\n    write_timeout: 30s\n    idle_timeout: 120s\n  shutdown_timeout: 15s\n  default_tenant_id: \"00000000-0000-0000-0000-000000000001\"\ndatabase:\n  url: postgres://localhost/hub\n"
				err := os.WriteFile(path, []byte(content), 0644)
				require.NoError(t, err)
				return path
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			path := tt.setup(t)
			cfg, err := config.LoadHub(path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
