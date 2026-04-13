package repo

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// MountFileServer registers HTTP routes to serve cached binaries from the cache directory.
// Route pattern: GET /repo/files/{os}/{filename}
func MountFileServer(r chi.Router, cacheDir string) {
	r.Get("/repo/files/{os}/{filename}", func(w http.ResponseWriter, r *http.Request) {
		osFamily := chi.URLParam(r, "os")
		filename := chi.URLParam(r, "filename")

		// Sanitize path components to prevent traversal.
		osFamily = filepath.Base(osFamily)
		filename = filepath.Base(filename)

		if osFamily == "." || osFamily == ".." || filename == "." || filename == ".." {
			http.NotFound(w, r)
			return
		}

		filePath := filepath.Join(cacheDir, osFamily, filename)

		// Verify resolved path is within cache directory.
		absCache, err := filepath.Abs(cacheDir)
		if err != nil {
			slog.ErrorContext(r.Context(), "repo file server: resolve cache dir", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		absFile, err := filepath.Abs(filePath)
		if err != nil {
			slog.ErrorContext(r.Context(), "repo file server: resolve file path", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !strings.HasPrefix(absFile, absCache+string(os.PathSeparator)) {
			http.NotFound(w, r)
			return
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}

		slog.InfoContext(r.Context(), "serving binary",
			"os", osFamily,
			"filename", filename,
		)

		http.ServeFile(w, r, filePath)
	})
}
