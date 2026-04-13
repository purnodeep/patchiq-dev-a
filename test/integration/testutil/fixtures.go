//go:build integration

package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fixtureDir returns the absolute path to the testdata directory,
// resolved relative to this source file so it works regardless of
// the caller's working directory.
func fixtureDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: runtime.Caller(0) failed in fixtureDir")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "testdata")
}

// APTPackagesFixture returns a minimal APT Packages file entry for
// the cowsay package at the given version.
func APTPackagesFixture(version string) string {
	return fmt.Sprintf(`Package: cowsay
Version: %s
Architecture: amd64
Maintainer: Test <test@example.com>
Installed-Size: 100
Filename: pool/main/c/cowsay/cowsay_%s_amd64.deb
Size: 25000
MD5sum: d41d8cd98f00b204e9800998ecf8427e
SHA256: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
Description: A configurable talking cow
`, version, version)
}

// ServeAPTPackages starts an httptest.Server that serves an APT Packages
// file for cowsay at the given version. The server is automatically
// closed when the test completes.
func ServeAPTPackages(t *testing.T, version string) *httptest.Server {
	t.Helper()
	content := APTPackagesFixture(version)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, content)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// LoadNVDFixture reads and returns the raw bytes of the NVD CVE
// fixture file (testdata/nvd_cve.json).
func LoadNVDFixture(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join(fixtureDir(), "nvd_cve.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load NVD fixture %s: %v", path, err)
	}
	return data
}

// WriteNVDFixtureDir copies the NVD fixture into a temporary directory
// and returns the directory path. This is suitable for passing to
// BulkImporter.ImportDirectory().
func WriteNVDFixtureDir(t *testing.T) string {
	t.Helper()
	data := LoadNVDFixture(t)
	dir := t.TempDir()
	dest := filepath.Join(dir, "nvd_cve.json")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		t.Fatalf("write NVD fixture to temp dir: %v", err)
	}
	return dir
}
