package tenant

import (
	"context"
	"errors"
)

type ctxKey struct{}

// WithTenantID returns a new context carrying the given tenant ID.
func WithTenantID(ctx context.Context, id string) context.Context {
	if id == "" {
		panic("tenant: WithTenantID called with empty tenant ID")
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// TenantIDFromContext extracts the tenant ID from ctx.
// Returns ("", false) if no tenant ID is set.
func TenantIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKey{}).(string)
	return id, ok
}

// ErrMissingTenantID is returned when no tenant ID is found in context.
var ErrMissingTenantID = errors.New("tenant: missing tenant ID in context")

// RequireTenantID extracts the tenant ID from ctx or returns an error.
// Prefer this over MustTenantID in code paths where panicking is unsafe
// (workers, gRPC handlers, background goroutines).
func RequireTenantID(ctx context.Context) (string, error) {
	id, ok := TenantIDFromContext(ctx)
	if !ok || id == "" {
		return "", ErrMissingTenantID
	}
	return id, nil
}

// MustTenantID extracts the tenant ID from ctx or panics.
// Use only in HTTP handlers behind chi's panic recovery middleware.
func MustTenantID(ctx context.Context) string {
	id, ok := TenantIDFromContext(ctx)
	if !ok || id == "" {
		panic("tenant: missing tenant ID in context")
	}
	return id
}
