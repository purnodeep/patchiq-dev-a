package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// ptr is a generic helper to get a pointer to any value.
func ptr[T any](v T) *T {
	return &v
}

func TestMergeConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		base    DeployConfig
		overlay DeployConfig
		want    DeployConfig
	}{
		{
			name: "nil field does not override",
			base: DeployConfig{
				AutoReboot: ptr(true),
			},
			overlay: DeployConfig{
				AutoReboot: nil,
			},
			want: DeployConfig{
				AutoReboot: ptr(true),
			},
		},
		{
			name: "non-nil scalar overrides",
			base: DeployConfig{
				WaveStrategy: ptr("sequential"),
			},
			overlay: DeployConfig{
				WaveStrategy: ptr("parallel"),
			},
			want: DeployConfig{
				WaveStrategy: ptr("parallel"),
			},
		},
		{
			name: "slice is additive",
			base: DeployConfig{
				ExcludedPackages: []string{"pkg-a"},
			},
			overlay: DeployConfig{
				ExcludedPackages: []string{"pkg-b"},
			},
			want: DeployConfig{
				ExcludedPackages: []string{"pkg-a", "pkg-b"},
			},
		},
		{
			name: "slice deduplicates",
			base: DeployConfig{
				ExcludedPackages: []string{"pkg-a", "pkg-b"},
			},
			overlay: DeployConfig{
				ExcludedPackages: []string{"pkg-a"},
			},
			want: DeployConfig{
				ExcludedPackages: []string{"pkg-a", "pkg-b"},
			},
		},
		{
			name: "struct pointer replaces entirely",
			base: DeployConfig{
				MaintenanceWindow: &TimeWindow{Day: "Monday", Start: "02:00", End: "04:00"},
			},
			overlay: DeployConfig{
				MaintenanceWindow: &TimeWindow{Day: "Sunday", Start: "01:00", End: "03:00"},
			},
			want: DeployConfig{
				MaintenanceWindow: &TimeWindow{Day: "Sunday", Start: "01:00", End: "03:00"},
			},
		},
		{
			name: "all-nil overlay changes nothing",
			base: DeployConfig{
				AutoReboot:        ptr(true),
				WaveStrategy:      ptr("sequential"),
				MaxConcurrent:     ptr(5),
				ExcludedPackages:  []string{"pkg-a"},
				MaintenanceWindow: &TimeWindow{Day: "Monday", Start: "02:00", End: "04:00"},
			},
			overlay: DeployConfig{},
			want: DeployConfig{
				AutoReboot:        ptr(true),
				WaveStrategy:      ptr("sequential"),
				MaxConcurrent:     ptr(5),
				ExcludedPackages:  []string{"pkg-a"},
				MaintenanceWindow: &TimeWindow{Day: "Monday", Start: "02:00", End: "04:00"},
			},
		},
		{
			name: "duration pointer override",
			base: DeployConfig{
				RebootDelay: &Duration{Duration: 5 * time.Minute},
			},
			overlay: DeployConfig{
				RebootDelay: &Duration{Duration: 10 * time.Minute},
			},
			want: DeployConfig{
				RebootDelay: &Duration{Duration: 10 * time.Minute},
			},
		},
		{
			name: "empty overlay slice preserves base",
			base: DeployConfig{
				ExcludedPackages: []string{"pkg-a", "pkg-b"},
			},
			overlay: DeployConfig{
				ExcludedPackages: []string{},
			},
			want: DeployConfig{
				ExcludedPackages: []string{"pkg-a", "pkg-b"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := MergeConfig(tc.base, tc.overlay)
			assertDeployConfigEqual(t, tc.want, got)
		})
	}
}

// assertDeployConfigEqual compares two DeployConfig values field by field.
func assertDeployConfigEqual(t *testing.T, want, got DeployConfig) {
	t.Helper()

	assertPtrBool(t, "AutoReboot", want.AutoReboot, got.AutoReboot)
	assertPtrString(t, "WaveStrategy", want.WaveStrategy, got.WaveStrategy)
	assertPtrString(t, "PreScript", want.PreScript, got.PreScript)
	assertPtrString(t, "PostScript", want.PostScript, got.PostScript)
	assertPtrString(t, "BandwidthLimit", want.BandwidthLimit, got.BandwidthLimit)
	assertPtrInt(t, "MaxConcurrent", want.MaxConcurrent, got.MaxConcurrent)
	assertPtrBool(t, "NotifyUser", want.NotifyUser, got.NotifyUser)
	assertStringSlice(t, "ExcludedPackages", want.ExcludedPackages, got.ExcludedPackages)
	assertTimeWindow(t, want.MaintenanceWindow, got.MaintenanceWindow)
	assertDuration(t, "RebootDelay", want.RebootDelay, got.RebootDelay)
}

func assertPtrBool(t *testing.T, field string, want, got *bool) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if (want == nil) != (got == nil) {
		t.Errorf("%s: want nil=%v, got nil=%v", field, want == nil, got == nil)
		return
	}
	if *want != *got {
		t.Errorf("%s: want %v, got %v", field, *want, *got)
	}
}

func assertPtrString(t *testing.T, field string, want, got *string) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if (want == nil) != (got == nil) {
		t.Errorf("%s: want nil=%v, got nil=%v", field, want == nil, got == nil)
		return
	}
	if *want != *got {
		t.Errorf("%s: want %q, got %q", field, *want, *got)
	}
}

func assertPtrInt(t *testing.T, field string, want, got *int) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if (want == nil) != (got == nil) {
		t.Errorf("%s: want nil=%v, got nil=%v", field, want == nil, got == nil)
		return
	}
	if *want != *got {
		t.Errorf("%s: want %d, got %d", field, *want, *got)
	}
}

func assertStringSlice(t *testing.T, field string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s: want len=%d %v, got len=%d %v", field, len(want), want, len(got), got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d]: want %q, got %q", field, i, want[i], got[i])
		}
	}
}

func assertTimeWindow(t *testing.T, want, got *TimeWindow) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if (want == nil) != (got == nil) {
		t.Errorf("MaintenanceWindow: want nil=%v, got nil=%v", want == nil, got == nil)
		return
	}
	if want.Day != got.Day || want.Start != got.Start || want.End != got.End {
		t.Errorf("MaintenanceWindow: want %+v, got %+v", *want, *got)
	}
}

func assertDuration(t *testing.T, field string, want, got *Duration) {
	t.Helper()
	if want == nil && got == nil {
		return
	}
	if (want == nil) != (got == nil) {
		t.Errorf("%s: want nil=%v, got nil=%v", field, want == nil, got == nil)
		return
	}
	if want.Duration != got.Duration {
		t.Errorf("%s: want %v, got %v", field, want.Duration, got.Duration)
	}
}

// mockStore is a test double for ConfigStore.
type mockStore struct {
	overrides map[string]json.RawMessage // key: "scopeType:scopeID:module"
	// errOnKey returns an error for a specific key (non-ErrNoOverride).
	errOnKey map[string]error
}

func (m *mockStore) GetOverride(_ context.Context, _, scopeType, scopeID, module string) (json.RawMessage, error) {
	key := scopeType + ":" + scopeID + ":" + module
	if m.errOnKey != nil {
		if err, ok := m.errOnKey[key]; ok {
			return nil, err
		}
	}
	if v, ok := m.overrides[key]; ok {
		return v, nil
	}
	return nil, ErrNoOverride
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestResolveConfig(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name      string
		store     *mockStore
		params    ResolveParams
		wantSched string
		wantMax   int
		wantSrc   map[string]SourceLevel
		wantErr   bool
	}{
		{
			name:  "no overrides — returns system defaults",
			store: &mockStore{},
			params: ResolveParams{
				TenantID: "t1", TagIDs: []string{"g1"}, EndpointID: "e1", Module: "scan",
			},
			wantSched: "0 2 * * *",
			wantMax:   5,
			wantSrc: map[string]SourceLevel{
				"scan_schedule":  SourceSystem,
				"max_concurrent": SourceSystem,
			},
		},
		{
			name: "tenant override",
			store: &mockStore{
				overrides: map[string]json.RawMessage{
					"tenant:t1:scan": mustJSON(t, ScanConfig{Schedule: ptr("0 */6 * * *")}),
				},
			},
			params: ResolveParams{
				TenantID: "t1", TagIDs: []string{"g1"}, EndpointID: "e1", Module: "scan",
			},
			wantSched: "0 */6 * * *",
			wantMax:   5,
			wantSrc: map[string]SourceLevel{
				"scan_schedule":  SourceTenant,
				"max_concurrent": SourceSystem,
			},
		},
		{
			name: "group overrides tenant",
			store: &mockStore{
				overrides: map[string]json.RawMessage{
					"tenant:t1:scan": mustJSON(t, ScanConfig{Schedule: ptr("0 */6 * * *")}),
					"tag:g1:scan":  mustJSON(t, ScanConfig{Schedule: ptr("0 */2 * * *")}),
				},
			},
			params: ResolveParams{
				TenantID: "t1", TagIDs: []string{"g1"}, EndpointID: "e1", Module: "scan",
			},
			wantSched: "0 */2 * * *",
			wantMax:   5,
			wantSrc: map[string]SourceLevel{
				"scan_schedule":  SourceTag,
				"max_concurrent": SourceSystem,
			},
		},
		{
			name: "endpoint overrides all",
			store: &mockStore{
				overrides: map[string]json.RawMessage{
					"tenant:t1:scan":   mustJSON(t, ScanConfig{Schedule: ptr("0 */6 * * *")}),
					"tag:g1:scan":    mustJSON(t, ScanConfig{Schedule: ptr("0 */2 * * *")}),
					"endpoint:e1:scan": mustJSON(t, ScanConfig{Schedule: ptr("@hourly")}),
				},
			},
			params: ResolveParams{
				TenantID: "t1", TagIDs: []string{"g1"}, EndpointID: "e1", Module: "scan",
			},
			wantSched: "@hourly",
			wantMax:   5,
			wantSrc: map[string]SourceLevel{
				"scan_schedule":  SourceEndpoint,
				"max_concurrent": SourceSystem,
			},
		},
		{
			name: "partial overrides at different levels",
			store: &mockStore{
				overrides: map[string]json.RawMessage{
					"tenant:t1:scan": mustJSON(t, ScanConfig{Schedule: ptr("0 */6 * * *")}),
					"tag:g1:scan":  mustJSON(t, ScanConfig{MaxConcurrent: ptr(20)}),
				},
			},
			params: ResolveParams{
				TenantID: "t1", TagIDs: []string{"g1"}, EndpointID: "e1", Module: "scan",
			},
			wantSched: "0 */6 * * *",
			wantMax:   20,
			wantSrc: map[string]SourceLevel{
				"scan_schedule":  SourceTenant,
				"max_concurrent": SourceTag,
			},
		},
		{
			name: "empty group ID skips group level",
			store: &mockStore{
				overrides: map[string]json.RawMessage{
					"tenant:t1:scan": mustJSON(t, ScanConfig{Schedule: ptr("0 */6 * * *")}),
					// group override present but should not be queried because GroupID is empty
					"tag::scan": mustJSON(t, ScanConfig{Schedule: ptr("@hourly")}),
				},
			},
			params: ResolveParams{
				TenantID: "t1", TagIDs: nil, EndpointID: "e1", Module: "scan",
			},
			wantSched: "0 */6 * * *",
			wantMax:   5,
			wantSrc: map[string]SourceLevel{
				"scan_schedule":  SourceTenant,
				"max_concurrent": SourceSystem,
			},
		},
		{
			name: "store error propagates",
			store: &mockStore{
				errOnKey: map[string]error{
					"tenant:t1:scan": errors.New("db connection lost"),
				},
			},
			params: ResolveParams{
				TenantID: "t1", TagIDs: []string{"g1"}, EndpointID: "e1", Module: "scan",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := ResolveConfig(ctx, tc.store, tc.params, DefaultScanConfig())

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check effective schedule.
			if result.Effective.Schedule == nil {
				t.Fatal("Effective.Schedule is nil")
			}
			if *result.Effective.Schedule != tc.wantSched {
				t.Errorf("Effective.Schedule: want %q, got %q", tc.wantSched, *result.Effective.Schedule)
			}

			// Check effective max_concurrent.
			if result.Effective.MaxConcurrent == nil {
				t.Fatal("Effective.MaxConcurrent is nil")
			}
			if *result.Effective.MaxConcurrent != tc.wantMax {
				t.Errorf("Effective.MaxConcurrent: want %d, got %d", tc.wantMax, *result.Effective.MaxConcurrent)
			}

			// Check source attribution.
			for field, wantLevel := range tc.wantSrc {
				gotLevel, ok := result.Sources[field]
				if !ok {
					t.Errorf("Sources[%q]: missing, want %q", field, wantLevel)
					continue
				}
				if gotLevel != wantLevel {
					t.Errorf("Sources[%q]: want %q, got %q", field, wantLevel, gotLevel)
				}
			}
		})
	}
}

func TestResolveConfigIntegration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("full 4-level DeployConfig resolution", func(t *testing.T) {
		t.Parallel()

		// System defaults: autoReboot=true, rebootDelay=5m, maxConcurrent=10,
		// waveStrategy="sequential", notifyUser=true, bwLimit="50mbps",
		// maintenanceWindow={sunday,02:00,06:00}
		//
		// Tenant override: changes scan_schedule analogue — here changes waveStrategy.
		// Group override: changes maintenanceWindow.
		// Endpoint override: changes autoReboot to false.

		store := &mockStore{
			overrides: map[string]json.RawMessage{
				"tenant:tenant-abc:deploy": mustJSON(t, DeployConfig{
					WaveStrategy: ptr("parallel"),
				}),
				"tag:group-xyz:deploy": mustJSON(t, DeployConfig{
					MaintenanceWindow: &TimeWindow{Day: "saturday", Start: "03:00", End: "05:00"},
				}),
				"endpoint:ep-001:deploy": mustJSON(t, DeployConfig{
					AutoReboot: ptr(false),
				}),
			},
		}

		params := ResolveParams{
			TenantID:   "tenant-abc",
			TagIDs:     []string{"group-xyz"},
			EndpointID: "ep-001",
			Module:     "deploy",
		}

		result, err := ResolveConfig(ctx, store, params, DefaultDeployConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// --- Verify effective values ---

		// auto_reboot: endpoint overrode to false
		if result.Effective.AutoReboot == nil {
			t.Fatal("Effective.AutoReboot is nil")
		}
		if *result.Effective.AutoReboot != false {
			t.Errorf("Effective.AutoReboot: want false, got %v", *result.Effective.AutoReboot)
		}

		// reboot_delay: unchanged from system default (5m)
		if result.Effective.RebootDelay == nil {
			t.Fatal("Effective.RebootDelay is nil")
		}
		if result.Effective.RebootDelay.Duration != 5*time.Minute {
			t.Errorf("Effective.RebootDelay: want 5m, got %v", result.Effective.RebootDelay.Duration)
		}

		// max_concurrent_installs: unchanged from system default (10)
		if result.Effective.MaxConcurrent == nil {
			t.Fatal("Effective.MaxConcurrent is nil")
		}
		if *result.Effective.MaxConcurrent != 10 {
			t.Errorf("Effective.MaxConcurrent: want 10, got %d", *result.Effective.MaxConcurrent)
		}

		// wave_strategy: tenant overrode to "parallel"
		if result.Effective.WaveStrategy == nil {
			t.Fatal("Effective.WaveStrategy is nil")
		}
		if *result.Effective.WaveStrategy != "parallel" {
			t.Errorf("Effective.WaveStrategy: want %q, got %q", "parallel", *result.Effective.WaveStrategy)
		}

		// notify_user_before_reboot: unchanged from system default (true)
		if result.Effective.NotifyUser == nil {
			t.Fatal("Effective.NotifyUser is nil")
		}
		if *result.Effective.NotifyUser != true {
			t.Errorf("Effective.NotifyUser: want true, got %v", *result.Effective.NotifyUser)
		}

		// bandwidth_limit: unchanged from system default ("50mbps")
		if result.Effective.BandwidthLimit == nil {
			t.Fatal("Effective.BandwidthLimit is nil")
		}
		if *result.Effective.BandwidthLimit != "50mbps" {
			t.Errorf("Effective.BandwidthLimit: want %q, got %q", "50mbps", *result.Effective.BandwidthLimit)
		}

		// maintenance_window: group overrode to saturday 03:00-05:00
		if result.Effective.MaintenanceWindow == nil {
			t.Fatal("Effective.MaintenanceWindow is nil")
		}
		wantMW := &TimeWindow{Day: "saturday", Start: "03:00", End: "05:00"}
		if result.Effective.MaintenanceWindow.Day != wantMW.Day ||
			result.Effective.MaintenanceWindow.Start != wantMW.Start ||
			result.Effective.MaintenanceWindow.End != wantMW.End {
			t.Errorf("Effective.MaintenanceWindow: want %+v, got %+v", *wantMW, *result.Effective.MaintenanceWindow)
		}

		// --- Verify source attribution for every field ---

		wantSources := map[string]SourceLevel{
			"auto_reboot":               SourceEndpoint,
			"reboot_delay":              SourceSystem,
			"max_concurrent_installs":   SourceSystem,
			"wave_strategy":             SourceTenant,
			"notify_user_before_reboot": SourceSystem,
			"bandwidth_limit":           SourceSystem,
			"maintenance_window":        SourceTag,
		}
		for field, wantLevel := range wantSources {
			gotLevel, ok := result.Sources[field]
			if !ok {
				t.Errorf("Sources[%q]: missing, want %q", field, wantLevel)
				continue
			}
			if gotLevel != wantLevel {
				t.Errorf("Sources[%q]: want %q, got %q", field, wantLevel, gotLevel)
			}
		}
	})

	t.Run("Duration JSON round-trip", func(t *testing.T) {
		t.Parallel()

		original := Duration{Duration: 90 * time.Second}

		b, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var decoded Duration
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if original.Duration != decoded.Duration {
			t.Errorf("round-trip: want %v, got %v", original.Duration, decoded.Duration)
		}
	})

	t.Run("invalid JSON returns unmarshal error", func(t *testing.T) {
		t.Parallel()

		// scan_schedule is a string field; storing a number causes unmarshal failure.
		store := &mockStore{
			overrides: map[string]json.RawMessage{
				"tenant:t-bad:deploy": json.RawMessage(`{"auto_reboot": "not-a-bool"}`),
			},
		}

		params := ResolveParams{
			TenantID:   "t-bad",
			TagIDs:     nil,
			EndpointID: "",
			Module:     "deploy",
		}

		_, err := ResolveConfig(ctx, store, params, DefaultDeployConfig())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unmarshal") {
			t.Errorf("error should contain %q, got: %v", "unmarshal", err)
		}
	})

	t.Run("all empty IDs returns system defaults", func(t *testing.T) {
		t.Parallel()

		store := &mockStore{}

		params := ResolveParams{
			TenantID:   "",
			TagIDs:     nil,
			EndpointID: "",
			Module:     "deploy",
		}

		result, err := ResolveConfig(ctx, store, params, DefaultDeployConfig())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		defaults := DefaultDeployConfig()

		// Every value should match system defaults.
		assertDeployConfigEqual(t, defaults, result.Effective)

		// Every source should be "system".
		for field, gotLevel := range result.Sources {
			if gotLevel != SourceSystem {
				t.Errorf("Sources[%q]: want %q, got %q", field, SourceSystem, gotLevel)
			}
		}
	})
}
