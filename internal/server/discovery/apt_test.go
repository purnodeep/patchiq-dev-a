package discovery

import (
	"bytes"
	"compress/gzip"
	"context"
	"testing"
)

func gzipString(t *testing.T, s string) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func TestAPTParser(t *testing.T) {
	raw := "Package: curl\nVersion: 7.81.0-1ubuntu1.16\nPriority: optional\nSection: web\nArchitecture: amd64\nFilename: pool/main/c/curl/curl_7.81.0-1ubuntu1.16_amd64.deb\nSize: 194560\nSHA256: abc123def456\nDescription: command line tool for transferring data with URL syntax\n\nPackage: openssl\nVersion: 3.0.2-0ubuntu1.15\nPriority: optional\nSection: utils\nArchitecture: amd64\nFilename: pool/main/o/openssl/openssl_3.0.2-0ubuntu1.15_amd64.deb\nSize: 1298432\nSHA256: def789abc012\nDescription: Secure Sockets Layer toolkit\n"

	tests := []struct {
		name         string
		wantName     string
		wantVer      string
		wantSHA      string
		wantSize     int64
		wantPriority string
		wantSection  string
		wantFilename string
		wantDesc     string
	}{
		{"first package", "curl", "7.81.0-1ubuntu1.16", "abc123def456", 194560, "optional", "web", "pool/main/c/curl/curl_7.81.0-1ubuntu1.16_amd64.deb", "command line tool for transferring data with URL syntax"},
		{"second package", "openssl", "3.0.2-0ubuntu1.15", "def789abc012", 1298432, "optional", "utils", "pool/main/o/openssl/openssl_3.0.2-0ubuntu1.15_amd64.deb", "Secure Sockets Layer toolkit"},
	}

	parser := &APTParser{OsFamily: "debian", OsDistro: "ubuntu-22.04", SourceRepo: "http://archive.ubuntu.com/ubuntu"}
	var patches []DiscoveredPatch

	for p, err := range parser.Parse(context.Background(), gzipString(t, raw)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		patches = append(patches, p)
	}

	if len(patches) != 2 {
		t.Fatalf("expected 2 patches, got %d", len(patches))
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := patches[i]
			if p.Name != tc.wantName {
				t.Errorf("name: got %q, want %q", p.Name, tc.wantName)
			}
			if p.Version != tc.wantVer {
				t.Errorf("version: got %q, want %q", p.Version, tc.wantVer)
			}
			if p.Checksum != tc.wantSHA {
				t.Errorf("checksum: got %q, want %q", p.Checksum, tc.wantSHA)
			}
			if p.Size != tc.wantSize {
				t.Errorf("size: got %d, want %d", p.Size, tc.wantSize)
			}
			if p.OsFamily != "debian" {
				t.Errorf("os_family: got %q, want debian", p.OsFamily)
			}
			if p.Priority != tc.wantPriority {
				t.Errorf("priority: got %q, want %q", p.Priority, tc.wantPriority)
			}
			if p.Section != tc.wantSection {
				t.Errorf("section: got %q, want %q", p.Section, tc.wantSection)
			}
			if p.Filename != tc.wantFilename {
				t.Errorf("filename: got %q, want %q", p.Filename, tc.wantFilename)
			}
			if p.Description != tc.wantDesc {
				t.Errorf("description: got %q, want %q", p.Description, tc.wantDesc)
			}
			if p.Summary != "" {
				t.Errorf("summary: expected empty for APT, got %q", p.Summary)
			}
		})
	}
}

func TestAPTParser_EmptyInput(t *testing.T) {
	parser := &APTParser{OsFamily: "debian", OsDistro: "ubuntu-22.04"}
	var count int
	for _, err := range parser.Parse(context.Background(), gzipString(t, "")) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 patches from empty input, got %d", count)
	}
}

func TestAPTParser_ContextCancellation(t *testing.T) {
	raw := "Package: curl\nVersion: 1.0\nArchitecture: amd64\n\nPackage: openssl\nVersion: 2.0\nArchitecture: amd64\n"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parser := &APTParser{OsFamily: "debian", OsDistro: "ubuntu-22.04"}
	var count int
	for _, err := range parser.Parse(ctx, gzipString(t, raw)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
		cancel()
	}
	if count != 1 {
		t.Fatalf("expected 1 patch before cancellation, got %d", count)
	}
}
