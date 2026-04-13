package workers

// River queue names. Jobs are split across queues to prevent
// low-priority work from starving time-sensitive operations.
const (
	// QueueCritical handles deployment execution, timeouts, wave dispatch,
	// schedule checking, and scans — anything that directly affects
	// patch deployment SLAs.
	QueueCritical = "critical"

	// QueueDefault handles notifications, compliance, CVE sync/matching,
	// catalog sync, workflow execution, policy scheduling, and user sync.
	QueueDefault = "default"

	// QueueBackground handles discovery and audit retention —
	// long-running, deferrable work.
	QueueBackground = "background"
)
