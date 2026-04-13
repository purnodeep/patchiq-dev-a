package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallerTypeOrFallback(t *testing.T) {
	tests := []struct {
		name          string
		installerType string
		osFamily      string
		wantSource    string
	}{
		{"wua from installer_type", "wua", "windows", "wua"},
		{"exe from installer_type", "exe", "windows", "exe"},
		{"fallback for empty windows", "", "windows", "msi"},
		{"fallback for linux", "", "linux-ubuntu", "apt"},
		{"apt from installer_type", "apt", "linux-debian", "apt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := installerTypeOrFallback(tt.installerType, tt.osFamily)
			assert.Equal(t, tt.wantSource, got)
		})
	}
}
