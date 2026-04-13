//go:build integration

package discovery

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_APT_UbuntuSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	fetcher := NewFetcher(60*time.Second, 3)
	url := "http://archive.ubuntu.com/ubuntu/dists/jammy-security/main/binary-amd64/Packages.gz"

	body, err := fetcher.Fetch(context.Background(), url)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	defer body.Close()

	parser := &APTParser{
		OsFamily:   "debian",
		OsDistro:   "ubuntu-22.04",
		SourceRepo: url,
	}

	var count int
	for p, err := range parser.Parse(context.Background(), body) {
		if err != nil {
			t.Fatalf("parse error after %d packages: %v", count, err)
		}
		if p.Name == "" {
			t.Fatal("parsed patch with empty name")
		}
		if p.Version == "" {
			t.Fatal("parsed patch with empty version")
		}
		count++
	}

	t.Logf("parsed %d packages from Ubuntu jammy-security", count)
	if count < 100 {
		t.Fatalf("expected at least 100 packages, got %d", count)
	}
}
