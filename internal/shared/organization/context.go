// Package organization provides context helpers and HTTP middleware for
// carrying the active organization ID through a request. An organization is
// the parent entity of one or more tenants; see docs/adr/025 for details.
//
// Semantics mirror internal/shared/tenant: the context key is private, both
// With/From round-trips and Must/Require accessors are provided, and errors
// propagate rather than panic for non-HTTP callers.
package organization

import (
	"context"
	"errors"
)

type ctxKey struct{}

// WithOrgID returns a new context carrying the given organization ID.
// Panics if id is empty — callers must validate before injecting.
func WithOrgID(ctx context.Context, id string) context.Context {
	if id == "" {
		panic("organization: WithOrgID called with empty organization ID")
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// OrgIDFromContext extracts the organization ID from ctx.
// Returns ("", false) if no organization ID is set.
func OrgIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok
}

// ErrMissingOrgID is returned when no organization ID is found in context.
var ErrMissingOrgID = errors.New("organization: missing organization ID in context")

// RequireOrgID extracts the organization ID from ctx or returns an error.
// Prefer this over MustOrgID in code paths where panicking is unsafe
// (workers, gRPC handlers, background goroutines).
func RequireOrgID(ctx context.Context) (string, error) {
	id, ok := OrgIDFromContext(ctx)
	if !ok || id == "" {
		return "", ErrMissingOrgID
	}
	return id, nil
}

// MustOrgID extracts the organization ID from ctx or panics.
// Use only in HTTP handlers behind chi's panic recovery middleware.
func MustOrgID(ctx context.Context) string {
	id, ok := OrgIDFromContext(ctx)
	if !ok || id == "" {
		panic("organization: missing organization ID in context")
	}
	return id
}
