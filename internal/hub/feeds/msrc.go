package feeds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const msrcBaseURL = "https://api.msrc.microsoft.com/cvrf/v3.0/updates"

// MSRCFeed fetches vulnerability data from the Microsoft Security Response Center API.
type MSRCFeed struct {
	client *http.Client
}

// NewMSRCFeed creates an MSRCFeed with the given HTTP client.
// If client is nil, a default client with a 30-second timeout is used.
func NewMSRCFeed(client *http.Client) *MSRCFeed {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &MSRCFeed{client: client}
}

// Name returns the feed identifier.
func (f *MSRCFeed) Name() string {
	return "msrc"
}

// Fetch retrieves security update entries from the MSRC API.
// The cursor is an update ID (e.g., "2024-Feb"); only updates after it are returned.
// Returns parsed entries, the next cursor (latest update ID), and any error.
func (f *MSRCFeed) Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, msrcBaseURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("msrc fetch: create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("msrc fetch: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("msrc fetch: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("msrc fetch: read response body: %w", err)
	}

	allEntries, err := f.parse(data)
	if err != nil {
		return nil, "", err
	}

	// Filter entries after cursor and determine next cursor.
	var filtered []RawEntry
	var nextCursor string
	cursorTime, cursorErr := parseMSRCUpdateID(cursor)
	if cursor != "" && cursorErr != nil {
		slog.WarnContext(ctx, "msrc fetch: invalid cursor, fetching all entries",
			"cursor", cursor, "error", cursorErr)
	}
	var nextCursorTime time.Time
	for _, entry := range allEntries {
		updateID := entry.Metadata["update_id"]
		t, err := parseMSRCUpdateID(updateID)
		if err != nil {
			slog.WarnContext(ctx, "msrc fetch: skipping entry with unparseable update_id",
				"update_id", updateID, "error", err)
			continue
		}
		if cursor != "" && !t.After(cursorTime) {
			continue
		}
		filtered = append(filtered, entry)
		if t.After(nextCursorTime) {
			nextCursorTime = t
			nextCursor = updateID
		}
	}

	if nextCursor == "" {
		nextCursor = cursor
	}

	return filtered, nextCursor, nil
}

// msrcResponse represents the top-level MSRC API response.
type msrcResponse struct {
	Value []msrcUpdate `json:"value"`
}

// msrcUpdate represents a monthly security update.
type msrcUpdate struct {
	ID                 string              `json:"ID"`
	DocumentTitle      string              `json:"DocumentTitle"`
	Severity           string              `json:"Severity"`
	InitialReleaseDate string              `json:"InitialReleaseDate"`
	CvrfURL            string              `json:"CvrfUrl"`
	Vulnerabilities    []msrcVulnerability `json:"Vulnerabilities"`
}

// msrcVulnerability represents a single vulnerability within an update.
type msrcVulnerability struct {
	CVE              string          `json:"CVE"`
	Title            string          `json:"Title"`
	Severity         string          `json:"Severity"`
	KBArticles       []msrcKBArticle `json:"KBArticles"`
	AffectedProducts []string        `json:"AffectedProducts"`
}

// msrcKBArticle represents a KB article reference.
type msrcKBArticle struct {
	ID  string `json:"ID"`
	URL string `json:"URL"`
}

// parse converts raw MSRC JSON into RawEntry slices.
func (f *MSRCFeed) parse(data []byte) ([]RawEntry, error) {
	var resp msrcResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("msrc parse: unmarshal response: %w", err)
	}

	var entries []RawEntry
	for _, update := range resp.Value {
		releaseDate, err := time.Parse(time.RFC3339, update.InitialReleaseDate)
		if err != nil {
			return nil, fmt.Errorf("msrc parse: parse release date %q for update %s: %w",
				update.InitialReleaseDate, update.ID, err)
		}

		for _, vuln := range update.Vulnerabilities {
			// Use KB article ID as the patch name (e.g., "KB5034763") if available,
			// otherwise fall back to the vulnerability title.
			patchName := vuln.Title
			if len(vuln.KBArticles) > 0 && vuln.KBArticles[0].ID != "" {
				patchName = "KB" + vuln.KBArticles[0].ID
			}

			entry := RawEntry{
				CVEs:          []string{vuln.CVE},
				Name:          patchName,
				Summary:       vuln.Title,
				Vendor:        "microsoft",
				OSFamily:      "windows",
				Severity:      strings.ToLower(vuln.Severity),
				ReleaseDate:   releaseDate,
				InstallerType: "wua",
				Metadata: map[string]string{
					"update_id": update.ID,
				},
			}

			if len(vuln.AffectedProducts) > 0 {
				entry.Product = vuln.AffectedProducts[0]
			}

			if len(vuln.KBArticles) > 0 {
				entry.SourceURL = vuln.KBArticles[0].URL
				entry.Metadata["kb_article"] = vuln.KBArticles[0].ID
			}

			for _, kb := range vuln.KBArticles {
				if kb.URL != "" {
					entry.References = append(entry.References, CVEReference{
						URL:    kb.URL,
						Source: "msrc",
					})
				}
			}

			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// parseMSRCUpdateID parses an MSRC update ID like "2025-Aug" into a time.Time
// for chronological comparison instead of broken lexicographic ordering.
func parseMSRCUpdateID(id string) (time.Time, error) {
	if id == "" {
		return time.Time{}, fmt.Errorf("msrc: empty update ID")
	}
	return time.Parse("2006-Jan", id)
}
