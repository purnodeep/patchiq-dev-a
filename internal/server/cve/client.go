package cve

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultNVDBaseURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"
	defaultKEVURL     = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
	nvdPageSize       = 2000
)

// NVDClient fetches CVE and KEV data from NVD and CISA APIs.
type NVDClient struct {
	baseURL    string
	kevURL     string
	apiKey     string
	httpClient *http.Client
}

// NewNVDClient creates an NVDClient. If baseURL is empty, the default NVD API URL is used.
func NewNVDClient(baseURL, apiKey string, timeout time.Duration) *NVDClient {
	if baseURL == "" {
		baseURL = defaultNVDBaseURL
	}
	return &NVDClient{
		baseURL: baseURL,
		kevURL:  defaultKEVURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchCVEs retrieves all CVEs modified since the given time, handling pagination.
func (c *NVDClient) FetchCVEs(ctx context.Context, since time.Time) ([]CVERecord, error) {
	var allRecords []CVERecord
	startIndex := 0
	endDate := time.Now().UTC()

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse NVD base URL: %w", err)
	}

	for {
		q := u.Query()
		q.Set("lastModStartDate", since.UTC().Format("2006-01-02T15:04:05.000"))
		q.Set("lastModEndDate", endDate.Format("2006-01-02T15:04:05.000"))
		q.Set("startIndex", fmt.Sprintf("%d", startIndex))
		q.Set("resultsPerPage", fmt.Sprintf("%d", nvdPageSize))
		u.RawQuery = q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create NVD request: %w", err)
		}
		if c.apiKey != "" {
			req.Header.Set("apiKey", c.apiKey)
		}

		slog.InfoContext(ctx, "nvd: fetching CVEs", "start_index", startIndex, "url", u.Redacted())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch NVD CVEs (startIndex=%d): %w", startIndex, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read NVD response body (status=%d): %w", resp.StatusCode, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("NVD API returned status %d: %s", resp.StatusCode, string(body))
		}

		nvdResp, err := ParseNVDResponse(body)
		if err != nil {
			return nil, fmt.Errorf("parse NVD page (startIndex=%d): %w", startIndex, err)
		}

		records := NVDResponseToCVERecords(nvdResp)
		allRecords = append(allRecords, records...)

		slog.InfoContext(ctx, "nvd: fetched page",
			"start_index", startIndex,
			"page_count", len(records),
			"total", nvdResp.TotalResults,
		)

		startIndex += nvdResp.ResultsPerPage
		if startIndex >= nvdResp.TotalResults {
			break
		}
	}

	return allRecords, nil
}

// FetchKEV retrieves the CISA Known Exploited Vulnerabilities catalog
// and returns it as a map keyed by CVE ID.
func (c *NVDClient) FetchKEV(ctx context.Context) (map[string]KEVVulnerability, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.kevURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create KEV request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch KEV catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KEV API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read KEV response: %w", err)
	}

	catalog, err := ParseKEVCatalog(body)
	if err != nil {
		return nil, fmt.Errorf("fetch KEV: parse catalog: %w", err)
	}

	return KEVCatalogToMap(catalog), nil
}
