package feeds

import (
	"os"
	"testing"
	"time"
)

func TestCISAKEVFeedName(t *testing.T) {
	t.Parallel()

	feed := &CISAKEVFeed{}
	if got := feed.Name(); got != "cisa_kev" {
		t.Fatalf("expected name %q, got %q", "cisa_kev", got)
	}
}

func TestCISAKEVFeedParse(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdataPath("cisa_kev_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	feed := &CISAKEVFeed{}
	entries, err := feed.parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got := len(entries); got != 2 {
		t.Fatalf("expected 2 entries, got %d", got)
	}

	tests := []struct {
		name             string
		entry            RawEntry
		wantCVE          string
		wantVendor       string
		wantProduct      string
		wantDate         time.Time
		wantRansom       string
		wantDueDate      string
		wantKEVDueDate   time.Time
		wantRansomRefLen int
	}{
		{
			name:             "fortinet entry",
			entry:            entries[0],
			wantCVE:          "CVE-2024-21762",
			wantVendor:       "fortinet",
			wantProduct:      "FortiOS",
			wantDate:         time.Date(2024, 2, 9, 0, 0, 0, 0, time.UTC),
			wantRansom:       "Known",
			wantDueDate:      "2024-03-01",
			wantKEVDueDate:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			wantRansomRefLen: 1,
		},
		{
			name:             "ivanti entry",
			entry:            entries[1],
			wantCVE:          "CVE-2023-46805",
			wantVendor:       "ivanti",
			wantProduct:      "Connect Secure",
			wantDate:         time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			wantRansom:       "Unknown",
			wantDueDate:      "2024-01-31",
			wantKEVDueDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			wantRansomRefLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := tt.entry

			// CVE
			if len(e.CVEs) != 1 || e.CVEs[0] != tt.wantCVE {
				t.Errorf("CVEs = %v, want [%s]", e.CVEs, tt.wantCVE)
			}

			// Vendor must be lowercase
			if e.Vendor != tt.wantVendor {
				t.Errorf("Vendor = %q, want %q", e.Vendor, tt.wantVendor)
			}

			// Product
			if e.Product != tt.wantProduct {
				t.Errorf("Product = %q, want %q", e.Product, tt.wantProduct)
			}

			// Severity always critical
			if e.Severity != "critical" {
				t.Errorf("Severity = %q, want %q", e.Severity, "critical")
			}

			// ReleaseDate
			if !e.ReleaseDate.Equal(tt.wantDate) {
				t.Errorf("ReleaseDate = %v, want %v", e.ReleaseDate, tt.wantDate)
			}

			// Summary non-empty
			if e.Summary == "" {
				t.Error("Summary is empty")
			}

			// CISAKEVDueDate must be set and correct
			if e.CISAKEVDueDate == nil {
				t.Error("CISAKEVDueDate is nil, want non-nil")
			} else if !e.CISAKEVDueDate.Equal(tt.wantKEVDueDate) {
				t.Errorf("CISAKEVDueDate = %v, want %v", *e.CISAKEVDueDate, tt.wantKEVDueDate)
			}

			// References: ransomware ref only when campaign use is "Known"
			if len(e.References) != tt.wantRansomRefLen {
				t.Errorf("len(References) = %d, want %d", len(e.References), tt.wantRansomRefLen)
			}
			if tt.wantRansomRefLen > 0 {
				ref := e.References[0]
				if ref.URL == "" {
					t.Error("ransomware Reference.URL is empty")
				}
				if ref.Source == "" {
					t.Error("ransomware Reference.Source is empty")
				}
			}

			// Metadata backward compatibility
			if e.Metadata["ransomware"] != tt.wantRansom {
				t.Errorf("Metadata[ransomware] = %q, want %q", e.Metadata["ransomware"], tt.wantRansom)
			}
			if e.Metadata["due_date"] != tt.wantDueDate {
				t.Errorf("Metadata[due_date] = %q, want %q", e.Metadata["due_date"], tt.wantDueDate)
			}
			if e.Metadata["required_action"] == "" {
				t.Error("Metadata[required_action] is empty")
			}
		})
	}
}
