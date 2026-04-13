package compliance

import (
	"testing"
	"time"
)

func TestComputeSLADeadline(t *testing.T) {
	published := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		slaDays  *int
		wantNil  bool
		wantDate time.Time
	}{
		{"15 day SLA", intPtr(15), false, time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)},
		{"30 day SLA", intPtr(30), false, time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)},
		{"nil SLA", nil, true, time.Time{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSLADeadline(published, tt.slaDays)
			if tt.wantNil {
				if got != nil {
					t.Errorf("ComputeSLADeadline() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("ComputeSLADeadline() = nil, want deadline")
				return
			}
			if !got.Equal(tt.wantDate) {
				t.Errorf("ComputeSLADeadline() = %v, want %v", got, tt.wantDate)
			}
		})
	}
}

func TestDaysRemaining(t *testing.T) {
	tests := []struct {
		name     string
		deadline time.Time
		now      time.Time
		want     int
	}{
		{
			"future deadline",
			time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
			10,
		},
		{
			"past deadline",
			time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			-5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DaysRemaining(tt.deadline, tt.now)
			if got != tt.want {
				t.Errorf("DaysRemaining() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEvaluateCVEState(t *testing.T) {
	published := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	threshold := 0.8 // 80% of SLA elapsed = at risk

	tests := []struct {
		name         string
		slaDays      *int
		remediatedAt *time.Time
		now          time.Time
		want         EvalState
	}{
		{
			"no SLA returns compliant",
			nil,
			nil,
			time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			StateCompliant,
		},
		{
			"patched within SLA",
			intPtr(30),
			timePtr(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
			time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			StateCompliant,
		},
		{
			"patched after SLA",
			intPtr(15),
			timePtr(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)),
			time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			StateLateRemediation,
		},
		{
			"not patched past deadline",
			intPtr(15),
			nil,
			time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
			StateNonCompliant,
		},
		{
			"not patched at risk threshold",
			intPtr(30),
			nil,
			// 80% of 30 days = 24 days elapsed. published + 24 = Jan 25.
			time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC),
			StateAtRisk,
		},
		{
			"not patched within SLA below threshold",
			intPtr(30),
			nil,
			// only 10 days elapsed out of 30 (33%) < 80% threshold
			time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC),
			StateCompliant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateCVEState(published, tt.slaDays, tt.remediatedAt, tt.now, threshold)
			if got != tt.want {
				t.Errorf("EvaluateCVEState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
