package inventory

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func packageNames(pkgs []*pb.PackageInfo) []string {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	return names
}

func TestAPTCollector_Name(t *testing.T) {
	c := &aptCollector{statusPath: "/dev/null"}
	if got := c.Name(); got != "apt" {
		t.Errorf("Name() = %q, want %q", got, "apt")
	}
}

func TestParseAPTStatus_Basic(t *testing.T) {
	f, err := os.Open("testdata/dpkg_status_basic")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	pkgs, err := parseAPTStatus(f)
	if err != nil {
		t.Fatalf("parseAPTStatus: %v", err)
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}

	// Verify first package fields.
	first := pkgs[0]
	if first.Name != "bash" {
		t.Errorf("first package Name = %q, want %q", first.Name, "bash")
	}
	if first.Version != "5.1-6ubuntu1.1" {
		t.Errorf("first package Version = %q, want %q", first.Version, "5.1-6ubuntu1.1")
	}
	if first.Architecture != "amd64" {
		t.Errorf("first package Architecture = %q, want %q", first.Architecture, "amd64")
	}
	if first.Source != "apt" {
		t.Errorf("first package Source = %q, want %q", first.Source, "apt")
	}
	if first.Status != "install ok installed" {
		t.Errorf("first package Status = %q, want %q", first.Status, "install ok installed")
	}
}

func TestParseAPTStatus_ExcludesDeinstalled(t *testing.T) {
	f, err := os.Open("testdata/dpkg_status_basic")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	pkgs, err := parseAPTStatus(f)
	if err != nil {
		t.Fatalf("parseAPTStatus: %v", err)
	}

	for _, p := range pkgs {
		if strings.Contains(p.Name, "linux-headers") {
			t.Errorf("expected linux-headers to be excluded, but found %q", p.Name)
		}
	}
}

func TestParseAPTStatus_EdgeCases(t *testing.T) {
	f, err := os.Open("testdata/dpkg_status_edge_cases")
	if err != nil {
		t.Fatalf("open testdata: %v", err)
	}
	defer f.Close()

	pkgs, err := parseAPTStatus(f)
	if err != nil {
		t.Fatalf("parseAPTStatus: %v", err)
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}

	// half-installed should be excluded because status third word is "half-installed" not "installed".
	names := packageNames(pkgs)
	for _, n := range names {
		if n == "half-installed" {
			t.Error("expected half-installed package to be excluded")
		}
	}

	// Verify the three included packages.
	want := map[string]bool{"no-version-pkg": true, "no-arch-pkg": true, "valid-after-broken": true}
	for _, n := range names {
		if !want[n] {
			t.Errorf("unexpected package %q in results", n)
		}
		delete(want, n)
	}
	for n := range want {
		t.Errorf("expected package %q not found in results", n)
	}
}

func TestParseAPTStatus_EmptyInput(t *testing.T) {
	pkgs, err := parseAPTStatus(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseAPTStatus on empty input: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestAPTCollector_Collect_Integration(t *testing.T) {
	c := &aptCollector{statusPath: "testdata/dpkg_status_basic"}

	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}
}

func TestAPTCollector_Collect_MissingFile(t *testing.T) {
	c := &aptCollector{statusPath: "testdata/nonexistent_file"}

	_, err := c.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func BenchmarkParseAPTStatus_1000Packages(b *testing.B) {
	var buf strings.Builder
	for i := range 1000 {
		fmt.Fprintf(&buf, "Package: pkg-%d\n", i)
		buf.WriteString("Status: install ok installed\n")
		buf.WriteString("Architecture: amd64\n")
		fmt.Fprintf(&buf, "Version: %d.0.0-1ubuntu1\n", i)
		buf.WriteString("Description: Test package\n\n")
	}
	data := buf.String()

	b.ResetTimer()
	for range b.N {
		pkgs, err := parseAPTStatus(strings.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		if len(pkgs) != 1000 {
			b.Fatalf("expected 1000, got %d", len(pkgs))
		}
	}
}
