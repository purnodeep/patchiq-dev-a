package cli

import "testing"

func TestDefaultServerAddress_DefaultsToEmpty(t *testing.T) {
	// In a development build (no -ldflags injection), DefaultServerAddress
	// must be empty so install validation forces an explicit --server.
	if DefaultServerAddress != "" {
		t.Errorf("DefaultServerAddress should be empty in dev builds, got %q", DefaultServerAddress)
	}
}

func TestResolveServerAddress(t *testing.T) {
	tests := []struct {
		name    string
		flag    string
		envVal  string
		baked   string
		want    string
		wantErr bool
	}{
		{name: "flag wins", flag: "flag.example:50051", envVal: "env.example:50051", baked: "baked.example:50051", want: "flag.example:50051"},
		{name: "env wins over baked", flag: "", envVal: "env.example:50051", baked: "baked.example:50051", want: "env.example:50051"},
		{name: "baked when no flag or env", flag: "", envVal: "", baked: "baked.example:50051", want: "baked.example:50051"},
		{name: "error when nothing set", flag: "", envVal: "", baked: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveServerAddress(tt.flag, tt.envVal, tt.baked)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
