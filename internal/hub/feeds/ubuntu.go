package feeds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const ubuntuUSNBaseURL = "https://ubuntu.com/security/notices.json"

// ubuntuPublishedFormats are the timestamp formats the Ubuntu API may return.
var ubuntuPublishedFormats = []string{
	"2006-01-02T15:04:05.999999",
	time.RFC3339,
	"2006-01-02T15:04:05",
}

// usnResponse represents the top-level Ubuntu USN API response.
// The API uses the "notices" key (not "entries").
type usnResponse struct {
	Notices []usnEntry `json:"notices"`
}

// usnCVERef represents a CVE reference object returned by the Ubuntu API.
type usnCVERef struct {
	ID       string `json:"id"`
	Priority string `json:"priority"`
}

// usnEntry represents a single Ubuntu Security Notice from the Ubuntu API.
type usnEntry struct {
	// ID is a string like "USN-8107-1", not an integer.
	ID              string              `json:"id"`
	Title           string              `json:"title"`
	Summary         string              `json:"summary"`
	Description     string              `json:"description"`
	Published       string              `json:"published"`
	CVEs            []usnCVERef         `json:"cves"`
	ReleasePackages map[string][]usnPkg `json:"release_packages"`
}

// usnPkg represents a package in a release-specific section of a USN entry.
type usnPkg struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// UbuntuFeed fetches vulnerability data from the Ubuntu USN API.
type UbuntuFeed struct {
	client *http.Client
}

// NewUbuntuFeed creates an UbuntuFeed with the given HTTP client.
// If client is nil, a default client with a 30-second timeout is used.
func NewUbuntuFeed(client *http.Client) *UbuntuFeed {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &UbuntuFeed{client: client}
}

// Name returns the feed identifier.
func (f *UbuntuFeed) Name() string {
	return "ubuntu_usn"
}

// Fetch retrieves USN entries from the Ubuntu API starting from the given cursor.
// The cursor is an RFC3339 timestamp; only entries published after it are returned.
// Returns parsed entries, the next cursor (latest published timestamp), and any error.
func (f *UbuntuFeed) Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ubuntuUSNBaseURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("ubuntu usn fetch: create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("ubuntu usn fetch: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("ubuntu usn fetch: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("ubuntu usn fetch: read response body: %w", err)
	}

	entries, err := f.parse(data)
	if err != nil {
		return nil, "", err
	}

	// Parse cursor as a time for filtering.
	var cursorTime time.Time
	if cursor != "" {
		cursorTime, err = time.Parse(time.RFC3339, cursor)
		if err != nil {
			return nil, "", fmt.Errorf("ubuntu usn fetch: parse cursor %q: %w", cursor, err)
		}
	}

	var filtered []RawEntry
	var maxPublished time.Time
	for _, entry := range entries {
		if !entry.ReleaseDate.IsZero() && entry.ReleaseDate.After(maxPublished) {
			maxPublished = entry.ReleaseDate
		}
		if cursorTime.IsZero() || entry.ReleaseDate.After(cursorTime) {
			filtered = append(filtered, entry)
		}
	}

	var nextCursor string
	if !maxPublished.IsZero() {
		nextCursor = maxPublished.UTC().Format(time.RFC3339)
	}

	return filtered, nextCursor, nil
}

// parseUbuntuTime tries to parse a timestamp from the Ubuntu API.
func parseUbuntuTime(s string) (time.Time, error) {
	for _, layout := range ubuntuPublishedFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("ubuntu usn: unparseable timestamp %q", s)
}

// parse converts raw Ubuntu USN JSON into RawEntry slices.
func (f *UbuntuFeed) parse(data []byte) ([]RawEntry, error) {
	var resp usnResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("ubuntu usn parse: unmarshal response: %w", err)
	}

	entries := make([]RawEntry, 0, len(resp.Notices))
	for _, usn := range resp.Notices {
		// Extract CVE IDs from the reference objects.
		cveIDs := make([]string, 0, len(usn.CVEs))
		for _, c := range usn.CVEs {
			if c.ID != "" {
				cveIDs = append(cveIDs, c.ID)
			}
		}

		// Use summary if description is empty.
		desc := usn.Description
		if desc == "" {
			desc = usn.Summary
		}

		// Build References: USN advisory URL + per-CVE URLs.
		refs := make([]CVEReference, 0, 1+len(cveIDs))
		if usn.ID != "" {
			refs = append(refs, CVEReference{
				URL:    "https://ubuntu.com/security/notices/" + usn.ID,
				Source: "ubuntu",
			})
		}
		for _, cveID := range cveIDs {
			refs = append(refs, CVEReference{
				URL:    "https://ubuntu.com/security/" + cveID,
				Source: "ubuntu",
			})
		}

		entry := RawEntry{
			CVEs:          cveIDs,
			Name:          usn.Title,
			Vendor:        "canonical",
			Severity:      deriveSeverity(usn.CVEs),
			OSFamily:      "linux",
			InstallerType: "deb",
			Summary:       desc,
			Metadata:      map[string]string{"usn_id": usn.ID},
			References:    refs,
		}

		// Published date — Ubuntu API returns without timezone.
		if usn.Published != "" {
			t, err := parseUbuntuTime(usn.Published)
			if err != nil {
				// Log-warn only: don't abort the whole fetch for a single bad timestamp.
				continue
			}
			entry.ReleaseDate = t
		}

		// Extract product and version from the first package of the first release
		// (sorted for determinism).
		releaseNames := make([]string, 0, len(usn.ReleasePackages))
		for name := range usn.ReleasePackages {
			releaseNames = append(releaseNames, name)
		}
		sort.Strings(releaseNames)

		if len(releaseNames) > 0 {
			pkgs := usn.ReleasePackages[releaseNames[0]]
			if len(pkgs) > 0 {
				entry.Product = pkgs[0].Name
				entry.Version = pkgs[0].Version
			}
		}

		// OS versions: release codenames (sorted for determinism).
		entry.OSVersions = releaseNames

		entries = append(entries, entry)
	}

	return entries, nil
}

// severityRank maps severity strings to a numeric rank for comparison.
var severityRank = map[string]int{
	"negligible": 1,
	"low":        2,
	"medium":     3,
	"high":       4,
	"critical":   5,
}

// deriveSeverity returns the highest severity from the CVE priority fields.
// Falls back to "medium" if no priorities are present.
func deriveSeverity(cves []usnCVERef) string {
	best := ""
	bestRank := 0
	for _, cve := range cves {
		p := strings.ToLower(cve.Priority)
		if r, ok := severityRank[p]; ok && r > bestRank {
			bestRank = r
			best = p
		}
	}
	if best == "" {
		return "medium"
	}
	return best
}
