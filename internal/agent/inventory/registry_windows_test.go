//go:build windows

package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRegistryReader struct {
	entries []registryEntry
	err     error
}

func (m *mockRegistryReader) ReadUninstallKeys() ([]registryEntry, error) {
	return m.entries, m.err
}

func TestRegistryCollector_Name(t *testing.T) {
	c := &registryCollector{}
	assert.Equal(t, "registry", c.Name())
}

func TestRegistryCollector_Collect(t *testing.T) {
	c := &registryCollector{
		reader: &mockRegistryReader{
			entries: []registryEntry{
				{DisplayName: "Google Chrome", DisplayVersion: "122.0.6261.94", Publisher: "Google LLC"},
				{DisplayName: "7-Zip", DisplayVersion: "23.01", Publisher: "Igor Pavlov"},
				{DisplayName: "", DisplayVersion: "1.0", Publisher: "System"},
			},
		},
	}

	pkgs, err := c.Collect(context.Background())
	require.NoError(t, err)
	assert.Len(t, pkgs, 2, "should skip entry with empty DisplayName")
	assert.Equal(t, "Google Chrome", pkgs[0].Name)
	assert.Equal(t, "122.0.6261.94", pkgs[0].Version)
	assert.Equal(t, "registry", pkgs[0].Source)
}

func TestRegistryCollector_Dedup(t *testing.T) {
	c := &registryCollector{
		reader: &mockRegistryReader{
			entries: []registryEntry{
				{DisplayName: "App", DisplayVersion: "1.0", Publisher: "Vendor", Is64Bit: true},
				{DisplayName: "App", DisplayVersion: "1.0", Publisher: "Vendor", Is64Bit: false},
			},
		},
	}

	pkgs, err := c.Collect(context.Background())
	require.NoError(t, err)
	assert.Len(t, pkgs, 1, "should deduplicate same app across 64/32-bit paths")
}
