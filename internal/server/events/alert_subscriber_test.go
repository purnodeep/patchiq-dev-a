package events

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

func TestAlertSubscriber_buildAlertParams(t *testing.T) {
	t.Parallel()

	tenantID := "11111111-1111-1111-1111-111111111111"
	ruleID := "22222222-2222-2222-2222-222222222222"

	makeSub := func(rules ...struct {
		tenantID  string
		eventType string
		rule      *cachedRule
	}) *AlertSubscriber {
		s := &AlertSubscriber{log: slog.Default()}
		for _, r := range rules {
			s.cache.Store(cacheKey(r.tenantID, r.eventType), r.rule)
		}
		return s
	}

	t.Run("matching event creates params", func(t *testing.T) {
		t.Parallel()

		sub := makeSub(struct {
			tenantID  string
			eventType string
			rule      *cachedRule
		}{
			tenantID:  tenantID,
			eventType: "endpoint.enrolled",
			rule: &cachedRule{
				ID:                  ruleID,
				TenantID:            tenantID,
				Severity:            "info",
				Category:            "endpoint",
				TitleTemplate:       "Endpoint enrolled",
				DescriptionTemplate: "A new endpoint was enrolled",
			},
		})

		event := domain.DomainEvent{
			ID:         "EVT001",
			Type:       "endpoint.enrolled",
			TenantID:   tenantID,
			Resource:   "endpoint",
			ResourceID: "ep-123",
			Payload:    map[string]any{"hostname": "srv-01"},
			Timestamp:  time.Now(),
		}

		params := sub.buildAlertParams(event)
		if params == nil {
			t.Fatal("expected non-nil params for matching event")
		}
		if params.Severity != "info" {
			t.Errorf("severity = %q, want %q", params.Severity, "info")
		}
		if params.EventID != "EVT001" {
			t.Errorf("event_id = %q, want %q", params.EventID, "EVT001")
		}
		if params.Category != "endpoint" {
			t.Errorf("category = %q, want %q", params.Category, "endpoint")
		}
		if params.Title != "Endpoint enrolled" {
			t.Errorf("title = %q, want %q", params.Title, "Endpoint enrolled")
		}
		if params.Resource != "endpoint" {
			t.Errorf("resource = %q, want %q", params.Resource, "endpoint")
		}
		if params.ResourceID != "ep-123" {
			t.Errorf("resource_id = %q, want %q", params.ResourceID, "ep-123")
		}
		if params.Status != "unread" {
			t.Errorf("status = %q, want %q", params.Status, "unread")
		}
		if params.ID == "" {
			t.Error("expected non-empty alert ID")
		}
	})

	t.Run("non-matching event returns nil", func(t *testing.T) {
		t.Parallel()

		sub := makeSub() // empty cache

		event := domain.DomainEvent{
			ID:       "EVT002",
			Type:     "patch.applied",
			TenantID: tenantID,
		}

		params := sub.buildAlertParams(event)
		if params != nil {
			t.Fatal("expected nil params for non-matching event")
		}
	})

	t.Run("alert event type skipped (loop guard)", func(t *testing.T) {
		t.Parallel()

		sub := makeSub(struct {
			tenantID  string
			eventType string
			rule      *cachedRule
		}{
			tenantID:  tenantID,
			eventType: "alert.created",
			rule: &cachedRule{
				ID:       ruleID,
				TenantID: tenantID,
				Severity: "critical",
				Category: "alert",
			},
		})

		event := domain.DomainEvent{
			ID:       "EVT003",
			Type:     "alert.created",
			TenantID: tenantID,
		}

		params := sub.buildAlertParams(event)
		if params != nil {
			t.Fatal("expected nil params for alert event type (loop guard)")
		}
	})

	t.Run("alert_rule event type skipped (loop guard)", func(t *testing.T) {
		t.Parallel()

		sub := makeSub(struct {
			tenantID  string
			eventType string
			rule      *cachedRule
		}{
			tenantID:  tenantID,
			eventType: "alert_rule.created",
			rule: &cachedRule{
				ID:       ruleID,
				TenantID: tenantID,
				Severity: "info",
				Category: "system",
			},
		})

		event := domain.DomainEvent{
			ID:       "EVT004",
			Type:     "alert_rule.created",
			TenantID: tenantID,
		}

		params := sub.buildAlertParams(event)
		if params != nil {
			t.Fatal("expected nil params for alert_rule event type (loop guard)")
		}
	})
}

func TestAlertSubscriber_Backfill_Validation(t *testing.T) {
	t.Parallel()

	s := &AlertSubscriber{log: slog.Default()}

	t.Run("missing rule id returns error", func(t *testing.T) {
		t.Parallel()
		_, err := s.Backfill(context.Background(), BackfillRule{
			TenantID:  "11111111-1111-1111-1111-111111111111",
			EventType: "endpoint.enrolled",
		}, 7*24*time.Hour)
		if err == nil {
			t.Fatal("expected error for missing rule id")
		}
	})

	t.Run("alert.* event type is skipped (loop guard)", func(t *testing.T) {
		t.Parallel()
		// Pool is nil, but Backfill must return before touching it for alert.* types.
		n, err := s.Backfill(context.Background(), BackfillRule{
			ID:        "22222222-2222-2222-2222-222222222222",
			TenantID:  "11111111-1111-1111-1111-111111111111",
			EventType: "alert.created",
		}, 7*24*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Fatalf("expected 0 inserts, got %d", n)
		}
	})

	t.Run("alert_rule.* event type is skipped (loop guard)", func(t *testing.T) {
		t.Parallel()
		n, err := s.Backfill(context.Background(), BackfillRule{
			ID:        "22222222-2222-2222-2222-222222222222",
			TenantID:  "11111111-1111-1111-1111-111111111111",
			EventType: "alert_rule.created",
		}, 7*24*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 0 {
			t.Fatalf("expected 0 inserts, got %d", n)
		}
	})
}

func TestBuildAlertParamsForRule_Idempotency(t *testing.T) {
	t.Parallel()

	// buildAlertParamsForRule must be deterministic with respect to inputs
	// that determine dedup (event_id + created_at). This guards the contract
	// relied on by the alerts unique index (event_id, created_at).
	rule := &cachedRule{
		ID:                  "22222222-2222-2222-2222-222222222222",
		TenantID:            "11111111-1111-1111-1111-111111111111",
		Severity:            "info",
		Category:            "agents",
		TitleTemplate:       "t",
		DescriptionTemplate: "d",
	}
	ts := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	event := domain.DomainEvent{
		ID:        "EVT-DEDUP",
		Type:      "endpoint.enrolled",
		TenantID:  rule.TenantID,
		Timestamp: ts,
		Payload:   map[string]any{"hostname": "h1"},
	}
	a := buildAlertParamsForRule(event, rule)
	b := buildAlertParamsForRule(event, rule)
	if a == nil || b == nil {
		t.Fatal("expected non-nil params")
	}
	if a.EventID != b.EventID || !a.CreatedAt.Time.Equal(b.CreatedAt.Time) {
		t.Fatalf("event_id/created_at must be stable for dedup; got a=(%s,%v) b=(%s,%v)",
			a.EventID, a.CreatedAt.Time, b.EventID, b.CreatedAt.Time)
	}
}

func TestToPayloadMap(t *testing.T) {
	t.Parallel()

	t.Run("with map", func(t *testing.T) {
		t.Parallel()

		input := map[string]any{"key": "value", "count": float64(42)}
		result := toPayloadMap(input)
		if result == nil {
			t.Fatal("expected non-nil map")
		}
		if result["key"] != "value" {
			t.Errorf("key = %v, want %q", result["key"], "value")
		}
	})

	t.Run("with nil", func(t *testing.T) {
		t.Parallel()

		result := toPayloadMap(nil)
		if result != nil {
			t.Fatal("expected nil for nil input")
		}
	})

	t.Run("with struct", func(t *testing.T) {
		t.Parallel()

		type payload struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		input := payload{Name: "test", Count: 5}
		result := toPayloadMap(input)
		if result == nil {
			t.Fatal("expected non-nil map")
		}
		if result["name"] != "test" {
			t.Errorf("name = %v, want %q", result["name"], "test")
		}
		if result["count"] != float64(5) {
			t.Errorf("count = %v, want %v", result["count"], float64(5))
		}
	})
}
