package feeds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "feeds", name)
}

func TestNVDFeedName(t *testing.T) {
	t.Parallel()

	feed := NewNVDFeed(nil, "")
	if got := feed.Name(); got != "nvd" {
		t.Fatalf("expected name %q, got %q", "nvd", got)
	}
}

func TestNVDFeedParse(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(testdataPath("nvd_sample.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	feed := NewNVDFeed(nil, "")
	_, entries, _, err := feed.parsePage(data)
	if err != nil {
		t.Fatalf("parsePage: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	tests := []struct {
		name         string
		entry        RawEntry
		wantCVEs     []string
		wantName     string
		wantVendor   string
		wantProduct  string
		wantSev      string
		wantScore    float64
		wantVector   string
		wantAV       string
		wantCweID    string
		wantRefCount int
		wantSummary  string
		wantURL      string
		wantDate     time.Time
		wantLastMod  time.Time
	}{
		{
			name:         "CVE with CPE config",
			entry:        entries[0],
			wantCVEs:     []string{"CVE-2024-21762"},
			wantName:     "CVE-2024-21762",
			wantVendor:   "fortinet",
			wantProduct:  "fortios",
			wantSev:      "critical",
			wantScore:    9.8,
			wantVector:   "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			wantAV:       "NETWORK",
			wantCweID:    "CWE-787",
			wantRefCount: 1,
			wantSummary:  "FortiOS out-of-bounds write vulnerability in HTTP/2 handler allows remote code execution",
			wantURL:      "https://fortiguard.fortinet.com/psirt/FG-IR-24-015",
			wantDate:     time.Date(2024, 2, 9, 0, 0, 0, 0, time.UTC),
			wantLastMod:  time.Date(2024, 2, 12, 0, 0, 0, 0, time.UTC),
		},
		{
			name:         "CVE without CPE config defaults vendor to nist",
			entry:        entries[1],
			wantCVEs:     []string{"CVE-2024-0056"},
			wantName:     "CVE-2024-0056",
			wantVendor:   "nist",
			wantProduct:  "",
			wantSev:      "medium",
			wantScore:    6.8,
			wantVector:   "CVSS:3.1/AV:N/AC:H/PR:H/UI:R/S:U/C:H/I:H/A:N",
			wantAV:       "NETWORK",
			wantCweID:    "",
			wantRefCount: 1,
			wantSummary:  "Microsoft.Data.SqlClient and System.Data.SqlClient SQL Data Provider Security Feature Bypass Vulnerability",
			wantURL:      "https://msrc.microsoft.com/update-guide/vulnerability/CVE-2024-0056",
			wantDate:     time.Date(2024, 1, 9, 18, 15, 0, 0, time.UTC),
			wantLastMod:  time.Date(2024, 1, 12, 0, 0, 0, 0, time.UTC),
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
			if e.CVSSScore != tt.wantScore {
				t.Errorf("CVSSScore: expected %f, got %f", tt.wantScore, e.CVSSScore)
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
			if e.CVSSv3Vector != tt.wantVector {
				t.Errorf("CVSSv3Vector: expected %q, got %q", tt.wantVector, e.CVSSv3Vector)
			}
			if e.AttackVector != tt.wantAV {
				t.Errorf("AttackVector: expected %q, got %q", tt.wantAV, e.AttackVector)
			}
			if e.CweID != tt.wantCweID {
				t.Errorf("CweID: expected %q, got %q", tt.wantCweID, e.CweID)
			}
			if len(e.References) != tt.wantRefCount {
				t.Errorf("References: expected %d, got %d", tt.wantRefCount, len(e.References))
			}
			if e.NVDLastModified == nil {
				t.Error("NVDLastModified: expected non-nil")
			} else if !e.NVDLastModified.Equal(tt.wantLastMod) {
				t.Errorf("NVDLastModified: expected %v, got %v", tt.wantLastMod, *e.NVDLastModified)
			}
		})
	}
}

func TestNVDFeedFetchPagination(t *testing.T) {
	t.Parallel()

	page1, err := os.ReadFile(testdataPath("nvd_paginated_page1.json"))
	if err != nil {
		t.Fatalf("read page1 fixture: %v", err)
	}
	page2, err := os.ReadFile(testdataPath("nvd_paginated_page2.json"))
	if err != nil {
		t.Fatalf("read page2 fixture: %v", err)
	}

	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		startIndex := r.URL.Query().Get("startIndex")
		w.Header().Set("Content-Type", "application/json")
		if startIndex == "1" {
			_, _ = w.Write(page2)
		} else {
			_, _ = w.Write(page1)
		}
	}))
	defer srv.Close()

	feed := NewNVDFeed(srv.Client(), "")
	feed.baseURL = srv.URL
	feed.pageDelay = 0 // disable rate-limit delay in tests

	entries, nextCursor, err := feed.Fetch(context.Background(), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if got := requestCount.Load(); got != 2 {
		t.Errorf("expected 2 HTTP requests, got %d", got)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	wantCVEs := []string{"CVE-2024-21762", "CVE-2024-0056"}
	for i, want := range wantCVEs {
		if entries[i].Name != want {
			t.Errorf("entry[%d].Name: expected %q, got %q", i, want, entries[i].Name)
		}
	}

	// Cursor must be the maximum lastModified across all pages (2024-02-12),
	// not the maximum published date (2024-02-09). Using lastModified ensures
	// CVEs updated after their publish date are not missed on the next sync.
	wantCursor := "2024-02-12T00:00:00Z"
	if nextCursor != wantCursor {
		t.Errorf("nextCursor: expected %q (max lastModified), got %q", wantCursor, nextCursor)
	}
}

// TestNVDBuildPageURLDateFormat verifies that buildPageURL formats cursor dates
// as "2006-01-02T15:04:05.000" (NVD requirement) and NOT as RFC3339 (which
// appends a "Z" suffix that causes NVD to return 404).
func TestNVDBuildPageURLDateFormat(t *testing.T) {
	t.Parallel()

	feed := NewNVDFeed(nil, "")
	feed.baseURL = "https://example.com/cves"

	contains := func(s, sub string) bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}

	// Cursor stored as RFC3339 (as it is in the DB).
	cursor := "2024-02-12T10:30:00Z"
	got := feed.buildPageURL(cursor, 0)

	// The lastModStartDate must be reformatted to NVD format (no Z suffix, has .000).
	wantStart := "lastModStartDate=2024-02-12T10:30:00.000"
	if !contains(got, wantStart) {
		t.Errorf("buildPageURL URL missing NVD-formatted lastModStartDate %q:\nURL: %s", wantStart, got)
	}

	// Must NOT contain the bare RFC3339 format (Z suffix without milliseconds).
	badFormat := "lastModStartDate=2024-02-12T10:30:00Z"
	if contains(got, badFormat) {
		t.Errorf("buildPageURL produced RFC3339 format (Z suffix causes NVD 404): %s", got)
	}
}

func TestNVDFeedFetchZeroResultsPerPage(t *testing.T) {
	t.Parallel()

	// A malformed response with resultsPerPage=0 and outstanding results
	// should return an error (not silently drop data).
	malformed := []byte(`{
		"resultsPerPage": 0,
		"startIndex": 0,
		"totalResults": 100,
		"vulnerabilities": []
	}`)

	var requestCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(malformed)
	}))
	defer srv.Close()

	feed := NewNVDFeed(srv.Client(), "")
	feed.baseURL = srv.URL

	_, _, err := feed.Fetch(context.Background(), "")
	if err == nil {
		t.Fatal("expected error when resultsPerPage=0 with outstanding results")
	}
	if got := requestCount.Load(); got != 1 {
		t.Errorf("expected 1 request (loop should break), got %d", got)
	}
}
