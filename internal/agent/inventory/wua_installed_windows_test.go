//go:build windows

package inventory

import (
	"context"
	"errors"
	"testing"
)

func TestWUAInstalledCollector_Name(t *testing.T) {
	c := &wuaInstalledCollector{}
	if c.Name() != "wua_installed" {
		t.Errorf("Name() = %q, want %q", c.Name(), "wua_installed")
	}
}

func TestWUAInstalledCollector_Collect_Success(t *testing.T) {
	mock := &mockSearcher{
		updates: []windowsUpdate{
			{KBID: "KB5034765", Title: "Security Update", Severity: "Critical"},
			{KBID: "KB5034766", Title: "Feature Update", Severity: "Important"},
		},
	}
	c := &wuaInstalledCollector{searcher: mock}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages, want 2", len(pkgs))
	}
	for _, p := range pkgs {
		if p.Source != "wua_installed" {
			t.Errorf("source = %q, want %q", p.Source, "wua_installed")
		}
	}
}

func TestWUAInstalledCollector_Collect_Empty(t *testing.T) {
	mock := &mockSearcher{updates: nil}
	c := &wuaInstalledCollector{searcher: mock}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("got %d packages, want 0", len(pkgs))
	}
}

func TestWUAInstalledCollector_Collect_Error(t *testing.T) {
	mock := &mockSearcher{err: errors.New("COM init failed")}
	c := &wuaInstalledCollector{searcher: mock}
	_, err := c.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
