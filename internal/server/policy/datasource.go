package policy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
	"github.com/skenzeriq/patchiq/internal/server/targeting"
)

// EvaluatorQuerier defines the sqlc queries needed by SQLDataSource.
// Endpoint targeting is no longer a sqlc query — it is delegated to an
// EndpointResolver (implemented by targeting.Resolver) which compiles
// the policy's tag selector into SQL at resolve time.
type EvaluatorQuerier interface {
	GetPolicyByID(ctx context.Context, arg sqlcgen.GetPolicyByIDParams) (sqlcgen.Policy, error)
	ListAvailablePatchesForEndpoint(ctx context.Context, arg sqlcgen.ListAvailablePatchesForEndpointParams) ([]sqlcgen.Patch, error)
	ListCVEsForPatches(ctx context.Context, arg sqlcgen.ListCVEsForPatchesParams) ([]sqlcgen.ListCVEsForPatchesRow, error)
	ListEndpointsByIDs(ctx context.Context, arg sqlcgen.ListEndpointsByIDsParams) ([]sqlcgen.ListEndpointsByIDsRow, error)
}

// EndpointResolver returns the set of endpoints a policy targets. Phase 2
// wires SQLDataSource through targeting.Resolver; tests can substitute a
// fake via this interface.
type EndpointResolver interface {
	ResolveForPolicy(ctx context.Context, tenantID, policyID string) ([]uuid.UUID, error)
}

// SQLDataSource implements DataSource using sqlc queries plus a tag
// selector resolver.
type SQLDataSource struct {
	q        EvaluatorQuerier
	resolver EndpointResolver
}

// NewSQLDataSource creates a SQLDataSource. resolver may be nil in unit
// tests that never call ListEndpointsForPolicy.
func NewSQLDataSource(q EvaluatorQuerier, resolver EndpointResolver) *SQLDataSource {
	if q == nil {
		panic("policy: NewSQLDataSource called with nil querier")
	}
	return &SQLDataSource{q: q, resolver: resolver}
}

// GetPolicy fetches a policy by ID and maps it to PolicyData.
func (ds *SQLDataSource) GetPolicy(ctx context.Context, tenantID, policyID string) (PolicyData, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return PolicyData{}, fmt.Errorf("get policy: parse tenant ID: %w", err)
	}
	pid, err := parseUUID(policyID)
	if err != nil {
		return PolicyData{}, fmt.Errorf("get policy: parse policy ID: %w", err)
	}

	p, err := ds.q.GetPolicyByID(ctx, sqlcgen.GetPolicyByIDParams{
		ID:       pid,
		TenantID: tid,
	})
	if err != nil {
		return PolicyData{}, fmt.Errorf("get policy: %w", err)
	}

	return mapPolicy(p), nil
}

// ListEndpointsForPolicy returns endpoints targeted by a policy's
// key=value tag selector. Requires the datasource to be constructed with
// a non-nil resolver; legacy group lookups were removed in migration 060.
func (ds *SQLDataSource) ListEndpointsForPolicy(ctx context.Context, tenantID, policyID string) ([]EndpointData, error) {
	if ds.resolver == nil {
		return nil, fmt.Errorf("list endpoints for policy: nil resolver")
	}
	ids, err := ds.resolver.ResolveForPolicy(ctx, tenantID, policyID)
	if err != nil {
		return nil, fmt.Errorf("list endpoints for policy: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}

	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list endpoints for policy: parse tenant ID: %w", err)
	}

	pgIDs := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		pgIDs[i] = pgtype.UUID{Bytes: id, Valid: true}
	}

	rows, err := ds.q.ListEndpointsByIDs(ctx, sqlcgen.ListEndpointsByIDsParams{
		TenantID: tid,
		Ids:      pgIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list endpoints for policy: hydrate: %w", err)
	}

	result := make([]EndpointData, len(rows))
	for i, ep := range rows {
		result[i] = EndpointData{
			ID:       pgUUIDToString(ep.ID),
			Hostname: ep.Hostname,
			OsFamily: ep.OsFamily,
		}
	}
	return result, nil
}

// ListAvailablePatches returns available patches for an endpoint's OS family.
func (ds *SQLDataSource) ListAvailablePatches(ctx context.Context, tenantID, _ string, osFamily string) ([]CandidatePatch, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list available patches: parse tenant ID: %w", err)
	}

	patches, err := ds.q.ListAvailablePatchesForEndpoint(ctx, sqlcgen.ListAvailablePatchesForEndpointParams{
		TenantID: tid,
		OsFamily: osFamily,
	})
	if err != nil {
		return nil, fmt.Errorf("list available patches: %w", err)
	}

	result := make([]CandidatePatch, len(patches))
	for i, p := range patches {
		result[i] = CandidatePatch{
			PatchID:  pgUUIDToString(p.ID),
			Name:     p.Name,
			Version:  p.Version,
			Severity: strings.ToLower(p.Severity),
		}
	}
	return result, nil
}

// ListCVEsForPatches returns CVEs grouped by patch ID.
func (ds *SQLDataSource) ListCVEsForPatches(ctx context.Context, tenantID string, patchIDs []string) (map[string][]CVEInfo, error) {
	tid, err := parseUUID(tenantID)
	if err != nil {
		return nil, fmt.Errorf("list CVEs for patches: parse tenant ID: %w", err)
	}

	pgPatchIDs := make([]pgtype.UUID, len(patchIDs))
	for i, id := range patchIDs {
		pgPatchIDs[i], err = parseUUID(id)
		if err != nil {
			return nil, fmt.Errorf("list CVEs for patches: parse patch ID %s: %w", id, err)
		}
	}

	rows, err := ds.q.ListCVEsForPatches(ctx, sqlcgen.ListCVEsForPatchesParams{
		TenantID: tid,
		PatchIds: pgPatchIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list CVEs for patches: %w", err)
	}

	result := make(map[string][]CVEInfo, len(patchIDs))
	for _, row := range rows {
		patchID := pgUUIDToString(row.PatchID)
		result[patchID] = append(result[patchID], CVEInfo{
			CVEID:    row.CveID,
			Severity: strings.ToLower(row.Severity),
		})
	}
	return result, nil
}

// mapPolicy converts a sqlcgen.Policy to a PolicyData.
func mapPolicy(p sqlcgen.Policy) PolicyData {
	pd := PolicyData{
		ID:                 pgUUIDToString(p.ID),
		Name:               p.Name,
		SelectionMode:      p.SelectionMode,
		CVEIDs:             p.CveIds,
		DeploymentStrategy: p.DeploymentStrategy,
		Enabled:            p.Enabled,
		HasMwWindow:        p.MwStart.Valid,
	}

	if p.MinSeverity.Valid {
		pd.MinSeverity = p.MinSeverity.String
	}
	if p.PackageRegex.Valid {
		pd.PackageRegex = p.PackageRegex.String
	}
	if p.ExcludePackages != nil {
		pd.ExcludePackages = p.ExcludePackages
	}
	if p.MwStart.Valid {
		pd.MwStart = microsToTimeDuration(p.MwStart.Microseconds)
	}
	if p.MwEnd.Valid {
		pd.MwEnd = microsToTimeDuration(p.MwEnd.Microseconds)
	}

	return pd
}

func microsToTimeDuration(micros int64) time.Duration {
	return time.Duration(micros) * time.Microsecond
}

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func pgUUIDToString(u pgtype.UUID) string {
	return uuid.UUID(u.Bytes).String()
}

// Compile-time assertion that targeting.Resolver satisfies EndpointResolver.
var _ EndpointResolver = (*targeting.Resolver)(nil)
