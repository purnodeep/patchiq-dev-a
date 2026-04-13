package deployment_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/deployment"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// fakeEvalQuerier implements deployment.EvalQuerier for unit tests,
// backed by the new key=value / resolver world. `endpoints` holds the
// hydrated rows that ListEndpointsByIDs returns.
type fakeEvalQuerier struct {
	policy       sqlcgen.Policy
	policyErr    error
	endpoints    []sqlcgen.ListEndpointsByIDsRow
	endpointsErr error
	patches      []sqlcgen.Patch
	patchesErr   error
}

func (f *fakeEvalQuerier) GetPolicyByID(_ context.Context, _ sqlcgen.GetPolicyByIDParams) (sqlcgen.Policy, error) {
	return f.policy, f.policyErr
}

func (f *fakeEvalQuerier) ListEndpointsByIDs(_ context.Context, _ sqlcgen.ListEndpointsByIDsParams) ([]sqlcgen.ListEndpointsByIDsRow, error) {
	return f.endpoints, f.endpointsErr
}

func (f *fakeEvalQuerier) ListPatchesForPolicyFilters(_ context.Context, _ sqlcgen.ListPatchesForPolicyFiltersParams) ([]sqlcgen.Patch, error) {
	return f.patches, f.patchesErr
}

// fakeResolver returns a fixed list of endpoint UUIDs regardless of input.
type fakeResolver struct {
	ids []uuid.UUID
	err error
}

func (f *fakeResolver) ResolveForPolicy(_ context.Context, _, _ string) ([]uuid.UUID, error) {
	return f.ids, f.err
}

var (
	tenantID = validUUID("00000000-0000-0000-0000-000000000001")
	policyID = validUUID("00000000-0000-0000-0000-000000000002")
	ep1ID    = validUUID("00000000-0000-0000-0000-000000000010")
	ep2ID    = validUUID("00000000-0000-0000-0000-000000000011")
	patch1ID = validUUID("00000000-0000-0000-0000-000000000020")
)

// rowFromID builds a minimal ListEndpointsByIDsRow for a given endpoint.
func rowFromID(id pgtype.UUID, os string) sqlcgen.ListEndpointsByIDsRow {
	return sqlcgen.ListEndpointsByIDsRow{ID: id, Hostname: "host", OsFamily: os, Status: "online"}
}

func resolverForIDs(ids ...pgtype.UUID) *fakeResolver {
	out := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		out[i] = uuid.UUID(id.Bytes)
	}
	return &fakeResolver{ids: out}
}

func TestEvaluate_Success(t *testing.T) {
	t.Parallel()
	q := &fakeEvalQuerier{
		policy: sqlcgen.Policy{
			ID:             policyID,
			TenantID:       tenantID,
			Enabled:        true,
			SeverityFilter: []string{"critical"},
		},
		endpoints: []sqlcgen.ListEndpointsByIDsRow{rowFromID(ep1ID, "linux")},
		patches:   []sqlcgen.Patch{{ID: patch1ID, OsFamily: "linux"}},
	}

	eval := deployment.NewEvaluator(resolverForIDs(ep1ID))
	result, err := eval.Evaluate(context.Background(), q, policyID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(result.Targets))
	}
	if result.Targets[0].EndpointID != ep1ID {
		t.Errorf("expected endpoint ID %v, got %v", ep1ID, result.Targets[0].EndpointID)
	}
	if result.Targets[0].PatchID != patch1ID {
		t.Errorf("expected patch ID %v, got %v", patch1ID, result.Targets[0].PatchID)
	}
}

func TestEvaluate_DisabledPolicy(t *testing.T) {
	t.Parallel()
	q := &fakeEvalQuerier{policy: sqlcgen.Policy{Enabled: false}}

	eval := deployment.NewEvaluator(resolverForIDs())
	_, err := eval.Evaluate(context.Background(), q, policyID, tenantID)
	if err == nil {
		t.Fatal("expected error for disabled policy")
	}
	if !contains(err.Error(), "disabled") {
		t.Errorf("expected error to contain 'disabled', got: %v", err)
	}
}

func TestEvaluate_NoEndpoints(t *testing.T) {
	t.Parallel()
	q := &fakeEvalQuerier{
		policy:    sqlcgen.Policy{Enabled: true},
		endpoints: []sqlcgen.ListEndpointsByIDsRow{},
	}

	// Resolver returns zero IDs — the evaluator must short-circuit with
	// ErrNoEndpoints before reaching ListEndpointsByIDs.
	eval := deployment.NewEvaluator(resolverForIDs())
	_, err := eval.Evaluate(context.Background(), q, policyID, tenantID)
	if err == nil {
		t.Fatal("expected error for no endpoints")
	}
	if !contains(err.Error(), "no endpoints") {
		t.Errorf("expected error to contain 'no endpoints', got: %v", err)
	}
}

func TestEvaluate_NoPatches(t *testing.T) {
	t.Parallel()
	q := &fakeEvalQuerier{
		policy:    sqlcgen.Policy{Enabled: true, SeverityFilter: []string{"critical"}},
		endpoints: []sqlcgen.ListEndpointsByIDsRow{rowFromID(ep1ID, "linux")},
		patches:   []sqlcgen.Patch{},
	}

	eval := deployment.NewEvaluator(resolverForIDs(ep1ID))
	_, err := eval.Evaluate(context.Background(), q, policyID, tenantID)
	if err == nil {
		t.Fatal("expected error for no patches")
	}
	if !contains(err.Error(), "no patches") {
		t.Errorf("expected error to contain 'no patches', got: %v", err)
	}
}

func TestEvaluate_FiltersByOSFamily(t *testing.T) {
	t.Parallel()
	q := &fakeEvalQuerier{
		policy: sqlcgen.Policy{Enabled: true, SeverityFilter: []string{"critical"}},
		endpoints: []sqlcgen.ListEndpointsByIDsRow{
			rowFromID(ep1ID, "linux"),
			rowFromID(ep2ID, "windows"),
		},
		patches: []sqlcgen.Patch{{ID: patch1ID, OsFamily: "linux"}},
	}

	eval := deployment.NewEvaluator(resolverForIDs(ep1ID, ep2ID))
	result, err := eval.Evaluate(context.Background(), q, policyID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(result.Targets))
	}
	if result.Targets[0].EndpointID != ep1ID {
		t.Errorf("expected linux endpoint %v, got %v", ep1ID, result.Targets[0].EndpointID)
	}
}

func TestEvaluate_GetPolicyError(t *testing.T) {
	t.Parallel()
	q := &fakeEvalQuerier{policyErr: errors.New("db connection failed")}

	eval := deployment.NewEvaluator(resolverForIDs())
	_, err := eval.Evaluate(context.Background(), q, policyID, tenantID)
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "get policy") {
		t.Errorf("expected error to contain 'get policy', got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Compile-time assertions for the fakes.
var (
	_ deployment.EvalQuerier      = (*fakeEvalQuerier)(nil)
	_ deployment.EndpointResolver = (*fakeResolver)(nil)
)

func TestBuildSeverityFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		policy   sqlcgen.Policy
		expected []string
	}{
		{
			name: "pass-through when SeverityFilter already populated",
			policy: sqlcgen.Policy{
				SelectionMode:  "by_severity",
				MinSeverity:    pgtype.Text{String: "high", Valid: true},
				SeverityFilter: []string{"critical"},
			},
			expected: []string{"critical"},
		},
		{
			name:     "all_available returns nil",
			policy:   sqlcgen.Policy{SelectionMode: "all_available"},
			expected: nil,
		},
		{
			name: "by_severity with min_severity=critical",
			policy: sqlcgen.Policy{
				SelectionMode: "by_severity",
				MinSeverity:   pgtype.Text{String: "critical", Valid: true},
			},
			expected: []string{"critical"},
		},
		{
			name: "by_severity with min_severity=high",
			policy: sqlcgen.Policy{
				SelectionMode: "by_severity",
				MinSeverity:   pgtype.Text{String: "high", Valid: true},
			},
			expected: []string{"critical", "high"},
		},
		{
			name: "by_severity with min_severity=medium",
			policy: sqlcgen.Policy{
				SelectionMode: "by_severity",
				MinSeverity:   pgtype.Text{String: "medium", Valid: true},
			},
			expected: []string{"critical", "high", "medium"},
		},
		{
			name: "by_severity with min_severity=low",
			policy: sqlcgen.Policy{
				SelectionMode: "by_severity",
				MinSeverity:   pgtype.Text{String: "low", Valid: true},
			},
			expected: []string{"critical", "high", "low", "medium"},
		},
		{
			name: "by_severity with invalid min_severity",
			policy: sqlcgen.Policy{
				SelectionMode: "by_severity",
				MinSeverity:   pgtype.Text{String: "unknown", Valid: true},
			},
			expected: nil,
		},
		{
			name: "by_severity with empty min_severity",
			policy: sqlcgen.Policy{
				SelectionMode: "by_severity",
				MinSeverity:   pgtype.Text{String: "", Valid: false},
			},
			expected: nil,
		},
		{
			name:     "by_cve_list returns nil",
			policy:   sqlcgen.Policy{SelectionMode: "by_cve_list"},
			expected: nil,
		},
		{
			name:     "unknown selection_mode returns nil",
			policy:   sqlcgen.Policy{SelectionMode: "custom_unknown"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deployment.BuildSeverityFilter(tt.policy)

			if tt.expected == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected %v, got nil", tt.expected)
			}

			sortedExpected := make([]string, len(tt.expected))
			copy(sortedExpected, tt.expected)
			sort.Strings(sortedExpected)

			sortedGot := make([]string, len(got))
			copy(sortedGot, got)
			sort.Strings(sortedGot)

			if len(sortedGot) != len(sortedExpected) {
				t.Fatalf("expected %v, got %v", sortedExpected, sortedGot)
			}
			for i := range sortedExpected {
				if sortedGot[i] != sortedExpected[i] {
					t.Fatalf("expected %v, got %v", sortedExpected, sortedGot)
				}
			}
		})
	}
}
