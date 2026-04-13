package cve

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestIntegration_NVDFetchParseScore(t *testing.T) {
	testdata, err := os.ReadFile("testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(testdata) //nolint:errcheck
	}))
	defer srv.Close()

	client := NewNVDClient(srv.URL, "", 5*time.Second)
	records, err := client.FetchCVEs(context.Background(), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchCVEs: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	r := records[0]
	if r.CVEID != "CVE-2024-1234" {
		t.Errorf("CVEID = %q, want CVE-2024-1234", r.CVEID)
	}
	if r.CVSSv3Score != 9.8 {
		t.Errorf("CVSSv3Score = %.1f, want 9.8", r.CVSSv3Score)
	}
	if len(r.AffectedPackages) != 1 || r.AffectedPackages[0].PackageName != "openssl" {
		t.Errorf("unexpected affected packages: %+v", r.AffectedPackages)
	}

	now := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	score := ComputeRiskScore(r.CVSSv3Score, false, false, r.PublishedAt, now)
	if score < 9.0 {
		t.Errorf("expected high risk score for critical CVE, got %.2f", score)
	}
}

func TestIntegration_BulkImportAndCorrelate(t *testing.T) {
	importer := NewBulkImporter()
	records, err := importer.ImportFile(context.Background(), "testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	lister := &mockPatchLister{
		patches: map[string][]PatchInfo{
			"openssl": {{ID: "patch-1", Name: "openssl", Version: "3.0.13"}},
		},
	}
	linker := &mockCVELinker{}
	correlator := NewCorrelator(lister, linker)

	cveDBIDs := make(map[string]string)
	for i, rec := range records {
		cveDBIDs[rec.CVEID] = fmt.Sprintf("db-id-%d", i)
	}

	linked, err := correlator.Correlate(context.Background(), "tenant-1", records, cveDBIDs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if linked != 1 {
		t.Errorf("expected 1 link, got %d", linked)
	}
}

func TestIntegration_EndToEnd_MatchEndpoint(t *testing.T) {
	now := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	importer := NewBulkImporter()
	records, err := importer.ImportFile(context.Background(), "testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	pkgLister := &mockEndpointPackageLister{
		packages: []EndpointPackage{
			{Name: "openssl", Version: "3.0.2"},
		},
	}

	cveLookup := &mockCVELookup{cves: map[string][]MatchableCVE{}}
	for _, rec := range records {
		for _, pkg := range rec.AffectedPackages {
			cveLookup.cves[pkg.PackageName] = append(cveLookup.cves[pkg.PackageName], MatchableCVE{
				CVEDBID:             "db-" + rec.CVEID,
				CVEID:               rec.CVEID,
				CVSSv3Score:         rec.CVSSv3Score,
				PublishedAt:         rec.PublishedAt,
				VersionEndExcluding: pkg.VersionEndExcluding,
			})
		}
	}

	upserter := &mockEndpointCVEUpserter{}
	matcher := NewMatcher(pkgLister, cveLookup, upserter)

	result, err := matcher.MatchEndpoint(context.Background(), "tenant-1", "endpoint-1", "", now)
	if err != nil {
		t.Fatalf("MatchEndpoint: %v", err)
	}

	if result.Affected != 1 {
		t.Errorf("Affected = %d, want 1", result.Affected)
	}

	if len(upserter.upserted) == 0 {
		t.Fatal("expected at least 1 upserted record")
	}
	affected := upserter.upserted[0]
	if affected.RiskScore <= 0 {
		t.Errorf("expected positive risk score, got %.2f", affected.RiskScore)
	}
}
