package patcher

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDownloadBinary(t *testing.T) {
	content := []byte("fake-installer-binary-content-12345")
	checksum := sha256hex(content)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repo/files/windows/patch.msi" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dl := NewDownloader(http.DefaultClient, tmpDir)

	localPath, err := dl.Download(t.Context(), srv.URL+"/repo/files/windows/patch.msi", checksum)
	if err != nil {
		t.Fatalf("Download() error: %v", err)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != string(content) {
		t.Error("downloaded content does not match")
	}
}

func TestDownloadBinaryChecksumMismatch(t *testing.T) {
	content := []byte("binary-content")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dl := NewDownloader(http.DefaultClient, tmpDir)

	_, err := dl.Download(t.Context(), srv.URL+"/file.msi", "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("Download() should fail on checksum mismatch")
	}
}

func TestDownloadBinaryHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dl := NewDownloader(http.DefaultClient, tmpDir)

	_, err := dl.Download(t.Context(), srv.URL+"/fail.msi", "abc")
	if err == nil {
		t.Fatal("Download() should fail on HTTP error")
	}
}

func TestDownloadBinaryEmptyChecksum(t *testing.T) {
	content := []byte("binary-content-no-checksum-verification")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	dl := NewDownloader(http.DefaultClient, tmpDir)

	// Empty checksum should skip verification.
	localPath, err := dl.Download(t.Context(), srv.URL+"/file.pkg", "")
	if err != nil {
		t.Fatalf("Download() error: %v", err)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	if string(data) != string(content) {
		t.Error("downloaded content does not match")
	}
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
