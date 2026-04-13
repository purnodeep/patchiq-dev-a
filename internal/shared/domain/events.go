package domain

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// ActorType constants matching the DB CHECK constraint on audit_events.actor_type.
const (
	ActorUser        = "user"
	ActorSystem      = "system"
	ActorAIAssistant = "ai_assistant"
)

// EventMeta carries request-scoped metadata for tracing and audit.
type EventMeta struct {
	TraceID   string `json:"trace_id"`
	RequestID string `json:"request_id"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

// DomainEvent is the canonical event envelope emitted by every write operation.
// The struct maps 1:1 to the audit_events table columns.
type DomainEvent struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	TenantID   string    `json:"tenant_id"`
	ActorID    string    `json:"actor_id"`
	ActorType  string    `json:"actor_type"`
	Resource   string    `json:"resource"`
	ResourceID string    `json:"resource_id"`
	Action     string    `json:"action"`
	Payload    any       `json:"payload"`
	Metadata   EventMeta `json:"metadata"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewEventID generates a time-ordered, globally unique ULID string (26 chars).
func NewEventID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}
