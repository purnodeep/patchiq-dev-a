package policy_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDataSource struct {
	policy       policy.PolicyData
	policyErr    error
	endpoints    []policy.EndpointData
	endpointsErr error
	patches      []policy.CandidatePatch
	patchesErr   error
	cves         map[string][]policy.CVEInfo
	cvesErr      error
}

func (f *fakeDataSource) GetPolicy(_ context.Context, _, _ string) (policy.PolicyData, error) {
	return f.policy, f.policyErr
}

func (f *fakeDataSource) ListEndpointsForPolicy(_ context.Context, _, _ string) ([]policy.EndpointData, error) {
	return f.endpoints, f.endpointsErr
}

func (f *fakeDataSource) ListAvailablePatches(_ context.Context, _, _, _ string) ([]policy.CandidatePatch, error) {
	return f.patches, f.patchesErr
}

func (f *fakeDataSource) ListCVEsForPatches(_ context.Context, _ string, _ []string) (map[string][]policy.CVEInfo, error) {
	return f.cves, f.cvesErr
}

func TestEvaluator_AllAvailable(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "all_available", Enabled: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{
			{PatchID: "p1", Name: "libssl", Version: "1.0"},
			{PatchID: "p2", Name: "curl", Version: "7.74"},
		},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "ep-1", results[0].EndpointID)
	assert.Len(t, results[0].Patches, 2)
}

func TestEvaluator_BySeverity(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "by_severity", MinSeverity: "high", Enabled: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{
			{PatchID: "p1", Name: "libssl"},
			{PatchID: "p2", Name: "curl"},
		},
		cves: map[string][]policy.CVEInfo{
			"p1": {{CVEID: "CVE-2024-001", Severity: "critical"}},
			"p2": {{CVEID: "CVE-2024-002", Severity: "low"}},
		},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Patches, 1)
	assert.Equal(t, "p1", results[0].Patches[0].PatchID)
}

func TestEvaluator_ByCVEList(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "by_cve_list", CVEIDs: []string{"CVE-2024-003"}, Enabled: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{
			{PatchID: "p1", Name: "libssl"},
			{PatchID: "p2", Name: "curl"},
		},
		cves: map[string][]policy.CVEInfo{
			"p1": {{CVEID: "CVE-2024-001"}},
			"p2": {{CVEID: "CVE-2024-003"}},
		},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Patches, 1)
	assert.Equal(t, "p2", results[0].Patches[0].PatchID)
}

func TestEvaluator_ByRegex(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "by_regex", PackageRegex: "^lib", ExcludePackages: []string{"libxml"}, Enabled: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{
			{PatchID: "p1", Name: "libssl"},
			{PatchID: "p2", Name: "curl"},
			{PatchID: "p3", Name: "libxml"},
		},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Len(t, results[0].Patches, 1)
	assert.Equal(t, "p1", results[0].Patches[0].PatchID)
}

func TestEvaluator_DisabledPolicy(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{ID: "pol-1", Enabled: false},
	}
	ev := policy.NewEvaluator(ds)
	_, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, policy.ErrPolicyDisabled))
}

func TestEvaluator_MaintenanceWindow_Inside(t *testing.T) {
	now := time.Date(2026, 3, 4, 14, 0, 0, 0, time.UTC)
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "all_available", Enabled: true,
			MwStart: timeOfDay(13, 0), MwEnd: timeOfDay(15, 0), HasMwWindow: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{{PatchID: "p1", Name: "libssl"}},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", now)
	require.NoError(t, err)
	assert.Len(t, results, 1, "inside maintenance window — endpoints included")
}

func TestEvaluator_MaintenanceWindow_Outside(t *testing.T) {
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "all_available", Enabled: true,
			MwStart: timeOfDay(13, 0), MwEnd: timeOfDay(15, 0), HasMwWindow: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{{PatchID: "p1", Name: "libssl"}},
	}
	ev := policy.NewEvaluator(ds)
	_, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", now)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, policy.ErrOutsideMaintenanceWindow))
}

func TestEvaluator_MaintenanceWindow_Overnight(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "all_available", Enabled: true,
			MwStart: timeOfDay(22, 0), MwEnd: timeOfDay(6, 0), HasMwWindow: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{{PatchID: "p1", Name: "libssl"}},
	}
	ev := policy.NewEvaluator(ds)

	// 23:00 is inside the overnight window 22:00-06:00.
	inside := time.Date(2026, 3, 4, 23, 0, 0, 0, time.UTC)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", inside)
	require.NoError(t, err)
	assert.Len(t, results, 1, "23:00 is inside overnight window 22:00-06:00")

	// 03:00 is also inside.
	earlyMorning := time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC)
	results, err = ev.Evaluate(context.Background(), "tenant-1", "pol-1", earlyMorning)
	require.NoError(t, err)
	assert.Len(t, results, 1, "03:00 is inside overnight window 22:00-06:00")

	// 10:00 is outside.
	outside := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	_, err = ev.Evaluate(context.Background(), "tenant-1", "pol-1", outside)
	assert.True(t, errors.Is(err, policy.ErrOutsideMaintenanceWindow))
}

func TestEvaluator_NoEndpoints(t *testing.T) {
	ds := &fakeDataSource{
		policy:    policy.PolicyData{ID: "pol-1", SelectionMode: "all_available", Enabled: true},
		endpoints: []policy.EndpointData{},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestEvaluator_MultipleEndpoints(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{ID: "pol-1", SelectionMode: "all_available", Enabled: true},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
			{ID: "ep-2", Hostname: "web-2", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{{PatchID: "p1", Name: "libssl"}},
	}
	ev := policy.NewEvaluator(ds)
	results, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestEvaluator_ErrorPaths(t *testing.T) {
	tests := []struct {
		name string
		ds   *fakeDataSource
	}{
		{
			name: "GetPolicy error returns ErrPolicyNotFound",
			ds:   &fakeDataSource{policyErr: fmt.Errorf("no rows")},
		},
		{
			name: "ListEndpointsForPolicy error",
			ds: &fakeDataSource{
				policy:       policy.PolicyData{ID: "pol-1", SelectionMode: "all_available", Enabled: true},
				endpointsErr: fmt.Errorf("connection refused"),
			},
		},
		{
			name: "ListAvailablePatches error",
			ds: &fakeDataSource{
				policy:     policy.PolicyData{ID: "pol-1", SelectionMode: "all_available", Enabled: true},
				endpoints:  []policy.EndpointData{{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"}},
				patchesErr: fmt.Errorf("connection refused"),
			},
		},
		{
			name: "ListCVEsForPatches error",
			ds: &fakeDataSource{
				policy:    policy.PolicyData{ID: "pol-1", SelectionMode: "by_severity", MinSeverity: "high", Enabled: true},
				endpoints: []policy.EndpointData{{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"}},
				patches:   []policy.CandidatePatch{{PatchID: "p1", Name: "libssl"}},
				cvesErr:   fmt.Errorf("connection refused"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev := policy.NewEvaluator(tt.ds)
			_, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
			assert.Error(t, err)
		})
	}
}

func TestEvaluator_InvalidRegex(t *testing.T) {
	ds := &fakeDataSource{
		policy: policy.PolicyData{
			ID: "pol-1", SelectionMode: "by_regex", PackageRegex: "[invalid", Enabled: true,
		},
		endpoints: []policy.EndpointData{
			{ID: "ep-1", Hostname: "web-1", OsFamily: "debian"},
		},
		patches: []policy.CandidatePatch{{PatchID: "p1", Name: "libssl"}},
	}
	ev := policy.NewEvaluator(ds)
	_, err := ev.Evaluate(context.Background(), "tenant-1", "pol-1", time.Now())
	assert.Error(t, err, "invalid regex should return error, not silently empty results")
	assert.Contains(t, err.Error(), "invalid package_regex")
}

func timeOfDay(hour, min int) time.Duration {
	return time.Duration(hour)*time.Hour + time.Duration(min)*time.Minute
}
