package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type ServerConfig struct {
	Server    ServerSettings    `koanf:"server"`
	Database  DatabaseSettings  `koanf:"database"`
	Valkey    ValkeySettings    `koanf:"valkey"`
	Watermill WatermillSettings `koanf:"watermill"`
	River     RiverSettings     `koanf:"river"`
	Log       LogSettings       `koanf:"log"`
	OTel      OTelSettings      `koanf:"otel"`
	IAM       IAMSettings       `koanf:"iam"`
	Env       string            `koanf:"env"`
}

// ValkeySettings holds Valkey (Redis-compatible) connection settings.
type ValkeySettings struct {
	URL string `koanf:"url"`
}

type ServerSettings struct {
	HTTP            HTTPSettings  `koanf:"http"`
	GRPC            GRPCSettings  `koanf:"grpc"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
	CORSOrigins     []string      `koanf:"cors_origins"`
	RepoCacheDir    string        `koanf:"repo_cache_dir"`
	MinIO           MinIOSettings `koanf:"minio"`
}

type HTTPSettings struct {
	Port         int           `koanf:"port"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
	IdleTimeout  time.Duration `koanf:"idle_timeout"`
}

type GRPCSettings struct {
	Port       int  `koanf:"port"`
	Reflection bool `koanf:"reflection"`
}

type DatabaseSettings struct {
	URL      string `koanf:"url"`
	MaxConns int    `koanf:"max_conns"`
	MinConns int    `koanf:"min_conns"`
}

type WatermillSettings struct {
	PostgresURL string `koanf:"postgres_url"`
}

type RiverSettings struct {
	MaxWorkers int `koanf:"max_workers"`
}

type LogSettings struct {
	Level      string `koanf:"level"`
	Format     string `koanf:"format"`
	File       string `koanf:"file"`
	MaxSizeMB  int    `koanf:"max_size_mb"`
	MaxBackups int    `koanf:"max_backups"`
	MaxAgeDays int    `koanf:"max_age_days"`
}

type OTelSettings struct {
	Endpoint string `koanf:"endpoint"`
	Insecure bool   `koanf:"insecure"`
}

// IAMSettings holds Zitadel IAM integration configuration.
type IAMSettings struct {
	Zitadel ZitadelSettings `koanf:"zitadel"`
	Session SessionSettings `koanf:"session"`
}

// ZitadelSettings holds Zitadel connection configuration.
type ZitadelSettings struct {
	Domain            string `koanf:"domain"`
	Secure            bool   `koanf:"secure"`
	ClientID          string `koanf:"client_id"`
	ClientSecret      string `koanf:"client_secret"`
	ServiceAccountKey string `koanf:"service_account_key"`
	RedirectURI       string `koanf:"redirect_uri"`
}

// SessionSettings holds session/cookie configuration.
type SessionSettings struct {
	CookieName    string        `koanf:"cookie_name"`
	CookieSecure  bool          `koanf:"cookie_secure"`
	CookieDomain  string        `koanf:"cookie_domain"`
	AccessTTL     time.Duration `koanf:"access_token_ttl"`
	RefreshTTL    time.Duration `koanf:"refresh_token_ttl"`
	RememberMeTTL time.Duration `koanf:"remember_me_ttl"`
	PostLoginURL  string        `koanf:"post_login_url"`
}

// knownKeys maps dot-separated env var candidates (all underscores converted to
// dots) to their canonical Koanf key paths. Only snake_case field names need an
// entry here — simple fields where the candidate already equals the canonical
// path are handled by the fallback in the env provider callback.
var knownKeys = map[string]string{
	"server.http.read.timeout":        "server.http.read_timeout",
	"server.http.write.timeout":       "server.http.write_timeout",
	"server.http.idle.timeout":        "server.http.idle_timeout",
	"server.shutdown.timeout":         "server.shutdown_timeout",
	"database.max.conns":              "database.max_conns",
	"database.min.conns":              "database.min_conns",
	"watermill.postgres.url":          "watermill.postgres_url",
	"river.max.workers":               "river.max_workers",
	"log.max.size.mb":                 "log.max_size_mb",
	"log.max.backups":                 "log.max_backups",
	"log.max.age.days":                "log.max_age_days",
	"iam.zitadel.client.id":           "iam.zitadel.client_id",
	"iam.zitadel.client.secret":       "iam.zitadel.client_secret",
	"iam.zitadel.service.account.key": "iam.zitadel.service_account_key",
	"iam.zitadel.redirect.uri":        "iam.zitadel.redirect_uri",
	"iam.session.cookie.name":         "iam.session.cookie_name",
	"iam.session.cookie.secure":       "iam.session.cookie_secure",
	"iam.session.cookie.domain":       "iam.session.cookie_domain",
	"iam.session.access.token.ttl":    "iam.session.access_token_ttl",
	"iam.session.refresh.token.ttl":   "iam.session.refresh_token_ttl",
	"iam.session.remember.me.ttl":     "iam.session.remember_me_ttl",
	"iam.session.post.login.url":      "iam.session.post_login_url",
	"server.repo.cache.dir":           "server.repo_cache_dir",
	"server.minio.access.key":         "server.minio.access_key",
	"server.minio.secret.key":         "server.minio.secret_key",
	"server.minio.use.ssl":            "server.minio.use_ssl",
}

// Load reads configuration from a YAML file and overlays environment variables
// prefixed with PATCHIQ_. Env vars use underscore as delimiter mapped to dots.
// Example: PATCHIQ_SERVER_HTTP_PORT -> server.http.port
// Example: PATCHIQ_DATABASE_MAX_CONNS -> database.max_conns
func Load(configPath string) (*ServerConfig, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("load config file %s: %w", configPath, err)
	}

	if err := k.Load(env.Provider("PATCHIQ_", ".", func(s string) string {
		// Strip prefix and lowercase: PATCHIQ_DATABASE_MAX_CONNS -> database_max_conns
		key := strings.ToLower(strings.TrimPrefix(s, "PATCHIQ_"))
		// Replace all underscores with dots to get a candidate path.
		candidate := strings.ReplaceAll(key, "_", ".")
		// Look up the canonical Koanf key (handles snake_case field names).
		if canonical, ok := knownKeys[candidate]; ok {
			return canonical
		}
		return candidate
	}), nil); err != nil {
		return nil, fmt.Errorf("load env config: %w", err)
	}

	var cfg ServerConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

type HubConfig struct {
	Hub       HubSettings       `koanf:"hub"`
	Database  DatabaseSettings  `koanf:"database"`
	Watermill WatermillSettings `koanf:"watermill"`
	Log       LogSettings       `koanf:"log"`
	OTel      OTelSettings      `koanf:"otel"`
	Env       string            `koanf:"env"`
	IAM       IAMSettings       `koanf:"iam"`
}

type HubSettings struct {
	HTTP            HTTPSettings  `koanf:"http"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
	CORSOrigins     []string      `koanf:"cors_origins"`
	MinIO           MinIOSettings `koanf:"minio"`
	DefaultTenantID string        `koanf:"default_tenant_id"`
}

// MinIOSettings holds MinIO (S3-compatible) object storage configuration.
type MinIOSettings struct {
	Endpoint  string `koanf:"endpoint"`
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Bucket    string `koanf:"bucket"`
	UseSSL    bool   `koanf:"use_ssl"`
}

// hubKnownKeys maps dot-separated env var candidates (all underscores converted
// to dots) to their canonical Koanf key paths for the Hub Manager config.
var hubKnownKeys = map[string]string{
	"hub.http.read.timeout":           "hub.http.read_timeout",
	"hub.http.write.timeout":          "hub.http.write_timeout",
	"hub.http.idle.timeout":           "hub.http.idle_timeout",
	"hub.shutdown.timeout":            "hub.shutdown_timeout",
	"log.max.size.mb":                 "log.max_size_mb",
	"log.max.backups":                 "log.max_backups",
	"log.max.age.days":                "log.max_age_days",
	"database.max.conns":              "database.max_conns",
	"database.min.conns":              "database.min_conns",
	"watermill.postgres.url":          "watermill.postgres_url",
	"iam.zitadel.client.id":           "iam.zitadel.client_id",
	"iam.zitadel.client.secret":       "iam.zitadel.client_secret",
	"iam.zitadel.service.account.key": "iam.zitadel.service_account_key",
	"iam.zitadel.redirect.uri":        "iam.zitadel.redirect_uri",
	"iam.session.cookie.name":         "iam.session.cookie_name",
	"iam.session.cookie.secure":       "iam.session.cookie_secure",
	"iam.session.cookie.domain":       "iam.session.cookie_domain",
	"iam.session.access.token.ttl":    "iam.session.access_token_ttl",
	"iam.session.refresh.token.ttl":   "iam.session.refresh_token_ttl",
	"iam.session.remember.me.ttl":     "iam.session.remember_me_ttl",
	"iam.session.post.login.url":      "iam.session.post_login_url",
	"hub.default.tenant.id":           "hub.default_tenant_id",
	"hub.minio.access.key":            "hub.minio.access_key",
	"hub.minio.secret.key":            "hub.minio.secret_key",
	"hub.minio.use.ssl":               "hub.minio.use_ssl",
}

// LoadHub reads configuration from a YAML file and overlays environment
// variables prefixed with PATCHIQ_HUB_. Env vars use underscore as delimiter
// mapped to dots.
// Example: PATCHIQ_HUB_HUB_HTTP_PORT -> hub.http.port
// Example: PATCHIQ_HUB_DATABASE_MAX_CONNS -> database.max_conns
func LoadHub(configPath string) (*HubConfig, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("load config file %s: %w", configPath, err)
	}

	if err := k.Load(env.Provider("PATCHIQ_HUB_", ".", func(s string) string {
		// Strip prefix and lowercase: PATCHIQ_HUB_DATABASE_MAX_CONNS -> database_max_conns
		key := strings.ToLower(strings.TrimPrefix(s, "PATCHIQ_HUB_"))
		// Replace all underscores with dots to get a candidate path.
		candidate := strings.ReplaceAll(key, "_", ".")
		// Look up the canonical Koanf key (handles snake_case field names).
		if canonical, ok := hubKnownKeys[candidate]; ok {
			return canonical
		}
		return candidate
	}), nil); err != nil {
		return nil, fmt.Errorf("load env config: %w", err)
	}

	var cfg HubConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func (c *HubConfig) validate() error {
	if c.Hub.HTTP.Port <= 0 || c.Hub.HTTP.Port > 65535 {
		return fmt.Errorf("hub.http.port must be between 1 and 65535, got %d", c.Hub.HTTP.Port)
	}
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}
	if c.Hub.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("hub.http.read_timeout must be positive, got %s", c.Hub.HTTP.ReadTimeout)
	}
	if c.Hub.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("hub.http.write_timeout must be positive, got %s", c.Hub.HTTP.WriteTimeout)
	}
	if c.Hub.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("hub.http.idle_timeout must be positive, got %s", c.Hub.HTTP.IdleTimeout)
	}
	if c.Hub.ShutdownTimeout <= 0 {
		return fmt.Errorf("hub.shutdown_timeout must be positive, got %s", c.Hub.ShutdownTimeout)
	}
	if c.Hub.DefaultTenantID == "" {
		return fmt.Errorf("hub.default_tenant_id is required")
	}
	if _, err := uuid.Parse(c.Hub.DefaultTenantID); err != nil {
		return fmt.Errorf("hub.default_tenant_id is not a valid UUID: %w", err)
	}
	if c.Watermill.PostgresURL == "" {
		return fmt.Errorf("validate hub config: watermill.postgres_url is required")
	}
	return nil
}

// validLogLevels defines the accepted log level strings.
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

func (c *ServerConfig) validate() error {
	if c.Server.HTTP.Port <= 0 || c.Server.HTTP.Port > 65535 {
		return fmt.Errorf("validate server config: server.http.port must be between 1 and 65535, got %d", c.Server.HTTP.Port)
	}
	if c.Server.GRPC.Port <= 0 || c.Server.GRPC.Port > 65535 {
		return fmt.Errorf("validate server config: server.grpc.port must be between 1 and 65535, got %d", c.Server.GRPC.Port)
	}
	if c.Server.HTTP.Port == c.Server.GRPC.Port {
		return fmt.Errorf("validate server config: server.http.port and server.grpc.port must differ, both set to %d", c.Server.HTTP.Port)
	}
	if c.Database.URL == "" {
		return fmt.Errorf("validate server config: database.url is required")
	}
	if c.Watermill.PostgresURL == "" {
		return fmt.Errorf("validate server config: watermill.postgres_url is required")
	}
	if c.Valkey.URL == "" {
		return fmt.Errorf("validate server config: valkey.url is required")
	}
	if c.Server.HTTP.ReadTimeout <= 0 {
		return fmt.Errorf("validate server config: server.http.read_timeout must be positive, got %s", c.Server.HTTP.ReadTimeout)
	}
	if c.Server.HTTP.WriteTimeout <= 0 {
		return fmt.Errorf("validate server config: server.http.write_timeout must be positive, got %s", c.Server.HTTP.WriteTimeout)
	}
	if c.Server.HTTP.IdleTimeout <= 0 {
		return fmt.Errorf("validate server config: server.http.idle_timeout must be positive, got %s", c.Server.HTTP.IdleTimeout)
	}
	if c.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("validate server config: server.shutdown_timeout must be positive, got %s", c.Server.ShutdownTimeout)
	}
	if len(c.Server.CORSOrigins) == 0 {
		slog.Warn("validate server config: server.cors_origins is empty, CORS will reject browser requests")
	}
	if c.Log.Level != "" && !validLogLevels[c.Log.Level] {
		return fmt.Errorf("validate server config: log.level must be one of debug/info/warn/error, got %q", c.Log.Level)
	}
	return nil
}
