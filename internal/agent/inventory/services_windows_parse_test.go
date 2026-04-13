package inventory

import (
	"os"
	"testing"
)

func TestParseWinServices(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/services.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	services := parseWinServices(string(data))

	if len(services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(services))
	}

	wu := services[0]
	if wu.Name != "wuauserv" {
		t.Errorf("services[0].Name = %q", wu.Name)
	}
	if wu.Description != "Windows Update" {
		t.Errorf("services[0].Description = %q", wu.Description)
	}
	if wu.ActiveState != "active" {
		t.Errorf("services[0].ActiveState = %q, want active", wu.ActiveState)
	}
	if wu.SubState != "running" {
		t.Errorf("services[0].SubState = %q, want running", wu.SubState)
	}
	if wu.Enabled {
		t.Error("services[0].Enabled should be false (Manual start)")
	}
	if wu.Category != "Package Management" {
		t.Errorf("services[0].Category = %q, want Package Management", wu.Category)
	}

	wd := services[1]
	if !wd.Enabled {
		t.Error("services[1].Enabled should be true (Automatic)")
	}
	if wd.Category != "Security" {
		t.Errorf("services[1].Category = %q, want Security", wd.Category)
	}

	sp := services[2]
	if sp.ActiveState != "inactive" {
		t.Errorf("services[2].ActiveState = %q, want inactive", sp.ActiveState)
	}
	if sp.SubState != "dead" {
		t.Errorf("services[2].SubState = %q, want dead", sp.SubState)
	}

	ms := services[3]
	if ms.Category != "Database" {
		t.Errorf("services[3].Category = %q, want Database", ms.Category)
	}
}

func TestParseWinServicesEmpty(t *testing.T) {
	services := parseWinServices("")
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}
