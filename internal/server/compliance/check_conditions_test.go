package compliance

import (
	"encoding/json"
	"testing"
)

func TestGetValue(t *testing.T) {
	tests := []struct {
		name       string
		conditions map[string]ConditionValue
		key        string
		fallback   float64
		want       float64
	}{
		{
			name:       "returns zero when IsSet is true",
			conditions: map[string]ConditionValue{"rate": {Enabled: true, Value: 0, IsSet: true}},
			key:        "rate",
			fallback:   80,
			want:       0,
		},
		{
			name:       "returns fallback when key not present",
			conditions: map[string]ConditionValue{},
			key:        "missing",
			fallback:   42,
			want:       42,
		},
		{
			name:       "returns fallback when IsSet is false",
			conditions: map[string]ConditionValue{"key": {Enabled: true, Value: 0, IsSet: false}},
			key:        "key",
			fallback:   99,
			want:       99,
		},
		{
			name:       "returns non-zero value when IsSet is true",
			conditions: map[string]ConditionValue{"key": {Enabled: true, Value: 50, IsSet: true}},
			key:        "key",
			fallback:   99,
			want:       50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CheckConfig{Conditions: tt.conditions}
			got := cfg.GetValue(tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("GetValue(%q, %v) = %v, want %v", tt.key, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestParseCheckConfig(t *testing.T) {
	tests := []struct {
		name      string
		checkType string
		raw       any
		check     func(t *testing.T, cfg CheckConfig)
	}{
		{
			name:      "pass threshold zero is respected",
			checkType: "asset_inventory",
			raw:       map[string]any{"pass_threshold": 0},
			check: func(t *testing.T, cfg CheckConfig) {
				if cfg.PassThreshold != 0 {
					t.Errorf("PassThreshold = %v, want 0", cfg.PassThreshold)
				}
			},
		},
		{
			name:      "missing pass threshold uses default",
			checkType: "asset_inventory",
			raw:       map[string]any{"conditions": map[string]any{}},
			check: func(t *testing.T, cfg CheckConfig) {
				if cfg.PassThreshold != 95 {
					t.Errorf("PassThreshold = %v, want 95 (default)", cfg.PassThreshold)
				}
			},
		},
		{
			name:      "user overlay sets IsSet true",
			checkType: "asset_inventory",
			raw: map[string]any{
				"conditions": map[string]any{
					"heartbeat_freshness": map[string]any{"enabled": true, "value": 0},
				},
			},
			check: func(t *testing.T, cfg CheckConfig) {
				cv, ok := cfg.Conditions["heartbeat_freshness"]
				if !ok {
					t.Fatal("condition heartbeat_freshness not found")
				}
				if !cv.IsSet {
					t.Error("IsSet should be true for user-provided condition")
				}
				if cv.Value != 0 {
					t.Errorf("Value = %v, want 0", cv.Value)
				}
			},
		},
		{
			name:      "default conditions have IsSet true",
			checkType: "asset_inventory",
			raw:       nil,
			check: func(t *testing.T, cfg CheckConfig) {
				for k, cv := range cfg.Conditions {
					if !cv.IsSet {
						t.Errorf("default condition %q should have IsSet=true", k)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			if tt.raw != nil {
				raw, _ = json.Marshal(tt.raw)
			}
			cfg := ParseCheckConfig(tt.checkType, raw)
			tt.check(t, cfg)
		})
	}
}
