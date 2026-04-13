package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// cachedRule holds fields needed from alert_rules for matching and rendering.
type cachedRule struct {
	ID                  string
	TenantID            string
	Severity            string
	Category            string
	TitleTemplate       string
	DescriptionTemplate string
}

// AlertSubscriber materializes domain events into the alerts table.
type AlertSubscriber struct {
	pool  *pgxpool.Pool
	log   *slog.Logger
	cache sync.Map // key: "tenant_id:event_type" -> *cachedRule
}

// NewAlertSubscriber creates a subscriber that creates alerts from domain events.
func NewAlertSubscriber(pool *pgxpool.Pool, logger *slog.Logger) *AlertSubscriber {
	return &AlertSubscriber{pool: pool, log: logger}
}

// StartCacheRefresh launches a background goroutine that refreshes the rule cache.
func (s *AlertSubscriber) StartCacheRefresh(ctx context.Context, interval time.Duration) {
	s.refreshCache(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.refreshCache(ctx)
			}
		}
	}()
}

func (s *AlertSubscriber) refreshCache(ctx context.Context) {
	// Raw query bypasses RLS -- loads rules for ALL tenants.
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, tenant_id::text, event_type, severity, category, title_template, description_template
		 FROM alert_rules WHERE enabled = true`)
	if err != nil {
		s.log.ErrorContext(ctx, "refresh alert rule cache", "error", err)
		return
	}
	defer rows.Close()

	newKeys := make(map[string]struct{})
	count := 0
	for rows.Next() {
		var r cachedRule
		var eventType string
		if err := rows.Scan(&r.ID, &r.TenantID, &eventType, &r.Severity, &r.Category, &r.TitleTemplate, &r.DescriptionTemplate); err != nil {
			s.log.ErrorContext(ctx, "scan alert rule", "error", err)
			continue
		}
		key := cacheKey(r.TenantID, eventType)
		s.cache.Store(key, &r)
		newKeys[key] = struct{}{}
		count++
	}

	// Remove stale entries.
	s.cache.Range(func(k, _ any) bool {
		key, _ := k.(string)
		if _, ok := newKeys[key]; !ok {
			s.cache.Delete(k)
		}
		return true
	})
	s.log.Debug("alert rule cache refreshed", "rule_count", count)
}

// buildAlertParams checks cache and builds insert params if a rule matches.
// Returns nil if event should be skipped. Separated from Handle for testability.
func (s *AlertSubscriber) buildAlertParams(event domain.DomainEvent) *sqlcgen.InsertAlertParams {
	if isAlertEventType(event.Type) {
		return nil
	}
	if event.TenantID == "" {
		return nil
	}

	key := cacheKey(event.TenantID, event.Type)
	val, ok := s.cache.Load(key)
	if !ok {
		return nil
	}
	rule, _ := val.(*cachedRule)
	return buildAlertParamsForRule(event, rule)
}

// buildAlertParamsForRule builds insert params for a specific rule, bypassing the cache.
// Used by backfill, which operates before/independently of cache warming.
// Returns nil if the event should be skipped (alert loop guard).
func buildAlertParamsForRule(event domain.DomainEvent, rule *cachedRule) *sqlcgen.InsertAlertParams {
	if rule == nil {
		return nil
	}
	if isAlertEventType(event.Type) {
		return nil
	}

	payloadMap := toPayloadMap(event.Payload)
	title := renderTemplate(rule.TitleTemplate, payloadMap, event.Type)
	desc := renderTemplate(rule.DescriptionTemplate, payloadMap, "")

	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		payloadBytes = []byte("{}")
	}

	var tenantUUID, ruleUUID pgtype.UUID
	_ = tenantUUID.Scan(event.TenantID)
	_ = ruleUUID.Scan(rule.ID)

	var ts pgtype.Timestamptz
	ts.Time = event.Timestamp
	ts.Valid = true

	return &sqlcgen.InsertAlertParams{
		ID:          domain.NewEventID(),
		TenantID:    tenantUUID,
		RuleID:      ruleUUID,
		EventID:     event.ID,
		Severity:    rule.Severity,
		Category:    rule.Category,
		Title:       title,
		Description: desc,
		Resource:    event.Resource,
		ResourceID:  event.ResourceID,
		Status:      "unread",
		Payload:     payloadBytes,
		CreatedAt:   ts,
	}
}

// BackfillRule is the minimal rule shape needed by Backfill.
// It mirrors the fields in cachedRule but is exported so callers
// (e.g. the alert-rules HTTP handler) can pass freshly created rules
// without depending on the internal cache being warm.
type BackfillRule struct {
	ID                  string
	TenantID            string
	EventType           string
	Severity            string
	Category            string
	TitleTemplate       string
	DescriptionTemplate string
}

// backfillMaxRows caps the number of historical events materialized per rule.
// Prevents runaway inserts if a tenant has many matching events in the window.
const backfillMaxRows = 1000

// Backfill scans audit_events for events matching the given rule within the
// last `window` duration and inserts corresponding rows into the alerts table.
// It reuses buildAlertParamsForRule, so template rendering and loop-guard
// behaviour stay identical to the streaming Handle() path.
//
// Dedup is handled by InsertAlert's ON CONFLICT (event_id, created_at) DO NOTHING
// backed by the unique index idx_alerts_event_id, so re-running is idempotent.
//
// TODO(PIQ): move to a River job if windows/volumes grow beyond synchronous
// execution in the rule create/update HTTP handlers.
func (s *AlertSubscriber) Backfill(ctx context.Context, rule BackfillRule, window time.Duration) (int, error) {
	if rule.ID == "" || rule.TenantID == "" || rule.EventType == "" {
		return 0, fmt.Errorf("backfill: rule id, tenant_id, event_type are required")
	}
	if isAlertEventType(rule.EventType) {
		// Don't backfill alert.*/alert_rule.* — would create a loop.
		return 0, nil
	}
	if window <= 0 {
		window = 7 * 24 * time.Hour
	}
	since := time.Now().Add(-window)

	s.log.InfoContext(ctx, "alert backfill started",
		"rule_id", rule.ID, "tenant_id", rule.TenantID,
		"event_type", rule.EventType, "window", window.String())

	// Raw query bypasses RLS (same pattern as refreshCache) so we can scan
	// audit_events by tenant without needing to set app.current_tenant_id on
	// a connection that may be shared with the pool.
	rows, err := s.pool.Query(ctx,
		`SELECT id, type, tenant_id::text, COALESCE(actor_id,''), COALESCE(actor_type,''),
		        COALESCE(resource,''), COALESCE(resource_id,''), COALESCE(action,''),
		        COALESCE(payload, '{}'::jsonb), timestamp
		   FROM audit_events
		  WHERE tenant_id = $1::uuid AND type = $2 AND timestamp >= $3
		  ORDER BY timestamp DESC
		  LIMIT $4`,
		rule.TenantID, rule.EventType, since, backfillMaxRows+1)
	if err != nil {
		return 0, fmt.Errorf("query audit_events for backfill: %w", err)
	}

	cr := &cachedRule{
		ID:                  rule.ID,
		TenantID:            rule.TenantID,
		Severity:            rule.Severity,
		Category:            rule.Category,
		TitleTemplate:       rule.TitleTemplate,
		DescriptionTemplate: rule.DescriptionTemplate,
	}

	var events []domain.DomainEvent
	for rows.Next() {
		var ev domain.DomainEvent
		var payloadBytes []byte
		if err := rows.Scan(&ev.ID, &ev.Type, &ev.TenantID, &ev.ActorID, &ev.ActorType,
			&ev.Resource, &ev.ResourceID, &ev.Action, &payloadBytes, &ev.Timestamp); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan audit_event row: %w", err)
		}
		// Decode payload JSONB into map[string]any so toPayloadMap renders templates.
		var pm map[string]any
		if len(payloadBytes) > 0 {
			if err := json.Unmarshal(payloadBytes, &pm); err == nil {
				ev.Payload = pm
			}
		}
		events = append(events, ev)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate audit_events: %w", err)
	}

	capped := false
	if len(events) > backfillMaxRows {
		events = events[:backfillMaxRows]
		capped = true
	}

	if len(events) == 0 {
		s.log.InfoContext(ctx, "alert backfill complete (no matching events)",
			"rule_id", rule.ID, "event_type", rule.EventType)
		return 0, nil
	}

	// Single tx for the whole backfill: cheaper and atomic per-rule.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin backfill tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)", rule.TenantID,
	); err != nil {
		return 0, fmt.Errorf("set tenant context for backfill: %w", err)
	}

	q := sqlcgen.New(tx)
	inserted := 0
	for _, ev := range events {
		params := buildAlertParamsForRule(ev, cr)
		if params == nil {
			continue
		}
		if err := q.InsertAlert(ctx, *params); err != nil {
			return 0, fmt.Errorf("insert backfilled alert for event %s: %w", ev.ID, err)
		}
		inserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit backfill tx: %w", err)
	}

	s.log.InfoContext(ctx, "alert backfill complete",
		"rule_id", rule.ID, "event_type", rule.EventType,
		"scanned", len(events), "inserted", inserted, "capped", capped)
	return inserted, nil
}

// Handle processes a domain event. Same tx+RLS pattern as AuditSubscriber.
func (s *AlertSubscriber) Handle(ctx context.Context, event domain.DomainEvent) error {
	params := s.buildAlertParams(event)
	if params == nil {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin alert tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant_id', $1, true)", event.TenantID,
	); err != nil {
		return fmt.Errorf("set tenant context for alert: %w", err)
	}

	queries := sqlcgen.New(tx)
	if err := queries.InsertAlert(ctx, *params); err != nil {
		return fmt.Errorf("insert alert for event %s: %w", event.ID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit alert tx: %w", err)
	}

	s.log.DebugContext(ctx, "alert created",
		"event_id", event.ID, "event_type", event.Type, "severity", params.Severity)
	return nil
}

func cacheKey(tenantID, eventType string) string {
	return tenantID + ":" + eventType
}

// toPayloadMap converts an event payload to map[string]any for template rendering.
func toPayloadMap(payload any) map[string]any {
	if payload == nil {
		return nil
	}
	if m, ok := payload.(map[string]any); ok {
		return m
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}
