package v1

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

// buildTestTarGz creates a .tar.gz in memory with the given file entries.
func buildTestTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0755,
			Size: int64(len(content)),
		}); err != nil {
			t.Fatalf("write tar header for %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content for %s: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}

// extractTarGz extracts a .tar.gz and returns a map of filename -> content.
func extractTarGz(t *testing.T, data []byte) map[string]string {
	t.Helper()
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	result := make(map[string]string)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}
		content, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("read tar entry content: %v", err)
		}
		result[hdr.Name] = string(content)
	}
	return result
}

// setupHandler creates an AgentBinariesHandler with a chi router wired for testing.
func setupHandler(t *testing.T, dir string) *chi.Mux {
	t.Helper()
	h := NewAgentBinariesHandler(dir, "grpc.example.com:50051")
	r := chi.NewRouter()
	r.Get("/api/v1/agent-binaries/{filename}/download", h.Download)
	return r
}

func TestDownload_LinuxTarball(t *testing.T) {
	dir := t.TempDir()

	origFiles := map[string]string{
		"patchiq-agent": "FAKE_BINARY_CONTENT",
		"README.txt":    "This is a readme.",
	}
	tarData := buildTestTarGz(t, origFiles)
	filename := "patchiq-agent-linux-amd64.tar.gz"
	if err := os.WriteFile(filepath.Join(dir, filename), tarData, 0644); err != nil {
		t.Fatalf("write test tarball: %v", err)
	}

	r := setupHandler(t, dir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-binaries/"+filename+"/download", nil)
	req.Host = "patch.example.com"
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/gzip" {
		t.Errorf("expected Content-Type application/gzip, got %q", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != "attachment; filename="+filename {
		t.Errorf("unexpected Content-Disposition: %q", cd)
	}

	extracted := extractTarGz(t, rec.Body.Bytes())

	// Check original files preserved
	for name, content := range origFiles {
		got, ok := extracted[name]
		if !ok {
			t.Errorf("missing original entry %q in repacked tarball", name)
			continue
		}
		if got != content {
			t.Errorf("entry %q: expected %q, got %q", name, content, got)
		}
	}

	// Check server.txt injected with the configured gRPC address
	serverTxt, ok := extracted["server.txt"]
	if !ok {
		t.Fatal("server.txt not found in repacked tarball")
	}
	if serverTxt != "grpc.example.com:50051\n" {
		t.Errorf("server.txt content: expected %q, got %q", "grpc.example.com:50051\n", serverTxt)
	}
}

func TestDownload_NonLinuxFile(t *testing.T) {
	dir := t.TempDir()

	filename := "patchiq-agent-darwin-amd64.tar.gz"
	content := []byte("NOT_A_REAL_TARBALL_JUST_BYTES")
	if err := os.WriteFile(filepath.Join(dir, filename), content, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	r := setupHandler(t, dir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-binaries/"+filename+"/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// http.ServeFile may return the bytes; verify content matches
	got := rec.Body.Bytes()
	if !bytes.Equal(got, content) {
		t.Errorf("non-linux file not served verbatim: got %d bytes, expected %d", len(got), len(content))
	}
}

func TestDownload_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := setupHandler(t, dir)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-binaries/nonexistent.tar.gz/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDownload_GRPCAddrInServerTxt(t *testing.T) {
	dir := t.TempDir()
	filename := "patchiq-agent-linux-amd64.tar.gz"
	tarData := buildTestTarGz(t, map[string]string{"agent": "bin"})
	if err := os.WriteFile(filepath.Join(dir, filename), tarData, 0644); err != nil {
		t.Fatalf("write test tarball: %v", err)
	}

	customAddr := "myserver.internal:50151"
	h := NewAgentBinariesHandler(dir, customAddr)
	r := chi.NewRouter()
	r.Get("/api/v1/agent-binaries/{filename}/download", h.Download)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent-binaries/"+filename+"/download", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	extracted := extractTarGz(t, rec.Body.Bytes())
	serverTxt, ok := extracted["server.txt"]
	if !ok {
		t.Fatal("server.txt not found")
	}
	want := customAddr + "\n"
	if serverTxt != want {
		t.Errorf("server.txt: expected %q, got %q", want, serverTxt)
	}
}
