package inventory

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHotFixOutput_Basic(t *testing.T) {
	data, err := os.ReadFile("testdata/hotfix_basic.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs, err := parseHotFixOutput(data)
	if err != nil {
		t.Fatalf("parseHotFixOutput: %v", err)
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}

	tests := []struct {
		idx     int
		name    string
		version string
		source  string
		status  string
	}{
		{0, "KB5034441", "2024-01-15T00:00:00Z", "hotfix", "Security Update"},
		{1, "KB5033372", "2024-01-10T00:00:00Z", "hotfix", "Update"},
		{2, "KB5032190", "2023-12-20T00:00:00Z", "hotfix", "Security Update"},
	}

	for _, tt := range tests {
		p := pkgs[tt.idx]
		if p.Name != tt.name {
			t.Errorf("pkgs[%d].Name = %q, want %q", tt.idx, p.Name, tt.name)
		}
		if p.Version != tt.version {
			t.Errorf("pkgs[%d].Version = %q, want %q", tt.idx, p.Version, tt.version)
		}
		if p.Source != tt.source {
			t.Errorf("pkgs[%d].Source = %q, want %q", tt.idx, p.Source, tt.source)
		}
		if p.Status != tt.status {
			t.Errorf("pkgs[%d].Status = %q, want %q", tt.idx, p.Status, tt.status)
		}
	}
}

func TestParseHotFixOutput_SingleObject(t *testing.T) {
	data, err := os.ReadFile("testdata/hotfix_single.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs, err := parseHotFixOutput(data)
	if err != nil {
		t.Fatalf("parseHotFixOutput: %v", err)
	}

	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}

	if pkgs[0].Name != "KB5034441" {
		t.Errorf("Name = %q, want %q", pkgs[0].Name, "KB5034441")
	}
	if pkgs[0].Source != "hotfix" {
		t.Errorf("Source = %q, want %q", pkgs[0].Source, "hotfix")
	}
}

func TestParseHotFixOutput_Empty(t *testing.T) {
	data, err := os.ReadFile("testdata/hotfix_empty.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs, err := parseHotFixOutput(data)
	if err != nil {
		t.Fatalf("parseHotFixOutput: %v", err)
	}

	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestParseHotFixOutput_InvalidJSON(t *testing.T) {
	_, err := parseHotFixOutput([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseHotFixOutput_SkipsEmptyHotFixID(t *testing.T) {
	data := []byte(`[
		{"HotFixID": "KB5034441", "Description": "Update", "InstalledOn": "2024-01-15T00:00:00Z"},
		{"HotFixID": "", "Description": "Update", "InstalledOn": "2024-01-10T00:00:00Z"},
		{"HotFixID": "KB5032190", "Description": "Update", "InstalledOn": "2023-12-20T00:00:00Z"}
	]`)

	pkgs, err := parseHotFixOutput(data)
	if err != nil {
		t.Fatalf("parseHotFixOutput: %v", err)
	}

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages (skipping empty ID), got %d", len(pkgs))
	}

	if pkgs[0].Name != "KB5034441" {
		t.Errorf("pkgs[0].Name = %q, want %q", pkgs[0].Name, "KB5034441")
	}
	if pkgs[1].Name != "KB5032190" {
		t.Errorf("pkgs[1].Name = %q, want %q", pkgs[1].Name, "KB5032190")
	}
}

func TestParseHotFixOutput_EmptyInput(t *testing.T) {
	pkgs, err := parseHotFixOutput([]byte{})
	if err != nil {
		t.Fatalf("parseHotFixOutput: %v", err)
	}
	if pkgs != nil {
		t.Errorf("expected nil for empty input, got %v", pkgs)
	}
}

func TestParseHotFixOutput_NewFields(t *testing.T) {
	input := `[{"HotFixID":"KB5034441","Description":"Security Update","InstalledOn":"1/15/2024"}]`
	pkgs, err := parseHotFixOutput([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	p := pkgs[0]
	if p.KbArticle != "KB5034441" {
		t.Errorf("KbArticle = %q, want KB5034441", p.KbArticle)
	}
	if p.InstallDate != "1/15/2024" {
		t.Errorf("InstallDate = %q, want 1/15/2024", p.InstallDate)
	}
	if p.Source != "hotfix" {
		t.Errorf("Source = %q, want hotfix", p.Source)
	}
}

func TestParseHotFixOutput_ObjectInstalledOn(t *testing.T) {
	data := []byte(`[
		{"HotFixID": "KB5034441", "Description": "Update", "InstalledOn": {"value": "\/Date(1705276800000)\/", "DisplayHint": 2, "DateTime": "Monday, January 15, 2024 12:00:00 AM"}},
		{"HotFixID": "KB5033372", "Description": "Security Update", "InstalledOn": "2024-01-10T00:00:00Z"}
	]`)

	pkgs, err := parseHotFixOutput(data)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
	assert.Equal(t, "KB5034441", pkgs[0].Name)
	assert.Equal(t, "Monday, January 15, 2024 12:00:00 AM", pkgs[0].Version)
	assert.Equal(t, "KB5033372", pkgs[1].Name)
	assert.Equal(t, "2024-01-10T00:00:00Z", pkgs[1].Version)
}
