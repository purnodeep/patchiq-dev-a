package cve

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBulkImporter_ImportDirectory(t *testing.T) {
	dir := t.TempDir()
	testdata, err := os.ReadFile("testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nvd-2024-part1.json"), testdata, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nvd-2024-part2.json"), testdata, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("skip"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	importer := NewBulkImporter()
	records, err := importer.ImportDirectory(context.Background(), dir)
	if err != nil {
		t.Fatalf("ImportDirectory: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 deduplicated records, got %d", len(records))
	}
}

func TestBulkImporter_ImportDirectory_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	importer := NewBulkImporter()
	records, err := importer.ImportDirectory(context.Background(), dir)
	if err != nil {
		t.Fatalf("ImportDirectory: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0, got %d", len(records))
	}
}

func TestBulkImporter_ImportDirectory_InvalidDir(t *testing.T) {
	importer := NewBulkImporter()
	_, err := importer.ImportDirectory(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBulkImporter_ImportFile(t *testing.T) {
	importer := NewBulkImporter()
	records, err := importer.ImportFile(context.Background(), "testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2, got %d", len(records))
	}
}

func TestBulkImporter_ImportFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badFile, []byte(`{invalid`), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	importer := NewBulkImporter()
	_, err := importer.ImportFile(context.Background(), badFile)
	if err == nil {
		t.Fatal("expected error")
	}
}
