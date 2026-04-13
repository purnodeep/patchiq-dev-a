package cve

import (
	"context"
	"errors"
	"testing"
)

type mockPatchLister struct {
	patches map[string][]PatchInfo
	err     error  // if set, return this error for any call
	errPkg  string // if set, only error for this package name
}

func (m *mockPatchLister) ListPatchesByName(ctx context.Context, tenantID, packageName string) ([]PatchInfo, error) {
	if m.err != nil && (m.errPkg == "" || m.errPkg == packageName) {
		return nil, m.err
	}
	return m.patches[packageName], nil
}

type mockCVELinker struct {
	links []CVEPatchLink
	err   error  // if set, return this error for any call
	errID string // if set, only error for this patch ID
}

func (m *mockCVELinker) LinkPatchCVE(ctx context.Context, tenantID, patchID, cveDBID, versionEndExcluding, versionEndIncluding string) error {
	if m.err != nil && (m.errID == "" || m.errID == patchID) {
		return m.err
	}
	m.links = append(m.links, CVEPatchLink{PatchID: patchID, CVEDBID: cveDBID})
	return nil
}

func TestCorrelator_Correlate(t *testing.T) {
	lister := &mockPatchLister{
		patches: map[string][]PatchInfo{
			"openssl": {{ID: "patch-1", Name: "openssl", Version: "3.0.13"}},
			"curl":    {{ID: "patch-2", Name: "curl", Version: "8.6.0"}},
		},
	}
	linker := &mockCVELinker{}
	c := NewCorrelator(lister, linker)

	records := []CVERecord{
		{CVEID: "CVE-2024-1234", AffectedPackages: []AffectedPackage{{PackageName: "openssl"}}},
		{CVEID: "CVE-2024-5678", AffectedPackages: []AffectedPackage{{PackageName: "curl"}}},
		{CVEID: "CVE-2024-0000", AffectedPackages: []AffectedPackage{{PackageName: "nginx"}}},
	}
	cveDBIDs := map[string]string{
		"CVE-2024-1234": "db-id-1",
		"CVE-2024-5678": "db-id-2",
		"CVE-2024-0000": "db-id-3",
	}

	linked, err := c.Correlate(context.Background(), "tenant-1", records, cveDBIDs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if linked != 2 {
		t.Errorf("linked = %d, want 2", linked)
	}
	if len(linker.links) != 2 {
		t.Errorf("len(links) = %d, want 2", len(linker.links))
	}
}

func TestCorrelator_Correlate_NoPatchesFound(t *testing.T) {
	lister := &mockPatchLister{patches: map[string][]PatchInfo{}}
	linker := &mockCVELinker{}
	c := NewCorrelator(lister, linker)

	records := []CVERecord{
		{CVEID: "CVE-2024-1234", AffectedPackages: []AffectedPackage{{PackageName: "openssl"}}},
	}
	cveDBIDs := map[string]string{"CVE-2024-1234": "db-id-1"}

	linked, err := c.Correlate(context.Background(), "tenant-1", records, cveDBIDs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if linked != 0 {
		t.Errorf("expected 0, got %d", linked)
	}
}

func TestCorrelator_Correlate_NoAffectedPackages(t *testing.T) {
	lister := &mockPatchLister{}
	linker := &mockCVELinker{}
	c := NewCorrelator(lister, linker)

	records := []CVERecord{{CVEID: "CVE-2024-1234", AffectedPackages: nil}}
	cveDBIDs := map[string]string{"CVE-2024-1234": "db-id-1"}

	linked, err := c.Correlate(context.Background(), "tenant-1", records, cveDBIDs)
	if err != nil {
		t.Fatalf("Correlate: %v", err)
	}
	if linked != 0 {
		t.Errorf("expected 0, got %d", linked)
	}
}

func TestCorrelator_Correlate_ContinuesOnListerError(t *testing.T) {
	listerErr := errors.New("db connection failed")
	lister := &mockPatchLister{
		patches: map[string][]PatchInfo{
			"curl": {{ID: "patch-2", Name: "curl", Version: "8.6.0"}},
		},
		err:    listerErr,
		errPkg: "openssl", // only fail for openssl
	}
	linker := &mockCVELinker{}
	c := NewCorrelator(lister, linker)

	records := []CVERecord{
		{CVEID: "CVE-2024-1234", AffectedPackages: []AffectedPackage{{PackageName: "openssl"}}},
		{CVEID: "CVE-2024-5678", AffectedPackages: []AffectedPackage{{PackageName: "curl"}}},
	}
	cveDBIDs := map[string]string{
		"CVE-2024-1234": "db-id-1",
		"CVE-2024-5678": "db-id-2",
	}

	linked, err := c.Correlate(context.Background(), "tenant-1", records, cveDBIDs)

	// Should still link curl even though openssl failed
	if linked != 1 {
		t.Errorf("linked = %d, want 1 (should continue past errors)", linked)
	}
	// Should return an error that includes the lister failure
	if err == nil {
		t.Fatal("expected aggregate error, got nil")
	}
	if !errors.Is(err, listerErr) {
		t.Errorf("expected error to wrap listerErr, got: %v", err)
	}
}

func TestCorrelator_Correlate_ContinuesOnLinkerError(t *testing.T) {
	linkerErr := errors.New("link insert failed")
	lister := &mockPatchLister{
		patches: map[string][]PatchInfo{
			"openssl": {{ID: "patch-1", Name: "openssl", Version: "3.0.13"}},
			"curl":    {{ID: "patch-2", Name: "curl", Version: "8.6.0"}},
		},
	}
	linker := &mockCVELinker{
		err:   linkerErr,
		errID: "patch-1", // only fail for patch-1
	}
	c := NewCorrelator(lister, linker)

	records := []CVERecord{
		{CVEID: "CVE-2024-1234", AffectedPackages: []AffectedPackage{{PackageName: "openssl"}}},
		{CVEID: "CVE-2024-5678", AffectedPackages: []AffectedPackage{{PackageName: "curl"}}},
	}
	cveDBIDs := map[string]string{
		"CVE-2024-1234": "db-id-1",
		"CVE-2024-5678": "db-id-2",
	}

	linked, err := c.Correlate(context.Background(), "tenant-1", records, cveDBIDs)

	// curl should still be linked
	if linked != 1 {
		t.Errorf("linked = %d, want 1 (should continue past linker errors)", linked)
	}
	if err == nil {
		t.Fatal("expected aggregate error, got nil")
	}
	if !errors.Is(err, linkerErr) {
		t.Errorf("expected error to wrap linkerErr, got: %v", err)
	}
}
