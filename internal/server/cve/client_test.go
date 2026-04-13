package cve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNVDClient_FetchCVEs(t *testing.T) {
	testdata, err := os.ReadFile("testdata/nvd_response.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Query().Get("lastModStartDate") == "" {
			t.Error("expected lastModStartDate query param")
		}
		if r.URL.Query().Get("lastModEndDate") == "" {
			t.Error("expected lastModEndDate query param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(testdata) //nolint:errcheck
	}))
	defer srv.Close()

	client := NewNVDClient(srv.URL, "", 5*time.Second)
	records, err := client.FetchCVEs(context.Background(), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchCVEs: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(records))
	}
	if callCount < 1 {
		t.Errorf("expected at least 1 API call, got %d", callCount)
	}
}

func TestNVDClient_FetchCVEs_WithAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("apiKey")
		if apiKey != "test-key" {
			t.Errorf("expected apiKey header 'test-key', got %q", apiKey)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"resultsPerPage":0,"startIndex":0,"totalResults":0,"format":"NVD_CVE","version":"2.0","vulnerabilities":[]}`)) //nolint:errcheck
	}))
	defer srv.Close()

	client := NewNVDClient(srv.URL, "test-key", 5*time.Second)
	_, err := client.FetchCVEs(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("FetchCVEs: %v", err)
	}
}

func TestNVDClient_FetchCVEs_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := NewNVDClient(srv.URL, "", 5*time.Second)
	_, err := client.FetchCVEs(context.Background(), time.Now().Add(-24*time.Hour))
	if err == nil {
		t.Fatal("expected error for HTTP 503")
	}
}

func TestNVDClient_FetchCVEs_Pagination(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if page == 0 {
			page++
			w.Write([]byte(`{"resultsPerPage":1,"startIndex":0,"totalResults":2,"format":"NVD_CVE","version":"2.0","vulnerabilities":[{"cve":{"id":"CVE-2024-0001","published":"2024-01-01T00:00:00.000","lastModified":"2024-01-01T00:00:00.000","descriptions":[{"lang":"en","value":"test"}],"metrics":{},"configurations":[]}}]}`)) //nolint:errcheck
		} else {
			w.Write([]byte(`{"resultsPerPage":1,"startIndex":1,"totalResults":2,"format":"NVD_CVE","version":"2.0","vulnerabilities":[{"cve":{"id":"CVE-2024-0002","published":"2024-01-02T00:00:00.000","lastModified":"2024-01-02T00:00:00.000","descriptions":[{"lang":"en","value":"test2"}],"metrics":{},"configurations":[]}}]}`)) //nolint:errcheck
		}
	}))
	defer srv.Close()

	client := NewNVDClient(srv.URL, "", 5*time.Second)
	records, err := client.FetchCVEs(context.Background(), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchCVEs: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("expected 2 records from pagination, got %d", len(records))
	}
}

func TestNVDClient_FetchKEV(t *testing.T) {
	testdata, err := os.ReadFile("testdata/kev_catalog.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(testdata) //nolint:errcheck
	}))
	defer srv.Close()

	client := NewNVDClient("", "", 5*time.Second)
	client.kevURL = srv.URL

	kevMap, err := client.FetchKEV(context.Background())
	if err != nil {
		t.Fatalf("FetchKEV: %v", err)
	}
	if len(kevMap) != 2 {
		t.Errorf("len(kevMap) = %d, want 2", len(kevMap))
	}
}
