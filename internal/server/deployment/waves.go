package deployment

import (
	"encoding/json"
	"fmt"
)

// WaveConfig defines a single wave in a deployment's rollout plan.
type WaveConfig struct {
	Percentage       int     `json:"percentage"`
	SuccessThreshold float64 `json:"success_threshold"`
	ErrorRateMax     float64 `json:"error_rate_max"`
	DelayMinutes     int     `json:"delay_minutes"`
}

// ParseWaveConfig parses JSONB wave config from the database.
// Returns a single 100% wave if data is nil or empty (backward compatible).
func ParseWaveConfig(data []byte) ([]WaveConfig, error) {
	if len(data) == 0 {
		return []WaveConfig{{
			Percentage:       100,
			SuccessThreshold: 0.80,
			ErrorRateMax:     DefaultFailureThreshold,
			DelayMinutes:     0,
		}}, nil
	}

	var waves []WaveConfig
	if err := json.Unmarshal(data, &waves); err != nil {
		return nil, fmt.Errorf("parse wave config: %w", err)
	}
	if len(waves) == 0 {
		return nil, fmt.Errorf("parse wave config: empty wave list")
	}
	return waves, nil
}

// AssignTargetsToWaves distributes targetCount across waves by percentage.
// The last wave gets any remainder to ensure all targets are assigned.
func AssignTargetsToWaves(waves []WaveConfig, targetCount int) []int {
	assignments := make([]int, len(waves))
	assigned := 0

	for i, w := range waves {
		if i == len(waves)-1 {
			assignments[i] = targetCount - assigned
		} else {
			count := (targetCount * w.Percentage) / 100
			assignments[i] = count
			assigned += count
		}
	}
	return assignments
}
