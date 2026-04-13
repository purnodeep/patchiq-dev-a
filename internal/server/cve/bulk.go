package cve

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// BulkImporter reads NVD JSON files from disk and converts them to CVERecords.
type BulkImporter struct{}

// NewBulkImporter creates a new BulkImporter.
func NewBulkImporter() *BulkImporter {
	return &BulkImporter{}
}

// ImportDirectory reads all .json files in dir, parses them as NVD responses,
// and returns deduplicated CVERecords.
func (b *BulkImporter) ImportDirectory(ctx context.Context, dir string) ([]CVERecord, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	seen := make(map[string]struct{})
	var records []CVERecord
	var skippedFiles int
	var lastErr error

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		fileRecords, err := b.ImportFile(ctx, path)
		if err != nil {
			slog.WarnContext(ctx, "bulk import: skipping file", "path", path, "error", err)
			skippedFiles++
			lastErr = err
			continue
		}
		for _, r := range fileRecords {
			if _, ok := seen[r.CVEID]; ok {
				continue
			}
			seen[r.CVEID] = struct{}{}
			records = append(records, r)
		}
		slog.InfoContext(ctx, "bulk import: parsed file", "path", path, "cves", len(fileRecords))
	}

	if skippedFiles > 0 && len(records) == 0 {
		return nil, fmt.Errorf("bulk import: all %d JSON files failed, last error: %w", skippedFiles, lastErr)
	}
	if skippedFiles > 0 {
		slog.WarnContext(ctx, "bulk import: some files skipped", "skipped", skippedFiles, "imported_cves", len(records))
	}

	return records, nil
}

// ImportFile reads a single NVD JSON file and returns the parsed CVERecords.
func (b *BulkImporter) ImportFile(ctx context.Context, path string) ([]CVERecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}
	resp, err := ParseNVDResponse(data)
	if err != nil {
		return nil, fmt.Errorf("parse file %s: %w", path, err)
	}
	return NVDResponseToCVERecords(resp), nil
}
