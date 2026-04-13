package cve

import (
	"math"
	"testing"
	"time"
)

func TestComputeRiskScore(t *testing.T) {
	now := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		cvss             float64
		cisaKEV          bool
		exploitAvailable bool
		publishedAt      time.Time
		want             float64
	}{
		{
			name:        "base CVSS only, recent",
			cvss:        7.5,
			publishedAt: now.AddDate(0, 0, -10),
			want:        7.5,
		},
		{
			name:        "CISA KEV boost",
			cvss:        7.5,
			cisaKEV:     true,
			publishedAt: now.AddDate(0, 0, -10),
			want:        9.0, // 7.5 * 1.2
		},
		{
			name:             "exploit available boost",
			cvss:             7.5,
			exploitAvailable: true,
			publishedAt:      now.AddDate(0, 0, -10),
			want:             8.25, // 7.5 * 1.1
		},
		{
			name:        "30-90 day age step",
			cvss:        7.5,
			publishedAt: now.AddDate(0, 0, -60),
			want:        8.25, // 7.5 * 1.1
		},
		{
			name:        "90+ day age step",
			cvss:        7.5,
			publishedAt: now.AddDate(0, 0, -120),
			want:        9.0, // 7.5 * 1.2
		},
		{
			name:             "all factors combined, capped at 10",
			cvss:             8.0,
			cisaKEV:          true,
			exploitAvailable: true,
			publishedAt:      now.AddDate(0, 0, -120),
			want:             10.0, // 8.0 * 1.5 = 12.0 → capped
		},
		{
			name:             "zero CVSS",
			cvss:             0.0,
			cisaKEV:          true,
			exploitAvailable: true,
			publishedAt:      now.AddDate(0, 0, -120),
			want:             0.0,
		},
		{
			name:        "exact 30 day boundary (in 30-90 tier)",
			cvss:        5.0,
			publishedAt: now.AddDate(0, 0, -30),
			want:        5.5, // 5.0 * 1.1
		},
		{
			name:        "exact 90 day boundary (in 90+ tier)",
			cvss:        5.0,
			publishedAt: now.AddDate(0, 0, -90),
			want:        6.0, // 5.0 * 1.2
		},
		{
			name:        "day 29 still in 0-30 tier",
			cvss:        5.0,
			publishedAt: now.AddDate(0, 0, -29),
			want:        5.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeRiskScore(tt.cvss, tt.cisaKEV, tt.exploitAvailable, tt.publishedAt, now)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("ComputeRiskScore() = %.2f, want %.2f", got, tt.want)
			}
		})
	}
}
