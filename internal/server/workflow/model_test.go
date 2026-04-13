package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTriggerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  TriggerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid cron trigger",
			config:  TriggerConfig{TriggerType: "cron", CronExpression: "0 2 * * *"},
			wantErr: false,
		},
		{
			name:    "valid cve_severity trigger",
			config:  TriggerConfig{TriggerType: "cve_severity", SeverityThreshold: "critical"},
			wantErr: false,
		},
		{
			name:    "valid manual trigger",
			config:  TriggerConfig{TriggerType: "manual"},
			wantErr: false,
		},
		{
			name:    "valid policy_evaluation trigger",
			config:  TriggerConfig{TriggerType: "policy_evaluation"},
			wantErr: false,
		},
		{
			name:    "empty trigger_type",
			config:  TriggerConfig{TriggerType: ""},
			wantErr: true,
			errMsg:  "invalid trigger_type",
		},
		{
			name:    "invalid trigger_type",
			config:  TriggerConfig{TriggerType: "unknown"},
			wantErr: true,
			errMsg:  "invalid trigger_type",
		},
		{
			name:    "cron without expression",
			config:  TriggerConfig{TriggerType: "cron"},
			wantErr: true,
			errMsg:  "cron_expression required",
		},
		{
			name:    "cve_severity without threshold",
			config:  TriggerConfig{TriggerType: "cve_severity"},
			wantErr: true,
			errMsg:  "severity_threshold required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestFilterConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  FilterConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid with os_types",
			config:  FilterConfig{OSTypes: []string{"linux", "windows"}},
			wantErr: false,
		},
		{
			name:    "valid with tags",
			config:  FilterConfig{Tags: []string{"env=prod"}},
			wantErr: false,
		},
		{
			name:    "valid with min_severity",
			config:  FilterConfig{MinSeverity: "high"},
			wantErr: false,
		},
		{
			name:    "valid with package_regex",
			config:  FilterConfig{PackageRegex: "^openssl"},
			wantErr: false,
		},
		{
			name:    "empty filter",
			config:  FilterConfig{},
			wantErr: true,
			errMsg:  "at least one filter criterion",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestApprovalConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ApprovalConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid",
			config:  ApprovalConfig{ApproverRoles: []string{"admin"}, TimeoutHours: 24},
			wantErr: false,
		},
		{
			name:    "no approver_roles",
			config:  ApprovalConfig{TimeoutHours: 24},
			wantErr: true,
			errMsg:  "approver_roles is required",
		},
		{
			name:    "zero timeout",
			config:  ApprovalConfig{ApproverRoles: []string{"admin"}, TimeoutHours: 0},
			wantErr: true,
			errMsg:  "timeout_hours must be positive",
		},
		{
			name:    "negative timeout",
			config:  ApprovalConfig{ApproverRoles: []string{"admin"}, TimeoutHours: -1},
			wantErr: true,
			errMsg:  "timeout_hours must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestDeploymentWaveConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DeploymentWaveConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid 10 percent",
			config:  DeploymentWaveConfig{Percentage: 10},
			wantErr: false,
		},
		{
			name:    "valid 100 percent",
			config:  DeploymentWaveConfig{Percentage: 100},
			wantErr: false,
		},
		{
			name:    "valid 1 percent",
			config:  DeploymentWaveConfig{Percentage: 1},
			wantErr: false,
		},
		{
			name:    "zero percentage",
			config:  DeploymentWaveConfig{Percentage: 0},
			wantErr: true,
			errMsg:  "percentage must be between 1 and 100",
		},
		{
			name:    "over 100 percentage",
			config:  DeploymentWaveConfig{Percentage: 101},
			wantErr: true,
			errMsg:  "percentage must be between 1 and 100",
		},
		{
			name:    "negative percentage",
			config:  DeploymentWaveConfig{Percentage: -5},
			wantErr: true,
			errMsg:  "percentage must be between 1 and 100",
		},
		{
			name:    "negative success_threshold",
			config:  DeploymentWaveConfig{Percentage: 50, SuccessThreshold: -1},
			wantErr: true,
			errMsg:  "success_threshold must be between 0 and 100",
		},
		{
			name:    "success_threshold over 100",
			config:  DeploymentWaveConfig{Percentage: 50, SuccessThreshold: 101},
			wantErr: true,
			errMsg:  "success_threshold must be between 0 and 100",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestGateConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  GateConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid",
			config:  GateConfig{WaitMinutes: 15, FailureThreshold: 5},
			wantErr: false,
		},
		{
			name:    "zero wait_minutes",
			config:  GateConfig{WaitMinutes: 0},
			wantErr: true,
			errMsg:  "wait_minutes must be positive",
		},
		{
			name:    "negative wait_minutes",
			config:  GateConfig{WaitMinutes: -1},
			wantErr: true,
			errMsg:  "wait_minutes must be positive",
		},
		{
			name:    "negative failure_threshold",
			config:  GateConfig{WaitMinutes: 10, FailureThreshold: -1},
			wantErr: true,
			errMsg:  "failure_threshold must be non-negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestScriptConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ScriptConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid shell script",
			config:  ScriptConfig{ScriptBody: "echo hello", ScriptType: "shell", TimeoutMinutes: 5, FailureBehavior: "continue"},
			wantErr: false,
		},
		{
			name:    "valid powershell script",
			config:  ScriptConfig{ScriptBody: "Write-Host hello", ScriptType: "powershell", TimeoutMinutes: 5, FailureBehavior: "halt"},
			wantErr: false,
		},
		{
			name:    "empty script_body",
			config:  ScriptConfig{ScriptType: "shell", TimeoutMinutes: 5, FailureBehavior: "continue"},
			wantErr: true,
			errMsg:  "script_body is required",
		},
		{
			name:    "zero timeout_minutes",
			config:  ScriptConfig{ScriptBody: "echo", ScriptType: "shell", TimeoutMinutes: 0, FailureBehavior: "continue"},
			wantErr: true,
			errMsg:  "timeout_minutes must be positive",
		},
		{
			name:    "negative timeout_minutes",
			config:  ScriptConfig{ScriptBody: "echo", ScriptType: "shell", TimeoutMinutes: -1, FailureBehavior: "continue"},
			wantErr: true,
			errMsg:  "timeout_minutes must be positive",
		},
		{
			name:    "invalid script_type",
			config:  ScriptConfig{ScriptBody: "echo", ScriptType: "python", TimeoutMinutes: 5, FailureBehavior: "continue"},
			wantErr: true,
			errMsg:  "invalid script_type",
		},
		{
			name:    "invalid failure_behavior",
			config:  ScriptConfig{ScriptBody: "echo", ScriptType: "shell", TimeoutMinutes: 5, FailureBehavior: "retry"},
			wantErr: true,
			errMsg:  "invalid failure_behavior",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestNotificationConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  NotificationConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid email",
			config:  NotificationConfig{Channel: "email", Target: "admin@example.com"},
			wantErr: false,
		},
		{
			name:    "valid slack",
			config:  NotificationConfig{Channel: "slack", Target: "#ops"},
			wantErr: false,
		},
		{
			name:    "valid webhook",
			config:  NotificationConfig{Channel: "webhook", Target: "https://example.com/hook"},
			wantErr: false,
		},
		{
			name:    "valid pagerduty",
			config:  NotificationConfig{Channel: "pagerduty", Target: "service-key"},
			wantErr: false,
		},
		{
			name:    "invalid channel",
			config:  NotificationConfig{Channel: "sms", Target: "+1234567890"},
			wantErr: true,
			errMsg:  "invalid channel",
		},
		{
			name:    "empty target",
			config:  NotificationConfig{Channel: "email", Target: ""},
			wantErr: true,
			errMsg:  "target is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestRollbackConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RollbackConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid snapshot_restore",
			config:  RollbackConfig{Strategy: "snapshot_restore", FailureThreshold: 10},
			wantErr: false,
		},
		{
			name:    "valid package_downgrade",
			config:  RollbackConfig{Strategy: "package_downgrade", FailureThreshold: 5},
			wantErr: false,
		},
		{
			name:    "valid script strategy",
			config:  RollbackConfig{Strategy: "script", RollbackScript: "apt-get rollback"},
			wantErr: false,
		},
		{
			name:    "script without rollback_script",
			config:  RollbackConfig{Strategy: "script"},
			wantErr: true,
			errMsg:  "rollback_script required for script strategy",
		},
		{
			name:    "invalid strategy",
			config:  RollbackConfig{Strategy: "magic"},
			wantErr: true,
			errMsg:  "invalid strategy",
		},
		{
			name:    "negative failure_threshold",
			config:  RollbackConfig{Strategy: "snapshot_restore", FailureThreshold: -1},
			wantErr: true,
			errMsg:  "failure_threshold must be non-negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestDecisionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  DecisionConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid equals",
			config:  DecisionConfig{Field: "os_type", Operator: "equals", Value: "linux"},
			wantErr: false,
		},
		{
			name:    "valid not_equals",
			config:  DecisionConfig{Field: "severity", Operator: "not_equals", Value: "low"},
			wantErr: false,
		},
		{
			name:    "valid in",
			config:  DecisionConfig{Field: "group", Operator: "in", Value: "prod,staging"},
			wantErr: false,
		},
		{
			name:    "valid gt",
			config:  DecisionConfig{Field: "count", Operator: "gt", Value: "10"},
			wantErr: false,
		},
		{
			name:    "valid lt",
			config:  DecisionConfig{Field: "count", Operator: "lt", Value: "5"},
			wantErr: false,
		},
		{
			name:    "empty field",
			config:  DecisionConfig{Operator: "equals", Value: "linux"},
			wantErr: true,
			errMsg:  "field is required",
		},
		{
			name:    "invalid operator",
			config:  DecisionConfig{Field: "os_type", Operator: "like", Value: "linux"},
			wantErr: true,
			errMsg:  "invalid operator",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestCompleteConfig_Validate(t *testing.T) {
	tests := []struct {
		name   string
		config CompleteConfig
	}{
		{
			name:   "empty config",
			config: CompleteConfig{},
		},
		{
			name:   "with generate_report",
			config: CompleteConfig{GenerateReport: true},
		},
		{
			name:   "with notify_on_complete",
			config: CompleteConfig{NotifyOnComplete: true},
		},
		{
			name:   "with both flags",
			config: CompleteConfig{GenerateReport: true, NotifyOnComplete: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestValidateNodeConfig(t *testing.T) {
	tests := []struct {
		name     string
		nodeType NodeType
		config   string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid trigger config",
			nodeType: NodeTrigger,
			config:   `{"trigger_type":"manual"}`,
			wantErr:  false,
		},
		{
			name:     "invalid trigger config",
			nodeType: NodeTrigger,
			config:   `{"trigger_type":""}`,
			wantErr:  true,
			errMsg:   "invalid trigger_type",
		},
		{
			name:     "valid filter config",
			nodeType: NodeFilter,
			config:   `{"os_types":["linux"]}`,
			wantErr:  false,
		},
		{
			name:     "valid approval config",
			nodeType: NodeApproval,
			config:   `{"approver_roles":["admin"],"timeout_hours":24}`,
			wantErr:  false,
		},
		{
			name:     "valid deployment_wave config",
			nodeType: NodeDeploymentWave,
			config:   `{"percentage":50}`,
			wantErr:  false,
		},
		{
			name:     "valid gate config",
			nodeType: NodeGate,
			config:   `{"wait_minutes":10,"failure_threshold":5}`,
			wantErr:  false,
		},
		{
			name:     "valid script config",
			nodeType: NodeScript,
			config:   `{"script_body":"echo hi","script_type":"shell","timeout_minutes":5,"failure_behavior":"halt"}`,
			wantErr:  false,
		},
		{
			name:     "valid notification config",
			nodeType: NodeNotification,
			config:   `{"channel":"email","target":"admin@example.com"}`,
			wantErr:  false,
		},
		{
			name:     "valid rollback config",
			nodeType: NodeRollback,
			config:   `{"strategy":"snapshot_restore","failure_threshold":10}`,
			wantErr:  false,
		},
		{
			name:     "valid decision config",
			nodeType: NodeDecision,
			config:   `{"field":"os_type","operator":"equals","value":"linux"}`,
			wantErr:  false,
		},
		{
			name:     "valid complete config",
			nodeType: NodeComplete,
			config:   `{"generate_report":true}`,
			wantErr:  false,
		},
		{
			name:     "invalid json",
			nodeType: NodeTrigger,
			config:   `{not json}`,
			wantErr:  true,
			errMsg:   "unmarshal trigger config",
		},
		{
			name:     "nil config",
			nodeType: NodeTrigger,
			config:   "",
			wantErr:  true,
			errMsg:   "config is required",
		},
		{
			name:     "unknown node type",
			nodeType: NodeType("unknown"),
			config:   `{}`,
			wantErr:  true,
			errMsg:   "unknown node type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNodeConfig(tt.nodeType, json.RawMessage(tt.config))
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNodeConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !strings.Contains(got, tt.errMsg) {
					t.Errorf("error message %q does not contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestConfigRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		nodeType NodeType
		config   string
	}{
		{"trigger", NodeTrigger, `{"trigger_type":"cron","cron_expression":"0 2 * * *"}`},
		{"filter", NodeFilter, `{"os_types":["linux","windows"]}`},
		{"approval", NodeApproval, `{"approver_roles":["admin","manager"],"timeout_hours":48}`},
		{"deployment_wave", NodeDeploymentWave, `{"percentage":25,"max_parallel":10}`},
		{"gate", NodeGate, `{"wait_minutes":30,"failure_threshold":5,"health_check":true}`},
		{"script", NodeScript, `{"script_body":"echo ok","script_type":"shell","timeout_minutes":5,"failure_behavior":"continue"}`},
		{"notification", NodeNotification, `{"channel":"slack","target":"#ops","message_template":"Deploy done"}`},
		{"rollback", NodeRollback, `{"strategy":"snapshot_restore","failure_threshold":10}`},
		{"decision", NodeDecision, `{"field":"severity","operator":"gt","value":"7"}`},
		{"complete", NodeComplete, `{"generate_report":true,"notify_on_complete":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := json.RawMessage(tt.config)

			// Validate should pass
			if err := ValidateNodeConfig(tt.nodeType, raw); err != nil {
				t.Fatalf("ValidateNodeConfig() unexpected error: %v", err)
			}

			// Round-trip through Node struct
			node := Node{
				ID:       "test-node",
				NodeType: tt.nodeType,
				Label:    "Test",
				Config:   raw,
			}
			data, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("json.Marshal(Node) error: %v", err)
			}
			var decoded Node
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal(Node) error: %v", err)
			}
			if decoded.NodeType != tt.nodeType {
				t.Errorf("NodeType = %q, want %q", decoded.NodeType, tt.nodeType)
			}
			// Validate the round-tripped config
			if err := ValidateNodeConfig(decoded.NodeType, decoded.Config); err != nil {
				t.Errorf("ValidateNodeConfig after round-trip error: %v", err)
			}
		})
	}
}
