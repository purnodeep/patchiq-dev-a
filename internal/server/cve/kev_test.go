package cve

import (
	"os"
	"testing"
)

func TestParseKEVCatalog(t *testing.T) {
	data, err := os.ReadFile("testdata/kev_catalog.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	catalog, err := ParseKEVCatalog(data)
	if err != nil {
		t.Fatalf("ParseKEVCatalog: %v", err)
	}
	if catalog.Count != 2 {
		t.Errorf("Count = %d, want 2", catalog.Count)
	}
	if len(catalog.Vulnerabilities) != 2 {
		t.Fatalf("len(Vulnerabilities) = %d, want 2", len(catalog.Vulnerabilities))
	}
	v1 := catalog.Vulnerabilities[0]
	if v1.CveID != "CVE-2024-1234" {
		t.Errorf("v1.CveID = %q", v1.CveID)
	}
	if v1.DueDate != "2024-02-10" {
		t.Errorf("v1.DueDate = %q", v1.DueDate)
	}
}

func TestKEVCatalogToMap(t *testing.T) {
	data, err := os.ReadFile("testdata/kev_catalog.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	catalog, err := ParseKEVCatalog(data)
	if err != nil {
		t.Fatalf("ParseKEVCatalog: %v", err)
	}
	kevMap := KEVCatalogToMap(catalog)
	if len(kevMap) != 2 {
		t.Fatalf("len(kevMap) = %d, want 2", len(kevMap))
	}
	entry, ok := kevMap["CVE-2024-1234"]
	if !ok {
		t.Fatal("expected CVE-2024-1234 in map")
	}
	if entry.DueDate != "2024-02-10" {
		t.Errorf("DueDate = %q", entry.DueDate)
	}
	_, ok = kevMap["CVE-2023-9999"]
	if !ok {
		t.Fatal("expected CVE-2023-9999 in map")
	}
}

func TestParseKEVCatalog_InvalidJSON(t *testing.T) {
	_, err := ParseKEVCatalog([]byte(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
