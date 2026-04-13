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

const nvdBaseURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"

// nvdPageDelay is the pause between paginated NVD API requests to
// respect the NVD rate limit (5 requests per 30 seconds without API key).
const nvdPageDelay = 6 * time.Second

// nvdPageDelayWithKey is the pause between requests when an API key is configured.
// NVD allows 50 requests per 30 seconds with a key (0.6s spacing is safe).
const nvdPageDelayWithKey = 600 * time.Millisecond

// nvdResponse represents the top-level NVD API 2.0 response.
type nvdResponse struct {
	ResultsPerPage  int                `json:"resultsPerPage"`
	StartIndex      int                `json:"startIndex"`
	TotalResults    int                `json:"totalResults"`
	Vulnerabilities []nvdVulnerability `json:"vulnerabilities"`
}

// nvdVulnerability wraps a single CVE entry.
type nvdVulnerability struct {
	CVE nvdCVE `json:"cve"`
}

// nvdCVE represents a CVE record from the NVD API.
type nvdCVE struct {
	ID             string           `json:"id"`
	Published      string           `json:"published"`
	LastModified   string           `json:"lastModified"`
	Descriptions   []nvdDescription `json:"descriptions"`
	Metrics        nvdMetrics       `json:"metrics"`
	Weaknesses     []nvdWeakness    `json:"weaknesses"`
	Configurations []nvdConfig      `json:"configurations"`
	References     []nvdReference   `json:"references"`
}

type nvdWeakness struct {
	Description []nvdDescription `json:"description"`
}

type nvdDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

type nvdMetrics struct {
	CvssMetricV40 []nvdCVSSMetric `json:"cvssMetricV40"`
	CvssMetricV31 []nvdCVSSMetric `json:"cvssMetricV31"`
	CvssMetricV2  []nvdCVSSMetric `json:"cvssMetricV2"`
}

type nvdCVSSMetric struct {
	Source   string      `json:"source"`
	Type     string      `json:"type"`
	CvssData nvdCVSSData `json:"cvssData"`
}

type nvdCVSSData struct {
	Version      string  `json:"version"`
	VectorString string  `json:"vectorString"`
	BaseScore    float64 `json:"baseScore"`
	BaseSeverity string  `json:"baseSeverity"`
}

type nvdConfig struct {
	Nodes []nvdNode `json:"nodes"`
}

type nvdNode struct {
	Operator string        `json:"operator"`
	Negate   bool          `json:"negate"`
	CpeMatch []nvdCPEMatch `json:"cpeMatch"`
}

type nvdCPEMatch struct {
	Vulnerable bool   `json:"vulnerable"`
	Criteria   string `json:"criteria"`
}

type nvdReference struct {
	URL string `json:"url"`
}

// NVDFeed fetches vulnerability data from the NVD CVE 2.0 API.
type NVDFeed struct {
	client    *http.Client
	baseURL   string
	pageDelay time.Duration
	apiKey    string
}

// NewNVDFeed creates an NVDFeed with the given HTTP client.
// If client is nil, a default client with a 30-second timeout is used.
// apiKey is the NVD API key for higher rate limits (50 req/30s vs 5 req/30s).
// Pass an empty string to use the unauthenticated rate limit.
func NewNVDFeed(client *http.Client, apiKey string) *NVDFeed {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	delay := nvdPageDelay
	if apiKey != "" {
		delay = nvdPageDelayWithKey
	}
	return &NVDFeed{client: client, baseURL: nvdBaseURL, pageDelay: delay, apiKey: apiKey}
}

// Name returns the feed identifier.
func (f *NVDFeed) Name() string {
	return "nvd"
}

// Fetch retrieves CVE entries from the NVD API starting from the given cursor.
// The cursor is an RFC3339 timestamp used as lastModStartDate.
// Returns parsed entries, the next cursor (latest published timestamp), and any error.
// Fetch loops through all pages using startIndex until all results are consumed.
func (f *NVDFeed) Fetch(ctx context.Context, cursor string) ([]RawEntry, string, error) {
	var allEntries []RawEntry
	var maxLastModified time.Time
	startIndex := 0
	totalResults := -1 // -1 signals "not yet known"

	for totalResults == -1 || startIndex < totalResults {
		// Rate-limit between pages (skip delay for the first request).
		if startIndex > 0 && f.pageDelay > 0 {
			select {
			case <-ctx.Done():
				return nil, "", fmt.Errorf("nvd fetch: context cancelled during rate-limit delay: %w", ctx.Err())
			case <-time.After(f.pageDelay):
			}
		}

		pageURL := f.buildPageURL(cursor, startIndex)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, "", fmt.Errorf("nvd fetch: create request: %w", err)
		}
		if f.apiKey != "" {
			req.Header.Set("apiKey", f.apiKey)
		}

		resp, err := f.client.Do(req)
		if err != nil {
			return nil, "", fmt.Errorf("nvd fetch: execute request: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, "", fmt.Errorf("nvd fetch: unexpected status %d", resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, "", fmt.Errorf("nvd fetch: read response body: %w", err)
		}

		page, entries, pageMaxMod, err := f.parsePage(data)
		if err != nil {
			return nil, "", err
		}
		totalResults = page.TotalResults
		allEntries = append(allEntries, entries...)
		if pageMaxMod.After(maxLastModified) {
			maxLastModified = pageMaxMod
		}

		// Guard against infinite loop if API returns 0 resultsPerPage.
		if page.ResultsPerPage == 0 {
			if startIndex < totalResults {
				return nil, "", fmt.Errorf("nvd fetch: API returned resultsPerPage=0 with %d/%d results remaining",
					totalResults-startIndex, totalResults)
			}
			break
		}
		startIndex += page.ResultsPerPage
	}

	// Derive the next cursor from the maximum lastModified timestamp seen across
	// all pages. The cursor is passed as lastModStartDate on the next sync, so
	// using lastModified (rather than published) ensures that CVEs updated after
	// their publish date are not missed.
	var nextCursor string
	if maxLastModified.After(time.Time{}) {
		nextCursor = maxLastModified.Format(time.RFC3339)
	}

	return allEntries, nextCursor, nil
}

// nvdDateFormat is the timestamp format required by the NVD API.
// NVD requires milliseconds with NO timezone suffix — e.g. "2026-01-02T15:04:05.000".
// DO NOT use time.RFC3339 here; the trailing "Z" suffix causes NVD to return 404.
const nvdDateFormat = "2006-01-02T15:04:05.000"

// buildPageURL constructs the NVD API URL for a given cursor and page offset.
// The cursor is stored as RFC3339 in the DB but must be reformatted to nvdDateFormat
// when used as a query parameter, because NVD rejects dates with a timezone suffix.
func (f *NVDFeed) buildPageURL(cursor string, startIndex int) string {
	if cursor != "" {
		// Parse the cursor (stored as RFC3339) and reformat for NVD.
		startDate := cursor
		if t, err := time.Parse(time.RFC3339, cursor); err == nil {
			startDate = t.UTC().Format(nvdDateFormat)
		}
		endDate := time.Now().UTC().Format(nvdDateFormat)
		return fmt.Sprintf("%s?lastModStartDate=%s&lastModEndDate=%s&startIndex=%d",
			f.baseURL, startDate, endDate, startIndex)
	}
	if startIndex > 0 {
		return fmt.Sprintf("%s?startIndex=%d", f.baseURL, startIndex)
	}
	return f.baseURL
}

// nvdTimeFormats are the timestamp formats the NVD API may return.
var nvdTimeFormats = []string{
	"2006-01-02T15:04:05.000",
	time.RFC3339,
	"2006-01-02T15:04:05",
}

func parseNVDTime(s string) (time.Time, error) {
	for _, layout := range nvdTimeFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse NVD time %q: no matching format", s)
}

// parsePage unmarshals a single NVD API response page, returning the response
// metadata (for pagination), the parsed entries, and the maximum lastModified
// timestamp seen on the page (used to advance the sync cursor).
func (f *NVDFeed) parsePage(data []byte) (nvdResponse, []RawEntry, time.Time, error) {
	var resp nvdResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nvdResponse{}, nil, time.Time{}, fmt.Errorf("nvd parse: unmarshal response: %w", err)
	}

	var pageMaxMod time.Time
	entries := make([]RawEntry, 0, len(resp.Vulnerabilities))
	for _, v := range resp.Vulnerabilities {
		entry := RawEntry{
			CVEs:    []string{v.CVE.ID},
			Name:    v.CVE.ID,
			CVEOnly: true,
		}

		// English description.
		for _, d := range v.CVE.Descriptions {
			if d.Lang == "en" {
				entry.Summary = d.Value
				break
			}
		}

		// CVSS metrics: prefer v3.1, fall back to v4.0, then v2.
		// Within each version, prefer Primary type.
		cvssMetrics := v.CVE.Metrics.CvssMetricV31
		if len(cvssMetrics) == 0 {
			cvssMetrics = v.CVE.Metrics.CvssMetricV40
		}
		if len(cvssMetrics) == 0 {
			cvssMetrics = v.CVE.Metrics.CvssMetricV2
		}
		for _, m := range cvssMetrics {
			entry.CVSSScore = m.CvssData.BaseScore
			entry.Severity = strings.ToLower(m.CvssData.BaseSeverity)
			entry.CVSSv3Vector = m.CvssData.VectorString
			entry.AttackVector = extractAttackVector(m.CvssData.VectorString)
			if m.Type == "Primary" {
				break
			}
		}

		// CWE ID from weaknesses.
		for _, w := range v.CVE.Weaknesses {
			for _, d := range w.Description {
				if d.Lang == "en" && d.Value != "" && d.Value != "NVD-CWE-noinfo" && d.Value != "NVD-CWE-Other" {
					entry.CweID = d.Value
					break
				}
			}
			if entry.CweID != "" {
				break
			}
		}

		// All references.
		for _, ref := range v.CVE.References {
			entry.References = append(entry.References, CVEReference{
				URL:    ref.URL,
				Source: "nvd",
			})
		}

		// Extract vendor and product from first vulnerable CPE match.
		vendor, product := extractCPEVendorProduct(v.CVE.Configurations)
		if vendor == "" {
			vendor = "nist"
		}
		entry.Vendor = vendor
		entry.Product = product

		// First reference URL.
		if len(v.CVE.References) > 0 {
			entry.SourceURL = v.CVE.References[0].URL
		}

		// Published date (ReleaseDate in the catalog).
		if v.CVE.Published != "" {
			t, err := parseNVDTime(v.CVE.Published)
			if err != nil {
				return nvdResponse{}, nil, time.Time{}, fmt.Errorf("nvd parse CVE %s: %w", v.CVE.ID, err)
			}
			entry.ReleaseDate = t
		}

		// Track max lastModified for cursor advancement and set NVDLastModified.
		if v.CVE.LastModified != "" {
			t, err := parseNVDTime(v.CVE.LastModified)
			if err != nil {
				return nvdResponse{}, nil, time.Time{}, fmt.Errorf("nvd parse CVE %s lastModified: %w", v.CVE.ID, err)
			}
			entry.NVDLastModified = &t
			if t.After(pageMaxMod) {
				pageMaxMod = t
			}
		}

		entries = append(entries, entry)
	}

	return resp, entries, pageMaxMod, nil
}

// extractAttackVector parses the attack vector component from a CVSS v3 vector string.
// Example: "CVSS:3.1/AV:N/AC:L/..." → "NETWORK".
func extractAttackVector(vector string) string {
	avMap := map[string]string{
		"N": "NETWORK",
		"A": "ADJACENT_NETWORK",
		"L": "LOCAL",
		"P": "PHYSICAL",
	}
	for _, part := range strings.Split(vector, "/") {
		if strings.HasPrefix(part, "AV:") {
			if val, ok := avMap[strings.TrimPrefix(part, "AV:")]; ok {
				return val
			}
		}
	}
	return ""
}

// extractCPEVendorProduct finds the vendor (index 3) and product (index 4)
// from the first vulnerable CPE match in the configuration nodes.
func extractCPEVendorProduct(configs []nvdConfig) (string, string) {
	for _, cfg := range configs {
		for _, node := range cfg.Nodes {
			for _, match := range node.CpeMatch {
				if !match.Vulnerable {
					continue
				}
				parts := strings.Split(match.Criteria, ":")
				var vendor, product string
				if len(parts) > 3 {
					vendor = parts[3]
				}
				if len(parts) > 4 {
					product = parts[4]
				}
				if vendor != "" {
					return vendor, product
				}
			}
		}
	}
	return "", ""
}
