package policy_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/policy"
	"github.com/stretchr/testify/assert"
)

func TestAllAvailableStrategy(t *testing.T) {
	s := policy.AllAvailableStrategy{}
	patches := []policy.CandidatePatch{
		{PatchID: "p1", Name: "libssl", Version: "1.1.1k-1"},
		{PatchID: "p2", Name: "curl", Version: "7.74.0-1"},
	}
	result := s.Select(patches, policy.PolicyCriteria{})
	assert.Len(t, result, 2, "all_available returns all patches")
}

func TestBySeverityStrategy(t *testing.T) {
	s := policy.BySeverityStrategy{}
	patches := []policy.CandidatePatch{
		{PatchID: "p1", Name: "libssl", CVEs: []policy.CVEInfo{{CVEID: "CVE-2024-001", Severity: "critical"}}},
		{PatchID: "p2", Name: "curl", CVEs: []policy.CVEInfo{{CVEID: "CVE-2024-002", Severity: "low"}}},
		{PatchID: "p3", Name: "zlib", CVEs: []policy.CVEInfo{{CVEID: "CVE-2024-003", Severity: "high"}}},
		{PatchID: "p4", Name: "nopatch"},
	}

	tests := []struct {
		name        string
		minSeverity string
		wantIDs     []string
	}{
		{"critical only", "critical", []string{"p1"}},
		{"high and above", "high", []string{"p1", "p3"}},
		{"medium and above", "medium", []string{"p1", "p3"}},
		{"low and above includes all with CVEs", "low", []string{"p1", "p2", "p3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Select(patches, policy.PolicyCriteria{MinSeverity: tt.minSeverity})
			gotIDs := make([]string, len(result))
			for i, p := range result {
				gotIDs[i] = p.PatchID
			}
			assert.ElementsMatch(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestByCVEListStrategy(t *testing.T) {
	s := policy.ByCVEListStrategy{}
	patches := []policy.CandidatePatch{
		{PatchID: "p1", Name: "libssl", CVEs: []policy.CVEInfo{{CVEID: "CVE-2024-001"}}},
		{PatchID: "p2", Name: "curl", CVEs: []policy.CVEInfo{{CVEID: "CVE-2024-002"}, {CVEID: "CVE-2024-003"}}},
		{PatchID: "p3", Name: "zlib", CVEs: []policy.CVEInfo{{CVEID: "CVE-2024-004"}}},
	}

	tests := []struct {
		name    string
		cveIDs  []string
		wantIDs []string
	}{
		{"matches single CVE", []string{"CVE-2024-001"}, []string{"p1"}},
		{"matches patch with multiple CVEs", []string{"CVE-2024-003"}, []string{"p2"}},
		{"matches multiple patches", []string{"CVE-2024-001", "CVE-2024-004"}, []string{"p1", "p3"}},
		{"no matches", []string{"CVE-9999-999"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Select(patches, policy.PolicyCriteria{CVEIDs: tt.cveIDs})
			gotIDs := make([]string, len(result))
			for i, p := range result {
				gotIDs[i] = p.PatchID
			}
			assert.ElementsMatch(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestByRegexStrategy(t *testing.T) {
	s := policy.ByRegexStrategy{}
	patches := []policy.CandidatePatch{
		{PatchID: "p1", Name: "libssl1.1"},
		{PatchID: "p2", Name: "curl"},
		{PatchID: "p3", Name: "libssl3"},
		{PatchID: "p4", Name: "zlib"},
	}

	tests := []struct {
		name            string
		regex           string
		excludePackages []string
		wantIDs         []string
	}{
		{"matches regex", "^libssl", nil, []string{"p1", "p3"}},
		{"matches regex with exclusion", "^libssl", []string{"libssl3"}, []string{"p1"}},
		{"no matches", "^nginx", nil, nil},
		{"match all", ".*", nil, []string{"p1", "p2", "p3", "p4"}},
		{"match all with exclusions", ".*", []string{"curl", "zlib"}, []string{"p1", "p3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.Select(patches, policy.PolicyCriteria{
				PackageRegex:    tt.regex,
				ExcludePackages: tt.excludePackages,
			})
			gotIDs := make([]string, len(result))
			for i, p := range result {
				gotIDs[i] = p.PatchID
			}
			assert.ElementsMatch(t, tt.wantIDs, gotIDs)
		})
	}
}

func TestByRegexStrategy_InvalidRegex(t *testing.T) {
	s := policy.ByRegexStrategy{}
	patches := []policy.CandidatePatch{
		{PatchID: "p1", Name: "libssl"},
	}
	result := s.Select(patches, policy.PolicyCriteria{PackageRegex: "[invalid"})
	assert.Empty(t, result, "invalid regex returns no matches")
}

func TestStrategyFor(t *testing.T) {
	tests := []struct {
		mode    string
		wantErr bool
	}{
		{"all_available", false},
		{"by_severity", false},
		{"by_cve_list", false},
		{"by_regex", false},
		{"unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			s, err := policy.StrategyFor(tt.mode)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, s)
			}
		})
	}
}
