package discovery

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// APTParser streams packages from a gzip-compressed APT Packages file (Debian control format).
type APTParser struct {
	OsFamily   string
	OsDistro   string
	SourceRepo string
}

// Parse reads a gzip-compressed Packages file and yields DiscoveredPatch values.
func (p *APTParser) Parse(ctx context.Context, r io.Reader) func(yield func(DiscoveredPatch, error) bool) {
	return func(yield func(DiscoveredPatch, error) bool) {
		gz, err := gzip.NewReader(r)
		if err != nil {
			yield(DiscoveredPatch{}, fmt.Errorf("apt parser: open gzip: %w", err))
			return
		}
		defer gz.Close()

		scanner := bufio.NewScanner(gz)
		fields := make(map[string]string)

		flush := func() bool {
			if len(fields) == 0 {
				return true
			}
			patch := p.toPatch(fields)
			ok := yield(patch, nil)
			for k := range fields {
				delete(fields, k)
			}
			return ok
		}

		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}
			line := scanner.Text()
			if line == "" {
				if !flush() {
					return
				}
				continue
			}
			if key, value, ok := strings.Cut(line, ": "); ok {
				fields[key] = value
			}
		}
		if err := scanner.Err(); err != nil {
			yield(DiscoveredPatch{}, fmt.Errorf("apt parser: scan: %w", err))
			return
		}
		flush()
	}
}

func (p *APTParser) toPatch(fields map[string]string) DiscoveredPatch {
	size, _ := strconv.ParseInt(fields["Size"], 10, 64)
	return DiscoveredPatch{
		Name:        fields["Package"],
		Version:     fields["Version"],
		Arch:        fields["Architecture"],
		OsFamily:    p.OsFamily,
		OsDistro:    p.OsDistro,
		Priority:    fields["Priority"],
		Section:     fields["Section"],
		Description: fields["Description"],
		Filename:    fields["Filename"],
		Size:        size,
		Checksum:    fields["SHA256"],
		SourceRepo:  p.SourceRepo,
	}
}
