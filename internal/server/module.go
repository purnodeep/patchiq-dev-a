package server

import (
	"context"
	"io/fs"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

// Module is the interface that all server-side modules must implement.
type Module interface {
	Name() string
	Version() string

	Init(ctx context.Context, deps ModuleDeps) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	RegisterRoutes(router chi.Router)
	RegisterGRPCServices(server *grpc.Server)

	MigrationSource() fs.FS

	EventSubscriptions() map[string]EventHandler
}

// EventHandler processes a domain event.
type EventHandler func(ctx context.Context, payload []byte) error

// ModuleDeps contains dependencies injected into server modules during Init().
type ModuleDeps struct {
	Logger *slog.Logger
}
