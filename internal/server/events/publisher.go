package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// WatermillEventBus implements domain.EventBus using Watermill with PostgreSQL transport.
type WatermillEventBus struct {
	publisher  message.Publisher
	router     *message.Router
	subFactory SubscriberFactory
	logger     watermill.LoggerAdapter

	mu       sync.RWMutex
	handlers map[string][]domain.EventHandler // pattern -> handlers
}

var _ domain.EventBus = (*WatermillEventBus)(nil)

// NewWatermillEventBus creates an EventBus backed by Watermill.
// Each Subscribe call creates a new subscriber with a unique consumer group
// so that multiple handlers on the same topic each receive every message.
func NewWatermillEventBus(
	publisher message.Publisher,
	subFactory SubscriberFactory,
	router *message.Router,
	logger watermill.LoggerAdapter,
) *WatermillEventBus {
	return &WatermillEventBus{
		publisher:  publisher,
		router:     router,
		subFactory: subFactory,
		logger:     logger,
		handlers:   make(map[string][]domain.EventHandler),
	}
}

// Emit publishes a domain event to the Watermill topic matching the event type.
// The event type must be registered in AllTopics() to ensure wildcard subscribers
// (like the audit subscriber) receive it. Emitting an unregistered topic returns
// an error to prevent silent audit gaps (Foundation 2 requirement).
func (b *WatermillEventBus) Emit(ctx context.Context, event domain.DomainEvent) error {
	if !isRegisteredTopic(event.Type) {
		return fmt.Errorf("emit event: topic %q is not registered in AllTopics(); add it to prevent audit gaps", event.Type)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal domain event: %w", err)
	}

	msg := message.NewMessage(event.ID, payload)
	msg.Metadata.Set("tenant_id", event.TenantID)
	msg.Metadata.Set("event_type", event.Type)

	if err := b.publisher.Publish(event.Type, msg); err != nil {
		return fmt.Errorf("publish event %s: %w", event.Type, err)
	}
	return nil
}

// isRegisteredTopic checks whether a topic is in the AllTopics() registry.
func isRegisteredTopic(topic string) bool {
	_, ok := TopicSet()[topic]
	return ok
}

// Subscribe registers a handler for events matching the given pattern.
// Patterns: "*" (all), "resource.*" (prefix), "resource.action" (exact).
//
// Subscribe must be called before the Watermill router is started (before Run).
// Calling Subscribe after the router is running will panic.
func (b *WatermillEventBus) Subscribe(pattern string, handler domain.EventHandler) error {
	b.mu.Lock()
	existing := len(b.handlers[pattern]) > 0
	b.handlers[pattern] = append(b.handlers[pattern], handler)
	b.mu.Unlock()

	// If router handlers already exist for this pattern, the new handler is
	// automatically invoked via the fan-out loop — no new router registration needed.
	if existing {
		return nil
	}

	topics := MatchingTopics(pattern)

	for _, topic := range topics {
		topicCopy := topic
		patternCopy := pattern
		handlerName := fmt.Sprintf("handler-%s-%s", patternCopy, topicCopy)
		sub, err := b.subFactory(handlerName)
		if err != nil {
			return fmt.Errorf("create subscriber for %s: %w", handlerName, err)
		}
		b.router.AddConsumerHandler(
			handlerName,
			topicCopy,
			sub,
			func(msg *message.Message) error {
				var evt domain.DomainEvent
				if err := json.Unmarshal(msg.Payload, &evt); err != nil {
					b.logger.Error("unmarshal event", err, watermill.LogFields{"topic": topicCopy})
					return nil // ack bad messages to avoid poison queue
				}

				b.mu.RLock()
				handlers := b.handlers[patternCopy]
				b.mu.RUnlock()

				for _, h := range handlers {
					if err := h(msg.Context(), evt); err != nil {
						return fmt.Errorf("handle event %s: %w", evt.Type, err)
					}
				}
				return nil
			},
		)
	}
	return nil
}

// Close shuts down the router and publisher. Subscribers are closed by the router.
func (b *WatermillEventBus) Close() error {
	var errs []error
	if err := b.router.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close router: %w", err))
	}
	if err := b.publisher.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close publisher: %w", err))
	}
	return errors.Join(errs...)
}

// MatchingTopics returns the list of known topics that match the given pattern.
func MatchingTopics(pattern string) []string {
	if pattern == "*" {
		return AllTopics()
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		var matched []string
		for _, t := range AllTopics() {
			if strings.HasPrefix(t, prefix+".") {
				matched = append(matched, t)
			}
		}
		return matched
	}
	return []string{pattern}
}
