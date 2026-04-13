package domain

import (
	"log/slog"
	"time"
)

// NewAuditEvent creates a DomainEvent with a generated ULID and current timestamp.
// Logs a warning when tenantID is empty, as tenant-scoped events should always carry a tenant.
func NewAuditEvent(
	eventType string,
	tenantID string,
	actorID string,
	actorType string,
	resource string,
	resourceID string,
	action string,
	payload any,
	meta EventMeta,
) DomainEvent {
	if tenantID == "" {
		slog.Warn("NewAuditEvent called with empty tenantID", "event_type", eventType, "resource", resource)
	}
	return DomainEvent{
		ID:         NewEventID(),
		Type:       eventType,
		TenantID:   tenantID,
		ActorID:    actorID,
		ActorType:  actorType,
		Resource:   resource,
		ResourceID: resourceID,
		Action:     action,
		Payload:    payload,
		Metadata:   meta,
		Timestamp:  time.Now().UTC(),
	}
}

// NewSystemEvent creates a DomainEvent attributed to the system actor.
func NewSystemEvent(
	eventType string,
	tenantID string,
	resource string,
	resourceID string,
	action string,
	payload any,
) DomainEvent {
	return NewAuditEvent(
		eventType,
		tenantID,
		ActorSystem,
		ActorSystem,
		resource,
		resourceID,
		action,
		payload,
		EventMeta{},
	)
}
