package feeds

import (
	"os"
	"sort"
	"testing"
	"time"
)

func TestUbuntuFeedParseReferences(t *testing.T) {
	t.Parallel()

	const jsonData = `{"notices":[{"id":"USN-6543-1","title":"USN-6543-1: OpenSSL vulnerabilities","summary":"Several security issues were fixed in OpenSSL.","description":"","published":"2024-02-15T00:00:00","cves":[{"id":"CVE-2024-0727","priority":"high"},{"id":"CVE-2023-6237","priority":"medium"}],"release_packages":{"jammy":[{"name":"openssl","version":"3.0.2-0ubuntu1.14"}]}}]}`

	feed := NewUbuntuFeed(nil)
	entries, err := feed.parse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	refs := entries[0].References
	// Expect: 1 USN URL + 2 CVE URLs = 3 total.
	if len(refs) != 3 {
		t.Fatalf("expected 3 references, got %d: %v", len(refs), refs)
	}

	if refs[0].URL != "https://ubuntu.com/security/notices/USN-6543-1" {
		t.Errorf("refs[0].URL: expected USN URL, got %q", refs[0].URL)
	}
	if refs[0].Source != "ubuntu" {
		t.Errorf("refs[0].Source: expected %q, got %q", "ubuntu", refs[0].Source)
	}

	if refs[1].URL != "https://ubuntu.com/security/CVE-2024-0727" {
		t.Errorf("refs[1].URL: expected CVE-2024-0727 URL, got %q", refs[1].URL)
	}
	if refs[1].Source != "ubuntu" {
		t.Errorf("refs[1].Source: expected %q, got %q", "ubuntu", refs[1].Source)
	}

	if refs[2].URL != "https://ubuntu.com/security/CVE-2023-6237" {
		t.Errorf("refs[2].URL: expected CVE-2023-6237 URL, got %q", refs[2].URL)
	}
	if refs[2].Source != "ubuntu" {
		t.Errorf("refs[2].Source: expected %q, got %q", "ubuntu", refs[2].Source)
	}
}

func TestUbuntuFeedName(t *testing.T) {
	t.Parallel()

	feed := NewUbuntuFeed(nil)
	if got := feed.Name(); got != "ubuntu_usn" {
		t.Fatalf("expected name %q, got %q", "ubuntu_usn", got)
	}
}

func TestUbuntuFeedParse(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdataPath("ubuntu_usn_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	feed := NewUbuntuFeed(nil)
	entries, err := feed.parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	tests := []struct {
		name           string
		entry          RawEntry
		wantCVECount   int
		wantCVEs       []string
		wantName       string
		wantVendor     string
		wantProduct    string
		wantVersion    string
		wantSev        string
		wantOSFamily   string
		wantOSVersions []string
		wantInstaller  string
		wantSummary    string
		wantDate       time.Time
		wantMetadata   map[string]string
	}{
		{
			name:           "USN with multiple CVEs and multiple releases",
			entry:          entries[0],
			wantCVECount:   2,
			wantCVEs:       []string{"CVE-2024-0727", "CVE-2023-6237"},
			wantName:       "USN-6543-1: OpenSSL vulnerabilities",
			wantVendor:     "canonical",
			wantProduct:    "openssl",
			wantVersion:    "3.0.2-0ubuntu1.14",
			wantSev:        "high",
			wantOSFamily:   "linux",
			wantOSVersions: []string{"jammy", "noble"},
			wantInstaller:  "deb",
			wantSummary:    "Several security issues were fixed in OpenSSL.",
			wantDate:       time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			wantMetadata:   map[string]string{"usn_id": "USN-6543-1"},
		},
		{
			name:           "USN with single CVE and single release",
			entry:          entries[1],
			wantCVECount:   1,
			wantCVEs:       []string{"CVE-2024-0853"},
			wantName:       "USN-6540-1: curl vulnerability",
			wantVendor:     "canonical",
			wantProduct:    "curl",
			wantVersion:    "7.81.0-1ubuntu1.16",
			wantSev:        "low",
			wantOSFamily:   "linux",
			wantOSVersions: []string{"jammy"},
			wantInstaller:  "deb",
			wantSummary:    "curl could be made to expose sensitive information.",
			wantDate:       time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC),
			wantMetadata:   map[string]string{"usn_id": "USN-6540-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := tt.entry

			if len(e.CVEs) != tt.wantCVECount {
				t.Fatalf("CVEs count: expected %d, got %d (%v)", tt.wantCVECount, len(e.CVEs), e.CVEs)
			}
			for i, cve := range tt.wantCVEs {
				if e.CVEs[i] != cve {
					t.Errorf("CVE[%d]: expected %q, got %q", i, cve, e.CVEs[i])
				}
			}

			if e.Name != tt.wantName {
				t.Errorf("Name: expected %q, got %q", tt.wantName, e.Name)
			}
			if e.Vendor != tt.wantVendor {
				t.Errorf("Vendor: expected %q, got %q", tt.wantVendor, e.Vendor)
			}
			if e.Product != tt.wantProduct {
				t.Errorf("Product: expected %q, got %q", tt.wantProduct, e.Product)
			}
			if e.Version != tt.wantVersion {
				t.Errorf("Version: expected %q, got %q", tt.wantVersion, e.Version)
			}
			if e.Severity != tt.wantSev {
				t.Errorf("Severity: expected %q, got %q", tt.wantSev, e.Severity)
			}
			if e.OSFamily != tt.wantOSFamily {
				t.Errorf("OSFamily: expected %q, got %q", tt.wantOSFamily, e.OSFamily)
			}
			if e.InstallerType != tt.wantInstaller {
				t.Errorf("InstallerType: expected %q, got %q", tt.wantInstaller, e.InstallerType)
			}
			if !e.ReleaseDate.Equal(tt.wantDate) {
				t.Errorf("ReleaseDate: expected %v, got %v", tt.wantDate, e.ReleaseDate)
			}
			if e.Summary != tt.wantSummary {
				t.Errorf("Summary: expected %q, got %q", tt.wantSummary, e.Summary)
			}

			// OS versions: sort both for stable comparison since map iteration order is non-deterministic.
			gotVersions := make([]string, len(e.OSVersions))
			copy(gotVersions, e.OSVersions)
			sort.Strings(gotVersions)
			wantVersions := make([]string, len(tt.wantOSVersions))
			copy(wantVersions, tt.wantOSVersions)
			sort.Strings(wantVersions)

			if len(gotVersions) != len(wantVersions) {
				t.Fatalf("OSVersions: expected %v, got %v", wantVersions, gotVersions)
			}
			for i, v := range wantVersions {
				if gotVersions[i] != v {
					t.Errorf("OSVersions[%d]: expected %q, got %q", i, v, gotVersions[i])
				}
			}

			// Metadata.
			for k, wantV := range tt.wantMetadata {
				if gotV, ok := e.Metadata[k]; !ok {
					t.Errorf("Metadata[%q]: missing", k)
				} else if gotV != wantV {
					t.Errorf("Metadata[%q]: expected %q, got %q", k, wantV, gotV)
				}
			}
		})
	}
}

func TestDeriveSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cves []usnCVERef
		want string
	}{
		{"empty", nil, "medium"},
		{"no priorities", []usnCVERef{{ID: "CVE-1"}}, "medium"},
		{"single low", []usnCVERef{{ID: "CVE-1", Priority: "low"}}, "low"},
		{"single critical", []usnCVERef{{ID: "CVE-1", Priority: "critical"}}, "critical"},
		{"mixed high and low", []usnCVERef{
			{ID: "CVE-1", Priority: "low"},
			{ID: "CVE-2", Priority: "high"},
		}, "high"},
		{"mixed with empty", []usnCVERef{
			{ID: "CVE-1", Priority: ""},
			{ID: "CVE-2", Priority: "medium"},
		}, "medium"},
		{"all negligible", []usnCVERef{
			{ID: "CVE-1", Priority: "negligible"},
		}, "negligible"},
		{"case insensitive", []usnCVERef{
			{ID: "CVE-1", Priority: "High"},
		}, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deriveSeverity(tt.cves); got != tt.want {
				t.Errorf("deriveSeverity() = %q, want %q", got, tt.want)
			}
		})
	}
}
