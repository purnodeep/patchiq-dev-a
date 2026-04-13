package feeds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const cisaKEVURL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"

// CISAKEVFeed fetches and parses the CISA Known Exploited Vulnerabilities catalog.
type CISAKEVFeed struct {
	client *http.Client
}

// NewCISAKEVFeed creates a CISAKEVFeed with the given HTTP client.
// If client is nil, http.DefaultClient is used.
func NewCISAKEVFeed(client *http.Client) *CISAKEVFeed {
	if client == nil {
		client = http.DefaultClient
	}
	return &CISAKEVFeed{client: client}
}

// Name returns the feed identifier.
func (f *CISAKEVFeed) Name() string {
	return "cisa_kev"
}

// Fetch downloads the full CISA KEV catalog and returns entries newer than cursor.
// The cursor is the dateAdded (YYYY-MM-DD) of the last processed entry.
// The returned cursor is the latest dateAdded among the returned entries.
func (f *CISAKEVFeed) Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cisaKEVURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("cisa kev fetch: create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("cisa kev fetch: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("cisa kev fetch: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("cisa kev fetch: read body: %w", err)
	}

	allEntries, err := f.parse(body)
	if err != nil {
		return nil, "", fmt.Errorf("cisa kev fetch: %w", err)
	}

	var cursorTime time.Time
	if cursor != "" {
		cursorTime, err = time.Parse("2006-01-02", cursor)
		if err != nil {
			return nil, "", fmt.Errorf("cisa kev fetch: parse cursor %q: %w", cursor, err)
		}
	}

	var filtered []RawEntry
	var latestDate time.Time
	for _, entry := range allEntries {
		if !cursorTime.IsZero() && !entry.ReleaseDate.After(cursorTime) {
			continue
		}
		filtered = append(filtered, entry)
		if entry.ReleaseDate.After(latestDate) {
			latestDate = entry.ReleaseDate
		}
	}

	newCursor := cursor
	if !latestDate.IsZero() {
		newCursor = latestDate.Format("2006-01-02")
	}

	return filtered, newCursor, nil
}

// cisaKEVCatalog represents the top-level CISA KEV JSON structure.
type cisaKEVCatalog struct {
	Vulnerabilities []cisaKEVVuln `json:"vulnerabilities"`
}

// cisaKEVVuln represents a single vulnerability in the CISA KEV catalog.
type cisaKEVVuln struct {
	CVEID                      string `json:"cveID"`
	VendorProject              string `json:"vendorProject"`
	Product                    string `json:"product"`
	VulnerabilityName          string `json:"vulnerabilityName"`
	DateAdded                  string `json:"dateAdded"`
	ShortDescription           string `json:"shortDescription"`
	RequiredAction             string `json:"requiredAction"`
	DueDate                    string `json:"dueDate"`
	KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
}

// parse converts raw JSON bytes into RawEntry slices.
func (f *CISAKEVFeed) parse(data []byte) ([]RawEntry, error) {
	var catalog cisaKEVCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse cisa kev: unmarshal: %w", err)
	}

	entries := make([]RawEntry, 0, len(catalog.Vulnerabilities))
	for _, v := range catalog.Vulnerabilities {
		releaseDate, err := time.Parse("2006-01-02", v.DateAdded)
		if err != nil {
			return nil, fmt.Errorf("parse cisa kev: parse dateAdded %q for %s: %w", v.DateAdded, v.CVEID, err)
		}

		var kevDueDate *time.Time
		if v.DueDate != "" {
			t, err := time.Parse("2006-01-02", v.DueDate)
			if err != nil {
				return nil, fmt.Errorf("parse cisa kev: parse dueDate %q for %s: %w", v.DueDate, v.CVEID, err)
			}
			kevDueDate = &t
		}

		var refs []CVEReference
		if strings.EqualFold(v.KnownRansomwareCampaignUse, "Known") {
			refs = append(refs, CVEReference{
				URL:    "https://www.cisa.gov/known-exploited-vulnerabilities-catalog",
				Source: "CISA KEV ransomware: " + v.KnownRansomwareCampaignUse,
			})
		}

		entries = append(entries, RawEntry{
			CVEs:           []string{v.CVEID},
			Name:           v.VulnerabilityName,
			Vendor:         strings.ToLower(v.VendorProject),
			Product:        v.Product,
			Severity:       "critical",
			ReleaseDate:    releaseDate,
			Summary:        v.ShortDescription,
			CISAKEVDueDate: kevDueDate,
			References:     refs,
			CVEOnly:        true,
			Metadata: map[string]string{
				"ransomware":      v.KnownRansomwareCampaignUse,
				"due_date":        v.DueDate,
				"required_action": v.RequiredAction,
			},
		})
	}

	return entries, nil
}
