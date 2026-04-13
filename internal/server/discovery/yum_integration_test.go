package discovery

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/shared/config"
)

func gzipBytes(t *testing.T, data string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write([]byte(data)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestYUMIntegration_HTTPTestServer(t *testing.T) {
	yumXML := `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://linux.duke.edu/metadata/common"
          xmlns:rpm="http://linux.duke.edu/metadata/rpm" packages="1">
  <package type="rpm">
    <name>bash</name>
    <arch>x86_64</arch>
    <version epoch="0" ver="5.1.8" rel="9.el9"/>
    <checksum type="sha256" pkgid="YES">deadbeef</checksum>
    <summary>The GNU Bourne Again shell</summary>
    <description>The GNU Bourne Again shell (Bash) is a shell and command language interpreter.</description>
    <size package="1870000" installed="3500000" archive="4000000"/>
    <location href="Packages/bash-5.1.8-9.el9.x86_64.rpm"/>
  </package>
</metadata>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.Write(gzipBytes(t, yumXML)) //nolint:errcheck
	}))
	defer srv.Close()

	fetcher := NewFetcher(5*time.Second, 3)
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	svc := NewService(upserter, emitter, fetcher)

	repo := config.RepositoryConfig{
		Name:     "test-yum-repo",
		Type:     "yum",
		URL:      srv.URL + "/repodata/primary.xml.gz",
		OsFamily: "rhel",
		OsDistro: "rhel-9",
		Enabled:  true,
	}

	count, err := svc.DiscoverRepo(context.Background(), "tenant-1", repo)
	if err != nil {
		t.Fatalf("DiscoverRepo: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 patch, got %d", count)
	}
	if upserter.current == nil || len(upserter.current.patches) != 1 {
		t.Fatal("expected 1 upserted patch")
	}
	p := upserter.current.patches[0]
	if p.Name != "bash" {
		t.Errorf("name = %q, want bash", p.Name)
	}
	if p.Summary != "The GNU Bourne Again shell" {
		t.Errorf("summary = %q, want 'The GNU Bourne Again shell'", p.Summary)
	}
	if p.Version != "5.1.8-9.el9" {
		t.Errorf("version = %q, want 5.1.8-9.el9", p.Version)
	}
}
