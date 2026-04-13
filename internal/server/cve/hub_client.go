package cve

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// hubCVEFeed matches the CVEFeed shape returned by the Hub's /api/v1/sync/cves endpoint.
type hubCVEFeed struct {
	CVEID              string          `json:"cve_id"`
	Severity           string          `json:"severity"`
	Description        string          `json:"description"`
	PublishedAt        string          `json:"published_at"`
	Source             string          `json:"source"`
	CVSSv3Score        json.Number     `json:"cvss_v3_score"`
	CVSSv3Vector       string          `json:"cvss_v3_vector"`
	AttackVector       string          `json:"attack_vector"`
	CweID              string          `json:"cwe_id"`
	CisaKEVDueDate     string          `json:"cisa_kev_due_date"`
	ExternalReferences json.RawMessage `json:"external_references"`
	NVDLastModified    string          `json:"nvd_last_modified"`
	ExploitKnown       bool            `json:"exploit_known"`
	InKEV              bool            `json:"in_kev"`
}

// hubCVEResponse is the JSON envelope returned by GET /api/v1/sync/cves.
type hubCVEResponse struct {
	CVEs       []hubCVEFeed `json:"cves"`
	ServerTime string       `json:"server_time"`
}

// HubCVEClient fetches CVE data from the Hub Manager instead of directly from NVD.
// It implements CVEFetcher.
type HubCVEClient struct {
	hubURL string
	apiKey string
	client *http.Client
}

// NewHubCVEClient creates a HubCVEClient that talks to hubURL authenticated with apiKey.
func NewHubCVEClient(hubURL, apiKey string) *HubCVEClient {
	return &HubCVEClient{
		hubURL: hubURL,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// FetchCVEs retrieves CVEs from the Hub that have been modified since the given time.
// It calls GET {hubURL}/api/v1/sync/cves?since={RFC3339} and maps the response to
// []CVERecord.
func (c *HubCVEClient) FetchCVEs(ctx context.Context, since time.Time) ([]CVERecord, error) {
	reqURL := fmt.Sprintf("%s/api/v1/sync/cves?since=%s", c.hubURL, since.UTC().Format(time.RFC3339))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hub cve client: create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	slog.InfoContext(ctx, "hub cve client: fetching CVEs", "since", since.UTC().Format(time.RFC3339))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hub cve client: fetch CVEs: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("hub cve client: read response body (status=%d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hub cve client: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var hubResp hubCVEResponse
	if err := json.Unmarshal(body, &hubResp); err != nil {
		return nil, fmt.Errorf("hub cve client: parse response: %w", err)
	}

	records := make([]CVERecord, 0, len(hubResp.CVEs))
	for _, feed := range hubResp.CVEs {
		rec := CVERecord{
			CVEID:        feed.CVEID,
			Description:  feed.Description,
			Severity:     feed.Severity,
			CVSSv3Vector: feed.CVSSv3Vector,
			AttackVector: feed.AttackVector,
			CweID:        feed.CweID,
			Source:       feed.Source,
		}

		if feed.CVSSv3Score != "" {
			if score, err := feed.CVSSv3Score.Float64(); err == nil {
				rec.CVSSv3Score = score
			} else {
				slog.WarnContext(ctx, "hub cve client: parse cvss_v3_score",
					"cve_id", feed.CVEID, "value", string(feed.CVSSv3Score), "error", err)
			}
		}

		if feed.PublishedAt != "" {
			t, err := time.Parse(time.RFC3339, feed.PublishedAt)
			if err != nil {
				slog.WarnContext(ctx, "hub cve client: parse published_at",
					"cve_id", feed.CVEID, "value", feed.PublishedAt, "error", err)
			} else {
				rec.PublishedAt = t.UTC()
			}
		}

		if feed.NVDLastModified != "" {
			t, err := time.Parse(time.RFC3339, feed.NVDLastModified)
			if err != nil {
				slog.WarnContext(ctx, "hub cve client: parse nvd_last_modified",
					"cve_id", feed.CVEID, "value", feed.NVDLastModified, "error", err)
			} else {
				rec.LastModified = t.UTC()
			}
		}

		// Map KEV/exploit fields that the Hub already enriched.
		rec.ExploitAvailable = feed.ExploitKnown || feed.InKEV
		rec.CisaKEVDueDate = feed.CisaKEVDueDate

		// external_references arrives as a base64-encoded JSON string because
		// the Hub model uses []byte (pgtype JSONB) which Go json.Marshal base64-encodes.
		// Decode base64 → JSON → []CVEReference.
		if len(feed.ExternalReferences) > 0 {
			var b64str string
			if err := json.Unmarshal(feed.ExternalReferences, &b64str); err == nil && b64str != "" {
				if decoded, err := base64.StdEncoding.DecodeString(b64str); err == nil {
					var refs []CVEReference
					if err := json.Unmarshal(decoded, &refs); err == nil {
						rec.References = refs
					}
				}
			}
		}

		records = append(records, rec)
	}

	slog.InfoContext(ctx, "hub cve client: fetched CVEs", "count", len(records))

	return records, nil
}

// FetchKEV returns an empty map. The Hub already enriches CVEs with KEV data
// (exploit_known, in_kev, cisa_kev_due_date) so a separate KEV fetch is not
// required when using HubCVEClient.
func (c *HubCVEClient) FetchKEV(_ context.Context) (map[string]KEVVulnerability, error) {
	return map[string]KEVVulnerability{}, nil
}
