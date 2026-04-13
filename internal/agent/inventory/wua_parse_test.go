package inventory

import (
	"testing"
)

func TestMapWindowsUpdates_Basic(t *testing.T) {
	updates := []windowsUpdate{
		{KBID: "KB5034441", Title: "2024-01 Cumulative Update", Severity: "Critical", Categories: []string{"Security Updates"}},
		{KBID: "KB5033372", Title: "2023-12 Cumulative Update", Severity: "Important", Categories: []string{"Security Updates"}},
	}

	pkgs := mapWindowsUpdates(updates)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	first := pkgs[0]
	if first.Name != "2024-01 Cumulative Update" {
		t.Errorf("Name = %q, want %q", first.Name, "2024-01 Cumulative Update")
	}
	if first.Version != "KB5034441" {
		t.Errorf("Version = %q, want %q", first.Version, "KB5034441")
	}
	if first.Source != "wua" {
		t.Errorf("Source = %q, want %q", first.Source, "wua")
	}
	if first.KbArticle != "KB5034441" {
		t.Errorf("KbArticle = %q, want %q", first.KbArticle, "KB5034441")
	}
	if first.Severity != "Critical" {
		t.Errorf("Severity = %q, want %q", first.Severity, "Critical")
	}
}

func TestMapWindowsUpdates_Empty(t *testing.T) {
	pkgs := mapWindowsUpdates(nil)
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestMapWindowsUpdates_SkipsEmptyKBID(t *testing.T) {
	updates := []windowsUpdate{
		{KBID: "", Title: "Unknown Update", Severity: "Low"},
		{KBID: "KB5034441", Title: "Good Update", Severity: "Important"},
	}

	pkgs := mapWindowsUpdates(updates)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Name != "Good Update" {
		t.Errorf("Name = %q, want %q", pkgs[0].Name, "Good Update")
	}
}

func TestMapWindowsUpdates_NewFields(t *testing.T) {
	updates := []windowsUpdate{
		{
			KBID:       "KB5034441",
			Title:      "2024-01 Cumulative Update for Windows 11",
			Severity:   "Critical",
			Categories: []string{"Security Updates", "Windows 11"},
		},
	}
	pkgs := mapWindowsUpdates(updates)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	p := pkgs[0]
	if p.Name != "2024-01 Cumulative Update for Windows 11" {
		t.Errorf("Name = %q, want title", p.Name)
	}
	if p.KbArticle != "KB5034441" {
		t.Errorf("KbArticle = %q, want KB5034441", p.KbArticle)
	}
	if p.Severity != "Critical" {
		t.Errorf("Severity = %q, want Critical", p.Severity)
	}
	if p.Category != "Security Updates, Windows 11" {
		t.Errorf("Category = %q, want categories", p.Category)
	}
}
