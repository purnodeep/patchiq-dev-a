package compliance

import "encoding/json"

// ConditionDef describes one configurable condition within a check type.
type ConditionDef struct {
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"`    // "boolean", "threshold", "duration"
	Default     float64 `json:"default"` // default value (1=true for boolean)
	Unit        string  `json:"unit"`    // "hours", "days", "%", ""
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
}

// CheckTypeDef describes a check type and its available conditions.
type CheckTypeDef struct {
	Type                    string         `json:"type"`
	Name                    string         `json:"name"`
	Description             string         `json:"description"`
	Conditions              []ConditionDef `json:"conditions"`
	DefaultPassThreshold    float64        `json:"default_pass_threshold"`
	DefaultPartialThreshold float64        `json:"default_partial_threshold"`
}

// ConditionValue is a user's configuration for one condition.
type ConditionValue struct {
	Enabled bool    `json:"enabled"`
	Value   float64 `json:"value,omitempty"`
	IsSet   bool    `json:"is_set,omitempty"`
}

// CheckConfig is the parsed user configuration for a control's check type.
type CheckConfig struct {
	Conditions       map[string]ConditionValue `json:"conditions,omitempty"`
	PassThreshold    float64                   `json:"pass_threshold"`
	PartialThreshold float64                   `json:"partial_threshold"`
}

// IsEnabled returns true if the condition is enabled in the config.
// Returns true by default if the condition is not in the config (backward compat).
func (c CheckConfig) IsEnabled(key string) bool {
	if cv, ok := c.Conditions[key]; ok {
		return cv.Enabled
	}
	return true // default: enabled
}

// GetValue returns the configured value for a condition, or the fallback if not set.
func (c CheckConfig) GetValue(key string, fallback float64) float64 {
	if cv, ok := c.Conditions[key]; ok && cv.IsSet {
		return cv.Value
	}
	return fallback
}

// ParseCheckConfig parses JSONB bytes into a CheckConfig, applying defaults from the check type definition.
func ParseCheckConfig(checkType string, raw []byte) CheckConfig {
	def, ok := CheckTypeDefs[checkType]
	if !ok {
		return CheckConfig{PassThreshold: 95, PartialThreshold: 70}
	}

	// Start with defaults
	config := CheckConfig{
		Conditions:       make(map[string]ConditionValue),
		PassThreshold:    def.DefaultPassThreshold,
		PartialThreshold: def.DefaultPartialThreshold,
	}

	// Set default conditions from definition
	for _, cond := range def.Conditions {
		config.Conditions[cond.Key] = ConditionValue{
			Enabled: true,
			Value:   cond.Default,
			IsSet:   true,
		}
	}

	// userCheckConfig uses pointer types so we can distinguish "field absent"
	// from "field explicitly set to 0" during JSON unmarshalling.
	type userCheckConfig struct {
		Conditions       map[string]ConditionValue `json:"conditions,omitempty"`
		PassThreshold    *float64                  `json:"pass_threshold,omitempty"`
		PartialThreshold *float64                  `json:"partial_threshold,omitempty"`
	}

	// Overlay user config if provided
	if len(raw) > 0 && string(raw) != "{}" {
		var userConfig userCheckConfig
		if err := json.Unmarshal(raw, &userConfig); err == nil {
			if userConfig.PassThreshold != nil {
				config.PassThreshold = *userConfig.PassThreshold
			}
			if userConfig.PartialThreshold != nil {
				config.PartialThreshold = *userConfig.PartialThreshold
			}
			for k, v := range userConfig.Conditions {
				v.IsSet = true
				config.Conditions[k] = v
			}
		}
	}

	return config
}

// CheckTypeDefs is the registry of all available check types with their condition schemas.
var CheckTypeDefs = map[string]CheckTypeDef{
	"asset_inventory": {
		Type:        "asset_inventory",
		Name:        "Asset Inventory",
		Description: "Verify endpoints are enrolled with hardware data and recent heartbeat",
		Conditions: []ConditionDef{
			{Key: "endpoint_enrolled", Name: "Endpoint Enrolled", Description: "Endpoint is registered and active in the system", Type: "boolean", Default: 1},
			{Key: "has_hardware_data", Name: "Has Hardware Data", Description: "Endpoint has reported CPU, memory, and disk information", Type: "boolean", Default: 1},
			{Key: "heartbeat_freshness", Name: "Heartbeat Freshness", Description: "Agent has reported within this time window", Type: "duration", Default: 24, Unit: "hours", Min: 1, Max: 168},
		},
		DefaultPassThreshold:    95,
		DefaultPartialThreshold: 70,
	},
	"software_inventory": {
		Type:        "software_inventory",
		Name:        "Software Inventory",
		Description: "Verify endpoints have completed package scans within a configurable window",
		Conditions: []ConditionDef{
			{Key: "has_recent_scan", Name: "Recent Package Scan", Description: "Endpoint has completed a package inventory scan", Type: "boolean", Default: 1},
			{Key: "scan_max_age", Name: "Max Scan Age", Description: "Maximum age of the most recent package scan", Type: "duration", Default: 7, Unit: "days", Min: 1, Max: 90},
		},
		DefaultPassThreshold:    95,
		DefaultPartialThreshold: 70,
	},
	"vuln_scanning": {
		Type:        "vuln_scanning",
		Name:        "Vulnerability Scanning",
		Description: "Verify endpoints are covered by CVE vulnerability scanning",
		Conditions: []ConditionDef{
			{Key: "has_cve_data", Name: "CVE Scan Coverage", Description: "Endpoint has been evaluated for known CVE vulnerabilities", Type: "boolean", Default: 1},
		},
		DefaultPassThreshold:    95,
		DefaultPartialThreshold: 70,
	},
	"kev_compliance": {
		Type:        "kev_compliance",
		Name:        "CISA KEV Compliance",
		Description: "Verify no endpoints have CISA Known Exploited Vulnerabilities",
		Conditions: []ConditionDef{
			{Key: "zero_kev", Name: "Zero KEV Exposure", Description: "No CISA Known Exploited Vulnerabilities present on any endpoint", Type: "boolean", Default: 1},
		},
		DefaultPassThreshold:    100,
		DefaultPartialThreshold: 95,
	},
	"deployment_governance": {
		Type:        "deployment_governance",
		Name:        "Deployment Governance",
		Description: "Verify patch deployments have acceptable success rates",
		Conditions: []ConditionDef{
			{Key: "min_success_rate", Name: "Minimum Success Rate", Description: "Required deployment success percentage", Type: "threshold", Default: 80, Unit: "%", Min: 0, Max: 100},
			{Key: "lookback_days", Name: "Lookback Period", Description: "Days of deployment history to evaluate", Type: "duration", Default: 30, Unit: "days", Min: 1, Max: 365},
		},
		DefaultPassThreshold:    90,
		DefaultPartialThreshold: 70,
	},
	"agent_monitoring": {
		Type:        "agent_monitoring",
		Name:        "Agent Monitoring",
		Description: "Verify endpoints have recent agent heartbeats",
		Conditions: []ConditionDef{
			{Key: "heartbeat_freshness", Name: "Heartbeat Freshness", Description: "Agent must have reported within this time window", Type: "duration", Default: 24, Unit: "hours", Min: 1, Max: 168},
		},
		DefaultPassThreshold:    95,
		DefaultPartialThreshold: 80,
	},
	"critical_vuln_remediation": {
		Type:        "critical_vuln_remediation",
		Name:        "Critical Vulnerability Remediation",
		Description: "Verify no critical/high CVEs are unpatched beyond the allowed window",
		Conditions: []ConditionDef{
			{Key: "max_age_days", Name: "Max Unpatched Age", Description: "Maximum days a critical/high CVE can remain unpatched", Type: "duration", Default: 30, Unit: "days", Min: 1, Max: 365},
			{Key: "include_high", Name: "Include High Severity", Description: "Also check high-severity CVEs, not just critical", Type: "boolean", Default: 1},
		},
		DefaultPassThreshold:    100,
		DefaultPartialThreshold: 90,
	},
}
