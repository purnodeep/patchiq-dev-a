package discovery

import (
	"context"
	"fmt"
	"io"
)

// DiscoveredPatch is the normalized output from any repository metadata parser.
type DiscoveredPatch struct {
	Name        string
	Version     string
	Arch        string
	OsFamily    string // "debian" or "rhel"
	OsDistro    string // "ubuntu-22.04", "rhel-9"
	Priority    string
	Section     string
	Summary     string
	Description string
	Filename    string
	Size        int64
	Checksum    string // Checksum (SHA256 for APT, varies for YUM)
	SourceRepo  string
}

// Validate checks that required fields are present.
func (p DiscoveredPatch) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("discovered patch: name is empty")
	}
	if p.Version == "" {
		return fmt.Errorf("discovered patch: version is empty")
	}
	if p.OsFamily == "" {
		return fmt.Errorf("discovered patch: os_family is empty")
	}
	return nil
}

// Parser streams discovered patches from repository metadata.
type Parser interface {
	Parse(ctx context.Context, r io.Reader) func(yield func(DiscoveredPatch, error) bool)
}
