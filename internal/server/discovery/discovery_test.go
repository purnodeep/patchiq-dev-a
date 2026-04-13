package discovery

import (
	"testing"
)

func TestDiscoveredPatchZeroValue(t *testing.T) {
	var p DiscoveredPatch
	if p.Name != "" {
		t.Errorf("expected empty Name, got %q", p.Name)
	}
	if p.Size != 0 {
		t.Errorf("expected zero Size, got %d", p.Size)
	}
}

func TestParserInterfaceSatisfied(t *testing.T) {
	// Compile-time check that APTParser satisfies Parser.
	var _ Parser = (*APTParser)(nil)
}

func TestDiscoveredPatch_Validate(t *testing.T) {
	tests := []struct {
		name    string
		patch   DiscoveredPatch
		wantErr bool
	}{
		{
			name:    "valid patch",
			patch:   DiscoveredPatch{Name: "curl", Version: "7.81.0", OsFamily: "debian"},
			wantErr: false,
		},
		{
			name:    "empty name",
			patch:   DiscoveredPatch{Name: "", Version: "7.81.0", OsFamily: "debian"},
			wantErr: true,
		},
		{
			name:    "empty version",
			patch:   DiscoveredPatch{Name: "curl", Version: "", OsFamily: "debian"},
			wantErr: true,
		},
		{
			name:    "empty os_family",
			patch:   DiscoveredPatch{Name: "curl", Version: "7.81.0", OsFamily: ""},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.patch.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
