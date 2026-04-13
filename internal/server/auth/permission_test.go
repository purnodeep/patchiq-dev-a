package auth_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/auth"
)

func TestParsePermission(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    auth.Permission
		wantErr bool
	}{
		{"simple wildcard scope", "endpoints:read:*", auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"}, false},
		{"group scope", "deployments:approve:group:prod", auth.Permission{Resource: "deployments", Action: "approve", Scope: "group:prod"}, false},
		{"tenant scope", "endpoints:read:tenant:abc-123", auth.Permission{Resource: "endpoints", Action: "read", Scope: "tenant:abc-123"}, false},
		{"own scope", "reports:read:own", auth.Permission{Resource: "reports", Action: "read", Scope: "own"}, false},
		{"full wildcard", "*:*:*", auth.Permission{Resource: "*", Action: "*", Scope: "*"}, false},
		{"resource wildcard action", "policies:*:*", auth.Permission{Resource: "policies", Action: "*", Scope: "*"}, false},
		{"empty string", "", auth.Permission{}, true},
		{"too few parts", "endpoints:read", auth.Permission{}, true},
		{"five parts collapse into scope", "a:b:c:d:e", auth.Permission{Resource: "a", Action: "b", Scope: "c:d:e"}, false},
		{"empty resource", ":read:*", auth.Permission{}, true},
		{"empty action", "endpoints::*", auth.Permission{}, true},
		{"empty scope", "endpoints:read:", auth.Permission{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.ParsePermission(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePermission(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("ParsePermission(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPermissionString(t *testing.T) {
	tests := []struct {
		perm auth.Permission
		want string
	}{
		{auth.Permission{Resource: "endpoints", Action: "read", Scope: "*"}, "endpoints:read:*"},
		{auth.Permission{Resource: "deployments", Action: "approve", Scope: "group:prod"}, "deployments:approve:group:prod"},
		{auth.Permission{Resource: "*", Action: "*", Scope: "*"}, "*:*:*"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.perm.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPermissionCovers(t *testing.T) {
	tests := []struct {
		name     string
		held     string
		required string
		want     bool
	}{
		// Exact matches
		{"exact match wildcard scope", "endpoints:read:*", "endpoints:read:*", true},
		{"exact match group scope", "endpoints:read:group:prod", "endpoints:read:group:prod", true},
		{"exact match own scope", "reports:read:own", "reports:read:own", true},

		// Wildcard resource
		{"wildcard resource covers specific", "*:read:*", "endpoints:read:*", true},
		{"wildcard resource wrong action", "*:read:*", "endpoints:create:*", false},

		// Wildcard action
		{"wildcard action covers specific", "endpoints:*:*", "endpoints:read:*", true},
		{"wildcard action covers create", "endpoints:*:*", "endpoints:create:*", true},

		// Full wildcard
		{"super admin covers everything", "*:*:*", "deployments:approve:group:prod", true},
		{"super admin covers wildcard required", "*:*:*", "endpoints:read:*", true},

		// Scope coverage
		{"wildcard scope covers group", "endpoints:read:*", "endpoints:read:group:prod", true},
		{"group scope does not cover wildcard", "endpoints:read:group:prod", "endpoints:read:*", false},
		{"group scope does not cover other group", "endpoints:read:group:prod", "endpoints:read:group:staging", false},
		{"own does not cover wildcard", "endpoints:read:own", "endpoints:read:*", false},
		{"own does not cover group", "endpoints:read:own", "endpoints:read:group:prod", false},
		{"wildcard scope covers own", "endpoints:read:*", "endpoints:read:own", true},

		// Mismatches
		{"wrong resource", "policies:read:*", "endpoints:read:*", false},
		{"wrong action", "endpoints:create:*", "endpoints:read:*", false},

		// Tenant scope
		{"tenant scope exact match", "endpoints:read:tenant:abc", "endpoints:read:tenant:abc", true},
		{"tenant scope mismatch", "endpoints:read:tenant:abc", "endpoints:read:tenant:xyz", false},
		{"wildcard covers tenant", "endpoints:read:*", "endpoints:read:tenant:abc", true},
		{"tenant does not cover wildcard", "endpoints:read:tenant:abc", "endpoints:read:*", false},

		// Wildcard resource and action with narrow scope
		{"wildcard resource and action with narrow scope covers matching scope", "*:*:group:prod", "endpoints:read:group:prod", true},
		{"wildcard resource and action with narrow scope denies different scope", "*:*:group:prod", "endpoints:read:group:staging", false},
		{"wildcard resource and action with narrow scope denies wildcard required", "*:*:group:prod", "endpoints:read:*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			held, err := auth.ParsePermission(tt.held)
			if err != nil {
				t.Fatalf("parse held: %v", err)
			}
			required, err := auth.ParsePermission(tt.required)
			if err != nil {
				t.Fatalf("parse required: %v", err)
			}
			if got := held.Covers(required); got != tt.want {
				t.Errorf("(%q).Covers(%q) = %v, want %v", tt.held, tt.required, got, tt.want)
			}
		})
	}

	t.Run("zero-value permission denies", func(t *testing.T) {
		var zero auth.Permission
		required, err := auth.ParsePermission("endpoints:read:*")
		if err != nil {
			t.Fatalf("parse required: %v", err)
		}
		if zero.Covers(required) {
			t.Error("zero-value Permission{}.Covers() = true, want false")
		}
	})
}

func TestPermissionRoundTrip(t *testing.T) {
	tests := []string{
		"endpoints:read:*",
		"deployments:approve:group:prod",
		"patches:update:tenant:abc-123",
	}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			p, err := auth.ParsePermission(s)
			if err != nil {
				t.Fatalf("ParsePermission(%q) error: %v", s, err)
			}
			got := p.String()
			if got != s {
				t.Errorf("String() = %q, want %q", got, s)
			}
			p2, err := auth.ParsePermission(got)
			if err != nil {
				t.Fatalf("ParsePermission(String()) error: %v", err)
			}
			if p2 != p {
				t.Errorf("round-trip mismatch: got %+v, want %+v", p2, p)
			}
		})
	}
}
