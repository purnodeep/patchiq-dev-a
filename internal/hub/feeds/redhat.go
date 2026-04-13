package feeds

import (
	"compress/bzip2"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var redhatOVALURLs = []string{
	"https://www.redhat.com/security/data/oval/v2/RHEL7/rhel-7.oval.xml.bz2",
	"https://www.redhat.com/security/data/oval/v2/RHEL8/rhel-8.oval.xml.bz2",
	"https://www.redhat.com/security/data/oval/v2/RHEL9/rhel-9.oval.xml.bz2",
}

// ovalDefinitions represents the top-level OVAL XML structure.
type ovalDefinitions struct {
	XMLName     xml.Name    `xml:"oval_definitions"`
	Definitions ovalDefList `xml:"definitions"`
}

type ovalDefList struct {
	Definitions []ovalDefinition `xml:"definition"`
}

type ovalDefinition struct {
	ID       string       `xml:"id,attr"`
	Version  string       `xml:"version,attr"`
	Class    string       `xml:"class,attr"`
	Metadata ovalMetadata `xml:"metadata"`
}

type ovalMetadata struct {
	Title       string       `xml:"title"`
	Affected    ovalAffected `xml:"affected"`
	References  []ovalRef    `xml:"reference"`
	Advisory    ovalAdvisory `xml:"advisory"`
	Description string       `xml:"description"`
}

type ovalAffected struct {
	Family    string   `xml:"family,attr"`
	Platforms []string `xml:"platform"`
}

type ovalRef struct {
	Source string `xml:"source,attr"`
	RefID  string `xml:"ref_id,attr"`
	RefURL string `xml:"ref_url,attr"`
}

type ovalAdvisory struct {
	Severity string   `xml:"severity"`
	Issued   ovalDate `xml:"issued"`
	Updated  ovalDate `xml:"updated"`
}

type ovalDate struct {
	Date string `xml:"date,attr"`
}

// rhsaPattern extracts the RHSA ID from definition titles like
// "RHSA-2024:0893: python3 security update (Important)".
var rhsaPattern = regexp.MustCompile(`^(RHSA-\d{4}:\d+):\s+`)

// rhelVersionPattern extracts the major version from platform strings like
// "Red Hat Enterprise Linux 9".
var rhelVersionPattern = regexp.MustCompile(`Red Hat Enterprise Linux (\d+)`)

// RedHatFeed fetches and parses Red Hat OVAL vulnerability definitions.
type RedHatFeed struct {
	client *http.Client
	urls   []string
}

// NewRedHatFeed creates a RedHatFeed with the given HTTP client.
// If client is nil, a default client with a 30-second timeout is used.
func NewRedHatFeed(client *http.Client) *RedHatFeed {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &RedHatFeed{client: client, urls: redhatOVALURLs}
}

// Name returns the feed identifier.
func (f *RedHatFeed) Name() string {
	return "redhat_oval"
}

// Fetch downloads and parses Red Hat OVAL definitions.
// The cursor is the Last-Modified or ETag header value from the previous fetch.
// Returns parsed entries, the next cursor, and any error.
func (f *RedHatFeed) Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error) {
	var allEntries []RawEntry
	var latestCursor string
	var errs []error

	for _, url := range f.urls {
		entries, next, err := f.fetchOne(ctx, url, cursor)
		if err != nil {
			slog.ErrorContext(ctx, "redhat oval fetch: url failed, continuing with remaining",
				"url", url, "error", err)
			errs = append(errs, err)
			continue
		}
		allEntries = append(allEntries, entries...)
		if next != "" {
			latestCursor = next
		}
	}

	if len(errs) == len(f.urls) {
		return nil, "", fmt.Errorf("redhat oval fetch: all URLs failed: %w", errors.Join(errs...))
	}

	if latestCursor == "" {
		latestCursor = cursor
	}
	return allEntries, latestCursor, nil
}

func (f *RedHatFeed) fetchOne(ctx context.Context, url, cursor string) ([]RawEntry, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("redhat oval fetch %s: create request: %w", url, err)
	}

	if cursor != "" {
		req.Header.Set("If-Modified-Since", cursor)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("redhat oval fetch %s: execute request: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, cursor, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("redhat oval fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	bz2Reader := bzip2.NewReader(resp.Body)
	data, err := io.ReadAll(bz2Reader)
	if err != nil {
		return nil, "", fmt.Errorf("redhat oval fetch %s: decompress bzip2: %w", url, err)
	}

	entries, err := f.parse(data)
	if err != nil {
		return nil, "", err
	}

	nextCursor := resp.Header.Get("Last-Modified")
	if nextCursor == "" {
		nextCursor = resp.Header.Get("ETag")
	}

	return entries, nextCursor, nil
}

// maxSummaryLen is the maximum length for the Summary field.
const maxSummaryLen = 512

// parse converts raw OVAL XML bytes into RawEntry slices.
// Only definitions with class="patch" are included.
func (f *RedHatFeed) parse(data []byte) ([]RawEntry, error) {
	var defs ovalDefinitions
	if err := xml.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("redhat oval parse: unmarshal XML: %w", err)
	}

	entries := make([]RawEntry, 0, len(defs.Definitions.Definitions))
	for _, def := range defs.Definitions.Definitions {
		if def.Class != "patch" {
			continue
		}

		entry := RawEntry{
			Vendor:        "redhat",
			OSFamily:      "linux",
			InstallerType: "rpm",
		}

		// Extract RHSA name and product from title.
		if m := rhsaPattern.FindStringSubmatch(def.Metadata.Title); len(m) > 1 {
			entry.Name = m[1]
		}
		entry.Product = extractProductFromTitle(def.Metadata.Title)

		// Advisory URL from RHSA name.
		if entry.Name != "" {
			entry.References = append(entry.References, CVEReference{
				URL:    "https://access.redhat.com/errata/" + entry.Name,
				Source: "redhat",
			})
		}

		// Collect CVE references.
		for _, ref := range def.Metadata.References {
			if ref.Source == "CVE" {
				entry.CVEs = append(entry.CVEs, ref.RefID)
				if entry.SourceURL == "" {
					entry.SourceURL = ref.RefURL
				}
				if ref.RefURL != "" {
					entry.References = append(entry.References, CVEReference{
						URL:    ref.RefURL,
						Source: "cve",
					})
				}
			}
		}

		// Severity from advisory (lowercase).
		entry.Severity = strings.ToLower(def.Metadata.Advisory.Severity)

		// Release date from advisory issued date.
		if def.Metadata.Advisory.Issued.Date != "" {
			t, err := time.Parse("2006-01-02", def.Metadata.Advisory.Issued.Date)
			if err != nil {
				return nil, fmt.Errorf("redhat oval parse: parse issued date %q for %s: %w",
					def.Metadata.Advisory.Issued.Date, def.ID, err)
			}
			entry.ReleaseDate = t.UTC()
		}

		// Summary (truncated if too long).
		summary := def.Metadata.Description
		if len(summary) > maxSummaryLen {
			summary = summary[:maxSummaryLen]
		}
		entry.Summary = summary

		// OS versions from affected platforms.
		for _, platform := range def.Metadata.Affected.Platforms {
			if m := rhelVersionPattern.FindStringSubmatch(platform); len(m) > 1 {
				entry.OSVersions = append(entry.OSVersions, m[1])
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// extractProductFromTitle extracts the product name from an RHSA title.
// For "RHSA-2024:0893: python3 security update (Important)", returns "python3".
func extractProductFromTitle(title string) string {
	// Find the part after "RHSA-XXXX:YYYY: " and before " security update".
	idx := strings.Index(title, ": ")
	if idx < 0 {
		return ""
	}
	rest := title[idx+2:]

	secIdx := strings.Index(rest, " security update")
	if secIdx < 0 {
		return ""
	}
	return rest[:secIdx]
}
