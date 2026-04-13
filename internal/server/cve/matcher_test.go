package cve

import (
	"context"
	"testing"
	"time"
)

type mockEndpointPackageLister struct {
	packages []EndpointPackage
}

func (m *mockEndpointPackageLister) ListEndpointPackages(ctx context.Context, tenantID, endpointID string) ([]EndpointPackage, error) {
	return m.packages, nil
}

type mockCVELookup struct {
	cves map[string][]MatchableCVE
}

func (m *mockCVELookup) ListCVEsForPackage(ctx context.Context, tenantID, packageName string) ([]MatchableCVE, error) {
	return m.cves[packageName], nil
}

type mockEndpointCVEUpserter struct {
	upserted []EndpointCVERecord
}

func (m *mockEndpointCVEUpserter) UpsertEndpointCVE(ctx context.Context, tenantID string, rec EndpointCVERecord) error {
	m.upserted = append(m.upserted, rec)
	return nil
}

func TestMatcher_MatchEndpoint(t *testing.T) {
	now := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	pkgLister := &mockEndpointPackageLister{
		packages: []EndpointPackage{
			{Name: "openssl", Version: "3.0.2"},
			{Name: "curl", Version: "8.6.0"},
		},
	}
	cveLookup := &mockCVELookup{
		cves: map[string][]MatchableCVE{
			"openssl": {{
				CVEDBID: "db-1", CVEID: "CVE-2024-1234", CVSSv3Score: 9.8,
				CISAKev: true, PublishedAt: now.AddDate(0, 0, -120),
				VersionEndExcluding: "3.0.13",
			}},
			"curl": {{
				CVEDBID: "db-2", CVEID: "CVE-2024-5678", CVSSv3Score: 7.4,
				PublishedAt:         now.AddDate(0, 0, -10),
				VersionEndExcluding: "8.6.0", // endpoint has 8.6.0, NOT affected
			}},
		},
	}
	upserter := &mockEndpointCVEUpserter{}
	matcher := NewMatcher(pkgLister, cveLookup, upserter)

	result, err := matcher.MatchEndpoint(context.Background(), "tenant-1", "endpoint-1", "", now)
	if err != nil {
		t.Fatalf("MatchEndpoint: %v", err)
	}
	if result.Affected != 1 {
		t.Errorf("Affected = %d, want 1", result.Affected)
	}
	if result.Patched != 1 {
		t.Errorf("Patched = %d, want 1", result.Patched)
	}
	if len(upserter.upserted) != 2 {
		t.Fatalf("expected 2 upserts, got %d", len(upserter.upserted))
	}
	for _, rec := range upserter.upserted {
		if rec.Status == StatusAffected && rec.RiskScore != 10.0 {
			t.Errorf("risk_score = %.2f, want 10.0 (capped)", rec.RiskScore)
		}
	}
}

func TestMatcher_MatchEndpoint_NoPackages(t *testing.T) {
	pkgLister := &mockEndpointPackageLister{packages: nil}
	cveLookup := &mockCVELookup{cves: map[string][]MatchableCVE{}}
	upserter := &mockEndpointCVEUpserter{}
	matcher := NewMatcher(pkgLister, cveLookup, upserter)

	result, err := matcher.MatchEndpoint(context.Background(), "tenant-1", "endpoint-1", "", time.Now())
	if err != nil {
		t.Fatalf("MatchEndpoint: %v", err)
	}
	if result.Affected != 0 || result.Patched != 0 {
		t.Errorf("expected 0/0, got %d/%d", result.Affected, result.Patched)
	}
}

type mockOsFamilyCVELookup struct {
	cves map[string][]MatchableCVE
}

func (m *mockOsFamilyCVELookup) ListCVEsByOsFamily(ctx context.Context, tenantID, osFamily string) ([]MatchableCVE, error) {
	return m.cves[osFamily], nil
}

func TestMatcher_MatchEndpoint_WindowsOsFamily(t *testing.T) {
	now := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	// Windows endpoint: no packages that map to CVEs via package names.
	pkgLister := &mockEndpointPackageLister{
		packages: []EndpointPackage{
			{Name: "KB5079473", Version: ""},
		},
	}
	cveLookup := &mockCVELookup{cves: map[string][]MatchableCVE{}}
	osFamilyLookup := &mockOsFamilyCVELookup{
		cves: map[string][]MatchableCVE{
			"windows": {
				{CVEDBID: "db-win-1", CVEID: "CVE-2024-9999", CVSSv3Score: 8.1, PublishedAt: now.AddDate(0, 0, -30)},
				{CVEDBID: "db-win-2", CVEID: "CVE-2024-8888", CVSSv3Score: 6.5, CISAKev: true, PublishedAt: now.AddDate(0, 0, -60)},
			},
		},
	}
	upserter := &mockEndpointCVEUpserter{}
	matcher := NewMatcher(pkgLister, cveLookup, upserter).WithOsFamilyLookup(osFamilyLookup)

	result, err := matcher.MatchEndpoint(context.Background(), "tenant-1", "endpoint-win", "windows", now)
	if err != nil {
		t.Fatalf("MatchEndpoint: %v", err)
	}
	if result.Affected != 2 {
		t.Errorf("Affected = %d, want 2", result.Affected)
	}
	if result.Patched != 0 {
		t.Errorf("Patched = %d, want 0", result.Patched)
	}
	if len(upserter.upserted) != 2 {
		t.Fatalf("expected 2 upserts, got %d", len(upserter.upserted))
	}
	for _, rec := range upserter.upserted {
		if rec.Status != StatusAffected {
			t.Errorf("expected status %q, got %q", StatusAffected, rec.Status)
		}
		if rec.RiskScore <= 0 {
			t.Errorf("expected positive risk score, got %.2f for CVE DB ID %s", rec.RiskScore, rec.CVEDBID)
		}
	}
}

func TestMatcher_MatchEndpoint_OsFamilyLookupNilSkipped(t *testing.T) {
	now := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	pkgLister := &mockEndpointPackageLister{packages: nil}
	cveLookup := &mockCVELookup{cves: map[string][]MatchableCVE{}}
	upserter := &mockEndpointCVEUpserter{}
	// No osFamilyLookup set — os-family matching must be skipped even when osFamily is set.
	matcher := NewMatcher(pkgLister, cveLookup, upserter)

	result, err := matcher.MatchEndpoint(context.Background(), "tenant-1", "endpoint-win", "windows", now)
	if err != nil {
		t.Fatalf("MatchEndpoint: %v", err)
	}
	if result.Affected != 0 {
		t.Errorf("Affected = %d, want 0 (osFamilyLookup not set)", result.Affected)
	}
}

func TestIsVersionAffected(t *testing.T) {
	tests := []struct {
		name                string
		installed           string
		versionEndExcluding string
		versionEndIncluding string
		want                bool
	}{
		{"below end excluding", "3.0.2", "3.0.13", "", true},
		{"equal to end excluding", "3.0.13", "3.0.13", "", false},
		{"above end excluding", "3.1.0", "3.0.13", "", false},
		{"below end including", "3.0.12", "", "3.0.13", true},
		{"equal to end including", "3.0.13", "", "3.0.13", true},
		{"above end including", "3.1.0", "", "3.0.13", false},
		{"no version constraint", "3.0.2", "", "", true},
		{"epoch version affected", "1:3.0.2", "1:3.0.13", "", true},
		{"epoch version not affected", "1:3.1.0", "1:3.0.13", "", false},
		{"higher epoch wins", "1:9.0", "2:1.0", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsVersionAffected(tt.installed, tt.versionEndExcluding, tt.versionEndIncluding)
			if got != tt.want {
				t.Errorf("IsVersionAffected(%q, %q, %q) = %v, want %v",
					tt.installed, tt.versionEndExcluding, tt.versionEndIncluding, got, tt.want)
			}
		})
	}
}
