package cve

import (
	"os"
	"testing"
	"time"
)

func TestParseNVDResponse(t *testing.T) {
	data, err := os.ReadFile("testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	resp, err := ParseNVDResponse(data)
	if err != nil {
		t.Fatalf("ParseNVDResponse: %v", err)
	}

	if resp.TotalResults != 2 {
		t.Errorf("TotalResults = %d, want 2", resp.TotalResults)
	}
	if len(resp.Vulnerabilities) != 2 {
		t.Fatalf("len(Vulnerabilities) = %d, want 2", len(resp.Vulnerabilities))
	}

	cve1 := resp.Vulnerabilities[0].CVE
	if cve1.ID != "CVE-2024-1234" {
		t.Errorf("cve1.ID = %q, want CVE-2024-1234", cve1.ID)
	}
	if cve1.Published.Year() != 2024 || cve1.Published.Month() != time.January {
		t.Errorf("cve1.Published = %v, want 2024-01-15", cve1.Published)
	}

	cve2 := resp.Vulnerabilities[1].CVE
	if cve2.ID != "CVE-2024-5678" {
		t.Errorf("cve2.ID = %q, want CVE-2024-5678", cve2.ID)
	}
}

func TestParsedCVEToCVERecord(t *testing.T) {
	data, err := os.ReadFile("testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	resp, err := ParseNVDResponse(data)
	if err != nil {
		t.Fatalf("ParseNVDResponse: %v", err)
	}

	records := NVDResponseToCVERecords(resp)
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}

	r1 := records[0]
	if r1.CVEID != "CVE-2024-1234" {
		t.Errorf("r1.CVEID = %q", r1.CVEID)
	}
	if r1.CVSSv3Score != 9.8 {
		t.Errorf("r1.CVSSv3Score = %.1f, want 9.8", r1.CVSSv3Score)
	}
	if r1.Severity != "critical" {
		t.Errorf("r1.Severity = %q, want critical", r1.Severity)
	}
	if r1.CVSSv3Vector != "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H" {
		t.Errorf("r1.CVSSv3Vector = %q", r1.CVSSv3Vector)
	}
	if r1.Description != "A buffer overflow in openssl allows remote code execution." {
		t.Errorf("r1.Description = %q", r1.Description)
	}
	if len(r1.AffectedPackages) != 1 {
		t.Fatalf("len(r1.AffectedPackages) = %d, want 1", len(r1.AffectedPackages))
	}
	if r1.AffectedPackages[0].PackageName != "openssl" {
		t.Errorf("PackageName = %q, want openssl", r1.AffectedPackages[0].PackageName)
	}
	if r1.AffectedPackages[0].VersionEndExcluding != "3.0.13" {
		t.Errorf("VersionEndExcluding = %q, want 3.0.13", r1.AffectedPackages[0].VersionEndExcluding)
	}

	r2 := records[1]
	if r2.AffectedPackages[0].PackageName != "curl" {
		t.Errorf("r2 PackageName = %q, want curl", r2.AffectedPackages[0].PackageName)
	}
}

func TestParseNVDResponse_EmptyVulnerabilities(t *testing.T) {
	data := []byte(`{"resultsPerPage":0,"startIndex":0,"totalResults":0,"format":"NVD_CVE","version":"2.0","vulnerabilities":[]}`)
	resp, err := ParseNVDResponse(data)
	if err != nil {
		t.Fatalf("ParseNVDResponse: %v", err)
	}
	if len(resp.Vulnerabilities) != 0 {
		t.Errorf("expected empty, got %d", len(resp.Vulnerabilities))
	}
}

func TestParseNVDResponse_InvalidJSON(t *testing.T) {
	_, err := ParseNVDResponse([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestExtractPackageNameFromCPE(t *testing.T) {
	tests := []struct {
		cpe  string
		want string
	}{
		{"cpe:2.3:a:openssl:openssl:*:*:*:*:*:*:*:*", "openssl"},
		{"cpe:2.3:a:haxx:curl:*:*:*:*:*:*:*:*", "curl"},
		{"cpe:2.3:a:apache:http_server:*:*:*:*:*:*:*:*", "http_server"},
		{"cpe:2.3:o:linux:linux_kernel:*:*:*:*:*:*:*:*", "linux_kernel"},
		{"invalid-cpe", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.cpe, func(t *testing.T) {
			got := ExtractPackageNameFromCPE(tt.cpe)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNVDResponseToCVERecords_NoCVSSMetrics(t *testing.T) {
	data := []byte(`{
		"resultsPerPage":1,"startIndex":0,"totalResults":1,"format":"NVD_CVE","version":"2.0",
		"vulnerabilities":[{
			"cve":{
				"id":"CVE-2024-9999",
				"published":"2024-06-01T00:00:00.000",
				"lastModified":"2024-06-01T00:00:00.000",
				"descriptions":[{"lang":"en","value":"No CVSS yet"}],
				"metrics":{},
				"configurations":[]
			}
		}]
	}`)
	resp, err := ParseNVDResponse(data)
	if err != nil {
		t.Fatalf("ParseNVDResponse: %v", err)
	}
	records := NVDResponseToCVERecords(resp)
	if len(records) != 1 {
		t.Fatalf("expected 1, got %d", len(records))
	}
	if records[0].CVSSv3Score != 0.0 {
		t.Errorf("expected 0.0 CVSS, got %.1f", records[0].CVSSv3Score)
	}
	if records[0].Severity != "none" {
		t.Errorf("expected none, got %q", records[0].Severity)
	}
}
