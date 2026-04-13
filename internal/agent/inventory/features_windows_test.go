//go:build windows

package inventory

import (
	"testing"
)

func TestWindowsFeaturesCollector_Name(t *testing.T) {
	c := &windowsFeaturesCollector{}
	if c.Name() != "windows_features" {
		t.Errorf("Name() = %q, want %q", c.Name(), "windows_features")
	}
}

func TestParseWindowsOptionalFeatures_Array(t *testing.T) {
	data := []byte(`[{"FeatureName":"Containers","State":0},{"FeatureName":"Microsoft-Hyper-V","State":1},{"FeatureName":"TelnetClient","State":0}]`)
	pkgs, err := parseWindowsOptionalFeatures(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("got %d packages, want 3", len(pkgs))
	}
	// Check Hyper-V is enabled.
	found := false
	for _, p := range pkgs {
		if p.Name == "Microsoft-Hyper-V" {
			found = true
			if p.Status != "Enabled" {
				t.Errorf("Hyper-V status = %q, want Enabled", p.Status)
			}
			if p.Source != "windows_feature" {
				t.Errorf("source = %q, want windows_feature", p.Source)
			}
		}
	}
	if !found {
		t.Error("Microsoft-Hyper-V not found in results")
	}
}

func TestParseWindowsOptionalFeatures_SingleObject(t *testing.T) {
	data := []byte(`{"FeatureName":"TelnetClient","State":0}`)
	pkgs, err := parseWindowsOptionalFeatures(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	if pkgs[0].Name != "TelnetClient" {
		t.Errorf("name = %q, want TelnetClient", pkgs[0].Name)
	}
}

func TestParseWindowsOptionalFeatures_Empty(t *testing.T) {
	data := []byte(`[]`)
	pkgs, err := parseWindowsOptionalFeatures(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("got %d packages, want 0", len(pkgs))
	}
}

func TestParseWindowsOptionalFeatures_InvalidJSON(t *testing.T) {
	data := []byte(`not json`)
	_, err := parseWindowsOptionalFeatures(data)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseWindowsServerFeatures(t *testing.T) {
	data := []byte(`[{"Name":"Web-Server","InstallState":"Installed"},{"Name":"DNS","InstallState":"Available"}]`)
	pkgs, err := parseWindowsServerFeatures(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	if pkgs[0].Status != "Installed" {
		t.Errorf("first feature status = %q, want Installed", pkgs[0].Status)
	}
}
