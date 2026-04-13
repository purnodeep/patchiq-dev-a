package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/shared/tenant"
)

// AuditQuerier defines the sqlc queries needed by AuditHandler.
type AuditQuerier interface {
	ListAuditEventsFiltered(ctx context.Context, arg sqlcgen.ListAuditEventsFilteredParams) ([]sqlcgen.AuditEvent, error)
	CountAuditEventsFiltered(ctx context.Context, arg sqlcgen.CountAuditEventsFilteredParams) (int64, error)
}

// AuditHandler serves audit log REST API endpoints (read-only).
type AuditHandler struct {
	q AuditQuerier
}

// NewAuditHandler creates an AuditHandler.
func NewAuditHandler(q AuditQuerier) *AuditHandler {
	if q == nil {
		panic("audit: NewAuditHandler called with nil querier")
	}
	return &AuditHandler{q: q}
}

// auditEventResponse is the JSON representation of an audit event.
type auditEventResponse struct {
	ID         string          `json:"id"`
	TenantID   string          `json:"tenant_id"`
	Type       string          `json:"type"`
	ActorID    string          `json:"actor_id"`
	ActorType  string          `json:"actor_type"`
	Resource   string          `json:"resource"`
	ResourceID string          `json:"resource_id"`
	Action     string          `json:"action"`
	Payload    json.RawMessage `json:"payload"`
	Metadata   json.RawMessage `json:"metadata"`
	Timestamp  string          `json:"timestamp"`
}

// List handles GET /api/v1/audit with pagination and filters.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
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

	params := sqlcgen.ListAuditEventsFilteredParams{
		TenantID:        tid,
		ActorID:         q.Get("actor_id"),
		ActorType:       q.Get("actor_type"),
		Resource:        q.Get("resource"),
		ResourceID:      q.Get("resource_id"),
		Action:          q.Get("action"),
		EventType:       q.Get("type"),
		ExcludeType:     q.Get("exclude_type"),
		FromDate:        fromDate,
		ToDate:          toDate,
		Search:          q.Get("search"),
		CursorTimestamp: cursorTS,
		CursorID:        cursorID,
		PageLimit:       limit,
	}

	events, err := h.q.ListAuditEventsFiltered(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "list audit events", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list audit events")
		return
	}

	countParams := sqlcgen.CountAuditEventsFilteredParams{
		TenantID:    tid,
		ActorID:     params.ActorID,
		ActorType:   params.ActorType,
		Resource:    params.Resource,
		ResourceID:  params.ResourceID,
		Action:      params.Action,
		EventType:   params.EventType,
		ExcludeType: params.ExcludeType,
		FromDate:    params.FromDate,
		ToDate:      params.ToDate,
		Search:      params.Search,
	}
	total, err := h.q.CountAuditEventsFiltered(ctx, countParams)
	if err != nil {
		slog.ErrorContext(ctx, "count audit events", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to count audit events")
		return
	}

	items := make([]auditEventResponse, len(events))
	for i, e := range events {
		payload := json.RawMessage(e.Payload)
		if len(payload) == 0 {
			payload = json.RawMessage("{}")
		}
		metadata := json.RawMessage(e.Metadata)
		if len(metadata) == 0 {
			metadata = json.RawMessage("{}")
		}
		items[i] = auditEventResponse{
			ID:         e.ID,
			TenantID:   uuidToString(e.TenantID),
			Type:       e.Type,
			ActorID:    e.ActorID,
			ActorType:  e.ActorType,
			Resource:   e.Resource,
			ResourceID: e.ResourceID,
			Action:     e.Action,
			Payload:    payload,
			Metadata:   metadata,
			Timestamp:  e.Timestamp.Time.Format(time.RFC3339),
		}
	}

	var nextCursor string
	if len(events) == int(limit) {
		last := events[len(events)-1]
		nextCursor = EncodeCursor(last.Timestamp.Time, last.ID)
	}

	WriteList(w, items, nextCursor, total)
}

// Export handles GET /api/v1/audit/export — streams audit events as CSV or NDJSON.
func (h *AuditHandler) Export(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := tenant.MustTenantID(ctx)

	tid, err := scanUUID(tenantID)
	if err != nil {
		slog.ErrorContext(ctx, "export audit: invalid tenant ID in context", "tenant_id", tenantID, "error", err)
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid tenant ID in context")
		return
	}

	q := r.URL.Query()

	// Parse format (csv or json).
	format := q.Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		WriteError(w, http.StatusBadRequest, "INVALID_FORMAT", "format must be csv or json")
		return
	}

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

	// Set response headers.
	ts := time.Now().UTC().Format("20060102T150405Z")
	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-export-%s.csv"`, ts))
	} else {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-export-%s.json"`, ts))
	}
	w.WriteHeader(http.StatusOK)

	// Write CSV header.
	if format == "csv" {
		_, _ = fmt.Fprintln(w, "id,timestamp,type,actor_id,actor_type,resource,resource_id,action,payload,metadata")
	}

	// Cursor loop — page through all results.
	const batchSize int32 = 500
	var cursorTS pgtype.Timestamptz
	var cursorID string
	var totalExported int

	for {
		params := sqlcgen.ListAuditEventsFilteredParams{
			TenantID:        tid,
			ActorID:         q.Get("actor_id"),
			ActorType:       q.Get("actor_type"),
			Resource:        q.Get("resource"),
			ResourceID:      q.Get("resource_id"),
			Action:          q.Get("action"),
			EventType:       q.Get("type"),
			ExcludeType:     q.Get("exclude_type"),
			FromDate:        fromDate,
			ToDate:          toDate,
			Search:          q.Get("search"),
			CursorTimestamp: cursorTS,
			CursorID:        cursorID,
			PageLimit:       batchSize,
		}

		events, listErr := h.q.ListAuditEventsFiltered(ctx, params)
		if listErr != nil {
			slog.ErrorContext(ctx, "export audit: list events failed", "tenant_id", tenantID, "error", listErr)
			// Headers already sent; nothing we can do but stop.
			return
		}

		for _, e := range events {
			payload := string(e.Payload)
			if payload == "" {
				payload = "{}"
			}
			metadata := string(e.Metadata)
			if metadata == "" {
				metadata = "{}"
			}

			if format == "csv" {
				_, _ = fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
					e.ID,
					e.Timestamp.Time.Format(time.RFC3339),
					e.Type,
					e.ActorID,
					e.ActorType,
					e.Resource,
					e.ResourceID,
					e.Action,
					escapeCSVField(payload),
					escapeCSVField(metadata),
				)
			} else {
				resp := auditEventResponse{
					ID:         e.ID,
					TenantID:   uuidToString(e.TenantID),
					Type:       e.Type,
					ActorID:    e.ActorID,
					ActorType:  e.ActorType,
					Resource:   e.Resource,
					ResourceID: e.ResourceID,
					Action:     e.Action,
					Payload:    json.RawMessage(payload),
					Metadata:   json.RawMessage(metadata),
					Timestamp:  e.Timestamp.Time.Format(time.RFC3339),
				}
				line, _ := json.Marshal(resp)
				_, _ = fmt.Fprintf(w, "%s\n", line)
			}
		}

		totalExported += len(events)

		// Flush after each batch.
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		// Stop when page returns fewer than batchSize results.
		if len(events) < int(batchSize) {
			break
		}

		// Advance cursor to last event's timestamp + id.
		last := events[len(events)-1]
		cursorTS = last.Timestamp
		cursorID = last.ID
	}

	slog.InfoContext(ctx, "audit export completed", "tenant_id", tenantID, "format", format, "total_exported", totalExported)
}

// escapeCSVField wraps a field in double quotes and escapes internal double quotes.
func escapeCSVField(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
