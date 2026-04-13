package cve

import (
	"encoding/json"
	"fmt"
)

// KEVCatalog represents the CISA Known Exploited Vulnerabilities catalog.
type KEVCatalog struct {
	Title           string             `json:"title"`
	CatalogVersion  string             `json:"catalogVersion"`
	DateReleased    string             `json:"dateReleased"`
	Count           int                `json:"count"`
	Vulnerabilities []KEVVulnerability `json:"vulnerabilities"`
}

// KEVVulnerability represents a single entry in the CISA KEV catalog.
type KEVVulnerability struct {
	CveID                      string `json:"cveID"`
	VendorProject              string `json:"vendorProject"`
	Product                    string `json:"product"`
	VulnerabilityName          string `json:"vulnerabilityName"`
	DateAdded                  string `json:"dateAdded"`
	ShortDescription           string `json:"shortDescription"`
	RequiredAction             string `json:"requiredAction"`
	DueDate                    string `json:"dueDate"`
	KnownRansomwareCampaignUse string `json:"knownRansomwareCampaignUse"`
}

// ParseKEVCatalog parses raw JSON bytes into a KEVCatalog.
func ParseKEVCatalog(data []byte) (*KEVCatalog, error) {
	var catalog KEVCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse KEV catalog: %w", err)
	}
	return &catalog, nil
}

// KEVCatalogToMap converts a KEVCatalog into a map keyed by CVE ID.
func KEVCatalogToMap(catalog *KEVCatalog) map[string]KEVVulnerability {
	m := make(map[string]KEVVulnerability, len(catalog.Vulnerabilities))
	for _, v := range catalog.Vulnerabilities {
		m[v.CveID] = v
	}
	return m
}
