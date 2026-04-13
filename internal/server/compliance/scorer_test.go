package compliance

import (
	"math"
	"testing"
)

func TestComputeEndpointScore(t *testing.T) {
	tests := []struct {
		name       string
		states     []EvalState
		wantScore  float64
		wantCounts StateCounts
	}{
		{
			"all compliant",
			[]EvalState{StateCompliant, StateCompliant, StateCompliant},
			100.0,
			StateCounts{Total: 3, Compliant: 3},
		},
		{
			"mixed states",
			[]EvalState{StateCompliant, StateCompliant, StateAtRisk, StateNonCompliant, StateLateRemediation},
			40.0,
			StateCounts{Total: 5, Compliant: 2, AtRisk: 1, NonCompliant: 1, LateRemediation: 1},
		},
		{
			"no CVEs",
			[]EvalState{},
			100.0,
			StateCounts{},
		},
		{
			"all non-compliant",
			[]EvalState{StateNonCompliant, StateNonCompliant, StateNonCompliant},
			0.0,
			StateCounts{Total: 3, NonCompliant: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, counts := ComputeEndpointScore(tt.states)
			if !floatEqual(score, tt.wantScore) {
				t.Errorf("ComputeEndpointScore() score = %f, want %f", score, tt.wantScore)
			}
			if counts != tt.wantCounts {
				t.Errorf("ComputeEndpointScore() counts = %+v, want %+v", counts, tt.wantCounts)
			}
		})
	}
}

func TestComputeGroupScore(t *testing.T) {
	scores := []float64{95.0, 90.0, 100.0, 85.0}

	tests := []struct {
		name   string
		method string
		want   float64
	}{
		{
			"strictest method",
			"strictest",
			50.0, // 2 out of 4 endpoints >= 95 (95.0 and 100.0) = 50%
		},
		{
			"worst_case method",
			"worst_case",
			85.0, // minimum score
		},
		{
			"average method",
			"average",
			92.5, // (95+90+100+85)/4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeGroupScore(scores, tt.method)
			if !floatEqual(got, tt.want) {
				t.Errorf("ComputeGroupScore(%v, %q) = %f, want %f", scores, tt.method, got, tt.want)
			}
		})
	}
}

func TestComputeGroupScoreEmpty(t *testing.T) {
	got := ComputeGroupScore([]float64{}, "average")
	if got != 100.0 {
		t.Errorf("ComputeGroupScore(empty) = %f, want 100.0", got)
	}
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
