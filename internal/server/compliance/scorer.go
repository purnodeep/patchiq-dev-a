package compliance

// StateCounts holds the count of CVEs in each compliance state.
type StateCounts struct {
	Total           int
	Compliant       int
	AtRisk          int
	NonCompliant    int
	LateRemediation int
}

// ComputeEndpointScore calculates the compliance score for an endpoint.
// Score = (compliant / total) * 100. Returns 100.0 for empty input.
func ComputeEndpointScore(states []EvalState) (float64, StateCounts) {
	if len(states) == 0 {
		return 100.0, StateCounts{}
	}

	var counts StateCounts
	counts.Total = len(states)

	for _, s := range states {
		switch s {
		case StateCompliant:
			counts.Compliant++
		case StateAtRisk:
			counts.AtRisk++
		case StateNonCompliant:
			counts.NonCompliant++
		case StateLateRemediation:
			counts.LateRemediation++
		}
	}

	score := float64(counts.Compliant) / float64(counts.Total) * 100.0
	return score, counts
}

// ComputeGroupScore calculates a group-level compliance score from endpoint scores.
// Methods: "strictest" (% of endpoints >= 95), "worst_case" (minimum), "average" (mean).
// Returns 100.0 for empty input.
func ComputeGroupScore(endpointScores []float64, method string) float64 {
	if len(endpointScores) == 0 {
		return 100.0
	}

	switch method {
	case "strictest":
		passing := 0
		for _, s := range endpointScores {
			if s >= 95.0 {
				passing++
			}
		}
		return float64(passing) / float64(len(endpointScores)) * 100.0

	case "worst_case":
		min := endpointScores[0]
		for _, s := range endpointScores[1:] {
			if s < min {
				min = s
			}
		}
		return min

	case "weighted":
		// Harmonic mean — penalises low-scoring endpoints more heavily
		// than arithmetic mean, rewarding consistent compliance.
		sum := 0.0
		for _, s := range endpointScores {
			if s <= 0 {
				s = 0.1
			}
			sum += 1.0 / s
		}
		return float64(len(endpointScores)) / sum

	default: // "average"
		sum := 0.0
		for _, s := range endpointScores {
			sum += s
		}
		return sum / float64(len(endpointScores))
	}
}
