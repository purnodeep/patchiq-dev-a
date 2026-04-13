package repo

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

// BinaryCache manages a local directory of cached patch binaries organized by OS family.
type BinaryCache struct {
	dir    string
	client *http.Client
}

// NewBinaryCache creates a cache rooted at dir.
func NewBinaryCache(dir string, client *http.Client) *BinaryCache {
	return &BinaryCache{dir: dir, client: client}
}

// Download fetches a binary from url, stores it at {dir}/{osFamily}/{filename},
// and verifies the SHA256 checksum. If the file already exists with a matching
// checksum, the download is skipped.
func (c *BinaryCache) Download(ctx context.Context, url, osFamily, filename, expectedChecksum string) (string, error) {
	destDir := filepath.Join(c.dir, osFamily)
	destPath := filepath.Join(destDir, filename)

	// Check if already cached with correct checksum.
	if existing, err := verifyChecksum(destPath, expectedChecksum); err == nil && existing {
		slog.InfoContext(ctx, "binary already cached",
			"path", destPath,
			"checksum", expectedChecksum,
		)
		return destPath, nil
	}

	slog.InfoContext(ctx, "downloading binary to cache",
		"url", url,
		"dest", destPath,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("cache download %s: create request: %w", filename, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("cache download %s: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cache download %s: unexpected status %d", filename, resp.StatusCode)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("cache download %s: create dir: %w", filename, err)
	}

	tmpFile, err := os.CreateTemp(destDir, ".download-*")
	if err != nil {
		return "", fmt.Errorf("cache download %s: create temp file: %w", filename, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		// Clean up temp file on error.
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("cache download %s: write: %w", filename, err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("cache download %s: close temp: %w", filename, err)
	}

	gotChecksum := hex.EncodeToString(hasher.Sum(nil))
	if gotChecksum != expectedChecksum {
		return "", fmt.Errorf("cache download %s: checksum mismatch: got %s, want %s", filename, gotChecksum, expectedChecksum)
	}

	if err := os.Rename(tmpPath, destPath); err != nil {
		return "", fmt.Errorf("cache download %s: rename: %w", filename, err)
	}
	tmpPath = "" // Prevent cleanup of successfully renamed file.

	slog.InfoContext(ctx, "binary cached",
		"path", destPath,
		"checksum", gotChecksum,
	)

	return destPath, nil
}

// Dir returns the root cache directory.
func (c *BinaryCache) Dir() string {
	return c.dir
}

// verifyChecksum checks if a file exists and has the expected SHA256 checksum.
func verifyChecksum(path, expected string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false, err
	}

	got := hex.EncodeToString(hasher.Sum(nil))
	return got == expected, nil
}
