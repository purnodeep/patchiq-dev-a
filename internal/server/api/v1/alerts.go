package v1

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/events"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// AlertQuerier defines the sqlc queries needed by AlertHandler.
type AlertQuerier interface {
	ListAlertsFiltered(ctx context.Context, arg sqlcgen.ListAlertsFilteredParams) ([]sqlcgen.Alert, error)
	CountAlertsFiltered(ctx context.Context, arg sqlcgen.CountAlertsFilteredParams) (int64, error)
	CountUnreadAlerts(ctx context.Context, arg sqlcgen.CountUnreadAlertsParams) (sqlcgen.CountUnreadAlertsRow, error)
	GetAlertCreatedAt(ctx context.Context, arg sqlcgen.GetAlertCreatedAtParams) (pgtype.Timestamptz, error)
	UpdateAlertStatus(ctx context.Context, arg sqlcgen.UpdateAlertStatusParams) (sqlcgen.Alert, error)
	BulkUpdateAlertStatus(ctx context.Context, arg sqlcgen.BulkUpdateAlertStatusParams) (int64, error)
	ListAlertRules(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.AlertRule, error)
	GetAlertRule(ctx context.Context, arg sqlcgen.GetAlertRuleParams) (sqlcgen.AlertRule, error)
	CreateAlertRule(ctx context.Context, arg sqlcgen.CreateAlertRuleParams) (sqlcgen.AlertRule, error)
	UpdateAlertRule(ctx context.Context, arg sqlcgen.UpdateAlertRuleParams) (sqlcgen.AlertRule, error)
	DeleteAlertRule(ctx context.Context, arg sqlcgen.DeleteAlertRuleParams) (int64, error)
}

// AlertBackfiller materializes historical audit_events into alerts for a
// newly created or newly enabled rule. Satisfied by *events.AlertSubscriber.
type AlertBackfiller interface {
	Backfill(ctx context.Context, rule events.BackfillRule, window time.Duration) (int, error)
}

// defaultBackfillWindow is the lookback used when the handler triggers a backfill.
const defaultBackfillWindow = 365 * 24 * time.Hour

// AlertHandler serves alert REST API endpoints.
type AlertHandler struct {
	q          AlertQuerier
	pool       TxBeginner
	eventBus   domain.EventBus
	backfiller AlertBackfiller
}

// NewAlertHandler creates an AlertHandler.
// backfiller may be nil in tests; when nil, rule create/update will not
// trigger a backfill (the production constructor always supplies it).
func NewAlertHandler(q AlertQuerier, pool TxBeginner, eventBus domain.EventBus, backfiller AlertBackfiller) *AlertHandler {
	if q == nil {
		panic("alerts: NewAlertHandler called with nil querier")
	}
	return &AlertHandler{q: q, pool: pool, eventBus: eventBus, backfiller: backfiller}
}

// alertResponse is the JSON representation of an alert.
type alertResponse struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	RuleID         string          `json:"rule_id"`
	EventID        string          `json:"event_id"`
	Severity       string          `json:"severity"`
	Category       string          `json:"category"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Resource       string          `json:"resource"`
	ResourceID     string          `json:"resource_id"`
	Status         string          `json:"status"`
	Payload        json.RawMessage `json:"payload"`
	CreatedAt      string          `json:"created_at"`
	ReadAt         *string         `json:"read_at"`
	AcknowledgedAt *string         `json:"acknowledged_at"`
	DismissedAt    *string         `json:"dismissed_at"`
}

// alertCountResponse is the JSON representation unnamed alert counts.
type alertCountResponse struct {
	CriticalUnread int64 `json:"critical_unread"`
	WarningUnread  int64 `json:"warning_unread"`
	InfoUnread     int64 `json:"info_unread"`
	TotalUnread    int64 `json:"total_unread"`
}

// alertRuleResponse is the JSON representation of an alert rule.
type alertRuleResponse struct {
	ID                  string `json:"id"`
	TenantID            string `json:"tenant_id"`
	EventType           string `json:"event_type"`
	Severity            string `json:"severity"`
	Category            string `json:"category"`
	TitleTemplate       string `json:"title_template"`
	DescriptionTemplate string `json:"description_template"`
	Enabled             bool   `json:"enabled"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

func toAlertResponse(a sqlcgen.Alert) alertResponse {
	payload := json.RawMessage(a.Payload)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	resp := alertResponse{
		ID:          a.ID,
		TenantID:    uuidToString(a.TenantID),
		RuleID:      uuidToString(a.RuleID),
		EventID:     a.EventID,
		Severity:    a.Severity,
		Category:    a.Category,
		Title:       a.Title,
		Description: a.Description,
		Resource:    a.Resource,
		ResourceID:  a.ResourceID,
		Status:      a.Status,
		Payload:     payload,
		CreatedAt:   a.CreatedAt.Time.Format(time.RFC3339),
	}
	if a.ReadAt.Valid {
		s := a.ReadAt.Time.Format(time.RFC3339)
		resp.ReadAt = &s
	}
	if a.AcknowledgedAt.Valid {
		s := a.AcknowledgedAt.Time.Format(time.RFC3339)
		resp.AcknowledgedAt = &s
	}
	if a.DismissedAt.Valid {
		s := a.DismissedAt.Time.Format(time.RFC3339)
		resp.DismissedAt = &s
	}
	return resp
}

func toAlertRuleResponse(r sqlcgen.AlertRule) alertRuleResponse {
	return alertRuleResponse{
		ID:                  uuidToString(r.ID),
		TenantID:            uuidToString(r.TenantID),
		EventType:           r.EventType,
		Severity:            r.Severity,
		Category:            r.Category,
		TitleTemplate:       r.TitleTemplate,
		DescriptionTemplate: r.DescriptionTemplate,
		Enabled:             r.Enabled,
		CreatedAt:           r.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:           r.UpdatedAt.Time.Format(time.RFC3339),
	}
}

// List handles GET /api/v1/alerts with pagination and filters.
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	q := r.URL.Query()

	// Parse date filters.
	var fromDate, toDate pgtype.Timestamptz
	if fd := q.Get("from_date"); fd != "" {
		t, parseErr := time.Parse(time.RFC3339, fd)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_DATE", "invalid from_date: expected RFC3339 format")
			return
		}
		fromDate = pgtype.Timestamptz{Time: t, Valid: true}
	}
	if td := q.Get("to_date"); td != "" {
		t, parseErr := time.Parse(time.RFC3339, td)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_DATE", "invalid to_date: expected RFC3339 format")
			return
		}
		toDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	// Parse cursor.
	cursorTime, cursorID, err := DecodeCursor(q.Get("cursor"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "invalid pagination cursor")
		return
	}

	var cursorTS pgtype.Timestamptz
	if !cursorTime.IsZero() {
		cursorTS = pgtype.Timestamptz{Time: cursorTime, Valid: true}
	}

	limit := ParseLimit(q.Get("limit"))

	params := sqlcgen.ListAlertsFilteredParams{
		TenantID:        tid,
		Severity:        q.Get("severity"),
		Category:        q.Get("category"),
		Status:          q.Get("status"),
		FromDate:        fromDate,
		ToDate:          toDate,
		Search:          q.Get("search"),
		CursorTimestamp: cursorTS,
		CursorID:        cursorID,
		PageLimit:       limit,
	}

	alerts, err := h.q.ListAlertsFiltered(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list alerts", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list alerts")
		return
	}

	countParams := sqlcgen.CountAlertsFilteredParams{
		TenantID: tid,
		Severity: params.Severity,
		Category: params.Category,
		Status:   params.Status,
		FromDate: params.FromDate,
		ToDate:   params.ToDate,
		Search:   params.Search,
	}
	total, err := h.q.CountAlertsFiltered(ctx, countParams)
	if err != nil {
		slog.ErrorContext(ctx, "count alerts", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count alerts")
		return
	}

	items := make([]alertResponse, len(alerts))
	for i, a := range alerts {
		items[i] = toAlertResponse(a)
	}

	var nextCursor string
	if len(alerts) == int(limit) {
		last := alerts[len(alerts)-1]
		nextCursor = EncodeCursor(last.CreatedAt.Time, last.ID)
	}

	WriteList(w, items, nextCursor, total)
}

// Count handles GET /api/v1/alerts/count — returns unread alert counts by severity.
func (h *AlertHandler) Count(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	q := r.URL.Query()
	var fromDate, toDate pgtype.Timestamptz
	if fd := q.Get("from_date"); fd != "" {
		t, parseErr := time.Parse(time.RFC3339, fd)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_DATE", "invalid from_date: expected RFC3339 format")
			return
		}
		fromDate = pgtype.Timestamptz{Time: t, Valid: true}
	}
	if td := q.Get("to_date"); td != "" {
		t, parseErr := time.Parse(time.RFC3339, td)
		if parseErr != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_DATE", "invalid to_date: expected RFC3339 format")
			return
		}
		toDate = pgtype.Timestamptz{Time: t, Valid: true}
	}

	counts, err := h.q.CountUnreadAlerts(ctx, sqlcgen.CountUnreadAlertsParams{
		TenantID: tid,
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		slog.ErrorContext(ctx, "count unread alerts", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count unread alerts")
		return
	}

	WriteJSON(w, http.StatusOK, alertCountResponse{
		CriticalUnread: counts.CriticalUnread,
		WarningUnread:  counts.WarningUnread,
		InfoUnread:     counts.InfoUnread,
		TotalUnread:    counts.TotalUnread,
	})
}

// updateStatusRequest is the JSON body for PATCH /api/v1/alerts/{id}/status.
type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PATCH /api/v1/alerts/{id}/status.
func (h *AlertHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	alertID := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	if body.Status == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "status is required")
		return
	}

	// Look up created_at (needed for partitioned PK).
	createdAt, err := h.q.GetAlertCreatedAt(ctx, sqlcgen.GetAlertCreatedAtParams{
		ID:       alertID,
		TenantID: tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
			return
		}
		slog.ErrorContext(ctx, "get alert created_at", "alert_id", alertID, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to look up alert")
		return
	}

	updated, err := h.q.UpdateAlertStatus(ctx, sqlcgen.UpdateAlertStatusParams{
		Status:    body.Status,
		ID:        alertID,
		CreatedAt: createdAt,
		TenantID:  tid,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
			return
		}
		slog.ErrorContext(ctx, "update alert status", "alert_id", alertID, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update alert status")
		return
	}

	emitEvent(ctx, h.eventBus, events.AlertStatusUpdated, "alert", alertID, tenantID, map[string]string{
		"status": body.Status,
	})

	WriteJSON(w, http.StatusOK, toAlertResponse(updated))
}

// bulkUpdateStatusRequest is the JSON body for PATCH /api/v1/alerts/bulk-status.
type bulkUpdateStatusRequest struct {
	IDs    []string `json:"ids"`
	Status string   `json:"status"`
}

// BulkUpdateStatus handles PATCH /api/v1/alerts/bulk-status.
func (h *AlertHandler) BulkUpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body bulkUpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	if body.Status == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "status is required")
		return
	}
	if len(body.IDs) == 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "ids is required and must not be empty")
		return
	}

	count, err := h.q.BulkUpdateAlertStatus(ctx, sqlcgen.BulkUpdateAlertStatusParams{
		Status:   body.Status,
		TenantID: tid,
		Ids:      body.IDs,
	})
	if err != nil {
		slog.ErrorContext(ctx, "bulk update alert status", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to bulk update alert status")
		return
	}

	emitEvent(ctx, h.eventBus, events.AlertStatusUpdated, "alert", "bulk", tenantID, map[string]any{
		"ids":    body.IDs,
		"status": body.Status,
		"count":  count,
	})

	WriteJSON(w, http.StatusOK, map[string]int64{"updated_count": count})
}

// ListRules handles GET /api/v1/alert-rules.
func (h *AlertHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	rules, err := h.q.ListAlertRules(ctx, tid)
	if err != nil {
		slog.ErrorContext(ctx, "list alert rules", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list alert rules")
		return
	}

	items := make([]alertRuleResponse, len(rules))
	for i, r := range rules {
		items[i] = toAlertRuleResponse(r)
	}

	WriteJSON(w, http.StatusOK, items)
}

// createAlertRuleRequest is the JSON body for POST /api/v1/alert-rules.
type createAlertRuleRequest struct {
	EventType           string `json:"event_type"`
	Severity            string `json:"severity"`
	Category            string `json:"category"`
	TitleTemplate       string `json:"title_template"`
	DescriptionTemplate string `json:"description_template"`
	Enabled             bool   `json:"enabled"`
}

// CreateRule handles POST /api/v1/alert-rules.
func (h *AlertHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	var body createAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	if body.EventType == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "event_type is required")
		return
	}

	rule, err := h.q.CreateAlertRule(ctx, sqlcgen.CreateAlertRuleParams{
		TenantID:            tid,
		EventType:           body.EventType,
		Severity:            body.Severity,
		Category:            body.Category,
		TitleTemplate:       body.TitleTemplate,
		DescriptionTemplate: body.DescriptionTemplate,
		Enabled:             body.Enabled,
	})
	if err != nil {
		slog.ErrorContext(ctx, "create alert rule", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create alert rule")
		return
	}

	ruleID := uuidToString(rule.ID)
	emitEvent(ctx, h.eventBus, events.AlertRuleCreated, "alert_rule", ruleID, tenantID, nil)

	// Backfill historical audit_events that match this newly created rule.
	// Synchronous for now (capped + narrow window); see TODO on Backfill for
	// moving this to River if it becomes a latency problem.
	if rule.Enabled && h.backfiller != nil {
		if _, err := h.backfiller.Backfill(ctx, events.BackfillRule{
			ID:                  ruleID,
			TenantID:            tenantID,
			EventType:           rule.EventType,
			Severity:            rule.Severity,
			Category:            rule.Category,
			TitleTemplate:       rule.TitleTemplate,
			DescriptionTemplate: rule.DescriptionTemplate,
		}, defaultBackfillWindow); err != nil {
			slog.ErrorContext(ctx, "alert rule backfill failed", "rule_id", ruleID, "tenant_id", tenantID, "error", err)
			// Non-fatal: the rule is created, streaming events will still produce alerts.
		}
	}

	WriteJSON(w, http.StatusCreated, toAlertRuleResponse(rule))
}

// updateAlertRuleRequest is the JSON body for PUT /api/v1/alert-rules/{id}.
type updateAlertRuleRequest struct {
	EventType           string `json:"event_type"`
	Severity            string `json:"severity"`
	Category            string `json:"category"`
	TitleTemplate       string `json:"title_template"`
	DescriptionTemplate string `json:"description_template"`
	Enabled             bool   `json:"enabled"`
}

// UpdateRule handles PUT /api/v1/alert-rules/{id}.
func (h *AlertHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	ruleIDStr := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ruleID, err := scanUUID(ruleIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid alert rule ID")
		return
	}

	var body updateAlertRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_BODY", "invalid JSON body")
		return
	}

	// Fetch current rule to detect a disabled->enabled transition; backfill
	// should only run when the rule is being turned on (not on every edit).
	prev, prevErr := h.q.GetAlertRule(ctx, sqlcgen.GetAlertRuleParams{
		ID:       ruleID,
		TenantID: tid,
	})
	if prevErr != nil {
		if isNotFound(prevErr) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "alert rule not found")
			return
		}
		slog.ErrorContext(ctx, "get alert rule for update", "rule_id", ruleIDStr, "tenant_id", tenantID, "error", prevErr)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to look up alert rule")
		return
	}

	rule, err := h.q.UpdateAlertRule(ctx, sqlcgen.UpdateAlertRuleParams{
		ID:                  ruleID,
		TenantID:            tid,
		EventType:           body.EventType,
		Severity:            body.Severity,
		Category:            body.Category,
		TitleTemplate:       body.TitleTemplate,
		DescriptionTemplate: body.DescriptionTemplate,
		Enabled:             body.Enabled,
	})
	if err != nil {
		if isNotFound(err) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "alert rule not found")
			return
		}
		slog.ErrorContext(ctx, "update alert rule", "rule_id", ruleIDStr, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update alert rule")
		return
	}

	emitEvent(ctx, h.eventBus, events.AlertRuleUpdated, "alert_rule", ruleIDStr, tenantID, nil)

	// Backfill only on disabled -> enabled transition.
	if !prev.Enabled && rule.Enabled && h.backfiller != nil {
		if _, err := h.backfiller.Backfill(ctx, events.BackfillRule{
			ID:                  ruleIDStr,
			TenantID:            tenantID,
			EventType:           rule.EventType,
			Severity:            rule.Severity,
			Category:            rule.Category,
			TitleTemplate:       rule.TitleTemplate,
			DescriptionTemplate: rule.DescriptionTemplate,
		}, defaultBackfillWindow); err != nil {
			slog.ErrorContext(ctx, "alert rule backfill failed", "rule_id", ruleIDStr, "tenant_id", tenantID, "error", err)
		}
	}

	WriteJSON(w, http.StatusOK, toAlertRuleResponse(rule))
}

// DeleteRule handles DELETE /api/v1/alert-rules/{id}.
func (h *AlertHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)
	ruleIDStr := chi.URLParam(r, "id")

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	ruleID, err := scanUUID(ruleIDStr)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_ID", "invalid alert rule ID")
		return
	}

	count, err := h.q.DeleteAlertRule(ctx, sqlcgen.DeleteAlertRuleParams{
		ID:       ruleID,
		TenantID: tid,
	})
	if err != nil {
		slog.ErrorContext(ctx, "delete alert rule", "rule_id", ruleIDStr, "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete alert rule")
		return
	}

	if count == 0 {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "alert rule not found")
		return
	}

	emitEvent(ctx, h.eventBus, events.AlertRuleDeleted, "alert_rule", ruleIDStr, tenantID, nil)

	w.WriteHeader(http.StatusNoContent)
}
