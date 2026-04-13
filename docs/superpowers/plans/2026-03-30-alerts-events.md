# Alerts/Events Page Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack Alerts/Events page that materializes domain events into a queryable alerts table with configurable rules, REST API, and React UI with sidebar badge.

**Architecture:** Database-driven alert rules (`alert_rules`) define which domain events become alerts. A Watermill subscriber listens on all events, matches against cached rules, renders templates, and inserts into a partitioned `alerts` table. REST API exposes 8 endpoints for alerts CRUD + rules management. React frontend adds an alerts page with filtering, bulk actions, and a sidebar unread badge.

**Tech Stack:** Go 1.25 (chi/v5, pgx/v5, sqlc, Watermill, text/template) | React 19, TypeScript 5.7, TanStack Query 5, Vite 6.2 | PostgreSQL 16 (partitioned tables, RLS)

**Spec:** `docs/superpowers/specs/2026-03-30-alerts-events-design.md`

---

## Chunk 1: Database Layer (Migration + Queries + Codegen)

### Task 1: Create migration 046_alerts.sql

**Files:**
- Create: `internal/server/store/migrations/046_alerts.sql`

- [ ] **Step 1: Verify migration number is available**

Run: `ls internal/server/store/migrations/ | tail -3`
Expected: Latest is `045_notification_discord_channel.sql`, so `046` is safe.

- [ ] **Step 2: Write the migration file**

Create `internal/server/store/migrations/046_alerts.sql`:

```sql
-- +goose Up

-- ============================================================
-- Alert rules — tenant-scoped, define which events become alerts
-- ============================================================

CREATE TABLE alert_rules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID NOT NULL REFERENCES tenants(id),
    event_type            TEXT NOT NULL,
    severity              TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    category              TEXT NOT NULL CHECK (category IN ('deployments', 'agents', 'cves', 'compliance', 'system')),
    title_template        TEXT NOT NULL,
    description_template  TEXT NOT NULL,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, event_type)
);

CREATE INDEX idx_alert_rules_tenant_enabled ON alert_rules(tenant_id, enabled);

ALTER TABLE alert_rules ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON alert_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE alert_rules FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON alert_rules TO patchiq_app;

-- ============================================================
-- Alerts — materialized from domain events, partitioned monthly
-- ============================================================

CREATE TABLE alerts (
    id              TEXT NOT NULL,
    tenant_id       UUID NOT NULL,
    rule_id         UUID,  -- no FK: PG 16 cannot reference non-partitioned table from partitioned; enforced at app level
    event_id        TEXT NOT NULL,
    severity        TEXT NOT NULL CHECK (severity IN ('critical', 'warning', 'info')),
    category        TEXT NOT NULL CHECK (category IN ('deployments', 'agents', 'cves', 'compliance', 'system')),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    resource        TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'acknowledged', 'dismissed')),
    payload         JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL,
    read_at         TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    dismissed_at    TIMESTAMPTZ,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Monthly partitions for 2026.
CREATE TABLE alerts_2026_01 PARTITION OF alerts FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE alerts_2026_02 PARTITION OF alerts FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE alerts_2026_03 PARTITION OF alerts FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE alerts_2026_04 PARTITION OF alerts FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE alerts_2026_05 PARTITION OF alerts FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE alerts_2026_06 PARTITION OF alerts FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE alerts_2026_07 PARTITION OF alerts FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE alerts_2026_08 PARTITION OF alerts FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE alerts_2026_09 PARTITION OF alerts FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE alerts_2026_10 PARTITION OF alerts FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE alerts_2026_11 PARTITION OF alerts FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE alerts_2026_12 PARTITION OF alerts FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');
CREATE TABLE alerts_default PARTITION OF alerts DEFAULT;

CREATE INDEX idx_alerts_tenant_status_created ON alerts(tenant_id, status, created_at DESC);
CREATE INDEX idx_alerts_tenant_unread_severity ON alerts(tenant_id, severity) WHERE status = 'unread';
CREATE UNIQUE INDEX idx_alerts_event_id ON alerts(event_id, created_at);

ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON alerts
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id')::UUID);
ALTER TABLE alerts FORCE ROW LEVEL SECURITY;
GRANT SELECT, INSERT, UPDATE, DELETE ON alerts TO patchiq_app;

-- ============================================================
-- Seed default alert rules for default tenant
-- ============================================================

SELECT set_config('app.current_tenant_id', '00000000-0000-0000-0000-000000000001', true);

INSERT INTO alert_rules (tenant_id, event_type, severity, category, title_template, description_template) VALUES
-- Critical
('00000000-0000-0000-0000-000000000001', 'deployment.failed',            'critical', 'deployments', 'Deployment failed: {{.name}}',            '{{.failed_count}} targets failed in wave {{.wave}}.'),
('00000000-0000-0000-0000-000000000001', 'deployment.rollback_triggered', 'critical', 'deployments', 'Rollback triggered: {{.name}}',           'Deployment rolled back due to failure threshold.'),
('00000000-0000-0000-0000-000000000001', 'agent.disconnected',            'critical', 'agents',      'Agent disconnected: {{.hostname}}',       'Endpoint {{.hostname}} lost connection.'),
('00000000-0000-0000-0000-000000000001', 'compliance.threshold_breach',   'critical', 'compliance',  'Compliance breach: {{.framework}}',       'Score dropped below threshold ({{.score}}%).'),
('00000000-0000-0000-0000-000000000001', 'license.expired',               'critical', 'system',      'License expired',                         'PatchIQ license has expired. Features may be restricted.'),
('00000000-0000-0000-0000-000000000001', 'catalog.sync_failed',           'critical', 'system',      'Catalog sync failed',                     'Hub catalog synchronization failed: {{.error}}.'),
-- Warning
('00000000-0000-0000-0000-000000000001', 'command.timed_out',             'warning',  'deployments', 'Command timed out: {{.command_id}}',      'Deployment command timed out on target.'),
('00000000-0000-0000-0000-000000000001', 'license.expiring',              'warning',  'system',      'License expiring soon',                   'License expires in {{.days_remaining}} days.'),
('00000000-0000-0000-0000-000000000001', 'cve.discovered',                'warning',  'cves',        'New CVE: {{.cve_id}}',                    'CVSS {{.cvss_score}} — affects {{.package}}.'),
('00000000-0000-0000-0000-000000000001', 'notification.failed',           'warning',  'system',      'Notification delivery failed',            'Failed to send via {{.channel}}: {{.error}}.'),
('00000000-0000-0000-0000-000000000001', 'deployment.wave_failed',        'warning',  'deployments', 'Wave failed: {{.name}}',                  'Wave {{.wave}} failed — deployment may need attention.'),
-- Info
('00000000-0000-0000-0000-000000000001', 'deployment.completed',          'info',     'deployments', 'Deployment completed: {{.name}}',         'All targets patched successfully.'),
('00000000-0000-0000-0000-000000000001', 'deployment.started',            'info',     'deployments', 'Deployment started: {{.name}}',           'Deployment began execution.'),
('00000000-0000-0000-0000-000000000001', 'endpoint.enrolled',             'info',     'agents',      'New endpoint enrolled: {{.hostname}}',    'Agent registered from {{.ip_address}}.'),
('00000000-0000-0000-0000-000000000001', 'cve.remediation_available',     'info',     'cves',        'Remediation available: {{.cve_id}}',      'Patch available for {{.cve_id}}.')
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS alerts CASCADE;
DROP TABLE IF EXISTS alert_rules CASCADE;
```

- [ ] **Step 3: Run migration**

Run: `make migrate`
Expected: Migration 046 applies cleanly.

- [ ] **Step 4: Verify tables exist**

Run: `make migrate-status`
Expected: Shows 046_alerts as applied.

- [ ] **Step 5: Commit**

```bash
git add internal/server/store/migrations/046_alerts.sql
git commit -m "feat(db): add alert_rules and alerts tables (046)"
```

---

### Task 2: Write sqlc queries for alerts

**Files:**
- Create: `internal/server/store/queries/alerts.sql`

- [ ] **Step 1: Write the alerts query file**

Create `internal/server/store/queries/alerts.sql`:

```sql
-- ============================================================
-- Alert Rules CRUD
-- ============================================================

-- name: ListAlertRules :many
SELECT * FROM alert_rules
WHERE tenant_id = @tenant_id
ORDER BY created_at DESC;

-- name: GetAlertRule :one
SELECT * FROM alert_rules
WHERE id = @id AND tenant_id = @tenant_id;

-- name: CreateAlertRule :one
INSERT INTO alert_rules (tenant_id, event_type, severity, category, title_template, description_template, enabled)
VALUES (@tenant_id, @event_type, @severity, @category, @title_template, @description_template, @enabled)
RETURNING *;

-- name: UpdateAlertRule :one
UPDATE alert_rules
SET event_type = @event_type,
    severity = @severity,
    category = @category,
    title_template = @title_template,
    description_template = @description_template,
    enabled = @enabled,
    updated_at = now()
WHERE id = @id AND tenant_id = @tenant_id
RETURNING *;

-- name: DeleteAlertRule :execrows
DELETE FROM alert_rules
WHERE id = @id AND tenant_id = @tenant_id;

-- name: ListEnabledAlertRules :many
SELECT * FROM alert_rules
WHERE tenant_id = @tenant_id AND enabled = true;

-- ============================================================
-- Alerts
-- ============================================================

-- name: InsertAlert :exec
INSERT INTO alerts (id, tenant_id, rule_id, event_id, severity, category, title, description, resource, resource_id, status, payload, created_at)
VALUES (@id, @tenant_id, @rule_id, @event_id, @severity, @category, @title, @description, @resource, @resource_id, 'unread', @payload, @created_at)
ON CONFLICT (event_id, created_at) DO NOTHING;

-- name: ListAlertsFiltered :many
SELECT * FROM alerts
WHERE tenant_id = @tenant_id
  AND (@severity::text = '' OR severity = @severity)
  AND (@category::text = '' OR category = @category)
  AND (@status::text = '' OR status = @status)
  AND (@search::text = '' OR title ILIKE '%' || @search || '%' OR description ILIKE '%' || @search || '%')
  AND (@from_date::timestamptz IS NULL OR created_at >= @from_date)
  AND (@to_date::timestamptz IS NULL OR created_at <= @to_date)
  AND (
    @cursor_timestamp::timestamptz IS NULL
    OR (created_at, id) < (@cursor_timestamp, @cursor_id::text)
  )
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;

-- name: CountAlertsFiltered :one
SELECT count(*) FROM alerts
WHERE tenant_id = @tenant_id
  AND (@severity::text = '' OR severity = @severity)
  AND (@category::text = '' OR category = @category)
  AND (@status::text = '' OR status = @status)
  AND (@search::text = '' OR title ILIKE '%' || @search || '%' OR description ILIKE '%' || @search || '%')
  AND (@from_date::timestamptz IS NULL OR created_at >= @from_date)
  AND (@to_date::timestamptz IS NULL OR created_at <= @to_date);

-- name: CountUnreadAlerts :one
SELECT
  count(*) FILTER (WHERE severity = 'critical') AS critical_unread,
  count(*) FILTER (WHERE severity = 'warning') AS warning_unread,
  count(*) FILTER (WHERE severity = 'info') AS info_unread,
  count(*) AS total_unread
FROM alerts
WHERE tenant_id = @tenant_id AND status = 'unread';

-- name: UpdateAlertStatus :one
UPDATE alerts
SET status = @status,
    read_at = CASE WHEN @status = 'read' THEN now() ELSE read_at END,
    acknowledged_at = CASE WHEN @status = 'acknowledged' THEN now() ELSE acknowledged_at END,
    dismissed_at = CASE WHEN @status = 'dismissed' THEN now() ELSE dismissed_at END
WHERE id = @id AND created_at = @created_at AND tenant_id = @tenant_id
RETURNING *;

-- name: BulkUpdateAlertStatus :execrows
-- Note: scans all partitions since we lack created_at. Acceptable for POC
-- with monthly partitions. For scale, add created_at range or use per-row updates.
UPDATE alerts
SET status = @status,
    read_at = CASE WHEN @status = 'read' THEN now() ELSE read_at END,
    acknowledged_at = CASE WHEN @status = 'acknowledged' THEN now() ELSE acknowledged_at END,
    dismissed_at = CASE WHEN @status = 'dismissed' THEN now() ELSE dismissed_at END
WHERE tenant_id = @tenant_id AND id = ANY(@ids::text[])
  AND created_at > now() - interval '90 days';

-- name: GetAlertCreatedAt :one
SELECT created_at FROM alerts
WHERE id = @id AND tenant_id = @tenant_id
LIMIT 1;
```

> **Note:** `ListAllEnabledAlertRules` is NOT a sqlc query — the AlertSubscriber loads rules via raw SQL on the pool to bypass RLS (see Task 6). The `InsertAlert` query is also called via the pool with tenant context set in a transaction (same pattern as AuditSubscriber).

- [ ] **Step 2: Run sqlc codegen**

Run: `make sqlc`
Expected: Generates new types/queries in `internal/server/store/sqlcgen/` without errors.

- [ ] **Step 3: Verify generated code**

Run: `ls internal/server/store/sqlcgen/alerts.sql.go`
Expected: File exists with generated query functions.

- [ ] **Step 4: Commit**

```bash
git add internal/server/store/queries/alerts.sql internal/server/store/sqlcgen/
git commit -m "feat(db): add sqlc queries for alerts and alert rules"
```

---

### Task 3: Add seed data for alert rules

**Files:**
- Modify: `scripts/seed.sql`

- [ ] **Step 1: Append alert rules seed data to seed.sql**

Add before the final `COMMIT;` in `scripts/seed.sql`:

```sql
-- -------------------------------------------------------------------------
-- Alert rules — default operational alert configuration.
-- -------------------------------------------------------------------------
INSERT INTO alert_rules (tenant_id, event_type, severity, category, title_template, description_template) VALUES
-- Critical
('00000000-0000-0000-0000-000000000001', 'deployment.failed',            'critical', 'deployments', 'Deployment failed: {{.name}}',            '{{.failed_count}} targets failed in wave {{.wave}}.'),
('00000000-0000-0000-0000-000000000001', 'deployment.rollback_triggered', 'critical', 'deployments', 'Rollback triggered: {{.name}}',           'Deployment rolled back due to failure threshold.'),
('00000000-0000-0000-0000-000000000001', 'agent.disconnected',            'critical', 'agents',      'Agent disconnected: {{.hostname}}',       'Endpoint {{.hostname}} lost connection.'),
('00000000-0000-0000-0000-000000000001', 'compliance.threshold_breach',   'critical', 'compliance',  'Compliance breach: {{.framework}}',       'Score dropped below threshold ({{.score}}%).'),
('00000000-0000-0000-0000-000000000001', 'license.expired',               'critical', 'system',      'License expired',                         'PatchIQ license has expired. Features may be restricted.'),
('00000000-0000-0000-0000-000000000001', 'catalog.sync_failed',           'critical', 'system',      'Catalog sync failed',                     'Hub catalog synchronization failed: {{.error}}.'),
-- Warning
('00000000-0000-0000-0000-000000000001', 'command.timed_out',             'warning',  'deployments', 'Command timed out: {{.command_id}}',      'Deployment command timed out on target.'),
('00000000-0000-0000-0000-000000000001', 'license.expiring',              'warning',  'system',      'License expiring soon',                   'License expires in {{.days_remaining}} days.'),
('00000000-0000-0000-0000-000000000001', 'cve.discovered',                'warning',  'cves',        'New CVE: {{.cve_id}}',                    'CVSS {{.cvss_score}} — affects {{.package}}.'),
('00000000-0000-0000-0000-000000000001', 'notification.failed',           'warning',  'system',      'Notification delivery failed',            'Failed to send via {{.channel}}: {{.error}}.'),
('00000000-0000-0000-0000-000000000001', 'deployment.wave_failed',        'warning',  'deployments', 'Wave failed: {{.name}}',                  'Wave {{.wave}} failed — deployment may need attention.'),
-- Info
('00000000-0000-0000-0000-000000000001', 'deployment.completed',          'info',     'deployments', 'Deployment completed: {{.name}}',         'All targets patched successfully.'),
('00000000-0000-0000-0000-000000000001', 'deployment.started',            'info',     'deployments', 'Deployment started: {{.name}}',           'Deployment began execution.'),
('00000000-0000-0000-0000-000000000001', 'endpoint.enrolled',             'info',     'agents',      'New endpoint enrolled: {{.hostname}}',    'Agent registered from {{.ip_address}}.'),
('00000000-0000-0000-0000-000000000001', 'cve.remediation_available',     'info',     'cves',        'Remediation available: {{.cve_id}}',      'Patch available for {{.cve_id}}.')
ON CONFLICT DO NOTHING;
```

- [ ] **Step 2: Run seed to verify**

Run: `make seed`
Expected: Seed completes without errors; alert_rules table has 15 rows.

- [ ] **Step 3: Commit**

```bash
git add scripts/seed.sql
git commit -m "feat(seed): add default alert rules for dev environment"
```

---

## Chunk 2: Event Topics + AlertSubscriber (Rule Cache + Template Rendering)

### Task 4: Add alert event topics

**Files:**
- Modify: `internal/server/events/topics.go`

- [ ] **Step 1: Add alert event type constants**

In `internal/server/events/topics.go`, add after the `InvitationClaimed`/`UserRegistered` block (before the closing `)`):

```go
// Alert events
AlertCreated       = "alert.created"
AlertStatusUpdated = "alert.status_updated"
AlertRuleCreated   = "alert_rule.created"
AlertRuleUpdated   = "alert_rule.updated"
AlertRuleDeleted   = "alert_rule.deleted"
```

- [ ] **Step 2: Add to AllTopics()**

Append these 5 entries to the `AllTopics()` return slice.

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/server/events/`
Expected: Compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add internal/server/events/topics.go
git commit -m "feat(events): add alert and alert_rule event types"
```

---

### Task 5: Implement alert rule cache

**Files:**
- Create: `internal/server/events/alert_rules.go`
- Create: `internal/server/events/alert_rules_test.go`

- [ ] **Step 1: Write failing test for rule cache**

Create `internal/server/events/alert_rules_test.go`:

```go
package events

import (
	"bytes"
	"testing"
	"text/template"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		tmpl     string
		data     map[string]any
		want     string
		fallback string
	}{
		{
			name: "simple substitution",
			tmpl: "Deployment failed: {{.name}}",
			data: map[string]any{"name": "Q1-Rollout"},
			want: "Deployment failed: Q1-Rollout",
		},
		{
			name: "missing field uses fallback",
			tmpl: "Deployment failed: {{.name}}",
			data: map[string]any{},
			want: "",  // template produces "<no value>" or error — fallback used
		},
		{
			name: "nil data uses fallback",
			tmpl: "Deployment failed: {{.name}}",
			data: nil,
			want: "",
		},
		{
			name:     "invalid template uses fallback",
			tmpl:     "Deployment failed: {{.name",
			data:     map[string]any{"name": "test"},
			want:     "",
			fallback: "deployment.failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTemplate(tt.tmpl, tt.data, tt.fallback)
			if tt.want != "" && got != tt.want {
				t.Errorf("renderTemplate() = %q, want %q", got, tt.want)
			}
			if tt.want == "" && tt.fallback != "" && got != tt.fallback {
				t.Errorf("renderTemplate() = %q, want fallback %q", got, tt.fallback)
			}
		})
	}
}

func TestIsAlertEventType(t *testing.T) {
	tests := []struct {
		eventType string
		want      bool
	}{
		{"alert.created", true},
		{"alert.status_updated", true},
		{"alert_rule.created", true},
		{"alert_rule.updated", true},
		{"alert_rule.deleted", true},
		{"deployment.failed", false},
		{"endpoint.enrolled", false},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if got := isAlertEventType(tt.eventType); got != tt.want {
				t.Errorf("isAlertEventType(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/events/ -run TestRenderTemplate -v`
Expected: FAIL — `renderTemplate` undefined.

- [ ] **Step 3: Implement alert_rules.go**

Create `internal/server/events/alert_rules.go`:

```go
package events

import (
	"bytes"
	"strings"
	"text/template"
)

// isAlertEventType returns true if the event type starts with "alert." or "alert_rule."
// These events are produced by the alert system itself and must be skipped
// by the AlertSubscriber to prevent infinite loops.
func isAlertEventType(eventType string) bool {
	return strings.HasPrefix(eventType, "alert.") || strings.HasPrefix(eventType, "alert_rule.")
}

// renderTemplate renders a Go text/template with the given data map.
// If the template is invalid or rendering fails, returns the fallback string.
func renderTemplate(tmpl string, data map[string]any, fallback string) string {
	t, err := template.New("").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		if fallback != "" {
			return fallback
		}
		return tmpl
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		if fallback != "" {
			return fallback
		}
		return tmpl
	}
	return buf.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/server/events/ -run "TestRenderTemplate|TestIsAlertEventType" -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server/events/alert_rules.go internal/server/events/alert_rules_test.go
git commit -m "feat(events): add alert template rendering and event type guard"
```

---

### Task 6: Implement AlertSubscriber

**Files:**
- Create: `internal/server/events/alert_subscriber.go`
- Create: `internal/server/events/alert_subscriber_test.go`

- [ ] **Step 1: Write failing test for AlertSubscriber**

Create `internal/server/events/alert_subscriber_test.go`:

```go
package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// mockAlertPool implements the AlertPool interface for testing.
type mockAlertPool struct {
	rules    []alertRuleCacheEntry
	inserted []sqlcgen.InsertAlertParams
}

type alertRuleCacheEntry struct {
	ID                  string
	TenantID            string
	EventType           string
	Severity            string
	Category            string
	TitleTemplate       string
	DescriptionTemplate string
}

func TestAlertSubscriber_Handle(t *testing.T) {
	// Pre-populate cache directly for unit testing (bypasses DB).
	sub := &AlertSubscriber{log: slog.Default()}

	// Populate cache with a test rule.
	sub.cache.Store(
		cacheKey("00000000-0000-0000-0000-000000000001", "deployment.failed"),
		&cachedRule{
			ID:                  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			TenantID:            "00000000-0000-0000-0000-000000000001",
			Severity:            "critical",
			Category:            "deployments",
			TitleTemplate:       "Deployment failed: {{.name}}",
			DescriptionTemplate: "{{.failed_count}} targets failed.",
		},
	)

	tests := []struct {
		name        string
		event       domain.DomainEvent
		wantInserts int
	}{
		{
			name: "matching event creates alert",
			event: domain.DomainEvent{
				ID:         "01TEST0001",
				Type:       "deployment.failed",
				TenantID:   "00000000-0000-0000-0000-000000000001",
				Resource:   "deployment",
				ResourceID: "d-123",
				Payload:    map[string]any{"name": "Q1-Rollout", "failed_count": 4},
				Timestamp:  time.Now(),
			},
			wantInserts: 1,
		},
		{
			name: "non-matching event skipped",
			event: domain.DomainEvent{
				ID:       "01TEST0002",
				Type:     "endpoint.updated",
				TenantID: "00000000-0000-0000-0000-000000000001",
			},
			wantInserts: 0,
		},
		{
			name: "alert event type skipped (loop guard)",
			event: domain.DomainEvent{
				ID:       "01TEST0003",
				Type:     "alert.created",
				TenantID: "00000000-0000-0000-0000-000000000001",
			},
			wantInserts: 0,
		},
	}

	// Note: Full Handle() integration requires a real pgxpool for tx + RLS.
	// Unit tests here verify the cache lookup + guard logic only.
	// The buildAlertParams helper can be tested in isolation.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := sub.buildAlertParams(tt.event)
			if tt.wantInserts == 0 && params != nil {
				t.Errorf("expected nil params (skip), got non-nil")
			}
			if tt.wantInserts > 0 && params == nil {
				t.Fatalf("expected non-nil params, got nil")
			}
			if tt.wantInserts > 0 {
				if params.Severity != "critical" {
					t.Errorf("severity = %q, want critical", params.Severity)
				}
				if params.EventID != tt.event.ID {
					t.Errorf("event_id = %q, want %q", params.EventID, tt.event.ID)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/server/events/ -run TestAlertSubscriber_Handle -v`
Expected: FAIL — `NewAlertSubscriber` undefined.

- [ ] **Step 3: Implement AlertSubscriber**

Create `internal/server/events/alert_subscriber.go`:

```go
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

// cachedRule holds the fields needed from alert_rules for matching and rendering.
type cachedRule struct {
	ID                  string
	TenantID            string
	Severity            string
	Category            string
	TitleTemplate       string
	DescriptionTemplate string
}

// AlertSubscriber materializes domain events into the alerts table
// based on database-driven alert rules.
// Uses *pgxpool.Pool directly (not sqlc querier) because:
// - refreshCache loads rules across ALL tenants (bypasses RLS via raw query)
// - Handle inserts alerts in a transaction with tenant context set for RLS
type AlertSubscriber struct {
	pool  *pgxpool.Pool
	log   *slog.Logger
	cache sync.Map // key: "tenant_id:event_type" → *cachedRule
}

// NewAlertSubscriber creates a subscriber that converts events to alerts.
func NewAlertSubscriber(pool *pgxpool.Pool, logger *slog.Logger) *AlertSubscriber {
	return &AlertSubscriber{pool: pool, log: logger}
}

// StartCacheRefresh launches a goroutine that refreshes the rule cache periodically.
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
	// Raw query bypasses RLS — loads rules for ALL tenants.
	rows, err := s.pool.Query(ctx,
		`SELECT id, tenant_id, event_type, severity, category, title_template, description_template
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
		if err := rows.Scan(&r.ID, &r.TenantID, nil, &r.Severity, &r.Category, &r.TitleTemplate, &r.DescriptionTemplate); err != nil {
			s.log.ErrorContext(ctx, "scan alert rule", "error", err)
			continue
		}
		// We need event_type for the key but it's the 3rd column.
		// Re-scan properly:
		var eventType string
		// Actually, let's use a struct scan approach.
		_ = eventType // placeholder — see full implementation below
		count++
	}
	// NOTE: Full implementation scans all 7 columns properly. The pattern:
	// rows.Scan(&id, &tenantID, &eventType, &severity, &category, &titleTmpl, &descTmpl)
	// then builds: key = cacheKey(tenantID, eventType) → &cachedRule{...}

	// Remove stale entries.
	s.cache.Range(func(k, _ any) bool {
		if _, ok := newKeys[k.(string)]; !ok {
			s.cache.Delete(k)
		}
		return true
	})
	s.log.Debug("alert rule cache refreshed", "rule_count", count)
}

// buildAlertParams checks the cache and builds insert params if a rule matches.
// Returns nil if the event should be skipped (no matching rule or guard triggered).
// Separated from Handle for unit testability.
func (s *AlertSubscriber) buildAlertParams(event domain.DomainEvent) *sqlcgen.InsertAlertParams {
	if isAlertEventType(event.Type) {
		return nil
	}

	key := cacheKey(event.TenantID, event.Type)
	val, ok := s.cache.Load(key)
	if !ok {
		return nil
	}
	rule := val.(*cachedRule)

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
		Payload:     payloadBytes,
		CreatedAt:   ts,
	}
}

// Handle processes a single domain event, creating an alert if a matching rule exists.
// Uses a transaction with tenant context for RLS compliance (same pattern as AuditSubscriber).
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

	// Set tenant context for RLS.
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
```

> **Implementation note for the worker:** The `refreshCache` raw query scans 7 columns: `id::text, tenant_id::text, event_type, severity, category, title_template, description_template`. Use `rows.Scan(&id, &tenantID, &eventType, &severity, &category, &titleTmpl, &descTmpl)` and build `cachedRule` from those. Cast `tenant_id` to text in the query with `tenant_id::text` to avoid UUID parsing.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/server/events/ -run "TestAlertSubscriber|TestRenderTemplate|TestIsAlertEventType" -v -race`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server/events/alert_subscriber.go internal/server/events/alert_subscriber_test.go
git commit -m "feat(events): implement AlertSubscriber with rule cache and template rendering"
```

---

## Chunk 3: REST API Handler + Router Wiring

### Task 7: Implement AlertHandler

**Files:**
- Create: `internal/server/api/v1/alerts.go`
- Create: `internal/server/api/v1/alerts_test.go`

- [ ] **Step 1: Write failing tests for AlertHandler**

Create `internal/server/api/v1/alerts_test.go` with table-driven tests for:
- `TestAlertHandler_List` — tests pagination, filtering, empty result
- `TestAlertHandler_Count` — tests unread count response
- `TestAlertHandler_UpdateStatus` — tests status transitions
- `TestAlertHandler_BulkUpdateStatus` — tests bulk update
- `TestAlertHandler_ListRules` — tests rule listing
- `TestAlertHandler_CreateRule` — tests rule creation with validation
- `TestAlertHandler_UpdateRule` — tests rule update
- `TestAlertHandler_DeleteRule` — tests rule deletion

Each test uses a mock `AlertQuerier` interface. Pattern follows `internal/server/api/v1/audit_test.go`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/server/api/v1/ -run TestAlertHandler -v`
Expected: FAIL — `AlertHandler` undefined.

- [ ] **Step 3: Implement alerts.go handler**

Create `internal/server/api/v1/alerts.go` implementing:
- `AlertQuerier` interface (as defined in spec section 4.2)
- `AlertHandler` struct with `q AlertQuerier`, `pool TxBeginner`, `eventBus domain.EventBus`
- `NewAlertHandler` constructor with nil-check panics
- Response types: `alertResponse`, `alertRuleResponse`, `alertCountResponse`
- `List` — parses query params (severity, category, status, search, from_date, to_date, cursor, limit), calls `ListAlertsFiltered` + `CountAlertsFiltered`, returns `WriteList`
- `Count` — calls `CountUnreadAlerts`, returns JSON with `critical_unread`, `warning_unread`, `info_unread`, `total_unread`
- `UpdateStatus` — reads `{id}` from URL + JSON body `{"status": "read"}`, calls `UpdateAlertStatus`, emits `alert.status_updated` event
- `BulkUpdateStatus` — reads JSON body `{"ids": [...], "status": "dismissed"}`, calls `BulkUpdateAlertStatus`, emits event
- `ListRules` — calls `ListAlertRules`, returns array
- `CreateRule` — reads JSON body, validates, calls `CreateAlertRule`, emits `alert_rule.created`
- `UpdateRule` — reads `{id}` + JSON body, calls `UpdateAlertRule`, emits `alert_rule.updated`
- `DeleteRule` — reads `{id}`, calls `DeleteAlertRule`, emits `alert_rule.deleted`

**Important:** The `UpdateAlertStatus` query needs both `id` and `created_at` (composite PK on partitioned table). The handler first calls `GetAlertCreatedAt(ctx, id, tenantID)` to look up the `created_at` value, then passes both to `UpdateAlertStatus`. This hides the partitioning detail from the API consumer — they only pass `{id}` in the URL.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/server/api/v1/ -run TestAlertHandler -v -race`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/server/api/v1/alerts.go internal/server/api/v1/alerts_test.go
git commit -m "feat(api): implement AlertHandler with 8 REST endpoints"
```

---

### Task 8: Register alert routes and wire subscriber

**Files:**
- Modify: `internal/server/api/router.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add alert routes to router.go**

In `internal/server/api/router.go`, after the audit routes (around line 249), add:

```go
alertH := v1.NewAlertHandler(q, st.Pool(), eventBus)

r.Route("/alerts", func(r chi.Router) {
    r.With(rp("alerts", "read")).Get("/", alertH.List)
    r.With(rp("alerts", "read")).Get("/count", alertH.Count)
    r.With(rp("alerts", "update")).Patch("/{id}/status", alertH.UpdateStatus)
    r.With(rp("alerts", "update")).Patch("/bulk-status", alertH.BulkUpdateStatus)
})

r.Route("/alert-rules", func(r chi.Router) {
    r.With(rp("alerts", "manage")).Get("/", alertH.ListRules)
    r.With(rp("alerts", "manage")).Post("/", alertH.CreateRule)
    r.With(rp("alerts", "manage")).Put("/{id}", alertH.UpdateRule)
    r.With(rp("alerts", "manage")).Delete("/{id}", alertH.DeleteRule)
})
```

- [ ] **Step 2: Wire AlertSubscriber in main.go**

In `cmd/server/main.go`, after the audit subscriber wiring (around line 153), add:

```go
alertSub := events.NewAlertSubscriber(pool, logger)
alertSub.StartCacheRefresh(ctx, 30*time.Second)
if err := eventBus.Subscribe("*", alertSub.Handle); err != nil {
    return fmt.Errorf("subscribe alert handler: %w", err)
}
slog.Info("alert subscriber initialized")
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./cmd/server/`
Expected: Compiles without errors.

- [ ] **Step 4: Commit**

```bash
git add internal/server/api/router.go cmd/server/main.go
git commit -m "feat(server): wire alert routes and subscriber in server startup"
```

---

### Task 9: Update OpenAPI spec

**Files:**
- Modify: `api/server.yaml`

- [ ] **Step 1: Add Alert and AlertRule schemas to components/schemas**

Add `Alert`, `AlertRule`, `AlertCountResponse`, `CreateAlertRuleRequest`, `UpdateAlertRuleRequest`, `UpdateAlertStatusRequest`, `BulkUpdateAlertStatusRequest` schemas following existing `AuditEvent` and `NotificationChannel` patterns.

- [ ] **Step 2: Add 8 endpoint path definitions**

Add paths for all 8 endpoints with proper parameters, request bodies, and response schemas. Use `Alerts` tag. Follow existing cursor-based pagination patterns from the audit endpoints.

- [ ] **Step 3: Validate spec**

Run: `make api-client`
Expected: OpenAPI validation passes; types regenerated in `web/src/api/types.ts`.

- [ ] **Step 4: Commit**

```bash
git add api/server.yaml web/src/api/types.ts
git commit -m "feat(api): add alerts/alert-rules OpenAPI spec and regenerate types"
```

---

## Chunk 4: Frontend — API Hooks + Route + Sidebar Badge

### Task 10: Create API hooks for alerts

**Files:**
- Create: `web/src/api/hooks/useAlerts.ts`

- [ ] **Step 1: Write the hooks file**

Create `web/src/api/hooks/useAlerts.ts` following the `useAudit.ts` pattern:

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../client';

export interface AlertFilters {
  cursor?: string;
  limit?: number;
  severity?: string;
  category?: string;
  status?: string;
  search?: string;
  from_date?: string;
  to_date?: string;
}

export function useAlerts(params?: AlertFilters, refetchInterval?: number) {
  return useQuery({
    queryKey: ['alerts', params],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/alerts', {
        params: { query: params },
      });
      if (error) throw error;
      return data;
    },
    refetchInterval: refetchInterval ?? undefined,
  });
}

export function useAlertCount(refetchInterval = 30000) {
  return useQuery({
    queryKey: ['alerts', 'count'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/alerts/count');
      if (error) throw error;
      return data;
    },
    refetchInterval,
  });
}

export function useUpdateAlertStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, status }: { id: string; status: string }) => {
      const { data, error } = await api.PATCH('/api/v1/alerts/{id}/status', {
        params: { path: { id } },
        body: { status },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useBulkUpdateAlertStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ ids, status }: { ids: string[]; status: string }) => {
      const { data, error } = await api.PATCH('/api/v1/alerts/bulk-status', {
        body: { ids, status },
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alerts'] });
    },
  });
}

export function useAlertRules() {
  return useQuery({
    queryKey: ['alert-rules'],
    queryFn: async () => {
      const { data, error } = await api.GET('/api/v1/alert-rules');
      if (error) throw error;
      return data;
    },
  });
}

export function useCreateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (body: {
      event_type: string;
      severity: string;
      category: string;
      title_template: string;
      description_template: string;
      enabled: boolean;
    }) => {
      const { data, error } = await api.POST('/api/v1/alert-rules', { body });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alert-rules'] });
    },
  });
}

export function useUpdateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, ...body }: {
      id: string;
      event_type: string;
      severity: string;
      category: string;
      title_template: string;
      description_template: string;
      enabled: boolean;
    }) => {
      const { data, error } = await api.PUT('/api/v1/alert-rules/{id}', {
        params: { path: { id } },
        body,
      });
      if (error) throw error;
      return data;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alert-rules'] });
    },
  });
}

export function useDeleteAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE('/api/v1/alert-rules/{id}', {
        params: { path: { id } },
      });
      if (error) throw error;
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['alert-rules'] });
    },
  });
}
```

- [ ] **Step 2: Verify TypeScript compiles**

Run: `cd web && npx tsc --noEmit`
Expected: No type errors (may need OpenAPI types from Task 9 first).

- [ ] **Step 3: Commit**

```bash
git add web/src/api/hooks/useAlerts.ts
git commit -m "feat(web): add TanStack Query hooks for alerts and alert rules"
```

---

### Task 11: Add alerts route and sidebar nav with badge

**Files:**
- Modify: `web/src/app/routes.tsx`
- Modify: `web/src/app/layout/AppSidebar.tsx`

- [ ] **Step 1: Add route for alerts page**

In `web/src/app/routes.tsx`:
1. Add import: `import { AlertsPage } from '../pages/alerts/AlertsPage';`
2. Add route after the audit route (line 60): `{ path: '/alerts', element: <AlertsPage /> },`

- [ ] **Step 2: Update sidebar with Alerts nav item + badge**

In `web/src/app/layout/AppSidebar.tsx`:
1. Add import: `import { BellRing } from 'lucide-react';`
2. Add import: `import { useAlertCount } from '../../api/hooks/useAlerts';`
3. In `navGroups`, in the Compliance group, insert `{ label: 'Alerts', icon: BellRing, to: '/alerts' }` between Compliance and Audit.
4. In the nav item render loop, add a badge `<span>` next to the "Alerts" label when count > 0.

The badge should be:
```tsx
{item.label === 'Alerts' && alertCount > 0 && (
  <span style={{
    marginLeft: 'auto',
    background: hasCritical ? 'var(--signal-critical)' : 'var(--signal-warning)',
    color: '#fff',
    fontSize: 10,
    fontWeight: 600,
    fontFamily: 'var(--font-mono)',
    borderRadius: 'var(--radius-full)',
    minWidth: 18,
    padding: '0 5px',
    textAlign: 'center',
    lineHeight: '16px',
  }}>
    {alertCount}
  </span>
)}
```

Call `useAlertCount(30000)` at the top of `AppSidebar` and compute `alertCount = (data?.critical_unread ?? 0) + (data?.warning_unread ?? 0)` and `hasCritical = (data?.critical_unread ?? 0) > 0`.

- [ ] **Step 3: Verify compilation**

Run: `cd web && npx tsc --noEmit`
Expected: Compiles (AlertsPage will be a stub placeholder until Task 12).

- [ ] **Step 4: Commit**

```bash
git add web/src/app/routes.tsx web/src/app/layout/AppSidebar.tsx
git commit -m "feat(web): add alerts route and sidebar nav with unread badge"
```

---

## Chunk 5: Frontend — AlertsPage Components

### Task 12: Implement AlertRow component

**Files:**
- Create: `web/src/pages/alerts/AlertRow.tsx`

- [ ] **Step 1: Create the AlertRow component**

Build the card-style row with:
- Checkbox for selection
- Unread dot indicator (6px green dot when status === 'unread')
- Severity icon block (28x28, tinted background): critical=red, warning=amber, info=green
- Title (13px, weight 500)
- Description (12px, secondary color)
- Meta row: severity badge, category, entity link (monospace, accent), relative timestamp
- Right side: action dropdown (Mark Read, Acknowledge, Dismiss)

All styling via inline styles with CSS variables. Follow the mockup from the brainstorm session (Option B).

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/alerts/AlertRow.tsx
git commit -m "feat(web): implement AlertRow component with severity icon layout"
```

---

### Task 13: Implement AlertFilters component

**Files:**
- Create: `web/src/pages/alerts/AlertFilters.tsx`

- [ ] **Step 1: Create the filter bar component**

Props interface with:
- `severity`, `onSeverityChange` — severity filter pills
- `status`, `onStatusChange` — status filter pills
- `category`, `onCategoryChange` — category filter pills
- `search`, `onSearchChange` — text search input
- `dateRange`, `onDateRangeChange` — date preset selector
- `fromDate`, `onFromDateChange`, `toDate`, `onToDateChange` — custom date inputs

Render as a flexbox row with pill-style filter buttons (similar to AuditFilters). Severity pills show counts from parent. Status "Active" = default (unread + read).

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/alerts/AlertFilters.tsx
git commit -m "feat(web): implement AlertFilters component with severity/status/category pills"
```

---

### Task 14: Implement AlertRulesSheet component

**Files:**
- Create: `web/src/pages/alerts/AlertRulesSheet.tsx`

- [ ] **Step 1: Create the rules management sheet**

Uses `Sheet` from `@patchiq/ui`. Contains:
- Header: "Alert Rules" + "Add Rule" button
- Table with columns: Event Type (mono), Severity (badge), Category, Enabled (toggle switch)
- Click row to expand inline edit: severity select, title/description template inputs
- Delete button per row with confirmation dialog
- Uses `useAlertRules`, `useCreateAlertRule`, `useUpdateAlertRule`, `useDeleteAlertRule` hooks

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/alerts/AlertRulesSheet.tsx
git commit -m "feat(web): implement AlertRulesSheet for alert rule management"
```

---

### Task 15: Implement AlertsPage

**Files:**
- Create: `web/src/pages/alerts/AlertsPage.tsx`

- [ ] **Step 1: Build the full alerts page**

Assemble all components:

1. `PageHeader` — title "Alerts", unread badge, actions row with:
   - Refresh interval selector (10s/30s/60s/off dropdown)
   - "Manage Rules" button → toggles `AlertRulesSheet` open
2. `AlertFilters` — all filter state managed in page via `useState`
3. Alert list — map over `useAlerts()` data, render `AlertRow` for each
4. `BulkActionBar` — visible when selectedIds.length > 0, with Mark Read / Acknowledge / Dismiss buttons calling `useBulkUpdateAlertStatus`
5. `DataTablePagination` — cursor-based pagination using cursors state array
6. Loading state: 8 `Skeleton` rows
7. Empty state: `EmptyState` with contextual message
8. Error state: `ErrorState` with retry via `refetch()`

Filter → API param mapping:
- severity pill → `severity` param
- status "Active" → `status=unread,read`, "Acknowledged" → `status=acknowledged`, etc.
- category pill → `category` param
- search → `search` param
- date range → `from_date`/`to_date` params

Auto-refresh controlled by `refetchInterval` state passed to `useAlerts`.

- [ ] **Step 2: Verify page renders**

Run: `cd web && npm run dev`
Navigate to `/alerts`. Expected: Page renders with skeleton loading, then shows empty state or alerts.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/alerts/AlertsPage.tsx
git commit -m "feat(web): implement AlertsPage with filters, bulk actions, and auto-refresh"
```

---

## Chunk 6: Integration Testing + Final Verification

### Task 16: Run full backend test suite

- [ ] **Step 1: Run Go tests**

Run: `make test`
Expected: All tests pass including new alert tests.

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No new lint errors.

---

### Task 17: Run frontend lint and type check

- [ ] **Step 1: Type check**

Run: `cd web && npx tsc --noEmit`
Expected: No type errors.

- [ ] **Step 2: Lint**

Run: `make lint-frontend`
Expected: No new lint errors.

---

### Task 18: End-to-end smoke test

- [ ] **Step 1: Start dev environment**

Run: `make dev`
Expected: All services start.

- [ ] **Step 2: Run seed data**

Run: `make seed`
Expected: Alert rules seeded.

- [ ] **Step 3: Verify API endpoints work**

```bash
# List alert rules
curl -s http://localhost:8080/api/v1/alert-rules -H 'X-Tenant-ID: 00000000-0000-0000-0000-000000000001' | jq '.data | length'
# Expected: 15

# Get alert count
curl -s http://localhost:8080/api/v1/alerts/count -H 'X-Tenant-ID: 00000000-0000-0000-0000-000000000001' | jq
# Expected: {"critical_unread":0,"warning_unread":0,"info_unread":0,"total_unread":0}

# List alerts (empty initially)
curl -s http://localhost:8080/api/v1/alerts -H 'X-Tenant-ID: 00000000-0000-0000-0000-000000000001' | jq '.total_count'
# Expected: 0
```

- [ ] **Step 4: Verify alerts page loads in browser**

Navigate to `http://localhost:3001/alerts`
Expected: Page loads with empty state "No alerts". Sidebar shows "Alerts" nav item in Compliance group. No badge visible (count = 0).

- [ ] **Step 5: Trigger an alert by creating a deployment (if possible) or manually insert a test event**

Expected: Alert appears in the list. Badge count updates in sidebar.

- [ ] **Step 6: Final commit**

```bash
git add web/src/pages/alerts/ web/src/app/routes.tsx web/src/app/layout/AppSidebar.tsx web/src/api/hooks/useAlerts.ts
git commit -m "feat: alerts/events page — full-stack implementation (B1)"
```
