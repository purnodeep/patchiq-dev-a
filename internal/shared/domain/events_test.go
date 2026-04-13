package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

func TestNewEventID(t *testing.T) {
	t.Run("returns 26-char ULID", func(t *testing.T) {
		id := domain.NewEventID()
		if len(id) != 26 {
			t.Errorf("NewEventID() length = %d, want 26", len(id))
		}
	})

	t.Run("returns unique IDs", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			id := domain.NewEventID()
			if seen[id] {
				t.Fatalf("duplicate ID at iteration %d: %s", i, id)
			}
			seen[id] = true
		}
	})

	t.Run("IDs are time-ordered", func(t *testing.T) {
		id1 := domain.NewEventID()
		time.Sleep(2 * time.Millisecond)
		id2 := domain.NewEventID()
		if id1 >= id2 {
			t.Errorf("expected id1 < id2, got %s >= %s", id1, id2)
		}
	})
}

func TestDomainEvent_JSON(t *testing.T) {
	evt := domain.DomainEvent{
		ID:         domain.NewEventID(),
		Type:       "endpoint.created",
		TenantID:   "00000000-0000-0000-0000-000000000001",
		ActorID:    "user-1",
		ActorType:  domain.ActorUser,
		Resource:   "endpoint",
		ResourceID: "ep-1",
		Action:     "created",
		Payload:    map[string]any{"hostname": "web-01"},
		Metadata: domain.EventMeta{
			TraceID:   "abc123",
			RequestID: "req-1",
		},
		Timestamp: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded domain.DomainEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != evt.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, evt.Type)
	}
	if decoded.ActorType != domain.ActorUser {
		t.Errorf("ActorType = %q, want %q", decoded.ActorType, domain.ActorUser)
	}
	if decoded.TenantID != evt.TenantID {
		t.Errorf("TenantID = %q, want %q", decoded.TenantID, evt.TenantID)
	}
}

func TestNewAuditEvent(t *testing.T) {
	evt := domain.NewAuditEvent(
		"endpoint.created",
		"tenant-1",
		"user-1",
		domain.ActorUser,
		"endpoint",
		"ep-1",
		"created",
		map[string]any{"hostname": "web-01"},
		domain.EventMeta{TraceID: "t1", RequestID: "r1"},
	)

	if len(evt.ID) != 26 {
		t.Errorf("ID length = %d, want 26", len(evt.ID))
	}
	if evt.Type != "endpoint.created" {
		t.Errorf("Type = %q, want %q", evt.Type, "endpoint.created")
	}
	if evt.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", evt.TenantID, "tenant-1")
	}
	if evt.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestNewSystemEvent(t *testing.T) {
	evt := domain.NewSystemEvent(
		"license.validated",
		"tenant-1",
		"license",
		"lic-1",
		"validated",
		nil,
	)

	if evt.ActorID != "system" {
		t.Errorf("ActorID = %q, want %q", evt.ActorID, "system")
	}
	if evt.ActorType != domain.ActorSystem {
		t.Errorf("ActorType = %q, want %q", evt.ActorType, domain.ActorSystem)
	}
}
