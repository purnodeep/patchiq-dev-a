package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheStoreAndVerify(t *testing.T) {
	binaryContent := []byte("fake-patch-binary-data-for-testing")
	checksum := sha256sum(binaryContent)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(binaryContent)))
		_, _ = w.Write(binaryContent)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	cache := NewBinaryCache(cacheDir, http.DefaultClient)

	localPath, err := cache.Download(t.Context(), srv.URL+"/curl.deb", "linux", "curl.deb", checksum)
	if err != nil {
		t.Fatalf("Download() error: %v", err)
	}

	// Verify file exists at expected path.
	wantPath := filepath.Join(cacheDir, "linux", "curl.deb")
	if localPath != wantPath {
		t.Errorf("localPath = %q, want %q", localPath, wantPath)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != string(binaryContent) {
		t.Error("downloaded file content does not match")
	}
}

func TestCacheChecksumMismatch(t *testing.T) {
	binaryContent := []byte("fake-patch-binary-data")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(binaryContent)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	cache := NewBinaryCache(cacheDir, http.DefaultClient)

	_, err := cache.Download(t.Context(), srv.URL+"/curl.deb", "linux", "curl.deb", "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("Download() should fail on checksum mismatch")
	}
}

func TestCacheHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	cache := NewBinaryCache(cacheDir, http.DefaultClient)

	_, err := cache.Download(t.Context(), srv.URL+"/fail.deb", "linux", "fail.deb", "abc")
	if err == nil {
		t.Fatal("Download() should fail on HTTP error")
	}
}

func TestCacheAlreadyCached(t *testing.T) {
	binaryContent := []byte("already-cached-binary")
	checksum := sha256sum(binaryContent)

	cacheDir := t.TempDir()
	osDir := filepath.Join(cacheDir, "windows")
	if err := os.MkdirAll(osDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(osDir, "patch.msi"), binaryContent, 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewBinaryCache(cacheDir, http.DefaultClient)

	// Should return immediately without downloading.
	localPath, err := cache.Download(t.Context(), "http://should-not-be-called/patch.msi", "windows", "patch.msi", checksum)
	if err != nil {
		t.Fatalf("Download() error: %v", err)
	}
	if localPath != filepath.Join(osDir, "patch.msi") {
		t.Errorf("localPath = %q, want %q", localPath, filepath.Join(osDir, "patch.msi"))
	}
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
