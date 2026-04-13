package domain

import "context"

// EventHandler processes a single domain event.
type EventHandler func(ctx context.Context, event DomainEvent) error

// EventBus is the interface for publishing and subscribing to domain events.
// Implementations must support wildcard patterns:
//   - "*" matches all events
//   - "deployment.*" matches all deployment events
//   - "deployment.created" matches exactly that event type
type EventBus interface {
	// Emit publishes a domain event to all matching subscribers.
	Emit(ctx context.Context, event DomainEvent) error

	// Subscribe registers a handler for events matching the given pattern.
	Subscribe(pattern string, handler EventHandler) error

	// Close shuts down the event bus and releases resources.
	Close() error
}
