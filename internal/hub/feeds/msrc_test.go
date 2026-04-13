package feeds

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestMSRCFeedName(t *testing.T) {
	t.Parallel()

	feed := NewMSRCFeed(nil)
	if got := feed.Name(); got != "msrc" {
		t.Fatalf("expected name %q, got %q", "msrc", got)
	}
}

func TestMSRCFeed_InstallerType(t *testing.T) {
	t.Parallel()

	feed := NewMSRCFeed(nil)

	data := []byte(`{
		"value": [{
			"ID": "2024-Feb",
			"InitialReleaseDate": "2024-02-13T08:00:00Z",
			"Vulnerabilities": [{
				"CVE": "CVE-2024-1234",
				"Title": "Windows Kernel Elevation of Privilege Vulnerability",
				"Severity": "Important",
				"AffectedProducts": ["Windows 11"],
				"KBArticles": [{"ID": "5034765", "URL": "https://support.microsoft.com/kb/5034765"}]
			}]
		}]
	}`)

	entries, err := feed.parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].InstallerType != "wua" {
		t.Errorf("InstallerType: expected %q, got %q", "wua", entries[0].InstallerType)
	}
}

func TestMSRCFeedParse(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdataPath("msrc_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	feed := NewMSRCFeed(nil)
	entries, err := feed.parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	tests := []struct {
		name                string
		entry               RawEntry
		wantCVEs            []string
		wantName            string
		wantSummaryContains string
		wantVendor          string
		wantProduct         string
		wantSev             string
		wantOS              string
		wantURL             string
		wantDate            time.Time
		wantMeta            map[string]string
		wantReferences      []CVEReference
	}{
		{
			name:                "SmartScreen vulnerability",
			entry:               entries[0],
			wantCVEs:            []string{"CVE-2024-21351"},
			wantName:            "KB5034763",
			wantSummaryContains: "Windows SmartScreen Security Feature Bypass",
			wantVendor:          "microsoft",
			wantProduct:         "Windows 11 Version 23H2",
			wantSev:             "high",
			wantOS:              "windows",
			wantURL:             "https://support.microsoft.com/kb/5034763",
			wantDate:            time.Date(2024, 2, 13, 8, 0, 0, 0, time.UTC),
			wantMeta: map[string]string{
				"update_id":  "2024-Feb",
				"kb_article": "5034763",
			},
			wantReferences: []CVEReference{
				{URL: "https://support.microsoft.com/kb/5034763", Source: "msrc"},
			},
		},
		{
			name:                "Internet Shortcut Files vulnerability",
			entry:               entries[1],
			wantCVEs:            []string{"CVE-2024-21412"},
			wantName:            "KB5034763",
			wantSummaryContains: "Internet Shortcut Files Security Feature Bypass",
			wantVendor:          "microsoft",
			wantProduct:         "Windows 11 Version 23H2",
			wantSev:             "critical",
			wantOS:              "windows",
			wantURL:             "https://support.microsoft.com/kb/5034763",
			wantDate:            time.Date(2024, 2, 13, 8, 0, 0, 0, time.UTC),
			wantMeta: map[string]string{
				"update_id":  "2024-Feb",
				"kb_article": "5034763",
			},
			wantReferences: []CVEReference{
				{URL: "https://support.microsoft.com/kb/5034763", Source: "msrc"},
			},
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
			if tt.wantSummaryContains != "" && !strings.Contains(e.Summary, tt.wantSummaryContains) {
				t.Errorf("Summary: expected to contain %q, got %q", tt.wantSummaryContains, e.Summary)
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
			if e.OSFamily != tt.wantOS {
				t.Errorf("OSFamily: expected %q, got %q", tt.wantOS, e.OSFamily)
			}
			if e.SourceURL != tt.wantURL {
				t.Errorf("SourceURL: expected %q, got %q", tt.wantURL, e.SourceURL)
			}
			if !e.ReleaseDate.Equal(tt.wantDate) {
				t.Errorf("ReleaseDate: expected %v, got %v", tt.wantDate, e.ReleaseDate)
			}

			for k, wantV := range tt.wantMeta {
				if gotV, ok := e.Metadata[k]; !ok {
					t.Errorf("Metadata[%q]: missing", k)
				} else if gotV != wantV {
					t.Errorf("Metadata[%q]: expected %q, got %q", k, wantV, gotV)
				}
			}

			if len(e.References) != len(tt.wantReferences) {
				t.Fatalf("References: expected %d, got %d", len(tt.wantReferences), len(e.References))
			}
			for i, want := range tt.wantReferences {
				got := e.References[i]
				if got.URL != want.URL {
					t.Errorf("References[%d].URL: expected %q, got %q", i, want.URL, got.URL)
				}
				if got.Source != want.Source {
					t.Errorf("References[%d].Source: expected %q, got %q", i, want.Source, got.Source)
				}
			}
		})
	}
}

func TestParseMSRCUpdateID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		wantErr bool
		want    time.Time
	}{
		{"valid Jan", "2025-Jan", false, time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"valid Aug", "2025-Aug", false, time.Date(2025, time.August, 1, 0, 0, 0, 0, time.UTC)},
		{"valid Dec", "2024-Dec", false, time.Date(2024, time.December, 1, 0, 0, 0, 0, time.UTC)},
		{"empty", "", true, time.Time{}},
		{"invalid format", "2025-13", true, time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseMSRCUpdateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseMSRCUpdateID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("parseMSRCUpdateID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestMSRCCursorChronologicalComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		a, b   string
		aAfter bool
	}{
		{"Aug after Feb", "2025-Aug", "2025-Feb", true},
		{"Jan before Feb", "2025-Jan", "2025-Feb", false},
		{"Dec 2024 before Jan 2025", "2024-Dec", "2025-Jan", false},
		{"Jan 2026 after Dec 2025", "2026-Jan", "2025-Dec", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ta, err := parseMSRCUpdateID(tt.a)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.a, err)
			}
			tb, err := parseMSRCUpdateID(tt.b)
			if err != nil {
				t.Fatalf("parse %q: %v", tt.b, err)
			}
			if got := ta.After(tb); got != tt.aAfter {
				t.Errorf("%q.After(%q) = %v, want %v", tt.a, tt.b, got, tt.aAfter)
			}
		})
	}
}
