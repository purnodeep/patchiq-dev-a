package deployment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// Sentinel errors for evaluation failures.
var (
	ErrPolicyDisabled   = errors.New("policy is disabled")
	ErrNoEndpoints      = errors.New("no endpoints matched policy selector")
	ErrNoPatchesMatched = errors.New("no patches matched policy filters")
	ErrNoResolver       = errors.New("deployment: evaluator has no endpoint resolver")
)

// EvalQuerier defines queries needed for policy evaluation. Endpoint set
// resolution moved from `ListEndpointsForPolicyGroups` (removed in 060)
// to `ListEndpointsByIDs`, which hydrates the UUIDs produced by the tag
// selector resolver into the minimal endpoint shape the evaluator needs.
type EvalQuerier interface {
	GetPolicyByID(ctx context.Context, arg sqlcgen.GetPolicyByIDParams) (sqlcgen.Policy, error)
	ListEndpointsByIDs(ctx context.Context, arg sqlcgen.ListEndpointsByIDsParams) ([]sqlcgen.ListEndpointsByIDsRow, error)
	ListPatchesForPolicyFilters(ctx context.Context, arg sqlcgen.ListPatchesForPolicyFiltersParams) ([]sqlcgen.Patch, error)
}

// EndpointResolver returns the endpoint UUIDs a policy targets, evaluated
// from the policy's tag selector (policy_tag_selectors.expression).
// Implemented by targeting.Resolver in production; tests can pass a fake.
type EndpointResolver interface {
	ResolveForPolicy(ctx context.Context, tenantID, policyID string) ([]uuid.UUID, error)
}

// EvaluatedEndpoint is the compact shape the deployment target-matcher
// needs from each endpoint. Replaces `sqlcgen.Endpoint` in the result so
// the evaluator is decoupled from the endpoints table row schema.
type EvaluatedEndpoint struct {
	ID       pgtype.UUID
	Hostname string
	OsFamily string
	Status   string
}

// EvalResult holds the output of policy evaluation.
type EvalResult struct {
	Policy    sqlcgen.Policy
	Endpoints []EvaluatedEndpoint
	Patches   []sqlcgen.Patch
	Targets   []Target
}

// Target is an endpoint+patch pair for a deployment.
type Target struct {
	EndpointID pgtype.UUID
	PatchID    pgtype.UUID
}

// Evaluator resolves a policy into deployment targets. A nil resolver is
// legal for tests that never call Evaluate; production callers must wire
// one in or Evaluate returns ErrNoResolver.
type Evaluator struct {
	resolver EndpointResolver
}

// NewEvaluator constructs an Evaluator with an endpoint resolver. Pass
// nil only in tests that stub Evaluate wholesale or never invoke it.
func NewEvaluator(resolver EndpointResolver) *Evaluator {
	return &Evaluator{resolver: resolver}
}

// Evaluate resolves a policy to endpoints and applicable patches, matching by OS family.
func (e *Evaluator) Evaluate(ctx context.Context, q EvalQuerier, policyID, tenantID pgtype.UUID) (*EvalResult, error) {
	policy, err := q.GetPolicyByID(ctx, sqlcgen.GetPolicyByIDParams{ID: policyID, TenantID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("evaluate policy: get policy: %w", err)
	}
	if !policy.Enabled {
		return nil, fmt.Errorf("evaluate policy: %w", ErrPolicyDisabled)
	}

	if e.resolver == nil {
		return nil, fmt.Errorf("evaluate policy: %w", ErrNoResolver)
	}
	tenantStr := uuid.UUID(tenantID.Bytes).String()
	policyStr := uuid.UUID(policyID.Bytes).String()
	ids, err := e.resolver.ResolveForPolicy(ctx, tenantStr, policyStr)
	if err != nil {
		return nil, fmt.Errorf("evaluate policy: resolve endpoints: %w", err)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("evaluate policy: %w", ErrNoEndpoints)
	}

	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}
	rows, err := q.ListEndpointsByIDs(ctx, sqlcgen.ListEndpointsByIDsParams{
		TenantID: tenantID,
		Ids:      pgIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate policy: hydrate endpoints: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("evaluate policy: %w", ErrNoEndpoints)
	}
	endpoints := make([]EvaluatedEndpoint, len(rows))
	for i, r := range rows {
		endpoints[i] = EvaluatedEndpoint{
			ID:       r.ID,
			Hostname: r.Hostname,
			OsFamily: r.OsFamily,
			Status:   r.Status,
		}
	}

	osFamilies := uniqueOSFamilies(endpoints)
	severityFilter := BuildSeverityFilter(policy)

	patches, err := q.ListPatchesForPolicyFilters(ctx, sqlcgen.ListPatchesForPolicyFiltersParams{
		TenantID:       tenantID,
		SeverityFilter: severityFilter,
		OsFamilies:     osFamilies,
	})
	if err != nil {
		return nil, fmt.Errorf("evaluate policy: list patches: %w", err)
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("evaluate policy: %w", ErrNoPatchesMatched)
	}

	targets := matchTargets(endpoints, patches)

	return &EvalResult{
		Policy:    policy,
		Endpoints: endpoints,
		Patches:   patches,
		Targets:   targets,
	}, nil
}

func uniqueOSFamilies(endpoints []EvaluatedEndpoint) []string {
	seen := make(map[string]bool)
	var result []string
	for _, ep := range endpoints {
		if !seen[ep.OsFamily] {
			seen[ep.OsFamily] = true
			result = append(result, ep.OsFamily)
		}
	}
	return result
}

// BuildSeverityFilter converts a policy's selection_mode and min_severity into
// a concrete list of severity strings for patch filtering. This is the runtime
// fallback used by the deployment evaluator when the policy's severity_filter
// column is empty (e.g., for policies created before severity_filter was added).
//
// Returns nil when no severity filtering should be applied (all_available,
// by_cve_list, unknown modes, or when the policy already has a populated
// SeverityFilter).
func BuildSeverityFilter(policy sqlcgen.Policy) []string {
	// If the policy already has a severity filter, use it as-is.
	if len(policy.SeverityFilter) > 0 {
		return policy.SeverityFilter
	}

	if policy.SelectionMode != "by_severity" {
		return nil
	}

	minSeverity := ""
	if policy.MinSeverity.Valid {
		minSeverity = policy.MinSeverity.String
	}
	if minSeverity == "" {
		return nil
	}

	rank := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	allSevs := []string{"low", "medium", "high", "critical"}
	minRank, ok := rank[minSeverity]
	if !ok {
		return nil
	}
	var result []string
	for _, s := range allSevs {
		if rank[s] >= minRank {
			result = append(result, s)
		}
	}
	return result
}

func matchTargets(endpoints []EvaluatedEndpoint, patches []sqlcgen.Patch) []Target {
	byOS := make(map[string][]sqlcgen.Patch)
	for _, p := range patches {
		byOS[p.OsFamily] = append(byOS[p.OsFamily], p)
	}
	var targets []Target
	for _, ep := range endpoints {
		for _, p := range byOS[ep.OsFamily] {
			targets = append(targets, Target{EndpointID: ep.ID, PatchID: p.ID})
		}
	}
	return targets
}
