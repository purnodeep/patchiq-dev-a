package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HubAliasClient implements HubAliasReporter by calling the Hub's discover endpoint.
type HubAliasClient struct {
	hubURL string
	apiKey string
	client *http.Client
}

// NewHubAliasClient creates a client that reports aliases to the Hub.
func NewHubAliasClient(hubURL, apiKey string) *HubAliasClient {
	return &HubAliasClient{
		hubURL: hubURL,
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ReportAliases sends the discovered packages to the Hub's package-aliases/discover endpoint.
func (c *HubAliasClient) ReportAliases(ctx context.Context, packages []DiscoveredAlias) error {
	body, err := json.Marshal(map[string]any{"packages": packages})
	if err != nil {
		return fmt.Errorf("hub alias client: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.hubURL+"/api/v1/package-aliases/discover", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("hub alias client: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("hub alias client: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("hub alias client: unexpected status %d", resp.StatusCode)
	}
	return nil
}
