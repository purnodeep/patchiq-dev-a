package discovery

import (
	"context"
	"testing"
)

func TestYUMParser(t *testing.T) {
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://linux.duke.edu/metadata/common"
          xmlns:rpm="http://linux.duke.edu/metadata/rpm" packages="2">
  <package type="rpm">
    <name>openssl</name>
    <arch>x86_64</arch>
    <version epoch="1" ver="3.0.7" rel="27.el9"/>
    <checksum type="sha256" pkgid="YES">abc123</checksum>
    <summary>Utilities from the general purpose cryptography library</summary>
    <description>The OpenSSL toolkit.</description>
    <size package="1234567" installed="2345678" archive="3456789"/>
    <location href="Packages/openssl-3.0.7-27.el9.x86_64.rpm"/>
  </package>
  <package type="rpm">
    <name>curl</name>
    <arch>x86_64</arch>
    <version epoch="0" ver="7.76.1" rel="29.el9"/>
    <checksum type="sha256" pkgid="YES">def456</checksum>
    <summary>A utility for getting files from remote servers</summary>
    <description>curl transfers data with URL syntax.</description>
    <size package="301234" installed="401234" archive="501234"/>
    <location href="Packages/curl-7.76.1-29.el9.x86_64.rpm"/>
  </package>
</metadata>`

	tests := []struct {
		name        string
		wantName    string
		wantVersion string
		wantArch    string
		wantSHA     string
		wantSize    int64
		wantSummary string
	}{
		{"openssl", "openssl", "1:3.0.7-27.el9", "x86_64", "abc123", 1234567, "Utilities from the general purpose cryptography library"},
		{"curl", "curl", "7.76.1-29.el9", "x86_64", "def456", 301234, "A utility for getting files from remote servers"},
	}

	parser := &YUMParser{OsFamily: "rhel", OsDistro: "rhel-9", SourceRepo: "https://mirror.example.com/rhel9"}
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
			if p.Version != tc.wantVersion {
				t.Errorf("version: got %q, want %q", p.Version, tc.wantVersion)
			}
			if p.Arch != tc.wantArch {
				t.Errorf("arch: got %q, want %q", p.Arch, tc.wantArch)
			}
			if p.Checksum != tc.wantSHA {
				t.Errorf("checksum: got %q, want %q", p.Checksum, tc.wantSHA)
			}
			if p.Size != tc.wantSize {
				t.Errorf("size: got %d, want %d", p.Size, tc.wantSize)
			}
			if p.OsFamily != "rhel" {
				t.Errorf("os_family: got %q, want rhel", p.OsFamily)
			}
			if p.Summary != tc.wantSummary {
				t.Errorf("summary: got %q, want %q", p.Summary, tc.wantSummary)
			}
		})
	}
}

func TestYUMParser_EmptyInput(t *testing.T) {
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://linux.duke.edu/metadata/common" packages="0">
</metadata>`

	parser := &YUMParser{OsFamily: "rhel", OsDistro: "rhel-9"}
	var count int
	for _, err := range parser.Parse(context.Background(), gzipString(t, raw)) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 patches, got %d", count)
	}
}
