package catalog

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// APTPackageResolver resolves source package names to download URLs by parsing
// Ubuntu's Packages.gz indices. Results are cached in memory.
type APTPackageResolver struct {
	client *http.Client
	mu     sync.RWMutex
	// cache maps "source_package/version" → full download URL
	cache     map[string]string
	lastFetch time.Time
	cacheTTL  time.Duration
}

func NewAPTPackageResolver() *APTPackageResolver {
	return &APTPackageResolver{
		client:   &http.Client{Timeout: 2 * time.Minute},
		cache:    make(map[string]string),
		cacheTTL: 6 * time.Hour,
	}
}

// Resolve returns the download URL for a package given its source name and version.
// Returns empty string if not found.
func (r *APTPackageResolver) Resolve(ctx context.Context, sourcePackage, version string) string {
	key := sourcePackage + "/" + version

	r.mu.RLock()
	if url, ok := r.cache[key]; ok {
		r.mu.RUnlock()
		slog.DebugContext(ctx, "apt resolver: cache hit", "key", key)
		return url
	}
	if time.Since(r.lastFetch) < r.cacheTTL && len(r.cache) > 0 {
		r.mu.RUnlock()
		slog.DebugContext(ctx, "apt resolver: cache miss", "key", key)
		return "" // Cache is fresh, package not found
	}
	r.mu.RUnlock()

	// Refresh cache
	r.refresh(ctx)

	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cache[key]
}

func (r *APTPackageResolver) refresh(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check under write lock
	if time.Since(r.lastFetch) < r.cacheTTL && len(r.cache) > 0 {
		return
	}

	baseURLs := []string{
		// Security repos (USN patches)
		"http://security.ubuntu.com/ubuntu/dists/noble-security",  // 24.04
		"http://security.ubuntu.com/ubuntu/dists/jammy-security",  // 22.04
		"http://security.ubuntu.com/ubuntu/dists/focal-security",  // 20.04
		"http://security.ubuntu.com/ubuntu/dists/bionic-security", // 18.04
		// Update repos (broader coverage)
		"http://archive.ubuntu.com/ubuntu/dists/noble-updates",  // 24.04
		"http://archive.ubuntu.com/ubuntu/dists/jammy-updates",  // 22.04
		"http://archive.ubuntu.com/ubuntu/dists/focal-updates",  // 20.04
		"http://archive.ubuntu.com/ubuntu/dists/bionic-updates", // 18.04
	}
	sections := []string{"main", "universe"}
	arches := []string{"amd64", "arm64"}

	newCache := make(map[string]string, len(r.cache))

	for _, base := range baseURLs {
		for _, section := range sections {
			for _, arch := range arches {
				url := fmt.Sprintf("%s/%s/binary-%s/Packages.gz", base, section, arch)
				r.parsePackagesGz(ctx, url, newCache)
			}
		}
	}

	if len(newCache) > 0 {
		r.cache = newCache
		r.lastFetch = time.Now()
		slog.Info("apt resolver: refreshed package cache", "entries", len(newCache))
	}
}

func (r *APTPackageResolver) parsePackagesGz(ctx context.Context, url string, cache map[string]string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	resp, err := r.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return
	}
	defer resp.Body.Close()

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return
	}
	defer gz.Close()

	// Extract base URL (e.g., "http://security.ubuntu.com/ubuntu")
	// from the Packages.gz URL by removing "/dists/..."
	baseURL := url[:strings.Index(url, "/dists/")]

	scanner := bufio.NewScanner(gz)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
	var stanzaCount, storedCount int

	// Parse the Packages file format:
	// Package: dovecot-core
	// Source: dovecot (2:2.3.16+dfsg1-3ubuntu2.7)
	// Version: 2:2.3.16+dfsg1-3ubuntu2.7
	// Filename: pool/main/d/dovecot/dovecot-core_2.3.16+dfsg1-3ubuntu2.7_amd64.deb
	var pkgName, source, version, filename string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// If no Source field, the package name IS the source name
			if source == "" {
				source = pkgName
			}
			stanzaCount++
			// End of package stanza — store if we have source + version + filename
			if source != "" && version != "" && filename != "" {
				key := source + "/" + version
				if _, exists := cache[key]; !exists {
					cache[key] = baseURL + "/" + filename
					storedCount++
				}
			}
			pkgName = ""
			source = ""
			version = ""
			filename = ""
			continue
		}

		switch {
		case strings.HasPrefix(line, "Package: "):
			pkgName = strings.TrimPrefix(line, "Package: ")
		case strings.HasPrefix(line, "Source: "):
			// Source line may be "Source: dovecot" or "Source: dovecot (2:2.3.16+dfsg1-3ubuntu2.7)"
			s := strings.TrimPrefix(line, "Source: ")
			if idx := strings.Index(s, " ("); idx >= 0 {
				source = s[:idx]
			} else {
				source = s
			}
		case strings.HasPrefix(line, "Version: "):
			version = strings.TrimPrefix(line, "Version: ")
		case strings.HasPrefix(line, "Filename: "):
			filename = strings.TrimPrefix(line, "Filename: ")
		}
	}

	// Handle last stanza if file doesn't end with blank line
	if source == "" {
		source = pkgName
	}
	if source != "" && version != "" && filename != "" {
		key := source + "/" + version
		if _, exists := cache[key]; !exists {
			cache[key] = baseURL + "/" + filename
			storedCount++
		}
	}
	slog.Info("apt resolver: parsed index", "url", url, "stanzas", stanzaCount, "stored", storedCount)
}
