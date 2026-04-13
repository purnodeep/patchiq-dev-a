//go:build darwin

package inventory

import (
	"context"
	"testing"
)

func TestLookupAppleChipFreq(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantMax   float64
		wantMin   float64
		wantFound bool
	}{
		{
			name:      "exact match M1",
			model:     "Apple M1",
			wantMax:   3200,
			wantMin:   2064,
			wantFound: true,
		},
		{
			name:      "exact match M4 Pro",
			model:     "Apple M4 Pro",
			wantMax:   4500,
			wantMin:   2850,
			wantFound: true,
		},
		{
			name:      "prefix match with extra text",
			model:     "Apple M3 Max (16-core)",
			wantMax:   4050,
			wantMin:   2750,
			wantFound: true,
		},
		{
			name:      "unknown chip",
			model:     "Apple M5",
			wantMax:   0,
			wantMin:   0,
			wantFound: false,
		},
		{
			name:      "Intel chip",
			model:     "Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz",
			wantMax:   0,
			wantMin:   0,
			wantFound: false,
		},
		{
			name:      "empty model",
			model:     "",
			wantMax:   0,
			wantMin:   0,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMax, gotMin, gotOK := lookupAppleChipFreq(tt.model)
			if gotOK != tt.wantFound {
				t.Errorf("lookupAppleChipFreq(%q) ok = %v, want %v", tt.model, gotOK, tt.wantFound)
			}
			if gotMax != tt.wantMax {
				t.Errorf("lookupAppleChipFreq(%q) maxMHz = %v, want %v", tt.model, gotMax, tt.wantMax)
			}
			if gotMin != tt.wantMin {
				t.Errorf("lookupAppleChipFreq(%q) minMHz = %v, want %v", tt.model, gotMin, tt.wantMin)
			}
		})
	}
}

func TestParsePowermetricsFreq(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantNil     bool
		wantPClust  float64
		wantEClust  float64
		wantPerCore map[int]float64
	}{
		{
			name: "typical M3 Max output",
			input: `Machine model: Mac16,1

*** Sampled system activity (Sat Apr  5 10:00:00 2025 -0700) (1ms elapsed) ***

**** Processor usage ****

P0-Cluster HW active frequency: 4050 MHz
P0-Cluster HW active residency:   3.21%
E0-Cluster HW active frequency: 2748 MHz
E0-Cluster HW active residency:  12.54%
CPU 0 active frequency: 2748 MHz
CPU 1 active frequency: 2748 MHz
CPU 2 active frequency: 2748 MHz
CPU 3 active frequency: 2748 MHz
CPU 4 active frequency: 4050 MHz
CPU 5 active frequency: 4050 MHz
`,
			wantNil:    false,
			wantPClust: 4050,
			wantEClust: 2748,
			wantPerCore: map[int]float64{
				0: 2748, 1: 2748, 2: 2748, 3: 2748,
				4: 4050, 5: 4050,
			},
		},
		{
			name: "cluster only, no per-core",
			input: `P-Cluster HW active frequency: 4408 MHz
E-Cluster HW active frequency: 2856 MHz
`,
			wantNil:     false,
			wantPClust:  4408,
			wantEClust:  2856,
			wantPerCore: map[int]float64{},
		},
		{
			name: "multiple P clusters, take highest",
			input: `P0-Cluster HW active frequency: 3800 MHz
P1-Cluster HW active frequency: 4200 MHz
E0-Cluster HW active frequency: 2400 MHz
E1-Cluster HW active frequency: 2200 MHz
`,
			wantNil:     false,
			wantPClust:  4200,
			wantEClust:  2200,
			wantPerCore: map[int]float64{},
		},
		{
			name:    "empty output",
			input:   "",
			wantNil: true,
		},
		{
			name:    "unrelated output",
			input:   "some random text\nno frequencies here\n",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePowermetricsFreq([]byte(tt.input))
			if tt.wantNil {
				if result != nil {
					t.Fatalf("parsePowermetricsFreq() = %+v, want nil", result)
				}
				return
			}
			if result == nil {
				t.Fatal("parsePowermetricsFreq() = nil, want non-nil")
			}
			if result.PClusterMHz != tt.wantPClust {
				t.Errorf("PClusterMHz = %v, want %v", result.PClusterMHz, tt.wantPClust)
			}
			if result.EClusterMHz != tt.wantEClust {
				t.Errorf("EClusterMHz = %v, want %v", result.EClusterMHz, tt.wantEClust)
			}
			if tt.wantPerCore != nil {
				for coreID, wantFreq := range tt.wantPerCore {
					if gotFreq, ok := result.PerCoreMHz[coreID]; !ok {
						t.Errorf("PerCoreMHz[%d] not present, want %v", coreID, wantFreq)
					} else if gotFreq != wantFreq {
						t.Errorf("PerCoreMHz[%d] = %v, want %v", coreID, gotFreq, wantFreq)
					}
				}
			}
		})
	}
}

func TestFillAppleSiliconFreq_AlreadyPopulated(t *testing.T) {
	info := &CPUInfo{
		ModelName: "Apple M3 Pro",
		MaxMHz:    4050,
		MinMHz:    2750,
	}
	// Should be a no-op when both values are already set.
	fillAppleSiliconFreq(context.TODO(), info)
	if info.MaxMHz != 4050 || info.MinMHz != 2750 {
		t.Errorf("fillAppleSiliconFreq modified already-populated values: max=%v min=%v",
			info.MaxMHz, info.MinMHz)
	}
}
