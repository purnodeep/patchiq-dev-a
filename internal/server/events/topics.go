package events

import "sync"

// Standard event types. New types are added as features are built.
// Format: resource.action
const (
	EndpointCreated       = "endpoint.created"
	EndpointUpdated       = "endpoint.updated"
	EndpointDeleted       = "endpoint.deleted"
	EndpointEnrolled      = "endpoint.enrolled"
	EndpointScanRequested = "endpoint.scan_requested"
	EndpointConfigPushed  = "endpoint.config_pushed"

	HeartbeatReceived = "heartbeat.received"

	InventoryReceived      = "inventory.received"
	InventoryScanCompleted = "inventory.scan_completed"
	CommandResultReceived  = "command.result.received"

	TagCreated       = "tag.created"
	TagUpdated       = "tag.updated"
	TagDeleted       = "tag.deleted"
	EndpointTagged   = "endpoint.tagged"
	EndpointUntagged = "endpoint.untagged"

	TagKeyUpserted = "tag_key.upserted"
	TagKeyDeleted  = "tag_key.deleted"

	PolicyTargetSelectorUpdated = "policy.target_selector.updated"

	TagRuleCreated = "tag_rule.created"
	TagRuleUpdated = "tag_rule.updated"
	TagRuleDeleted = "tag_rule.deleted"

	PolicyCreated            = "policy.created"
	PolicyUpdated            = "policy.updated"
	PolicyDeleted            = "policy.deleted"
	PolicyEvaluated          = "policy.evaluated"
	PolicyEvaluationRecorded = "policy.evaluation_recorded"
	PolicyAutoDeployed       = "policy.auto_deployed"

	DeploymentStarted           = "deployment.started"
	DeploymentCompleted         = "deployment.completed"
	DeploymentCreated           = "deployment.created"
	DeploymentFailed            = "deployment.failed"
	DeploymentCancelled         = "deployment.cancelled"
	DeploymentEndpointCompleted = "deployment.endpoint_completed"
	DeploymentTargetSent        = "deployment_target.sent"
	CommandDispatched           = "command.dispatched"
	CommandTimedOut             = "command.timed_out"
	DeploymentTargetTimedOut    = "deployment_target.timed_out"
	ScanTriggered               = "scan.triggered"
	ScanDispatched              = "scan.dispatched"
	ScanCompleted               = "scan.completed"
	DeploymentWaveStarted       = "deployment.wave_started"
	DeploymentWaveCompleted     = "deployment.wave_completed"
	DeploymentWaveFailed        = "deployment.wave_failed"
	DeploymentRollbackTriggered = "deployment.rollback_triggered"
	DeploymentRolledBack        = "deployment.rolled_back"
	DeploymentRollbackFailed    = "deployment.rollback_failed"
	DeploymentRetryTriggered    = "deployment.retry_triggered"

	ScheduleCreated = "schedule.created"
	ScheduleUpdated = "schedule.updated"
	ScheduleDeleted = "schedule.deleted"

	PatchDiscovered  = "patch.discovered"
	RepositorySynced = "repository.synced"

	CatalogSynced      = "catalog.synced"
	CatalogSyncStarted = "catalog.sync_started"
	CatalogSyncFailed  = "catalog.sync_failed"

	HubSyncConfigUpdated = "hub_sync.config_updated"
	HubSyncTriggered     = "hub_sync.triggered"

	CVEDiscovered           = "cve.discovered"
	CVELinkedToEndpoint     = "cve.linked_to_endpoint"
	CVERemediationAvailable = "cve.remediation_available"

	RoleCreated      = "role.created"
	RoleUpdated      = "role.updated"
	RoleDeleted      = "role.deleted"
	UserRoleAssigned = "user_role.assigned"
	UserRoleRevoked  = "user_role.revoked"

	// Compliance framework events
	ComplianceFrameworkEnabled    = "compliance.framework_enabled"
	ComplianceFrameworkUpdated    = "compliance.framework_updated"
	ComplianceFrameworkDisabled   = "compliance.framework_disabled"
	ComplianceEvaluationCompleted = "compliance.evaluation_completed"

	// Custom compliance framework events
	CustomComplianceFrameworkCreated = "compliance.custom_framework_created"
	CustomComplianceFrameworkUpdated = "compliance.custom_framework_updated"
	CustomComplianceFrameworkDeleted = "compliance.custom_framework_deleted"
	CustomComplianceControlsUpdated  = "compliance.custom_controls_updated"

	// Notification channel events
	ChannelCreated                 = "channel.created"
	ChannelUpdated                 = "channel.updated"
	ChannelDeleted                 = "channel.deleted"
	ChannelTested                  = "channel.tested"
	DigestConfigUpdated            = "notification.digest_config.updated"
	NotificationPreferencesUpdated = "notification.preferences_updated"

	// Notification events
	ComplianceThresholdBreach = "compliance.threshold_breach"
	AgentDisconnected         = "agent.disconnected"
	NotificationSent          = "notification.sent"
	NotificationFailed        = "notification.failed"

	LicenseLoaded             = "license.loaded"
	LicenseExpiring           = "license.expiring"
	LicenseExpired            = "license.expired"
	LicenseGracePeriodEntered = "license.grace_period_entered"

	AuditExportTriggered         = "audit.export_triggered"
	AuditRetentionPurgeCompleted = "audit.retention_purge_completed"

	WorkflowCreated   = "workflow.created"
	WorkflowUpdated   = "workflow.updated"
	WorkflowPublished = "workflow.published"
	WorkflowDeleted   = "workflow.deleted"

	// Workflow execution events
	WorkflowExecutionStarted   = "workflow.execution_started"
	WorkflowExecutionPaused    = "workflow.execution_paused"
	WorkflowExecutionResumed   = "workflow.execution_resumed"
	WorkflowExecutionCompleted = "workflow.execution_completed"
	WorkflowExecutionFailed    = "workflow.execution_failed"
	WorkflowExecutionCancelled = "workflow.execution_cancelled"
	WorkflowNodeCompleted      = "workflow.node_completed"

	// Settings events
	SettingsGeneralUpdated      = "settings.general_updated"
	SettingsIAMUpdated          = "settings.iam.updated"
	SettingsIAMConnectionTested = "settings.iam.connection_tested"
	SettingsRoleMappingUpdated  = "settings.role_mapping.updated"

	// Registration events
	RegistrationCreated = "registration.created"
	RegistrationRevoked = "registration.revoked"

	// IAM events
	UserProvisioned = "user.provisioned"
	UserRoleMapped  = "user.role_mapped"
	UserSynced      = "user.synced"
	UserDisabled    = "user.disabled"
	AuthLogin       = "auth.login"
	AuthLogout      = "auth.logout"

	// Invitation events
	InvitationCreated = "invitation.created"
	InvitationClaimed = "invitation.claimed"
	UserRegistered    = "user.registered"

	// Alert events
	AlertCreated       = "alert.created"
	AlertStatusUpdated = "alert.status_updated"
	AlertRuleCreated   = "alert_rule.created"
	AlertRuleUpdated   = "alert_rule.updated"
	AlertRuleDeleted   = "alert_rule.deleted"
)

// ScanTriggeredPayload is the typed payload for ScanTriggered events.
type ScanTriggeredPayload struct {
	CommandID  string `json:"command_id"`
	EndpointID string `json:"endpoint_id"`
}

// ScanDispatchedPayload is a type alias: the dispatched event carries the same
// shape as the triggered event. Kept as a distinct name so emit/consume sites
// self-document which lifecycle step they handle.
type ScanDispatchedPayload = ScanTriggeredPayload

// ScanCompletedPayload is the typed payload for ScanCompleted events.
type ScanCompletedPayload struct {
	CommandID    string `json:"command_id"`
	EndpointID   string `json:"endpoint_id"`
	Succeeded    bool   `json:"succeeded"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// CommandResultPayload is the typed payload for CommandResultReceived events.
type CommandResultPayload struct {
	CommandID    string `json:"command_id"`
	AgentID      string `json:"agent_id"`
	Succeeded    bool   `json:"succeeded"`
	Output       string `json:"output"`
	Stderr       string `json:"stderr,omitempty"`
	ErrorMessage string `json:"error_message"`
	ExitCode     *int32 `json:"exit_code,omitempty"`
}

// AllTopics returns every known event topic. Used by the Watermill router
// to register subscribers, and by wildcard matching.
func AllTopics() []string {
	return []string{
		EndpointCreated,
		EndpointUpdated,
		EndpointDeleted,
		EndpointEnrolled,
		EndpointScanRequested,
		EndpointConfigPushed,
		HeartbeatReceived,
		InventoryReceived,
		InventoryScanCompleted,
		CommandResultReceived,
		TagCreated,
		TagKeyUpserted,
		TagKeyDeleted,
		PolicyTargetSelectorUpdated,
		TagUpdated,
		TagDeleted,
		EndpointTagged,
		EndpointUntagged,
		TagRuleCreated,
		TagRuleUpdated,
		TagRuleDeleted,
		PolicyCreated,
		PolicyUpdated,
		PolicyDeleted,
		PolicyEvaluated,
		PolicyEvaluationRecorded,
		PolicyAutoDeployed,
		DeploymentStarted,
		DeploymentCompleted,
		DeploymentCreated,
		DeploymentFailed,
		DeploymentCancelled,
		DeploymentEndpointCompleted,
		DeploymentTargetSent,
		CommandDispatched,
		CommandTimedOut,
		DeploymentTargetTimedOut,
		ScanTriggered,
		ScanDispatched,
		ScanCompleted,
		DeploymentWaveStarted,
		DeploymentWaveCompleted,
		DeploymentWaveFailed,
		DeploymentRollbackTriggered,
		DeploymentRolledBack,
		DeploymentRollbackFailed,
		DeploymentRetryTriggered,
		ScheduleCreated,
		ScheduleUpdated,
		ScheduleDeleted,
		PatchDiscovered,
		RepositorySynced,
		CatalogSynced,
		CatalogSyncStarted,
		CatalogSyncFailed,
		HubSyncConfigUpdated,
		HubSyncTriggered,
		CVEDiscovered,
		CVELinkedToEndpoint,
		CVERemediationAvailable,
		RoleCreated,
		RoleUpdated,
		RoleDeleted,
		UserRoleAssigned,
		UserRoleRevoked,
		ComplianceFrameworkEnabled,
		ComplianceFrameworkUpdated,
		ComplianceFrameworkDisabled,
		ComplianceEvaluationCompleted,
		CustomComplianceFrameworkCreated,
		CustomComplianceFrameworkUpdated,
		CustomComplianceFrameworkDeleted,
		CustomComplianceControlsUpdated,
		ChannelCreated,
		ChannelUpdated,
		ChannelDeleted,
		ChannelTested,
		DigestConfigUpdated,
		NotificationPreferencesUpdated,
		ComplianceThresholdBreach,
		AgentDisconnected,
		NotificationSent,
		NotificationFailed,
		LicenseLoaded,
		LicenseExpiring,
		LicenseExpired,
		LicenseGracePeriodEntered,
		AuditExportTriggered,
		AuditRetentionPurgeCompleted,
		WorkflowCreated,
		WorkflowUpdated,
		WorkflowPublished,
		WorkflowDeleted,
		WorkflowExecutionStarted,
		WorkflowExecutionPaused,
		WorkflowExecutionResumed,
		WorkflowExecutionCompleted,
		WorkflowExecutionFailed,
		WorkflowExecutionCancelled,
		WorkflowNodeCompleted,
		RegistrationCreated,
		RegistrationRevoked,
		UserProvisioned,
		UserRoleMapped,
		UserSynced,
		UserDisabled,
		AuthLogin,
		AuthLogout,
		SettingsGeneralUpdated,
		SettingsIAMUpdated,
		SettingsIAMConnectionTested,
		SettingsRoleMappingUpdated,
		InvitationCreated,
		InvitationClaimed,
		UserRegistered,
		AlertCreated,
		AlertStatusUpdated,
		AlertRuleCreated,
		AlertRuleUpdated,
		AlertRuleDeleted,
	}
}

var (
	topicMapOnce sync.Once
	topicMap     map[string]struct{}
)

// TopicSet returns a cached set of all registered topics for O(1) lookup.
func TopicSet() map[string]struct{} {
	topicMapOnce.Do(func() {
		topics := AllTopics()
		topicMap = make(map[string]struct{}, len(topics))
		for _, t := range topics {
			topicMap[t] = struct{}{}
		}
	})
	return topicMap
}
