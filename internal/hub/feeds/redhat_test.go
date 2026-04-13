package feeds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRedHatFeedParseReferences(t *testing.T) {
	t.Parallel()

	const xmlData = `<?xml version="1.0" encoding="UTF-8"?>
<oval_definitions xmlns="http://oval.mitre.org/XMLSchema/oval-definitions-5">
  <definitions>
    <definition id="oval:com.redhat.rhsa:def:20240893" version="1" class="patch">
      <metadata>
        <title>RHSA-2024:0893: python3 security update (Important)</title>
        <affected family="unix">
          <platform>Red Hat Enterprise Linux 9</platform>
        </affected>
        <reference source="RHSA" ref_id="RHSA-2024:0893" ref_url="https://access.redhat.com/errata/RHSA-2024:0893"/>
        <reference source="CVE" ref_id="CVE-2023-6597" ref_url="https://access.redhat.com/security/cve/CVE-2023-6597"/>
        <reference source="CVE" ref_id="CVE-2024-0450" ref_url="https://access.redhat.com/security/cve/CVE-2024-0450"/>
        <advisory>
          <severity>Important</severity>
          <issued date="2024-02-20"/>
          <updated date="2024-02-20"/>
        </advisory>
        <description>Python security fix.</description>
      </metadata>
    </definition>
  </definitions>
</oval_definitions>`

	feed := NewRedHatFeed(nil)
	entries, err := feed.parse([]byte(xmlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	refs := entries[0].References
	// Expect: 1 advisory ref + 2 CVE refs = 3 total.
	if len(refs) != 3 {
		t.Fatalf("expected 3 references, got %d: %v", len(refs), refs)
	}

	// First ref: advisory.
	if refs[0].URL != "https://access.redhat.com/errata/RHSA-2024:0893" {
		t.Errorf("refs[0].URL: expected advisory URL, got %q", refs[0].URL)
	}
	if refs[0].Source != "redhat" {
		t.Errorf("refs[0].Source: expected %q, got %q", "redhat", refs[0].Source)
	}

	// Second ref: first CVE.
	if refs[1].URL != "https://access.redhat.com/security/cve/CVE-2023-6597" {
		t.Errorf("refs[1].URL: expected CVE-2023-6597 URL, got %q", refs[1].URL)
	}
	if refs[1].Source != "cve" {
		t.Errorf("refs[1].Source: expected %q, got %q", "cve", refs[1].Source)
	}

	// Third ref: second CVE.
	if refs[2].URL != "https://access.redhat.com/security/cve/CVE-2024-0450" {
		t.Errorf("refs[2].URL: expected CVE-2024-0450 URL, got %q", refs[2].URL)
	}
	if refs[2].Source != "cve" {
		t.Errorf("refs[2].Source: expected %q, got %q", "cve", refs[2].Source)
	}
}

func TestRedHatFeedFetchBzip2(t *testing.T) {
	t.Parallel()

	compressed, err := os.ReadFile(testdataPath("redhat_oval_sample.xml.bz2"))
	if err != nil {
		t.Fatalf("read compressed fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Last-Modified", "Tue, 20 Feb 2024 12:00:00 GMT")
		_, _ = w.Write(compressed)
	}))
	defer srv.Close()

	feed := NewRedHatFeed(srv.Client())
	feed.urls = []string{srv.URL}

	entries, cursor, err := feed.Fetch(context.Background(), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "RHSA-2024:0893" {
		t.Errorf("entry[0].Name: expected RHSA-2024:0893, got %s", entries[0].Name)
	}
	if cursor != "Tue, 20 Feb 2024 12:00:00 GMT" {
		t.Errorf("cursor: expected Last-Modified value, got %q", cursor)
	}
}

func TestRedHatFeedName(t *testing.T) {
	t.Parallel()

	feed := NewRedHatFeed(nil)
	if got := feed.Name(); got != "redhat_oval" {
		t.Fatalf("expected name %q, got %q", "redhat_oval", got)
	}
}

func TestRedHatFeedParse(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdataPath("redhat_oval_sample.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	feed := NewRedHatFeed(nil)
	entries, err := feed.parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Only class="patch" definitions should be included (2 of 3 in fixture).
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	tests := []struct {
		name           string
		entry          RawEntry
		wantCVEs       []string
		wantName       string
		wantVendor     string
		wantProduct    string
		wantSev        string
		wantOSFamily   string
		wantOSVersions []string
		wantInstaller  string
		wantSummary    string
		wantURL        string
		wantDate       time.Time
	}{
		{
			name:           "RHSA with multiple CVEs",
			entry:          entries[0],
			wantCVEs:       []string{"CVE-2023-6597", "CVE-2024-0450"},
			wantName:       "RHSA-2024:0893",
			wantVendor:     "redhat",
			wantProduct:    "python3",
			wantSev:        "important",
			wantOSFamily:   "linux",
			wantOSVersions: []string{"9"},
			wantInstaller:  "rpm",
			wantSummary:    "Python is an interpreted, interactive, object-oriented programming language that supports modules, classes, exceptions, high-level dynamic data types, and dynamic typing.",
			wantURL:        "https://access.redhat.com/security/cve/CVE-2023-6597",
			wantDate:       time.Date(2024, 2, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name:           "RHSA with single CVE",
			entry:          entries[1],
			wantCVEs:       []string{"CVE-2023-42465"},
			wantName:       "RHSA-2024:0811",
			wantVendor:     "redhat",
			wantProduct:    "sudo",
			wantSev:        "moderate",
			wantOSFamily:   "linux",
			wantOSVersions: []string{"8"},
			wantInstaller:  "rpm",
			wantSummary:    "The sudo packages contain the sudo utility which allows system administrators to provide certain users with the permission to execute privileged commands.",
			wantURL:        "https://access.redhat.com/security/cve/CVE-2023-42465",
			wantDate:       time.Date(2024, 2, 14, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := tt.entry

			if len(e.CVEs) != len(tt.wantCVEs) {
				t.Fatalf("CVEs: expected %v, got %v", tt.wantCVEs, e.CVEs)
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
			if e.Severity != tt.wantSev {
				t.Errorf("Severity: expected %q, got %q", tt.wantSev, e.Severity)
			}
			if e.OSFamily != tt.wantOSFamily {
				t.Errorf("OSFamily: expected %q, got %q", tt.wantOSFamily, e.OSFamily)
			}
			if e.InstallerType != tt.wantInstaller {
				t.Errorf("InstallerType: expected %q, got %q", tt.wantInstaller, e.InstallerType)
			}
			if e.SourceURL != tt.wantURL {
				t.Errorf("SourceURL: expected %q, got %q", tt.wantURL, e.SourceURL)
			}
			if !e.ReleaseDate.Equal(tt.wantDate) {
				t.Errorf("ReleaseDate: expected %v, got %v", tt.wantDate, e.ReleaseDate)
			}
			if e.Summary != tt.wantSummary {
				t.Errorf("Summary: expected %q, got %q", tt.wantSummary, e.Summary)
			}

			if len(e.OSVersions) != len(tt.wantOSVersions) {
				t.Fatalf("OSVersions: expected %v, got %v", tt.wantOSVersions, e.OSVersions)
			}
			for i, v := range tt.wantOSVersions {
				if e.OSVersions[i] != v {
					t.Errorf("OSVersions[%d]: expected %q, got %q", i, v, e.OSVersions[i])
				}
			}
		})
	}
}
