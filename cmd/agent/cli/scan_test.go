package cli

import "testing"

func TestScanParseFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantDryRun bool
		wantErr    bool
	}{
		{
			name:       "defaults",
			args:       []string{},
			wantDryRun: false,
		},
		{
			name:       "dry-run flag",
			args:       []string{"--dry-run"},
			wantDryRun: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseScanFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts.dryRun != tt.wantDryRun {
				t.Errorf("dryRun = %v, want %v", opts.dryRun, tt.wantDryRun)
			}
		})
	}
}
