package comms_test

import (
	"testing"
	"time"

	"github.com/skenzeriq/patchiq/internal/agent/comms"
)

func TestBackoff_ExponentialIncrease(t *testing.T) {
	cfg := comms.ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		JitterFactor: 0.0,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
	}

	for _, tt := range tests {
		got := comms.CalculateBackoff(cfg, tt.attempt)
		if got != tt.expected {
			t.Errorf("attempt %d: expected %v, got %v", tt.attempt, tt.expected, got)
		}
	}
}

func TestBackoff_CapsAtMaxDelay(t *testing.T) {
	cfg := comms.ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		JitterFactor: 0.0,
	}

	got := comms.CalculateBackoff(cfg, 20)
	if got != 5*time.Minute {
		t.Errorf("expected max delay 5m, got %v", got)
	}
}

func TestGRPCEnrollerImplementsEnroller(t *testing.T) {
	var _ comms.Enroller = comms.NewGRPCEnroller(nil)
}

func TestGRPCHeartbeatStreamerImplementsHeartbeatStreamer(t *testing.T) {
	var _ comms.HeartbeatStreamer = comms.NewGRPCHeartbeatStreamer(nil)
}

func TestBackoff_JitterWithinBounds(t *testing.T) {
	cfg := comms.ReconnectConfig{
		InitialDelay: 10 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}

	base := 10 * time.Second
	low := time.Duration(float64(base) * 0.8)
	high := time.Duration(float64(base) * 1.2)

	for i := 0; i < 100; i++ {
		got := comms.CalculateBackoff(cfg, 0)
		if got < low || got > high {
			t.Errorf("attempt 0 iteration %d: got %v, expected between %v and %v", i, got, low, high)
		}
	}
}
