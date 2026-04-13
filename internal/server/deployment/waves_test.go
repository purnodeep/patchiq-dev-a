package deployment

import (
	"testing"
)

func TestParseWaveConfig(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    int // number of waves
		wantErr bool
	}{
		{
			name: "valid two-wave config",
			data: []byte(`[{"percentage":10,"success_threshold":0.95,"error_rate_max":0.10,"delay_minutes":30},{"percentage":90,"success_threshold":0.80,"error_rate_max":0.20,"delay_minutes":0}]`),
			want: 2,
		},
		{
			name: "nil means single 100% wave",
			data: nil,
			want: 1,
		},
		{
			name: "empty bytes means single 100% wave",
			data: []byte{},
			want: 1,
		},
		{
			name:    "invalid json",
			data:    []byte(`{invalid}`),
			wantErr: true,
		},
		{
			name:    "empty array",
			data:    []byte(`[]`),
			wantErr: true,
		},
		{
			name: "four-wave config",
			data: []byte(`[{"percentage":10,"success_threshold":0.95,"error_rate_max":0.10,"delay_minutes":30},{"percentage":25,"success_threshold":0.90,"error_rate_max":0.15,"delay_minutes":15},{"percentage":50,"success_threshold":0.90,"error_rate_max":0.20,"delay_minutes":0},{"percentage":15,"success_threshold":0.80,"error_rate_max":0.25,"delay_minutes":0}]`),
			want: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			waves, err := ParseWaveConfig(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseWaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(waves) != tt.want {
				t.Errorf("got %d waves, want %d", len(waves), tt.want)
			}
		})
	}
}

func TestParseWaveConfig_DefaultValues(t *testing.T) {
	waves, err := ParseWaveConfig(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(waves) != 1 {
		t.Fatalf("expected 1 wave, got %d", len(waves))
	}
	w := waves[0]
	if w.Percentage != 100 {
		t.Errorf("default percentage = %d, want 100", w.Percentage)
	}
	if w.SuccessThreshold != 0.80 {
		t.Errorf("default success_threshold = %f, want 0.80", w.SuccessThreshold)
	}
	if w.ErrorRateMax != DefaultFailureThreshold {
		t.Errorf("default error_rate_max = %f, want %f", w.ErrorRateMax, DefaultFailureThreshold)
	}
}

func TestAssignTargetsToWaves(t *testing.T) {
	tests := []struct {
		name        string
		waves       []WaveConfig
		targetCount int
		wantSizes   []int // expected assignments per wave
		wantTotal   int   // must equal targetCount
	}{
		{
			name:        "single 100% wave",
			waves:       []WaveConfig{{Percentage: 100}},
			targetCount: 50,
			wantSizes:   []int{50},
			wantTotal:   50,
		},
		{
			name: "10/25/65 split",
			waves: []WaveConfig{
				{Percentage: 10},
				{Percentage: 25},
				{Percentage: 65},
			},
			targetCount: 50,
			wantSizes:   []int{5, 12, 33}, // last wave gets remainder
			wantTotal:   50,
		},
		{
			name: "small target count with rounding",
			waves: []WaveConfig{
				{Percentage: 10},
				{Percentage: 90},
			},
			targetCount: 3,
			wantSizes:   []int{0, 3}, // 10% of 3 = 0.3, rounds to 0
			wantTotal:   3,
		},
		{
			name: "single target",
			waves: []WaveConfig{
				{Percentage: 10},
				{Percentage: 90},
			},
			targetCount: 1,
			wantSizes:   []int{0, 1},
			wantTotal:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignments := AssignTargetsToWaves(tt.waves, tt.targetCount)
			if len(assignments) != len(tt.waves) {
				t.Fatalf("got %d assignments, want %d", len(assignments), len(tt.waves))
			}

			total := 0
			for i, a := range assignments {
				total += a
				if tt.wantSizes != nil && a != tt.wantSizes[i] {
					t.Errorf("wave %d: got %d targets, want %d", i+1, a, tt.wantSizes[i])
				}
			}
			if total != tt.wantTotal {
				t.Errorf("total assigned = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}
