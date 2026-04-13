package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcherAPTFetchBinary(t *testing.T) {
	binaryContent := []byte("fake-deb-binary-content-1234567890")
	vendorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(binaryContent)))
		_, _ = w.Write(binaryContent)
	}))
	defer vendorServer.Close()

	store := newMockObjectStore()
	fetcher := NewAPTFetcher(store, "patches")

	ref, checksum, size, err := fetcher.FetchBinary(
		context.Background(),
		vendorServer.URL+"/curl_7.88.1_amd64.deb",
		"ubuntu",
		"22.04",
		"curl_7.88.1_amd64.deb",
	)
	if err != nil {
		t.Fatalf("FetchBinary() error: %v", err)
	}

	wantKey := "apt/22.04/curl_7.88.1_amd64.deb"
	if ref != wantKey {
		t.Errorf("binary_ref = %q, want %q", ref, wantKey)
	}

	h := sha256.Sum256(binaryContent)
	wantChecksum := hex.EncodeToString(h[:])
	if checksum != wantChecksum {
		t.Errorf("checksum = %q, want %q", checksum, wantChecksum)
	}

	wantSize := int64(len(binaryContent))
	if size != wantSize {
		t.Errorf("size = %d, want %d", size, wantSize)
	}

	storedKey := "patches/" + wantKey
	if _, ok := store.uploaded[storedKey]; !ok {
		t.Errorf("binary not found in object store at key %q", storedKey)
	}
}

func TestFetcherAPTHTTPError(t *testing.T) {
	vendorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer vendorServer.Close()

	store := newMockObjectStore()
	fetcher := NewAPTFetcher(store, "patches")

	_, _, _, err := fetcher.FetchBinary(context.Background(), vendorServer.URL+"/missing.deb", "ubuntu", "22.04", "missing.deb")
	if err == nil {
		t.Fatal("FetchBinary() should return error for HTTP 404")
	}
}

func TestFetcherAPTBadURL(t *testing.T) {
	store := newMockObjectStore()
	fetcher := NewAPTFetcher(store, "patches")

	_, _, _, err := fetcher.FetchBinary(context.Background(), "http://127.0.0.1:0/invalid", "ubuntu", "22.04", "bad.deb")
	if err == nil {
		t.Fatal("FetchBinary() should return error for unreachable URL")
	}
}
