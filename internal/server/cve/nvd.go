package cve

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// NVDLangString is a language-tagged string used in NVD weakness descriptions.
type NVDLangString struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

// NVDWeakness represents a CWE weakness entry from the NVD API.
type NVDWeakness struct {
	Source      string          `json:"source"`
	Type        string          `json:"type"`
	Description []NVDLangString `json:"description"`
}

// NVDReference represents a reference URL from the NVD API.
type NVDReference struct {
	URL    string   `json:"url"`
	Source string   `json:"source"`
	Tags   []string `json:"tags"`
}

// NVDResponse represents the top-level NVD API 2.0 response.
type NVDResponse struct {
	ResultsPerPage  int                `json:"resultsPerPage"`
	StartIndex      int                `json:"startIndex"`
	TotalResults    int                `json:"totalResults"`
	Vulnerabilities []NVDVulnerability `json:"vulnerabilities"`
}

// NVDVulnerability wraps a single CVE entry in the NVD response.
type NVDVulnerability struct {
	CVE NVDCVE `json:"cve"`
}

// NVDCVE represents a CVE record from the NVD API.
type NVDCVE struct {
	ID             string           `json:"id"`
	Published      NVDTime          `json:"published"`
	LastModified   NVDTime          `json:"lastModified"`
	Descriptions   []NVDDescription `json:"descriptions"`
	Metrics        NVDMetrics       `json:"metrics"`
	Configurations []NVDConfig      `json:"configurations"`
	Weaknesses     []NVDWeakness    `json:"weaknesses"`
	References     []NVDReference   `json:"references"`
}

// NVDTime handles NVD's timestamp format (2006-01-02T15:04:05.000).
type NVDTime struct {
	time.Time
}

func (t *NVDTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02T15:04:05.000", s)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return fmt.Errorf("parse NVD time %q: %w", s, err)
		}
	}
	t.Time = parsed.UTC()
	return nil
}

// NVDDescription holds a language-tagged description.
type NVDDescription struct {
	Lang  string `json:"lang"`
	Value string `json:"value"`
}

// NVDMetrics contains CVSS scoring data.
type NVDMetrics struct {
	CvssMetricV31 []NVDCVSSMetric `json:"cvssMetricV31"`
}

// NVDCVSSMetric represents a single CVSS v3.1 metric entry.
type NVDCVSSMetric struct {
	Source   string      `json:"source"`
	Type     string      `json:"type"`
	CvssData NVDCVSSData `json:"cvssData"`
}

// NVDCVSSData holds the CVSS score and vector.
type NVDCVSSData struct {
	Version      string  `json:"version"`
	VectorString string  `json:"vectorString"`
	BaseScore    float64 `json:"baseScore"`
	BaseSeverity string  `json:"baseSeverity"`
}

// NVDConfig represents a CPE applicability configuration.
type NVDConfig struct {
	Nodes []NVDNode `json:"nodes"`
}

// NVDNode represents a node in the CPE match tree.
type NVDNode struct {
	Operator string        `json:"operator"`
	Negate   bool          `json:"negate"`
	CpeMatch []NVDCPEMatch `json:"cpeMatch"`
}

// NVDCPEMatch represents a single CPE match criterion.
type NVDCPEMatch struct {
	Vulnerable          bool   `json:"vulnerable"`
	Criteria            string `json:"criteria"`
	VersionEndExcluding string `json:"versionEndExcluding,omitempty"`
	VersionEndIncluding string `json:"versionEndIncluding,omitempty"`
}

// CVEReference represents a reference URL associated with a CVE.
type CVEReference struct {
	URL    string `json:"url"`
	Source string `json:"source"`
}

// CVERecord is the normalized internal representation of a CVE.
type CVERecord struct {
	CVEID            string
	Description      string
	Severity         string
	CVSSv3Score      float64
	CVSSv3Vector     string
	PublishedAt      time.Time
	LastModified     time.Time
	AttackVector     string
	CweID            string
	Source           string
	References       []CVEReference
	AffectedPackages []AffectedPackage
	CisaKEVDueDate   string
	ExploitAvailable bool
}

// AffectedPackage describes a package affected by a CVE.
type AffectedPackage struct {
	PackageName         string
	Vendor              string
	VersionEndExcluding string
	VersionEndIncluding string
}

// ParseNVDResponse unmarshals raw JSON from the NVD API 2.0 into an NVDResponse.
func ParseNVDResponse(data []byte) (*NVDResponse, error) {
	var resp NVDResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse NVD response: %w", err)
	}
	return &resp, nil
}

// NVDResponseToCVERecords converts a parsed NVD response into normalized CVERecords.
func NVDResponseToCVERecords(resp *NVDResponse) []CVERecord {
	records := make([]CVERecord, 0, len(resp.Vulnerabilities))
	for _, v := range resp.Vulnerabilities {
		r := CVERecord{
			CVEID:        v.CVE.ID,
			PublishedAt:  v.CVE.Published.Time,
			LastModified: v.CVE.LastModified.Time,
		}

		for _, d := range v.CVE.Descriptions {
			if d.Lang == "en" {
				r.Description = d.Value
				break
			}
		}

		r.Source = "NVD"

		r.Severity = "none"
		for _, m := range v.CVE.Metrics.CvssMetricV31 {
			r.CVSSv3Score = m.CvssData.BaseScore
			r.CVSSv3Vector = m.CvssData.VectorString
			r.Severity = strings.ToLower(m.CvssData.BaseSeverity)
			r.AttackVector = attackVectorFromCVSS(m.CvssData.VectorString)
			if m.Type == "Primary" {
				break
			}
		}

		for _, w := range v.CVE.Weaknesses {
			for _, d := range w.Description {
				if d.Lang == "en" && d.Value != "" {
					r.CweID = d.Value
					break
				}
			}
			if r.CweID != "" {
				break
			}
		}

		for _, ref := range v.CVE.References {
			r.References = append(r.References, CVEReference{
				URL:    ref.URL,
				Source: ref.Source,
			})
		}

		for _, cfg := range v.CVE.Configurations {
			for _, node := range cfg.Nodes {
				for _, match := range node.CpeMatch {
					if !match.Vulnerable {
						continue
					}
					pkgName := ExtractPackageNameFromCPE(match.Criteria)
					if pkgName == "" {
						continue
					}
					r.AffectedPackages = append(r.AffectedPackages, AffectedPackage{
						PackageName:         pkgName,
						Vendor:              extractVendorFromCPE(match.Criteria),
						VersionEndExcluding: match.VersionEndExcluding,
						VersionEndIncluding: match.VersionEndIncluding,
					})
				}
			}
		}

		records = append(records, r)
	}
	return records
}

// attackVectorFromCVSS extracts the human-readable attack vector from a CVSS vector string.
func attackVectorFromCVSS(vector string) string {
	for _, part := range strings.Split(vector, "/") {
		switch part {
		case "AV:N":
			return "Network"
		case "AV:A":
			return "Adjacent"
		case "AV:L":
			return "Local"
		case "AV:P":
			return "Physical"
		}
	}
	return ""
}

// ExtractPackageNameFromCPE extracts the product name (index 4) from a CPE 2.3 URI.
func ExtractPackageNameFromCPE(cpe string) string {
	parts := strings.Split(cpe, ":")
	if len(parts) < 5 {
		return ""
	}
	return parts[4]
}

func extractVendorFromCPE(cpe string) string {
	parts := strings.Split(cpe, ":")
	if len(parts) < 4 {
		return ""
	}
	return parts[3]
}
