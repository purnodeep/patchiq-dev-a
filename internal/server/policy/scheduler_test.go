package policy

import (
	"context"
	"testing"
	"time"
)

func TestPolicyScheduler_NoAutomaticPolicies(t *testing.T) {
	ds := &mockSchedulerDataSource{
		policies: []AutoPolicyData{},
	}
	deployer := &mockDeployer{}
	emitter := &mockEventEmitter{}

	sched := NewPolicyScheduler(ds, deployer, emitter)
	err := sched.Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if deployer.createCount != 0 {
		t.Errorf("expected 0 deployments, got %d", deployer.createCount)
	}
}

func TestPolicyScheduler_NewPatchesFound(t *testing.T) {
	ds := &mockSchedulerDataSource{
		policies: []AutoPolicyData{
			{
				TenantID: "tenant-1",
				PolicyID: "policy-1",
			},
		},
		evalResults: []EvaluationResult{
			{
				EndpointID:   "ep-1",
				EndpointName: "host-1",
				Patches: []PatchMatch{
					{PatchID: "patch-1", Name: "Security Update", Version: "1.0"},
				},
			},
		},
		lastPatchIDs: map[string][]string{
			"policy-1": {}, // no previous patches
		},
	}
	deployer := &mockDeployer{}
	emitter := &mockEventEmitter{}

	sched := NewPolicyScheduler(ds, deployer, emitter)
	err := sched.Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if deployer.createCount != 1 {
		t.Errorf("expected 1 deployment, got %d", deployer.createCount)
	}
	if emitter.emitCount != 1 {
		t.Errorf("expected 1 event emitted, got %d", emitter.emitCount)
	}
}

func TestPolicyScheduler_NoDiffNoDeployment(t *testing.T) {
	ds := &mockSchedulerDataSource{
		policies: []AutoPolicyData{
			{
				TenantID: "tenant-1",
				PolicyID: "policy-1",
			},
		},
		evalResults: []EvaluationResult{
			{
				EndpointID:   "ep-1",
				EndpointName: "host-1",
				Patches: []PatchMatch{
					{PatchID: "patch-1", Name: "Security Update", Version: "1.0"},
				},
			},
		},
		lastPatchIDs: map[string][]string{
			"policy-1": {"patch-1"}, // already deployed
		},
	}
	deployer := &mockDeployer{}
	emitter := &mockEventEmitter{}

	sched := NewPolicyScheduler(ds, deployer, emitter)
	err := sched.Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if deployer.createCount != 0 {
		t.Errorf("expected 0 deployments (no diff), got %d", deployer.createCount)
	}
}

func TestPolicyScheduler_MultiplePolices(t *testing.T) {
	ds := &mockSchedulerDataSource{
		policies: []AutoPolicyData{
			{TenantID: "tenant-1", PolicyID: "policy-1"},
			{TenantID: "tenant-1", PolicyID: "policy-2"},
		},
		evalResults: []EvaluationResult{
			{
				EndpointID: "ep-1",
				Patches:    []PatchMatch{{PatchID: "patch-1"}},
			},
		},
		lastPatchIDs: map[string][]string{
			"policy-1": {},
			"policy-2": {"patch-1"}, // already deployed
		},
	}
	deployer := &mockDeployer{}
	emitter := &mockEventEmitter{}

	sched := NewPolicyScheduler(ds, deployer, emitter)
	err := sched.Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	// Only policy-1 should create a deployment (policy-2 has no diff)
	if deployer.createCount != 1 {
		t.Errorf("expected 1 deployment, got %d", deployer.createCount)
	}
}

func TestPolicyScheduler_EvalError_ContinuesOthers(t *testing.T) {
	ds := &mockSchedulerDataSource{
		policies: []AutoPolicyData{
			{TenantID: "tenant-1", PolicyID: "policy-err"},
			{TenantID: "tenant-1", PolicyID: "policy-ok"},
		},
		evalResults: []EvaluationResult{
			{
				EndpointID: "ep-1",
				Patches:    []PatchMatch{{PatchID: "patch-1"}},
			},
		},
		evalError: map[string]bool{"policy-err": true},
		lastPatchIDs: map[string][]string{
			"policy-err": {},
			"policy-ok":  {},
		},
	}
	deployer := &mockDeployer{}
	emitter := &mockEventEmitter{}

	sched := NewPolicyScheduler(ds, deployer, emitter)
	err := sched.Run(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	// policy-err fails evaluation, but policy-ok should still work
	if deployer.createCount != 1 {
		t.Errorf("expected 1 deployment (skipping errored policy), got %d", deployer.createCount)
	}
}

// --- mocks ---

type mockSchedulerDataSource struct {
	policies     []AutoPolicyData
	evalResults  []EvaluationResult
	evalError    map[string]bool
	lastPatchIDs map[string][]string
}

func (m *mockSchedulerDataSource) ListAutomaticPolicies(_ context.Context) ([]AutoPolicyData, error) {
	return m.policies, nil
}

func (m *mockSchedulerDataSource) Evaluate(_ context.Context, tenantID, policyID string, _ time.Time) ([]EvaluationResult, error) {
	if m.evalError != nil && m.evalError[policyID] {
		return nil, ErrPolicyDisabled
	}
	return m.evalResults, nil
}

func (m *mockSchedulerDataSource) LastDeployedPatchIDs(_ context.Context, tenantID, policyID string) ([]string, error) {
	return m.lastPatchIDs[policyID], nil
}

type mockDeployer struct {
	createCount int
}

func (m *mockDeployer) CreateAutoDeployment(_ context.Context, tenantID, policyID string, patchIDs []string) (string, error) {
	m.createCount++
	return "deployment-" + policyID, nil
}

type mockEventEmitter struct {
	emitCount int
}

func (m *mockEventEmitter) EmitPolicyAutoDeployed(_ context.Context, tenantID, policyID, deploymentID string, newPatchCount int) {
	m.emitCount++
}
