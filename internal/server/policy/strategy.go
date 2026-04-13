package policy

import (
	"fmt"
	"log/slog"
	"regexp"
)

// CandidatePatch represents a patch available for an endpoint.
type CandidatePatch struct {
	PatchID  string
	Name     string
	Version  string
	Severity string
	CVEs     []CVEInfo
}

// CVEInfo holds CVE data linked to a patch.
type CVEInfo struct {
	CVEID    string
	Severity string
}

// PolicyCriteria holds the filtering parameters from a policy.
type PolicyCriteria struct {
	SelectionMode   string
	MinSeverity     string
	CVEIDs          []string
	PackageRegex    string
	ExcludePackages []string
}

// SelectionStrategy filters candidate patches based on policy criteria.
type SelectionStrategy interface {
	Select(patches []CandidatePatch, criteria PolicyCriteria) []CandidatePatch
}

// StrategyFor returns the strategy implementation for the given selection mode.
func StrategyFor(mode string) (SelectionStrategy, error) {
	switch mode {
	case "all_available":
		return AllAvailableStrategy{}, nil
	case "by_severity":
		return BySeverityStrategy{}, nil
	case "by_cve_list":
		return ByCVEListStrategy{}, nil
	case "by_regex":
		return ByRegexStrategy{}, nil
	default:
		return nil, fmt.Errorf("unknown selection mode: %s", mode)
	}
}

var severityRank = map[string]int{
	"critical": 4,
	"high":     3,
	"medium":   2,
	"low":      1,
	"none":     0,
}

// AllAvailableStrategy returns all candidate patches.
type AllAvailableStrategy struct{}

// Select returns all patches without filtering.
func (AllAvailableStrategy) Select(patches []CandidatePatch, _ PolicyCriteria) []CandidatePatch {
	return patches
}

// BySeverityStrategy filters patches with CVE severity >= min_severity.
type BySeverityStrategy struct{}

// Select returns patches that have at least one CVE at or above the minimum severity.
func (BySeverityStrategy) Select(patches []CandidatePatch, criteria PolicyCriteria) []CandidatePatch {
	minRank := severityRank[criteria.MinSeverity]
	var result []CandidatePatch
	for _, p := range patches {
		for _, cve := range p.CVEs {
			if severityRank[cve.Severity] >= minRank {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// ByCVEListStrategy filters patches that fix any CVE in the specified list.
type ByCVEListStrategy struct{}

// Select returns patches that address at least one of the specified CVE IDs.
func (ByCVEListStrategy) Select(patches []CandidatePatch, criteria PolicyCriteria) []CandidatePatch {
	wanted := make(map[string]bool, len(criteria.CVEIDs))
	for _, id := range criteria.CVEIDs {
		wanted[id] = true
	}
	var result []CandidatePatch
	for _, p := range patches {
		for _, cve := range p.CVEs {
			if wanted[cve.CVEID] {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// ByRegexStrategy filters patches by name regex, minus exclusions.
type ByRegexStrategy struct{}

// Select returns patches whose name matches the regex and is not in the exclusion list.
func (ByRegexStrategy) Select(patches []CandidatePatch, criteria PolicyCriteria) []CandidatePatch {
	re, err := regexp.Compile(criteria.PackageRegex)
	if err != nil {
		slog.Error("policy regex compilation failed", "regex", criteria.PackageRegex, "error", err)
		return nil
	}
	excluded := make(map[string]bool, len(criteria.ExcludePackages))
	for _, name := range criteria.ExcludePackages {
		excluded[name] = true
	}
	var result []CandidatePatch
	for _, p := range patches {
		if re.MatchString(p.Name) && !excluded[p.Name] {
			result = append(result, p)
		}
	}
	return result
}
