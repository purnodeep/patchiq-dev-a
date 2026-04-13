package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Registry manages the lifecycle of agent modules.
type Registry struct {
	logger   *slog.Logger
	modules  []Module
	names    map[string]bool
	commands map[string]Module
}

// NewRegistry creates an empty module registry.
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		logger:   logger,
		names:    make(map[string]bool),
		commands: make(map[string]Module),
	}
}

// Register adds a module to the registry. Panics on duplicate module names.
func (r *Registry) Register(m Module) {
	name := m.Name()
	if r.names[name] {
		panic(fmt.Sprintf("duplicate module name: %s", name))
	}
	r.names[name] = true
	r.modules = append(r.modules, m)
	for _, cmd := range m.SupportedCommands() {
		r.commands[cmd] = m
	}
}

// InitAll initializes all modules in registration order. Stops on first error.
func (r *Registry) InitAll(ctx context.Context, deps ModuleDeps) error {
	for _, m := range r.modules {
		r.logger.Info("initializing module", "module", m.Name(), "version", m.Version())
		if err := m.Init(ctx, deps); err != nil {
			return fmt.Errorf("init module %s: %w", m.Name(), err)
		}
	}
	return nil
}

// StartAll starts all modules in registration order. If a module fails to start,
// all previously started modules are stopped in reverse order.
func (r *Registry) StartAll(ctx context.Context) error {
	for i, m := range r.modules {
		r.logger.Info("starting module", "module", m.Name())
		if err := m.Start(ctx); err != nil {
			// Roll back previously started modules in reverse order.
			for j := i - 1; j >= 0; j-- {
				prev := r.modules[j]
				r.logger.Info("rolling back module", "module", prev.Name())
				if stopErr := prev.Stop(ctx); stopErr != nil {
					r.logger.Error("rollback stop failed", "module", prev.Name(), "error", stopErr)
				}
			}
			return fmt.Errorf("start module %s: %w", m.Name(), err)
		}
	}
	return nil
}

// StopAll stops all modules in reverse registration order.
func (r *Registry) StopAll(ctx context.Context) error {
	var errs []error
	for i := len(r.modules) - 1; i >= 0; i-- {
		m := r.modules[i]
		r.logger.Info("stopping module", "module", m.Name())
		if err := m.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop module %s: %w", m.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// Capabilities returns the aggregated capabilities of all registered modules.
func (r *Registry) Capabilities() []string {
	var caps []string
	for _, m := range r.modules {
		caps = append(caps, m.Capabilities()...)
	}
	return caps
}

// HandleCommand dispatches a command to the module that supports it.
func (r *Registry) HandleCommand(ctx context.Context, cmd Command) (Result, error) {
	m, ok := r.commands[cmd.Type]
	if !ok {
		return Result{}, fmt.Errorf("no module handles command type %q", cmd.Type)
	}
	return m.HandleCommand(ctx, cmd)
}

// Modules returns all registered modules.
func (r *Registry) Modules() []Module {
	return r.modules
}
