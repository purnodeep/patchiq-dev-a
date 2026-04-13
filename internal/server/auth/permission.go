package auth

import (
	"fmt"
	"strings"
)

// Permission represents an RBAC permission as Resource + Action + Scope.
type Permission struct {
	Resource string
	Action   string
	Scope    string
}

// String returns the colon-delimited format: "resource:action:scope".
func (p Permission) String() string {
	return p.Resource + ":" + p.Action + ":" + p.Scope
}

// ParsePermission parses a colon-delimited permission string.
// Accepted formats:
//   - "resource:action:scope"          (3 parts, e.g., "endpoints:read:*")
//   - "resource:action:scopeType:val"  (4 parts, e.g., "deployments:approve:group:prod")
func ParsePermission(s string) (Permission, error) {
	if s == "" {
		return Permission{}, fmt.Errorf("parse permission: empty string")
	}

	resource, rest, ok := strings.Cut(s, ":")
	if !ok {
		return Permission{}, fmt.Errorf("parse permission %q: invalid format, expected resource:action:scope", s)
	}
	action, scope, ok := strings.Cut(rest, ":")
	if !ok {
		return Permission{}, fmt.Errorf("parse permission %q: invalid format, expected resource:action:scope", s)
	}
	if resource == "" || action == "" || scope == "" {
		return Permission{}, fmt.Errorf("parse permission %q: empty component", s)
	}

	return Permission{Resource: resource, Action: action, Scope: scope}, nil
}

// Covers reports whether the held permission (p) satisfies the required permission.
func (p Permission) Covers(required Permission) bool {
	if p.Resource == "" || p.Action == "" || p.Scope == "" {
		return false
	}
	if p.Resource != "*" && p.Resource != required.Resource {
		return false
	}
	if p.Action != "*" && p.Action != required.Action {
		return false
	}
	return scopeCovers(p.Scope, required.Scope)
}

// scopeCovers reports whether heldScope covers requiredScope.
func scopeCovers(held, required string) bool {
	if held == "*" {
		return true
	}
	return held == required
}
