package inventory

import (
	"os"
	"testing"
)

func TestParseBrewList_Basic(t *testing.T) {
	data, err := os.ReadFile("testdata/brew_list_basic")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs := parseBrewList(data)

	if len(pkgs) != 6 {
		t.Fatalf("expected 6 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}

	first := pkgs[0]
	if first.Name != "bash" {
		t.Errorf("first Name = %q, want %q", first.Name, "bash")
	}
	if first.Version != "5.2.21" {
		t.Errorf("first Version = %q, want %q", first.Version, "5.2.21")
	}
	if first.Source != "homebrew" {
		t.Errorf("first Source = %q, want %q", first.Source, "homebrew")
	}
	if first.Status != "installed" {
		t.Errorf("first Status = %q, want %q", first.Status, "installed")
	}

	openssl := pkgs[4]
	if openssl.Name != "openssl@3" {
		t.Errorf("openssl Name = %q, want %q", openssl.Name, "openssl@3")
	}
}

func TestParseBrewList_Empty(t *testing.T) {
	pkgs := parseBrewList([]byte{})
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseBrewOutdated_Basic(t *testing.T) {
	data, err := os.ReadFile("testdata/brew_outdated_basic")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs := parseBrewOutdated(data)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}

	first := pkgs[0]
	if first.Name != "git" {
		t.Errorf("first Name = %q, want %q", first.Name, "git")
	}
	if first.Version != "2.43.0" {
		t.Errorf("first Version = %q, want %q", first.Version, "2.43.0")
	}
	if first.Source != "homebrew" {
		t.Errorf("first Source = %q, want %q", first.Source, "homebrew")
	}
	if first.Status != "outdated" {
		t.Errorf("first Status = %q, want %q", first.Status, "outdated")
	}
}

func TestParseBrewOutdated_Empty(t *testing.T) {
	pkgs := parseBrewOutdated([]byte{})
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestHomebrewCollector_Name(t *testing.T) {
	c := &homebrewCollector{runner: fakeRunner{}}
	if got := c.Name(); got != "homebrew" {
		t.Errorf("Name() = %q, want %q", got, "homebrew")
	}
}
