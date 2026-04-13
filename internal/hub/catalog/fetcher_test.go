package catalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcherDownloadAndUpload(t *testing.T) {
	binaryContent := []byte("fake-patch-binary-1234567890")
	vendorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(binaryContent)))
		_, _ = w.Write(binaryContent)
	}))
	defer vendorServer.Close()

	objStore := newMockObjectStore()
	fetcher := NewBinaryFetcher(objStore, "patches", http.DefaultClient)

	ref, checksum, _, err := fetcher.FetchAndStore(context.Background(), vendorServer.URL+"/curl.deb", "ubuntu", "22.04", "curl.deb")
	if err != nil {
		t.Fatalf("FetchAndStore() error: %v", err)
	}
	if ref == "" {
		t.Error("binary_ref should not be empty")
	}
	if checksum == "" {
		t.Error("checksum should not be empty")
	}
	wantKey := "patches/ubuntu/22.04/curl.deb"
	if ref != wantKey {
		t.Errorf("binary_ref = %q, want %q", ref, wantKey)
	}
}

func TestFetcherDownloadHTTPError(t *testing.T) {
	vendorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer vendorServer.Close()

	objStore := newMockObjectStore()
	fetcher := NewBinaryFetcher(objStore, "patches", http.DefaultClient)

	_, _, _, err := fetcher.FetchAndStore(context.Background(), vendorServer.URL+"/missing.deb", "ubuntu", "22.04", "missing.deb")
	if err == nil {
		t.Fatal("FetchAndStore() should return error for HTTP 404")
	}
}

func TestFetcherDownloadBadURL(t *testing.T) {
	objStore := newMockObjectStore()
	fetcher := NewBinaryFetcher(objStore, "patches", http.DefaultClient)

	_, _, _, err := fetcher.FetchAndStore(context.Background(), "http://127.0.0.1:0/invalid", "ubuntu", "22.04", "bad.deb")
	if err == nil {
		t.Fatal("FetchAndStore() should return error for unreachable URL")
	}
}
