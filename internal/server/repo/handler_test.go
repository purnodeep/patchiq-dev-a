package repo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestFileServerServesFile(t *testing.T) {
	cacheDir := t.TempDir()

	// Create a test binary file.
	winDir := filepath.Join(cacheDir, "windows")
	if err := os.MkdirAll(winDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("fake-msi-installer-content")
	if err := os.WriteFile(filepath.Join(winDir, "patch.msi"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	MountFileServer(r, cacheDir)

	req := httptest.NewRequest(http.MethodGet, "/repo/files/windows/patch.msi", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != string(content) {
		t.Errorf("body = %q, want %q", w.Body.String(), string(content))
	}
}

func TestFileServerNotFound(t *testing.T) {
	cacheDir := t.TempDir()

	r := chi.NewRouter()
	MountFileServer(r, cacheDir)

	req := httptest.NewRequest(http.MethodGet, "/repo/files/linux/nonexistent.deb", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestFileServerPathTraversal(t *testing.T) {
	cacheDir := t.TempDir()

	r := chi.NewRouter()
	MountFileServer(r, cacheDir)

	req := httptest.NewRequest(http.MethodGet, "/repo/files/../../../etc/passwd", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should not serve files outside cache dir.
	if w.Code == http.StatusOK {
		t.Error("should not serve files via path traversal")
	}
}

func TestFileServerMacOS(t *testing.T) {
	cacheDir := t.TempDir()

	macDir := filepath.Join(cacheDir, "macos")
	if err := os.MkdirAll(macDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("fake-pkg-content")
	if err := os.WriteFile(filepath.Join(macDir, "update.pkg"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	MountFileServer(r, cacheDir)

	req := httptest.NewRequest(http.MethodGet, "/repo/files/macos/update.pkg", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != string(content) {
		t.Errorf("body = %q, want %q", w.Body.String(), string(content))
	}
}
