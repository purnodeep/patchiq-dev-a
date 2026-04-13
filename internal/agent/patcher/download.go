package patcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

// Downloader handles downloading patch binaries from the Server and verifying checksums.
type Downloader struct {
	client *http.Client
	tmpDir string
}

// NewDownloader creates a downloader that stores files in tmpDir.
func NewDownloader(client *http.Client, tmpDir string) *Downloader {
	return &Downloader{client: client, tmpDir: tmpDir}
}

// Download fetches a binary from url to a temp directory and verifies the SHA256 checksum.
// If expectedChecksum is empty, checksum verification is skipped.
// Returns the local file path of the downloaded binary.
func (d *Downloader) Download(ctx context.Context, url, expectedChecksum string) (string, error) {
	slog.InfoContext(ctx, "downloading binary from server",
		"url", url,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("download binary: create request: %w", err)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download binary: unexpected status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(d.tmpDir, 0o755); err != nil {
		return "", fmt.Errorf("download binary: create temp dir: %w", err)
	}

	// Extract filename from URL path for a meaningful temp file name.
	filename := filepath.Base(url)
	if filename == "" || filename == "." || filename == "/" {
		filename = "patch-binary"
	}

	tmpFile, err := os.CreateTemp(d.tmpDir, filename+"-*")
	if err != nil {
		return "", fmt.Errorf("download binary: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("download binary: write: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("download binary: close: %w", err)
	}

	if expectedChecksum != "" {
		gotChecksum := hex.EncodeToString(hasher.Sum(nil))
		if gotChecksum != expectedChecksum {
			os.Remove(tmpPath)
			return "", fmt.Errorf("download binary: checksum mismatch: got %s, want %s", gotChecksum, expectedChecksum)
		}
		slog.InfoContext(ctx, "binary checksum verified",
			"checksum", gotChecksum,
		)
	}

	slog.InfoContext(ctx, "binary downloaded",
		"path", tmpPath,
	)

	return tmpPath, nil
}
