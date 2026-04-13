package protocol_test

import (
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/protocol"
)

func TestNegotiateProtocolVersion(t *testing.T) {
	tests := []struct {
		name      string
		agent     uint32
		server    uint32
		serverMin uint32
		want      uint32
		wantErr   bool
	}{
		{"same version", 1, 1, 1, 1, false},
		{"agent newer", 2, 1, 1, 1, false},
		{"server newer", 1, 2, 1, 1, false},
		{"agent below min", 1, 3, 2, 0, true},
		{"zero agent", 0, 1, 1, 0, true},
		{"zero server", 1, 0, 1, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := protocol.NegotiateProtocolVersion(tt.agent, tt.server, tt.serverMin)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
