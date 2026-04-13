package feeds

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const appleBaseURL = "https://support.apple.com/en-us/100100"

// AppleFeed fetches security release data from Apple.
type AppleFeed struct {
	client *http.Client
}

// NewAppleFeed creates an AppleFeed with the given HTTP client.
// If client is nil, a default client with a 30-second timeout is used.
func NewAppleFeed(client *http.Client) *AppleFeed {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &AppleFeed{client: client}
}

// Name returns the feed identifier.
func (f *AppleFeed) Name() string {
	return "apple"
}

// Fetch retrieves security release entries from the Apple feed.
// The cursor is the release date of the last processed entry (format: "02 Jan 2006").
// Only entries newer than the cursor are returned.
// Returns parsed entries, the next cursor (latest release date), and any error.
func (f *AppleFeed) Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleBaseURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("apple fetch: create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("apple fetch: execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("apple fetch: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("apple fetch: read response body: %w", err)
	}

	// Strip UTF-8 BOM if present (\xEF\xBB\xBF).
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// The Apple page may return HTML instead of JSON. Detect and skip gracefully.
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, cursor, nil
	}

	allEntries, err := f.parse(data)
	if err != nil {
		return nil, "", err
	}

	var cursorTime time.Time
	if cursor != "" {
		cursorTime, err = time.Parse("02 Jan 2006", cursor)
		if err != nil {
			return nil, "", fmt.Errorf("apple fetch: parse cursor %q: %w", cursor, err)
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

	nextCursor := cursor
	if !latestDate.IsZero() {
		nextCursor = latestDate.Format("02 Jan 2006")
	}

	return filtered, nextCursor, nil
}

// appleRelease represents a single Apple security release from the JSON feed.
type appleRelease struct {
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	ReleaseDate string   `json:"releaseDate"`
	OS          string   `json:"os"`
	CVEs        []string `json:"cves"`
}

// versionRe matches a version number at the end of a release name (e.g., "14.4" from "macOS Sonoma 14.4").
var versionRe = regexp.MustCompile(`(\d+(?:\.\d+)+)\s*$`)

// osMapping maps Apple OS field values to lowercase OS family identifiers.
var osMapping = map[string]string{
	"macOS":    "macos",
	"iOS":      "ios",
	"iPadOS":   "ipados",
	"watchOS":  "watchos",
	"tvOS":     "tvos",
	"visionOS": "visionos",
}

// parse converts raw Apple JSON into RawEntry slices.
func (f *AppleFeed) parse(data []byte) ([]RawEntry, error) {
	var releases []appleRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, fmt.Errorf("apple parse: unmarshal response: %w", err)
	}

	var entries []RawEntry
	for _, rel := range releases {
		releaseDate, err := time.Parse("02 Jan 2006", rel.ReleaseDate)
		if err != nil {
			return nil, fmt.Errorf("apple parse: parse release date %q for %q: %w",
				rel.ReleaseDate, rel.Name, err)
		}

		osFamily := strings.ToLower(rel.OS)
		if mapped, ok := osMapping[rel.OS]; ok {
			osFamily = mapped
		}

		version := ""
		if match := versionRe.FindStringSubmatch(rel.Name); len(match) > 1 {
			version = match[1]
		}

		var refs []CVEReference
		if rel.URL != "" {
			refs = []CVEReference{{URL: rel.URL, Source: "apple"}}
		}

		entry := RawEntry{
			CVEs:          rel.CVEs,
			Name:          rel.Name,
			Vendor:        "apple",
			Product:       rel.Name,
			Version:       version,
			Severity:      "high",
			OSFamily:      osFamily,
			InstallerType: "pkg",
			ReleaseDate:   releaseDate,
			Summary:       rel.Name + " security update",
			SourceURL:     rel.URL,
			References:    refs,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
