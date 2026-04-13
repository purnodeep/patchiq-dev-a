package cve

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	StatusAffected = "affected"
	StatusPatched  = "patched"
)

// EndpointPackage represents an installed package on an endpoint.
type EndpointPackage struct {
	Name    string
	Version string
}

// MatchableCVE holds the CVE fields needed for version-range matching and risk scoring.
type MatchableCVE struct {
	CVEDBID             string
	CVEID               string
	CVSSv3Score         float64
	CISAKev             bool
	ExploitAvailable    bool
	PublishedAt         time.Time
	VersionEndExcluding string
	VersionEndIncluding string
}

// EndpointCVERecord is the result of matching a single CVE against an endpoint.
type EndpointCVERecord struct {
	EndpointID string
	CVEDBID    string
	Status     string // "affected" or "patched"
	RiskScore  float64
	DetectedAt time.Time
}

// MatchResult summarises how many CVEs affect or are patched on an endpoint.
type MatchResult struct {
	Affected int
	Patched  int
}

// EndpointPackageLister retrieves installed packages for an endpoint.
type EndpointPackageLister interface {
	ListEndpointPackages(ctx context.Context, tenantID, endpointID string) ([]EndpointPackage, error)
}

// CVELookup retrieves CVEs that may affect a given package name.
type CVELookup interface {
	ListCVEsForPackage(ctx context.Context, tenantID, packageName string) ([]MatchableCVE, error)
}

// OsFamilyCVELookup retrieves CVEs relevant to an OS family via description keyword matching.
type OsFamilyCVELookup interface {
	ListCVEsByOsFamily(ctx context.Context, tenantID, osFamily string) ([]MatchableCVE, error)
}

// EndpointCVEUpserter persists an endpoint-CVE association.
type EndpointCVEUpserter interface {
	UpsertEndpointCVE(ctx context.Context, tenantID string, rec EndpointCVERecord) error
}

// Matcher correlates installed packages with known CVEs.
type Matcher struct {
	pkgLister      EndpointPackageLister
	cveLookup      CVELookup
	upserter       EndpointCVEUpserter
	osFamilyLookup OsFamilyCVELookup
}

// NewMatcher creates a Matcher with the required dependencies.
func NewMatcher(pkgLister EndpointPackageLister, cveLookup CVELookup, upserter EndpointCVEUpserter) *Matcher {
	return &Matcher{pkgLister: pkgLister, cveLookup: cveLookup, upserter: upserter}
}

// WithOsFamilyLookup configures optional OS-family-based CVE matching (e.g. for Windows endpoints).
func (m *Matcher) WithOsFamilyLookup(l OsFamilyCVELookup) *Matcher {
	m.osFamilyLookup = l
	return m
}

// MatchEndpoint checks every installed package on an endpoint against known CVEs,
// upserts the results, and returns aggregate counts.
// osFamily is used for OS-family-based CVE matching (pass "" to skip).
func (m *Matcher) MatchEndpoint(ctx context.Context, tenantID, endpointID, osFamily string, now time.Time) (MatchResult, error) {
	packages, err := m.pkgLister.ListEndpointPackages(ctx, tenantID, endpointID)
	if err != nil {
		return MatchResult{}, err
	}

	var result MatchResult
	for _, pkg := range packages {
		cves, err := m.cveLookup.ListCVEsForPackage(ctx, tenantID, pkg.Name)
		if err != nil {
			return MatchResult{}, fmt.Errorf("matcher: list CVEs for package %q: %w", pkg.Name, err)
		}
		for _, cve := range cves {
			affected := IsVersionAffected(pkg.Version, cve.VersionEndExcluding, cve.VersionEndIncluding)
			status := StatusPatched
			var riskScore float64
			if affected {
				status = StatusAffected
				riskScore = ComputeRiskScore(cve.CVSSv3Score, cve.CISAKev, cve.ExploitAvailable, cve.PublishedAt, now)
				result.Affected++
			} else {
				result.Patched++
			}
			rec := EndpointCVERecord{
				EndpointID: endpointID,
				CVEDBID:    cve.CVEDBID,
				Status:     status,
				RiskScore:  riskScore,
				DetectedAt: now,
			}
			if err := m.upserter.UpsertEndpointCVE(ctx, tenantID, rec); err != nil {
				return MatchResult{}, fmt.Errorf("matcher: upsert endpoint CVE %q for endpoint %q: %w", cve.CVEID, endpointID, err)
			}
		}
	}

	// OS-family-based matching (for Windows endpoints where package names don't map to CPE names).
	if osFamily != "" && m.osFamilyLookup != nil {
		osCVEs, err := m.osFamilyLookup.ListCVEsByOsFamily(ctx, tenantID, osFamily)
		if err != nil {
			return MatchResult{}, fmt.Errorf("matcher: list CVEs by os family %q: %w", osFamily, err)
		}
		for _, cve := range osCVEs {
			// No version-range data for description-based matches; treat all as affected.
			riskScore := ComputeRiskScore(cve.CVSSv3Score, cve.CISAKev, cve.ExploitAvailable, cve.PublishedAt, now)
			result.Affected++
			rec := EndpointCVERecord{
				EndpointID: endpointID,
				CVEDBID:    cve.CVEDBID,
				Status:     StatusAffected,
				RiskScore:  riskScore,
				DetectedAt: now,
			}
			if err := m.upserter.UpsertEndpointCVE(ctx, tenantID, rec); err != nil {
				return MatchResult{}, fmt.Errorf("matcher: upsert os-family endpoint CVE %q for endpoint %q: %w", cve.CVEID, endpointID, err)
			}
		}
	}

	return result, nil
}

// IsVersionAffected returns true when the installed version falls within the
// vulnerable range defined by versionEndExcluding (exclusive upper bound) or
// versionEndIncluding (inclusive upper bound). If neither bound is set the
// package is assumed affected.
func IsVersionAffected(installed, versionEndExcluding, versionEndIncluding string) bool {
	if versionEndExcluding == "" && versionEndIncluding == "" {
		return true
	}
	if versionEndExcluding != "" {
		return compareVersions(installed, versionEndExcluding) < 0
	}
	return compareVersions(installed, versionEndIncluding) <= 0
}

// compareVersions does a segment-by-segment numeric+string comparison.
// It handles epoch prefixes (e.g. "1:3.0.2") by splitting on ":" to
// extract the epoch (defaulting to 0), comparing epochs first, then
// comparing dot-separated version segments.
func compareVersions(a, b string) int {
	aEpoch, aVer := splitEpoch(a)
	bEpoch, bVer := splitEpoch(b)
	if aEpoch != bEpoch {
		if aEpoch < bEpoch {
			return -1
		}
		return 1
	}
	aParts := strings.Split(aVer, ".")
	bParts := strings.Split(bVer, ".")
	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}
	for i := 0; i < maxLen; i++ {
		var aSegment, bSegment string
		if i < len(aParts) {
			aSegment = aParts[i]
		}
		if i < len(bParts) {
			bSegment = bParts[i]
		}
		cmp := compareSegments(aSegment, bSegment)
		if cmp != 0 {
			return cmp
		}
	}
	return 0
}

func compareSegments(a, b string) int {
	aNum, aRest := splitNumeric(a)
	bNum, bRest := splitNumeric(b)
	if aNum != bNum {
		if aNum < bNum {
			return -1
		}
		return 1
	}
	if aRest < bRest {
		return -1
	}
	if aRest > bRest {
		return 1
	}
	return 0
}

func splitEpoch(v string) (int, string) {
	if idx := strings.IndexByte(v, ':'); idx >= 0 {
		epoch, err := strconv.Atoi(v[:idx])
		if err != nil {
			return 0, v
		}
		return epoch, v[idx+1:]
	}
	return 0, v
}

func splitNumeric(s string) (int, string) {
	var num int
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		num = num*10 + int(s[i]-'0')
		i++
	}
	return num, s[i:]
}
