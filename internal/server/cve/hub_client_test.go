package cve

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHubCVEClient_FetchCVEs(t *testing.T) {
	published := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	modified := time.Date(2024, 3, 20, 12, 0, 0, 0, time.UTC)

	mockResp := hubCVEResponse{
		ServerTime: time.Now().UTC().Format(time.RFC3339),
		CVEs: []hubCVEFeed{
			{
				CVEID:              "CVE-2024-1234",
				Severity:           "high",
				Description:        "A critical buffer overflow vulnerability",
				PublishedAt:        published.Format(time.RFC3339),
				Source:             "NVD",
				CVSSv3Score:        "8.5",
				CVSSv3Vector:       "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
				AttackVector:       "Network",
				CweID:              "CWE-119",
				CisaKEVDueDate:     "2024-04-01",
				ExternalReferences: json.RawMessage(`"W3siVVJMIjoiaHR0cHM6Ly9udmQubmlzdC5nb3YvdnVsbi9kZXRhaWwvQ1ZFLTIwMjQtMTIzNCIsIlNvdXJjZSI6Im52ZCJ9LHsiVVJMIjoiaHR0cHM6Ly9leGFtcGxlLmNvbS9hZHZpc29yeSIsIlNvdXJjZSI6ImV4YW1wbGUifV0="`),
				NVDLastModified:    modified.Format(time.RFC3339),
				ExploitKnown:       true,
				InKEV:              true,
			},
			{
				CVEID:       "CVE-2024-5678",
				Severity:    "medium",
				Description: "An information disclosure flaw",
				PublishedAt: published.Format(time.RFC3339),
				Source:      "NVD",
				CVSSv3Score: "5.3",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/sync/cves" {
			http.NotFound(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Query().Get("since") == "" {
			http.Error(w, "missing since param", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockResp); err != nil {
			t.Errorf("encode mock response: %v", err)
		}
	}))
	defer srv.Close()

	client := NewHubCVEClient(srv.URL, "test-api-key")
	since := time.Now().UTC().Add(-24 * time.Hour)

	records, err := client.FetchCVEs(context.Background(), since)
	if err != nil {
		t.Fatalf("FetchCVEs returned unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 CVERecords, got %d", len(records))
	}

	r := records[0]
	if r.CVEID != "CVE-2024-1234" {
		t.Errorf("CVEID: want %q, got %q", "CVE-2024-1234", r.CVEID)
	}
	if r.Severity != "high" {
		t.Errorf("Severity: want %q, got %q", "high", r.Severity)
	}
	if r.Description != "A critical buffer overflow vulnerability" {
		t.Errorf("Description mismatch: got %q", r.Description)
	}
	if r.CVSSv3Score != 8.5 {
		t.Errorf("CVSSv3Score: want 8.5, got %f", r.CVSSv3Score)
	}
	if r.CVSSv3Vector != "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H" {
		t.Errorf("CVSSv3Vector mismatch: got %q", r.CVSSv3Vector)
	}
	if r.AttackVector != "Network" {
		t.Errorf("AttackVector: want %q, got %q", "Network", r.AttackVector)
	}
	if r.CweID != "CWE-119" {
		t.Errorf("CweID: want %q, got %q", "CWE-119", r.CweID)
	}
	if r.Source != "NVD" {
		t.Errorf("Source: want %q, got %q", "NVD", r.Source)
	}
	if !r.PublishedAt.Equal(published) {
		t.Errorf("PublishedAt: want %v, got %v", published, r.PublishedAt)
	}
	if !r.LastModified.Equal(modified) {
		t.Errorf("LastModified: want %v, got %v", modified, r.LastModified)
	}
	if len(r.References) != 2 {
		t.Errorf("References: want 2, got %d", len(r.References))
	}
	if r.References[0].URL != "https://nvd.nist.gov/vuln/detail/CVE-2024-1234" {
		t.Errorf("References[0].URL mismatch: got %q", r.References[0].URL)
	}

	r2 := records[1]
	if r2.CVEID != "CVE-2024-5678" {
		t.Errorf("CVEID: want %q, got %q", "CVE-2024-5678", r2.CVEID)
	}
	if r2.CVSSv3Score != 5.3 {
		t.Errorf("CVSSv3Score: want 5.3, got %f", r2.CVSSv3Score)
	}
}

func TestHubCVEClient_FetchCVEs_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := NewHubCVEClient(srv.URL, "wrong-key")
	_, err := client.FetchCVEs(context.Background(), time.Now().UTC().Add(-time.Hour))
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
}

func TestHubCVEClient_FetchKEV(t *testing.T) {
	client := NewHubCVEClient("http://example.com", "key")
	kevMap, err := client.FetchKEV(context.Background())
	if err != nil {
		t.Fatalf("FetchKEV returned unexpected error: %v", err)
	}
	if len(kevMap) != 0 {
		t.Errorf("expected empty KEV map, got %d entries", len(kevMap))
	}
}

// Verify HubCVEClient satisfies the CVEFetcher interface at compile time.
var _ CVEFetcher = (*HubCVEClient)(nil)
