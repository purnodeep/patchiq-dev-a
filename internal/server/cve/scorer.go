package cve

import (
	"math"
	"time"
)

// ComputeRiskScore calculates risk score using CVSS, CISA KEV, exploit availability, and age.
// Formula: score = CVSS * (1.0 + 0.2*CISA_KEV + 0.1*exploit_available + age_step)
// Age steps: 0-29 days = +0.0, 30-89 days = +0.1, 90+ days = +0.2
// Capped at 10.0.
func ComputeRiskScore(cvss float64, cisaKEV, exploitAvailable bool, publishedAt, now time.Time) float64 {
	multiplier := 1.0

	if cisaKEV {
		multiplier += 0.2
	}
	if exploitAvailable {
		multiplier += 0.1
	}

	daysSincePublished := int(now.Sub(publishedAt).Hours() / 24)
	switch {
	case daysSincePublished >= 90:
		multiplier += 0.2
	case daysSincePublished >= 30:
		multiplier += 0.1
	}

	score := cvss * multiplier
	return math.Min(score, 10.0)
}
