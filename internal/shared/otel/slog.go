package otel

import (
	"context"
	"io"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/shared/tenant"
	"github.com/skenzeriq/patchiq/internal/shared/user"
	"go.opentelemetry.io/otel/trace"
)

// Handler is a slog.Handler that injects trace_id, span_id, tenant_id,
// request_id, and user_id from context into every log record.
//
// Injected fields are always written at the top level of the JSON object,
// regardless of any WithGroup calls applied to the handler.
//
// Design: we maintain a root handler (no groups applied) and an explicit
// list of group names accumulated via WithGroup. On Handle, injected context
// attrs are added to the root handler via WithAttrs, then the stored groups
// are re-applied, ensuring context fields land before any group prefix.
type Handler struct {
	// root is the base JSONHandler with pre-attrs applied but NO groups.
	// Used to inject context fields at the top level.
	root slog.Handler

	// inner is the handler with the full group+attr chain, used to process
	// the original record attrs.
	inner slog.Handler

	// groups tracks group names accumulated via WithGroup, in order.
	groups []string
}

// NewHandler creates a Handler wrapping a slog.JSONHandler writing to w.
func NewHandler(w io.Writer, opts *slog.HandlerOptions) *Handler {
	base := slog.NewJSONHandler(w, opts)
	return &Handler{root: base, inner: base}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle injects contextual fields at the top level, then delegates to the inner handler.
// Fields are omitted when not present — no empty-string values appear in logs.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var inject []slog.Attr

	sc := trace.SpanContextFromContext(ctx)
	if sc.HasTraceID() {
		inject = append(inject, slog.String("trace_id", sc.TraceID().String()))
	}
	if sc.HasSpanID() {
		inject = append(inject, slog.String("span_id", sc.SpanID().String()))
	}

	if tid, ok := tenant.TenantIDFromContext(ctx); ok {
		inject = append(inject, slog.String("tenant_id", tid))
	}

	if rid, ok := RequestIDFromContext(ctx); ok {
		inject = append(inject, slog.String("request_id", rid))
	}

	if uid, ok := user.UserIDFromContext(ctx); ok {
		inject = append(inject, slog.String("user_id", uid))
	}

	if len(inject) == 0 {
		// Fast path: no context fields to inject.
		return h.inner.Handle(ctx, r)
	}

	// Inject context attrs at the top level by applying them to the root
	// handler (which has no groups), then re-apply the stored groups so that
	// the record's own attrs land inside the correct group prefix.
	base := h.root.WithAttrs(inject)
	for _, g := range h.groups {
		base = base.WithGroup(g)
	}
	return base.Handle(ctx, r)
}

// WithAttrs returns a new Handler with the given attrs pre-set.
// Pre-attrs are applied to both root and inner so they appear at the correct
// level relative to any groups.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		root:   h.root.WithAttrs(attrs),
		inner:  h.inner.WithAttrs(attrs),
		groups: h.groups,
	}
}

// WithGroup returns a new Handler with the given group name recorded.
// The injected context fields will still appear at the top level; the record's
// own attrs will be nested under the group.
func (h *Handler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &Handler{
		root:   h.root,
		inner:  h.inner.WithGroup(name),
		groups: newGroups,
	}
}
