package policy

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// AutoPolicyData represents an automatic-mode policy for scheduled evaluation.
type AutoPolicyData struct {
	TenantID string
	PolicyID string
}

// SchedulerDataSource provides data access for the policy scheduler.
type SchedulerDataSource interface {
	ListAutomaticPolicies(ctx context.Context) ([]AutoPolicyData, error)
	Evaluate(ctx context.Context, tenantID, policyID string, now time.Time) ([]EvaluationResult, error)
	LastDeployedPatchIDs(ctx context.Context, tenantID, policyID string) ([]string, error)
}

// AutoDeployer creates a deployment from a policy's matched patches.
type AutoDeployer interface {
	CreateAutoDeployment(ctx context.Context, tenantID, policyID string, patchIDs []string) (string, error)
}

// SchedulerEventEmitter emits events for the policy scheduler.
type SchedulerEventEmitter interface {
	EmitPolicyAutoDeployed(ctx context.Context, tenantID, policyID, deploymentID string, newPatchCount int)
}

// PolicyScheduler evaluates automatic-mode policies and creates deployments
// when new patches are found since the last evaluation.
type PolicyScheduler struct {
	ds       SchedulerDataSource
	deployer AutoDeployer
	emitter  SchedulerEventEmitter
}

// NewPolicyScheduler creates a PolicyScheduler.
func NewPolicyScheduler(ds SchedulerDataSource, deployer AutoDeployer, emitter SchedulerEventEmitter) *PolicyScheduler {
	if ds == nil {
		panic("policy: NewPolicyScheduler called with nil SchedulerDataSource")
	}
	if deployer == nil {
		panic("policy: NewPolicyScheduler called with nil AutoDeployer")
	}
	if emitter == nil {
		panic("policy: NewPolicyScheduler called with nil SchedulerEventEmitter")
	}
	return &PolicyScheduler{ds: ds, deployer: deployer, emitter: emitter}
}

// Run evaluates all automatic-mode policies and creates deployments for new patches.
func (s *PolicyScheduler) Run(ctx context.Context, now time.Time) error {
	policies, err := s.ds.ListAutomaticPolicies(ctx)
	if err != nil {
		return fmt.Errorf("policy scheduler: list automatic policies: %w", err)
	}

	if len(policies) == 0 {
		slog.InfoContext(ctx, "policy scheduler: no automatic policies found")
		return nil
	}

	slog.InfoContext(ctx, "policy scheduler: evaluating automatic policies",
		"policy_count", len(policies))

	var succeeded, failed int
	for _, pol := range policies {
		if err := s.evaluatePolicy(ctx, pol, now); err != nil {
			slog.ErrorContext(ctx, "policy scheduler: evaluation failed, continuing with next policy",
				"policy_id", pol.PolicyID, "tenant_id", pol.TenantID, "error", err)
			failed++
			continue
		}
		succeeded++
	}

	slog.InfoContext(ctx, "policy scheduler: run complete",
		"succeeded", succeeded, "failed", failed, "total", len(policies))

	if failed > 0 && succeeded == 0 {
		return fmt.Errorf("policy scheduler: all %d policy evaluations failed", failed)
	}
	return nil
}

func (s *PolicyScheduler) evaluatePolicy(ctx context.Context, pol AutoPolicyData, now time.Time) error {
	results, err := s.ds.Evaluate(ctx, pol.TenantID, pol.PolicyID, now)
	if err != nil {
		return fmt.Errorf("evaluate policy %s: %w", pol.PolicyID, err)
	}

	// Collect all unique patch IDs from the evaluation.
	currentPatches := make(map[string]struct{})
	for _, r := range results {
		for _, p := range r.Patches {
			currentPatches[p.PatchID] = struct{}{}
		}
	}

	if len(currentPatches) == 0 {
		slog.InfoContext(ctx, "policy scheduler: no patches matched",
			"policy_id", pol.PolicyID, "tenant_id", pol.TenantID)
		return nil
	}

	// Get last deployed patch IDs to diff against.
	lastPatchIDs, err := s.ds.LastDeployedPatchIDs(ctx, pol.TenantID, pol.PolicyID)
	if err != nil {
		return fmt.Errorf("get last deployed patches for policy %s: %w", pol.PolicyID, err)
	}

	lastSet := make(map[string]struct{}, len(lastPatchIDs))
	for _, pid := range lastPatchIDs {
		lastSet[pid] = struct{}{}
	}

	// Find new patches not in the last deployment.
	var newPatchIDs []string
	for pid := range currentPatches {
		if _, seen := lastSet[pid]; !seen {
			newPatchIDs = append(newPatchIDs, pid)
		}
	}

	if len(newPatchIDs) == 0 {
		slog.InfoContext(ctx, "policy scheduler: no new patches since last deployment",
			"policy_id", pol.PolicyID, "tenant_id", pol.TenantID,
			"total_patches", len(currentPatches))
		return nil
	}

	// Create deployment for new patches.
	deploymentID, err := s.deployer.CreateAutoDeployment(ctx, pol.TenantID, pol.PolicyID, newPatchIDs)
	if err != nil {
		return fmt.Errorf("create auto deployment for policy %s: %w", pol.PolicyID, err)
	}

	slog.InfoContext(ctx, "policy scheduler: auto deployment created",
		"policy_id", pol.PolicyID, "tenant_id", pol.TenantID,
		"deployment_id", deploymentID, "new_patch_count", len(newPatchIDs))

	s.emitter.EmitPolicyAutoDeployed(ctx, pol.TenantID, pol.PolicyID, deploymentID, len(newPatchIDs))

	return nil
}
