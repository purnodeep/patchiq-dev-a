package user

import (
	"context"
	"errors"
)

type userCtxKey struct{}

// WithUserID returns a new context carrying the given user ID.
func WithUserID(ctx context.Context, id string) context.Context {
	if id == "" {
		panic("user: WithUserID called with empty user ID")
	}
	return context.WithValue(ctx, userCtxKey{}, id)
}

// UserIDFromContext extracts the user ID from ctx.
// Returns ("", false) if no user ID is set.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userCtxKey{}).(string)
	return id, ok
}

// ErrMissingUserID is returned when no user ID is found in context.
var ErrMissingUserID = errors.New("user: missing user ID in context")

// RequireUserID extracts the user ID from ctx or returns an error.
// Prefer this over MustUserID in code paths where panicking is unsafe
// (workers, gRPC handlers, background goroutines).
func RequireUserID(ctx context.Context) (string, error) {
	id, ok := UserIDFromContext(ctx)
	if !ok || id == "" {
		return "", ErrMissingUserID
	}
	return id, nil
}

// MustUserID extracts the user ID from ctx or panics.
// Use only in HTTP handlers behind chi's panic recovery middleware.
func MustUserID(ctx context.Context) string {
	id, ok := UserIDFromContext(ctx)
	if !ok || id == "" {
		panic("user: missing user ID in context")
	}
	return id
}
