package feeds

import (
	"os"
	"testing"
	"time"
)

func TestAppleFeedParseReferences(t *testing.T) {
	t.Parallel()

	const jsonData = `[{"name":"macOS Sonoma 14.4","url":"https://support.apple.com/en-us/HT214084","releaseDate":"07 Mar 2024","os":"macOS","cves":["CVE-2024-23296"]}]`

	feed := NewAppleFeed(nil)
	entries, err := feed.parse([]byte(jsonData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	refs := entries[0].References
	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d: %v", len(refs), refs)
	}
	if refs[0].URL != "https://support.apple.com/en-us/HT214084" {
		t.Errorf("refs[0].URL: expected support article URL, got %q", refs[0].URL)
	}
	if refs[0].Source != "apple" {
		t.Errorf("refs[0].Source: expected %q, got %q", "apple", refs[0].Source)
	}
}

func TestAppleFeedName(t *testing.T) {
	t.Parallel()

	feed := NewAppleFeed(nil)
	if got := feed.Name(); got != "apple" {
		t.Fatalf("expected name %q, got %q", "apple", got)
	}
}

func TestAppleFeedParse(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdataPath("apple_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	feed := NewAppleFeed(nil)
	entries, err := feed.parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	tests := []struct {
		name          string
		entry         RawEntry
		wantCVECount  int
		wantFirstCVE  string
		wantName      string
		wantVendor    string
		wantProduct   string
		wantVersion   string
		wantOSFamily  string
		wantSeverity  string
		wantInstaller string
		wantSummary   string
		wantURL       string
		wantDate      time.Time
	}{
		{
			name:          "macOS Sonoma 14.4",
			entry:         entries[0],
			wantCVECount:  2,
			wantFirstCVE:  "CVE-2024-23296",
			wantName:      "macOS Sonoma 14.4",
			wantVendor:    "apple",
			wantProduct:   "macOS Sonoma 14.4",
			wantVersion:   "14.4",
			wantOSFamily:  "macos",
			wantSeverity:  "high",
			wantInstaller: "pkg",
			wantSummary:   "macOS Sonoma 14.4 security update",
			wantURL:       "https://support.apple.com/en-us/HT214084",
			wantDate:      time.Date(2024, 3, 7, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "iOS 17.4 and iPadOS 17.4",
			entry:         entries[1],
			wantCVECount:  1,
			wantFirstCVE:  "CVE-2024-23296",
			wantName:      "iOS 17.4 and iPadOS 17.4",
			wantVendor:    "apple",
			wantProduct:   "iOS 17.4 and iPadOS 17.4",
			wantVersion:   "17.4",
			wantOSFamily:  "ios",
			wantSeverity:  "high",
			wantInstaller: "pkg",
			wantSummary:   "iOS 17.4 and iPadOS 17.4 security update",
			wantURL:       "https://support.apple.com/en-us/HT214081",
			wantDate:      time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:          "Safari 17.4",
			entry:         entries[2],
			wantCVECount:  3,
			wantFirstCVE:  "CVE-2024-23252",
			wantName:      "Safari 17.4",
			wantVendor:    "apple",
			wantProduct:   "Safari 17.4",
			wantVersion:   "17.4",
			wantOSFamily:  "macos",
			wantSeverity:  "high",
			wantInstaller: "pkg",
			wantSummary:   "Safari 17.4 security update",
			wantURL:       "https://support.apple.com/en-us/HT214089",
			wantDate:      time.Date(2024, 3, 7, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := tt.entry

			if len(e.CVEs) != tt.wantCVECount {
				t.Fatalf("CVE count: expected %d, got %d", tt.wantCVECount, len(e.CVEs))
			}
			if e.CVEs[0] != tt.wantFirstCVE {
				t.Errorf("first CVE: expected %q, got %q", tt.wantFirstCVE, e.CVEs[0])
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
			if e.OSFamily != tt.wantOSFamily {
				t.Errorf("OSFamily: expected %q, got %q", tt.wantOSFamily, e.OSFamily)
			}
			if e.Severity != tt.wantSeverity {
				t.Errorf("Severity: expected %q, got %q", tt.wantSeverity, e.Severity)
			}
			if e.InstallerType != tt.wantInstaller {
				t.Errorf("InstallerType: expected %q, got %q", tt.wantInstaller, e.InstallerType)
			}
			if e.Summary != tt.wantSummary {
				t.Errorf("Summary: expected %q, got %q", tt.wantSummary, e.Summary)
			}
			if e.SourceURL != tt.wantURL {
				t.Errorf("SourceURL: expected %q, got %q", tt.wantURL, e.SourceURL)
			}
			if !e.ReleaseDate.Equal(tt.wantDate) {
				t.Errorf("ReleaseDate: expected %v, got %v", tt.wantDate, e.ReleaseDate)
			}
		})
	}
}
