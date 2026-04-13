package otel_test

import (
	"testing"

	piqotel "github.com/skenzeriq/patchiq/internal/shared/otel"
)

func TestGRPCHandlersNotNil(t *testing.T) {
	tests := []struct {
		name string
		fn   func() any
	}{
		{"ServerHandler", func() any { return piqotel.GRPCServerHandler() }},
		{"ClientHandler", func() any { return piqotel.GRPCClientHandler() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fn() == nil {
				t.Fatalf("%s should not be nil", tt.name)
			}
		})
	}
}
