package inventory

import (
	"os"
	"testing"
)

func TestParseSoftwareUpdate_Basic(t *testing.T) {
	data, err := os.ReadFile("testdata/softwareupdate_basic")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	updates := parseSoftwareUpdate(data)

	if len(updates) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(updates))
	}

	first := updates[0]
	if first.Name != "macOS Ventura 13.6.1" {
		t.Errorf("first Name = %q, want %q", first.Name, "macOS Ventura 13.6.1")
	}
	if first.Version != "13.6.1" {
		t.Errorf("first Version = %q, want %q", first.Version, "13.6.1")
	}
	if first.Source != "softwareupdate" {
		t.Errorf("first Source = %q, want %q", first.Source, "softwareupdate")
	}
	if first.Status != "available" {
		t.Errorf("first Status = %q, want %q", first.Status, "available")
	}

	second := updates[1]
	if second.Name != "Safari" {
		t.Errorf("second Name = %q, want %q", second.Name, "Safari")
	}
	if second.Version != "17.1" {
		t.Errorf("second Version = %q, want %q", second.Version, "17.1")
	}

	third := updates[2]
	if third.Name != "Command Line Tools for Xcode" {
		t.Errorf("third Name = %q, want %q", third.Name, "Command Line Tools for Xcode")
	}
	if third.Version != "15.1" {
		t.Errorf("third Version = %q, want %q", third.Version, "15.1")
	}
}

func TestParseSoftwareUpdate_NoUpdates(t *testing.T) {
	data, err := os.ReadFile("testdata/softwareupdate_no_updates")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	updates := parseSoftwareUpdate(data)

	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParseSoftwareUpdate_EmptyInput(t *testing.T) {
	updates := parseSoftwareUpdate([]byte{})
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
}

func TestParseSoftwareUpdate_RestartRequired(t *testing.T) {
	data, err := os.ReadFile("testdata/softwareupdate_restart")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	updates := parseSoftwareUpdate(data)

	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	if updates[0].Name != "macOS Sonoma 14.2.1" {
		t.Errorf("Name = %q, want %q", updates[0].Name, "macOS Sonoma 14.2.1")
	}
}

func TestMacOSCollector_Name(t *testing.T) {
	c := &macosCollector{runner: fakeRunner{}}
	if got := c.Name(); got != "softwareupdate" {
		t.Errorf("Name() = %q, want %q", got, "softwareupdate")
	}
}
