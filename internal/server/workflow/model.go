package workflow

import (
	"encoding/json"
	"fmt"
)

// NodeType identifies the kind of workflow node.
type NodeType string

const (
	NodeTrigger         NodeType = "trigger"
	NodeFilter          NodeType = "filter"
	NodeApproval        NodeType = "approval"
	NodeDeploymentWave  NodeType = "deployment_wave"
	NodeGate            NodeType = "gate"
	NodeScript          NodeType = "script"
	NodeNotification    NodeType = "notification"
	NodeRollback        NodeType = "rollback"
	NodeDecision        NodeType = "decision"
	NodeComplete        NodeType = "complete"
	NodeReboot          NodeType = "reboot"
	NodeScan            NodeType = "scan"
	NodeTagGate         NodeType = "tag_gate"
	NodeComplianceCheck NodeType = "compliance_check"
)

// validNodeTypes is the set of known node types.
var validNodeTypes = map[NodeType]struct{}{
	NodeTrigger: {}, NodeFilter: {}, NodeApproval: {}, NodeDeploymentWave: {},
	NodeGate: {}, NodeScript: {}, NodeNotification: {}, NodeRollback: {},
	NodeDecision: {}, NodeComplete: {},
	NodeReboot: {}, NodeScan: {}, NodeTagGate: {}, NodeComplianceCheck: {},
}

// IsValid reports whether nt is a known node type.
func (nt NodeType) IsValid() bool {
	_, ok := validNodeTypes[nt]
	return ok
}

// VersionStatus represents the lifecycle state of a workflow version.
type VersionStatus string

const (
	StatusDraft     VersionStatus = "draft"
	StatusPublished VersionStatus = "published"
	StatusArchived  VersionStatus = "archived"
)

// validVersionStatuses is the set of known version statuses.
var validVersionStatuses = map[VersionStatus]struct{}{
	StatusDraft: {}, StatusPublished: {}, StatusArchived: {},
}

// IsValid reports whether vs is a known version status.
func (vs VersionStatus) IsValid() bool {
	_, ok := validVersionStatuses[vs]
	return ok
}

// Node represents a workflow node for validation and template purposes.
type Node struct {
	ID        string          `json:"id"`
	NodeType  NodeType        `json:"node_type"`
	Label     string          `json:"label"`
	PositionX float64         `json:"position_x"`
	PositionY float64         `json:"position_y"`
	Config    json.RawMessage `json:"config"`
}

// Edge represents a workflow edge for validation and template purposes.
type Edge struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"source_node_id"`
	TargetNodeID string `json:"target_node_id"`
	Label        string `json:"label"`
}

// Config structs for each node type.

// TriggerConfig defines the configuration for a trigger node.
type TriggerConfig struct {
	TriggerType       string `json:"trigger_type"`
	CronExpression    string `json:"cron_expression,omitempty"`
	SeverityThreshold string `json:"severity_threshold,omitempty"`
}

// Validate checks that TriggerConfig has all required fields for its trigger type.
func (c TriggerConfig) Validate() error {
	switch c.TriggerType {
	case "cron":
		if c.CronExpression == "" {
			return fmt.Errorf("trigger config: cron_expression required for cron trigger")
		}
	case "cve_severity":
		if c.SeverityThreshold == "" {
			return fmt.Errorf("trigger config: severity_threshold required for cve_severity trigger")
		}
	case "manual", "policy_evaluation":
		// no additional fields required
	default:
		return fmt.Errorf("trigger config: invalid trigger_type %q", c.TriggerType)
	}
	return nil
}

// FilterConfig defines the configuration for a filter node. As of the
// tags-replace-groups migration, GroupIDs has been removed and Tags is
// now []string of "key=value" pairs (future: migrate to a structured
// *targeting.Selector for consistency with policies).
type FilterConfig struct {
	OSTypes      []string `json:"os_types,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	MinSeverity  string   `json:"min_severity,omitempty"`
	PackageRegex string   `json:"package_regex,omitempty"`
}

// Validate checks that FilterConfig has at least one filter criterion
// and that every Tags entry is a well-formed "key=value" pair with a
// non-empty key and non-empty value. Malformed entries are rejected at
// save time rather than silently skipped at execution — a filter that
// evaluates to "no tag predicate" would return the entire tenant, which
// is the same blast-radius footgun the targeting DSL prevents for
// policies (see internal/server/targeting/resolver.go).
func (c FilterConfig) Validate() error {
	if len(c.OSTypes) == 0 && len(c.Tags) == 0 && c.MinSeverity == "" && c.PackageRegex == "" {
		return fmt.Errorf("filter config: at least one filter criterion is required")
	}
	for i, kv := range c.Tags {
		eq := indexByte(kv, '=')
		if eq <= 0 || eq == len(kv)-1 {
			return fmt.Errorf("filter config: tags[%d] must be non-empty \"key=value\" (got %q)", i, kv)
		}
	}
	return nil
}

// indexByte avoids importing strings for a single call.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ApprovalConfig defines the configuration for an approval node.
type ApprovalConfig struct {
	ApproverRoles  []string `json:"approver_roles"`
	TimeoutHours   int      `json:"timeout_hours"`
	EscalationRole string   `json:"escalation_role,omitempty"`
	TimeoutAction  string   `json:"timeout_action,omitempty"`
}

// Validate checks that ApprovalConfig has required fields.
func (c ApprovalConfig) Validate() error {
	if len(c.ApproverRoles) == 0 {
		return fmt.Errorf("approval config: approver_roles is required")
	}
	if c.TimeoutHours <= 0 {
		return fmt.Errorf("approval config: timeout_hours must be positive")
	}
	if c.TimeoutAction != "" && c.TimeoutAction != "reject" && c.TimeoutAction != "escalate" {
		return fmt.Errorf("approval config: invalid timeout_action %q, must be reject or escalate", c.TimeoutAction)
	}
	return nil
}

// DeploymentWaveConfig defines the configuration for a deployment wave node.
type DeploymentWaveConfig struct {
	Percentage       int `json:"percentage"`
	MaxParallel      int `json:"max_parallel,omitempty"`
	TimeoutMinutes   int `json:"timeout_minutes,omitempty"`
	SuccessThreshold int `json:"success_threshold,omitempty"`
}

// Validate checks that DeploymentWaveConfig has a valid percentage and success threshold.
func (c DeploymentWaveConfig) Validate() error {
	if c.Percentage < 1 || c.Percentage > 100 {
		return fmt.Errorf("deployment_wave config: percentage must be between 1 and 100")
	}
	if c.SuccessThreshold < 0 || c.SuccessThreshold > 100 {
		return fmt.Errorf("deployment_wave config: success_threshold must be between 0 and 100")
	}
	return nil
}

// GateConfig defines the configuration for a gate node.
type GateConfig struct {
	WaitMinutes      int  `json:"wait_minutes"`
	FailureThreshold int  `json:"failure_threshold"`
	HealthCheck      bool `json:"health_check,omitempty"`
}

// Validate checks that GateConfig has a positive wait time and non-negative failure threshold.
func (c GateConfig) Validate() error {
	if c.WaitMinutes <= 0 {
		return fmt.Errorf("gate config: wait_minutes must be positive")
	}
	if c.FailureThreshold < 0 {
		return fmt.Errorf("gate config: failure_threshold must be non-negative")
	}
	return nil
}

// ScriptConfig defines the configuration for a script node.
type ScriptConfig struct {
	ScriptBody      string `json:"script_body"`
	ScriptType      string `json:"script_type"`
	TimeoutMinutes  int    `json:"timeout_minutes"`
	FailureBehavior string `json:"failure_behavior"`
}

// Validate checks that ScriptConfig has all required fields and valid enum values.
func (c ScriptConfig) Validate() error {
	if c.ScriptBody == "" {
		return fmt.Errorf("script config: script_body is required")
	}
	if c.TimeoutMinutes <= 0 {
		return fmt.Errorf("script config: timeout_minutes must be positive")
	}
	switch c.ScriptType {
	case "shell", "powershell":
	default:
		return fmt.Errorf("script config: invalid script_type %q, must be shell or powershell", c.ScriptType)
	}
	switch c.FailureBehavior {
	case "continue", "halt":
	default:
		return fmt.Errorf("script config: invalid failure_behavior %q, must be continue or halt", c.FailureBehavior)
	}
	return nil
}

// NotificationConfig defines the configuration for a notification node.
type NotificationConfig struct {
	Channel         string `json:"channel"`
	Target          string `json:"target"`
	MessageTemplate string `json:"message_template,omitempty"`
}

// Validate checks that NotificationConfig has a valid channel and non-empty target.
func (c NotificationConfig) Validate() error {
	switch c.Channel {
	case "email", "slack", "webhook", "pagerduty":
	default:
		return fmt.Errorf("notification config: invalid channel %q", c.Channel)
	}
	if c.Target == "" {
		return fmt.Errorf("notification config: target is required")
	}
	return nil
}

// RollbackConfig defines the configuration for a rollback node.
type RollbackConfig struct {
	Strategy         string `json:"strategy"`
	FailureThreshold int    `json:"failure_threshold"`
	RollbackScript   string `json:"rollback_script,omitempty"`
}

// Validate checks that RollbackConfig has a valid strategy and, for the script strategy,
// a non-empty rollback_script.
func (c RollbackConfig) Validate() error {
	switch c.Strategy {
	case "snapshot_restore", "package_downgrade":
	case "script":
		if c.RollbackScript == "" {
			return fmt.Errorf("rollback config: rollback_script required for script strategy")
		}
	default:
		return fmt.Errorf("rollback config: invalid strategy %q", c.Strategy)
	}
	if c.FailureThreshold < 0 {
		return fmt.Errorf("rollback config: failure_threshold must be non-negative")
	}
	return nil
}

// DecisionConfig defines the configuration for a decision node.
type DecisionConfig struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// Validate checks that DecisionConfig has a non-empty field and valid operator.
func (c DecisionConfig) Validate() error {
	if c.Field == "" {
		return fmt.Errorf("decision config: field is required")
	}
	switch c.Operator {
	case "equals", "not_equals", "in", "gt", "lt":
	default:
		return fmt.Errorf("decision config: invalid operator %q", c.Operator)
	}
	return nil
}

// CompleteConfig defines the configuration for a complete node.
type CompleteConfig struct {
	GenerateReport   bool `json:"generate_report,omitempty"`
	NotifyOnComplete bool `json:"notify_on_complete,omitempty"`
}

// Validate always returns nil for CompleteConfig as no fields are required.
func (c CompleteConfig) Validate() error {
	return nil
}

// RebootConfig defines the configuration for a reboot node.
type RebootConfig struct {
	RebootMode         string `json:"reboot_mode"`
	GracePeriodSeconds int    `json:"grace_period_seconds"`
}

// Validate checks that RebootConfig has a valid reboot mode and non-negative grace period.
func (c RebootConfig) Validate() error {
	switch c.RebootMode {
	case "immediate", "graceful", "scheduled":
	default:
		return fmt.Errorf("reboot config: invalid reboot_mode %q, must be immediate, graceful, or scheduled", c.RebootMode)
	}
	if c.GracePeriodSeconds < 0 {
		return fmt.Errorf("reboot config: grace_period_seconds must be non-negative")
	}
	return nil
}

// ScanConfig defines the configuration for a scan node.
type ScanConfig struct {
	ScanType string `json:"scan_type"`
}

// Validate checks that ScanConfig has a valid scan type.
func (c ScanConfig) Validate() error {
	switch c.ScanType {
	case "inventory", "compliance", "vulnerability":
	default:
		return fmt.Errorf("scan config: invalid scan_type %q, must be inventory, compliance, or vulnerability", c.ScanType)
	}
	return nil
}

// TagGateConfig defines the configuration for a tag gate node.
type TagGateConfig struct {
	TagExpression string `json:"tag_expression"`
}

// Validate checks that TagGateConfig has a non-empty tag expression.
func (c TagGateConfig) Validate() error {
	if c.TagExpression == "" {
		return fmt.Errorf("tag_gate config: tag_expression is required")
	}
	return nil
}

// ComplianceCheckConfig defines the configuration for a compliance check node.
type ComplianceCheckConfig struct {
	Framework string `json:"framework,omitempty"`
}

// Validate always returns nil for ComplianceCheckConfig (M2 stub).
func (c ComplianceCheckConfig) Validate() error {
	return nil
}

// ValidateNodeConfig unmarshals config JSON into the appropriate typed struct and validates it.
// Returns an error for unknown node types, missing/empty config, or invalid JSON.
func ValidateNodeConfig(nodeType NodeType, config json.RawMessage) error {
	if len(config) == 0 {
		return fmt.Errorf("config is required for node type %s", nodeType)
	}
	var validator interface{ Validate() error }
	switch nodeType {
	case NodeTrigger:
		validator = &TriggerConfig{}
	case NodeFilter:
		validator = &FilterConfig{}
	case NodeApproval:
		validator = &ApprovalConfig{}
	case NodeDeploymentWave:
		validator = &DeploymentWaveConfig{}
	case NodeGate:
		validator = &GateConfig{}
	case NodeScript:
		validator = &ScriptConfig{}
	case NodeNotification:
		validator = &NotificationConfig{}
	case NodeRollback:
		validator = &RollbackConfig{}
	case NodeDecision:
		validator = &DecisionConfig{}
	case NodeComplete:
		validator = &CompleteConfig{}
	case NodeReboot:
		validator = &RebootConfig{}
	case NodeScan:
		validator = &ScanConfig{}
	case NodeTagGate:
		validator = &TagGateConfig{}
	case NodeComplianceCheck:
		validator = &ComplianceCheckConfig{}
	default:
		return fmt.Errorf("unknown node type %q", nodeType)
	}
	if err := json.Unmarshal(config, validator); err != nil {
		return fmt.Errorf("unmarshal %s config: %w", nodeType, err)
	}
	return validator.Validate()
}
