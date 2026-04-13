package agent

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// Module is the interface that all agent modules must implement.
type Module interface {
	Name() string
	Version() string
	Capabilities() []string
	SupportedCommands() []string
	CollectInterval() time.Duration

	Init(ctx context.Context, deps ModuleDeps) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	HandleCommand(ctx context.Context, cmd Command) (Result, error)
	Collect(ctx context.Context) ([]OutboxItem, error)
	HealthCheck(ctx context.Context) error
}

// ModuleDeps contains dependencies injected into modules during Init().
type ModuleDeps struct {
	Logger         *slog.Logger
	LocalDB        *sql.DB
	Outbox         OutboxWriter
	ConfigProvider ConfigProvider
	EventEmitter   EventEmitter
	FileCache      FileCache
}

// Command represents a command received from the server.
type Command struct {
	ID      string
	Type    string
	Payload []byte
}

// Result represents the outcome of a command execution.
type Result struct {
	Output       []byte
	ErrorMessage string
}

// OutboxItem represents a message to be queued in the outbox.
type OutboxItem struct {
	MessageType string
	Payload     []byte
}

// OutboxWriter is the interface for writing messages to the outbox queue.
type OutboxWriter interface {
	Add(ctx context.Context, messageType string, payload []byte) (int64, error)
}

// ConfigProvider provides module-specific configuration values.
type ConfigProvider interface {
	GetString(key string) string
	GetInt(key string) int
	GetDuration(key string) time.Duration
}

// EventEmitter emits local events within the agent.
type EventEmitter interface {
	Emit(ctx context.Context, eventType string, payload any) error
}

// FileCache provides file download and caching from the server.
type FileCache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, data []byte) error
}
