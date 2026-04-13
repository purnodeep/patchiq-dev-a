package v1

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// AgentBinariesHandler serves pre-built agent binaries for download.
// For Linux tarballs, it repacks the archive to inject a server.txt file
// so the agent installer can auto-discover the server gRPC address.
type AgentBinariesHandler struct {
	dir      string // directory containing agent binary files
	grpcAddr string // gRPC address for agent enrollment (e.g. "myserver.com:50151")
}

// NewAgentBinariesHandler creates a handler that serves agent binaries from dir.
// grpcAddr is the server's gRPC address that agents should connect to for enrollment.
func NewAgentBinariesHandler(dir, grpcAddr string) *AgentBinariesHandler {
	return &AgentBinariesHandler{dir: dir, grpcAddr: grpcAddr}
}

// List returns the available agent binary files.
func (h *AgentBinariesHandler) List(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		slog.ErrorContext(r.Context(), "list agent binaries: read dir", "error", err, "dir", h.dir)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to list agent binaries")
		return
	}

	type binaryInfo struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}

	var files []binaryInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, binaryInfo{Name: e.Name(), Size: info.Size()})
	}
	if files == nil {
		files = []binaryInfo{}
	}

	WriteJSON(w, http.StatusOK, map[string]any{"data": files})
}

// Download serves an agent binary file. For Linux tarballs (patchiq-agent-linux-*.tar.gz),
// it repacks the archive with an injected server.txt containing the server URL derived
// from the request. Non-Linux files are served as-is.
func (h *AgentBinariesHandler) Download(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "filename is required")
		return
	}

	// Prevent path traversal
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || filename == ".." {
		WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid filename")
		return
	}

	srcPath := filepath.Join(h.dir, filename)
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("agent binary %q not found", filename))
		return
	} else if err != nil {
		slog.ErrorContext(r.Context(), "download agent binary: stat file", "error", err, "path", srcPath)
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "failed to access agent binary")
		return
	}

	// Non-Linux files: serve as-is
	if !isLinuxTarball(filename) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		http.ServeFile(w, r, srcPath)
		return
	}

	// Linux tarball: repack with server.txt containing the gRPC address
	serverURL := h.grpcAddr

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	if err := repackTarballWithServerTxt(w, srcPath, serverURL); err != nil {
		// Headers already sent; log the error
		slog.ErrorContext(r.Context(), "download agent binary: repack tarball", "error", err, "path", srcPath)
	}
}

// isLinuxTarball returns true if the filename matches patchiq-agent-linux-*.tar.gz.
func isLinuxTarball(filename string) bool {
	return strings.HasPrefix(filename, "patchiq-agent-linux-") && strings.HasSuffix(filename, ".tar.gz")
}

// repackTarballWithServerTxt reads a .tar.gz from srcPath, streams all entries to w,
// then appends a server.txt entry with the given serverURL.
func repackTarballWithServerTxt(w io.Writer, srcPath, serverURL string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source tarball: %w", err)
	}
	defer srcFile.Close()

	srcGz, err := gzip.NewReader(srcFile)
	if err != nil {
		return fmt.Errorf("open gzip reader: %w", err)
	}
	defer srcGz.Close()

	srcTar := tar.NewReader(srcGz)

	dstGz := gzip.NewWriter(w)
	defer dstGz.Close()

	dstTar := tar.NewWriter(dstGz)
	defer dstTar.Close()

	// Copy all existing entries
	for {
		hdr, err := srcTar.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		if err := dstTar.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header: %w", err)
		}

		if hdr.Size > 0 {
			if _, err := io.Copy(dstTar, srcTar); err != nil {
				return fmt.Errorf("copy tar entry data: %w", err)
			}
		}
	}

	// Append server.txt
	content := serverURL + "\n"
	if err := dstTar.WriteHeader(&tar.Header{
		Name: "server.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		return fmt.Errorf("write server.txt header: %w", err)
	}

	if _, err := dstTar.Write([]byte(content)); err != nil {
		return fmt.Errorf("write server.txt content: %w", err)
	}

	return nil
}
