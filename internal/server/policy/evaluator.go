package policy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"
)

// Sentinel errors for evaluator.
var (
	ErrPolicyDisabled           = errors.New("policy is disabled")
	ErrPolicyNotFound           = errors.New("policy not found")
	ErrOutsideMaintenanceWindow = errors.New("outside maintenance window")
)

// PolicyData represents a policy's evaluation-relevant fields.
type PolicyData struct {
	ID                 string
	Name               string
	SelectionMode      string
	MinSeverity        string
	CVEIDs             []string
	PackageRegex       string
	ExcludePackages    []string
	DeploymentStrategy string
	Enabled            bool
	MwStart            time.Duration
	MwEnd              time.Duration
	HasMwWindow        bool
}

// EndpointData represents an endpoint for evaluation.
type EndpointData struct {
	ID       string
	Hostname string
	OsFamily string
}

// EvaluationResult holds the evaluation output per endpoint.
type EvaluationResult struct {
	EndpointID   string       `json:"endpoint_id"`
	EndpointName string       `json:"endpoint_name"`
	Patches      []PatchMatch `json:"patches"`
}

// PatchMatch is a patch selected by the evaluator.
type PatchMatch struct {
	PatchID  string   `json:"patch_id"`
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Severity string   `json:"severity"`
	CVEIDs   []string `json:"cve_ids,omitempty"`
}

// DataSource abstracts DB access for the evaluator.
type DataSource interface {
	GetPolicy(ctx context.Context, tenantID, policyID string) (PolicyData, error)
	ListEndpointsForPolicy(ctx context.Context, tenantID, policyID string) ([]EndpointData, error)
	ListAvailablePatches(ctx context.Context, tenantID, endpointID, osFamily string) ([]CandidatePatch, error)
	ListCVEsForPatches(ctx context.Context, tenantID string, patchIDs []string) (map[string][]CVEInfo, error)
}

// Evaluator runs policy evaluation logic.
type Evaluator struct {
	ds DataSource
}

// NewEvaluator creates an Evaluator.
func NewEvaluator(ds DataSource) *Evaluator {
	if ds == nil {
		panic("policy: NewEvaluator called with nil DataSource")
	}
	return &Evaluator{ds: ds}
}

// Evaluate runs a dry-run evaluation of a policy.
func (e *Evaluator) Evaluate(ctx context.Context, tenantID, policyID string, now time.Time) ([]EvaluationResult, error) {
	pol, err := e.ds.GetPolicy(ctx, tenantID, policyID)
	if err != nil {
		return nil, fmt.Errorf("evaluate policy %s: %w", policyID, ErrPolicyNotFound)
	}
	if !pol.Enabled {
		return nil, fmt.Errorf("evaluate policy %s: %w", policyID, ErrPolicyDisabled)
	}

	if pol.HasMwWindow && !inMaintenanceWindow(now, pol.MwStart, pol.MwEnd) {
		slog.InfoContext(ctx, "policy evaluation skipped: outside maintenance window",
			"policy_id", policyID, "now", now, "mw_start", pol.MwStart, "mw_end", pol.MwEnd)
		return nil, fmt.Errorf("evaluate policy %s: %w", policyID, ErrOutsideMaintenanceWindow)
	}

	endpoints, err := e.ds.ListEndpointsForPolicy(ctx, tenantID, policyID)
	if err != nil {
		return nil, fmt.Errorf("evaluate policy %s: list endpoints: %w", policyID, err)
	}

	strategy, err := StrategyFor(pol.SelectionMode)
	if err != nil {
		return nil, fmt.Errorf("evaluate policy %s: %w", policyID, err)
	}

	// Pre-validate regex to surface errors rather than silently returning empty results.
	if pol.SelectionMode == "by_regex" {
		if _, reErr := regexp.Compile(pol.PackageRegex); reErr != nil {
			return nil, fmt.Errorf("evaluate policy %s: invalid package_regex %q: %w", policyID, pol.PackageRegex, reErr)
		}
	}

	criteria := PolicyCriteria{
		SelectionMode:   pol.SelectionMode,
		MinSeverity:     pol.MinSeverity,
		CVEIDs:          pol.CVEIDs,
		PackageRegex:    pol.PackageRegex,
		ExcludePackages: pol.ExcludePackages,
	}

	var results []EvaluationResult
	for _, ep := range endpoints {
		candidates, patchErr := e.ds.ListAvailablePatches(ctx, tenantID, ep.ID, ep.OsFamily)
		if patchErr != nil {
			return nil, fmt.Errorf("evaluate policy %s: list patches for endpoint %s: %w", policyID, ep.ID, patchErr)
		}

		if pol.SelectionMode == "by_severity" || pol.SelectionMode == "by_cve_list" {
			candidates, patchErr = e.enrichWithCVEs(ctx, tenantID, candidates)
			if patchErr != nil {
				return nil, fmt.Errorf("evaluate policy %s: enrich CVEs for endpoint %s: %w", policyID, ep.ID, patchErr)
			}
		}

		selected := strategy.Select(candidates, criteria)
		if len(selected) == 0 {
			continue
		}

		matches := make([]PatchMatch, len(selected))
		for i, cp := range selected {
			cveIDs := make([]string, 0, len(cp.CVEs))
			for _, c := range cp.CVEs {
				cveIDs = append(cveIDs, c.CVEID)
			}
			matches[i] = PatchMatch{
				PatchID:  cp.PatchID,
				Name:     cp.Name,
				Version:  cp.Version,
				Severity: cp.Severity,
				CVEIDs:   cveIDs,
			}
		}

		results = append(results, EvaluationResult{
			EndpointID:   ep.ID,
			EndpointName: ep.Hostname,
			Patches:      matches,
		})
	}

	return results, nil
}

func (e *Evaluator) enrichWithCVEs(ctx context.Context, tenantID string, candidates []CandidatePatch) ([]CandidatePatch, error) {
	if len(candidates) == 0 {
		return candidates, nil
	}
	patchIDs := make([]string, len(candidates))
	for i, c := range candidates {
		patchIDs[i] = c.PatchID
	}
	cveMap, err := e.ds.ListCVEsForPatches(ctx, tenantID, patchIDs)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		candidates[i].CVEs = cveMap[candidates[i].PatchID]
	}
	return candidates, nil
}

func inMaintenanceWindow(now time.Time, start, end time.Duration) bool {
	tod := time.Duration(now.Hour())*time.Hour + time.Duration(now.Minute())*time.Minute
	if start <= end {
		return tod >= start && tod < end
	}
	return tod >= start || tod < end
}
