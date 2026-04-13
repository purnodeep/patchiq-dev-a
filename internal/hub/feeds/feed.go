package feeds

import (
	"context"
	"fmt"
	"time"
)

// Feed represents a vendor vulnerability/patch data source.
type Feed interface {
	Name() string
	Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error)
}

// CVEReference is a URL reference associated with a CVE from a feed source.
type CVEReference struct {
	URL    string
	Source string
}

// RawEntry is the vendor-agnostic intermediate representation.
type RawEntry struct {
	CVEs            []string
	Name            string
	Vendor          string
	Product         string
	Version         string
	Severity        string
	CVSSScore       float64
	CVSSv3Vector    string
	AttackVector    string
	CweID           string
	CISAKEVDueDate  *time.Time
	References      []CVEReference
	NVDLastModified *time.Time
	OSFamily        string
	OSVersions      []string
	InstallerType   string
	ReleaseDate     time.Time
	Summary         string
	SourceURL       string
	SilentArgs      string
	Metadata        map[string]string
	// CVEOnly marks entries from vulnerability-only feeds (NVD, CISA KEV).
	// These entries contribute CVE data but should NOT create patch catalog entries.
	CVEOnly bool
}

var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
	"none":     true,
}

// severityAliases maps vendor-specific severity labels to canonical values.
var severityAliases = map[string]string{
	"important":     "high",
	"moderate":      "medium",
	"negligible":    "low",
	"informational": "none",
}

// Validate checks that required fields are present and values are valid.
// It normalizes vendor-specific severity labels (e.g. "important" → "high").
func (e *RawEntry) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("validate raw entry: name is required")
	}
	if e.Vendor == "" {
		return fmt.Errorf("validate raw entry %q: vendor is required", e.Name)
	}
	if e.Severity != "" {
		if alias, ok := severityAliases[e.Severity]; ok {
			e.Severity = alias
		}
		if !validSeverities[e.Severity] {
			return fmt.Errorf("validate raw entry %q: invalid severity %q", e.Name, e.Severity)
		}
	}
	return nil
}
