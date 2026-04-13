//go:build windows

package patcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstallerType_InnoSetup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "setup.exe")
	// Write a fake binary with an Inno Setup marker.
	if err := os.WriteFile(path, []byte("MZxxxxxxInno Setupxxxxxxx"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := detectInstallerType(path)
	if result != "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART" {
		t.Errorf("got %q, want Inno Setup args", result)
	}
}

func TestDetectInstallerType_NSIS(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "installer.exe")
	if err := os.WriteFile(path, []byte("MZxxxxxxNullsoftxxxxxxx"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := detectInstallerType(path)
	if result != "/S" {
		t.Errorf("got %q, want /S", result)
	}
}

func TestDetectInstallerType_Unknown(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "random.exe")
	if err := os.WriteFile(path, []byte("MZxxxxxxxxxxxxxxxxx"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := detectInstallerType(path)
	if result != "" {
		t.Errorf("got %q, want empty for unknown installer", result)
	}
}

func TestDetectInstallerType_FileNotFound(t *testing.T) {
	result := detectInstallerType("/nonexistent/path.exe")
	if result != "" {
		t.Errorf("got %q, want empty for missing file", result)
	}
}
