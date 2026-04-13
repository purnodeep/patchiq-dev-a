/**
 * PatchIQ Event Type Catalog
 *
 * Static registry mapping all PatchIQ event types to their metadata.
 * Powers the alert rule builder's dropdowns, field chips, and default templates.
 *
 * Fields MUST match actual Go event payloads (verified from domain event audit).
 * Do NOT hand-edit generated types — this file is the source of truth for alert UI.
 */

export interface FieldInfo {
  /** Payload field name, e.g. "error" */
  name: string;
  /** "string" | "number" | "boolean" */
  type: string;
  /** Sample value for preview, e.g. "connection refused" */
  sample: string;
  /** Tooltip text, e.g. "Error message from the failed operation" */
  description: string;
}

export interface EventTypeInfo {
  /** e.g. "deployment.failed" */
  type: string;
  /** e.g. "Deployment Failed" */
  label: string;
  /** e.g. "A deployment has failed during execution" */
  description: string;
  category: 'deployments' | 'agents' | 'cves' | 'compliance' | 'system';
  defaultSeverity: 'critical' | 'warning' | 'info';
  /** Available payload fields (empty array if nil payload) */
  fields: FieldInfo[];
  /** Default title template, may use Go template syntax e.g. "{{.hostname}}" */
  defaultTitle: string;
  /** Default description template */
  defaultDescription: string;
}

// ---------------------------------------------------------------------------
// Deployments
// All deployment events carry a nil payload — no fields.
// ---------------------------------------------------------------------------

const DEPLOYMENT_EVENTS: EventTypeInfo[] = [
  {
    type: 'deployment.created',
    label: 'Deployment Created',
    description: 'A new deployment has been created.',
    category: 'deployments',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Deployment created',
    defaultDescription: 'A new deployment has been created.',
  },
  {
    type: 'deployment.started',
    label: 'Deployment Started',
    description: 'A new deployment has started execution.',
    category: 'deployments',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Deployment started',
    defaultDescription: 'A new deployment has started execution.',
  },
  {
    type: 'deployment.completed',
    label: 'Deployment Completed',
    description: 'A deployment has completed successfully.',
    category: 'deployments',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Deployment completed',
    defaultDescription: 'A deployment has completed successfully.',
  },
  {
    type: 'deployment.failed',
    label: 'Deployment Failed',
    description: 'A deployment has failed. Check the deployment detail page.',
    category: 'deployments',
    defaultSeverity: 'critical',
    fields: [],
    defaultTitle: 'Deployment failed',
    defaultDescription: 'A deployment has failed. Check the deployment detail page.',
  },
  {
    type: 'deployment.cancelled',
    label: 'Deployment Cancelled',
    description: 'A deployment has been cancelled.',
    category: 'deployments',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Deployment cancelled',
    defaultDescription: 'A deployment has been cancelled.',
  },
  {
    type: 'deployment.rollback_triggered',
    label: 'Rollback Triggered',
    description: 'A deployment rollback has been triggered.',
    category: 'deployments',
    defaultSeverity: 'critical',
    fields: [],
    defaultTitle: 'Rollback triggered',
    defaultDescription: 'A deployment rollback has been triggered.',
  },
  {
    type: 'deployment.wave_started',
    label: 'Wave Started',
    description: 'A deployment wave has started.',
    category: 'deployments',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Wave started',
    defaultDescription: 'A deployment wave has started.',
  },
  {
    type: 'deployment.wave_completed',
    label: 'Wave Completed',
    description: 'A deployment wave has completed.',
    category: 'deployments',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Wave completed',
    defaultDescription: 'A deployment wave has completed.',
  },
  {
    type: 'deployment.wave_failed',
    label: 'Wave Failed',
    description: 'A deployment wave has failed.',
    category: 'deployments',
    defaultSeverity: 'warning',
    fields: [],
    defaultTitle: 'Wave failed',
    defaultDescription: 'A deployment wave has failed.',
  },
  {
    type: 'command.timed_out',
    label: 'Command Timed Out',
    description: 'A deployment command timed out.',
    category: 'deployments',
    defaultSeverity: 'warning',
    fields: [],
    defaultTitle: 'Command timed out',
    defaultDescription: 'A deployment command timed out.',
  },
];

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

const AGENT_EVENTS: EventTypeInfo[] = [
  {
    type: 'endpoint.enrolled',
    label: 'Endpoint Enrolled',
    description: 'A new endpoint agent has enrolled successfully.',
    category: 'agents',
    defaultSeverity: 'info',
    fields: [
      {
        name: 'hostname',
        type: 'string',
        sample: 'web-server-01',
        description: 'Hostname of the enrolled endpoint',
      },
    ],
    defaultTitle: 'New endpoint: {{.hostname}}',
    defaultDescription: 'Endpoint "{{.hostname}}" has enrolled successfully.',
  },
  {
    type: 'agent.disconnected',
    label: 'Agent Disconnected',
    description: 'An endpoint agent has lost connection.',
    category: 'agents',
    defaultSeverity: 'critical',
    fields: [],
    defaultTitle: 'Agent disconnected',
    defaultDescription: 'An endpoint agent has lost connection.',
  },
  {
    type: 'endpoint.created',
    label: 'Endpoint Created',
    description: 'A new endpoint record has been created.',
    category: 'agents',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Endpoint created',
    defaultDescription: 'A new endpoint record has been created.',
  },
  {
    type: 'endpoint.deleted',
    label: 'Endpoint Deleted',
    description: 'An endpoint record has been deleted.',
    category: 'agents',
    defaultSeverity: 'warning',
    fields: [],
    defaultTitle: 'Endpoint deleted',
    defaultDescription: 'An endpoint record has been deleted.',
  },
];

// ---------------------------------------------------------------------------
// CVEs
// ---------------------------------------------------------------------------

const CVE_EVENTS: EventTypeInfo[] = [
  {
    type: 'cve.discovered',
    label: 'CVE Discovered',
    description: 'A new CVE has been discovered affecting one or more endpoints.',
    category: 'cves',
    defaultSeverity: 'warning',
    fields: [
      {
        name: 'cve_id',
        type: 'string',
        sample: 'CVE-2026-1234',
        description: 'CVE identifier',
      },
      {
        name: 'severity',
        type: 'string',
        sample: 'critical',
        description: 'CVE severity level',
      },
      {
        name: 'cvss',
        type: 'number',
        sample: '8.1',
        description: 'CVSS score',
      },
    ],
    defaultTitle: 'New CVE: {{.cve_id}}',
    defaultDescription: 'CVE {{.cve_id}} discovered (CVSS {{.cvss}}, severity: {{.severity}}).',
  },
  {
    type: 'cve.remediation_available',
    label: 'CVE Remediation Available',
    description: 'A patch is now available for a known CVE.',
    category: 'cves',
    defaultSeverity: 'info',
    fields: [
      {
        name: 'cve_id',
        type: 'string',
        sample: 'CVE-2026-1234',
        description: 'CVE identifier',
      },
      {
        name: 'package_name',
        type: 'string',
        sample: 'openssl',
        description: 'Package name with available fix',
      },
      {
        name: 'patch_id',
        type: 'string',
        sample: 'p-abc123',
        description: 'Patch ID',
      },
    ],
    defaultTitle: 'Remediation available: {{.cve_id}}',
    defaultDescription: 'A patch ({{.package_name}}) is now available for CVE {{.cve_id}}.',
  },
];

// ---------------------------------------------------------------------------
// Compliance
// ---------------------------------------------------------------------------

const COMPLIANCE_EVENTS: EventTypeInfo[] = [
  {
    type: 'compliance.threshold_breach',
    label: 'Compliance Threshold Breach',
    description: 'A compliance framework score has dropped below its configured threshold.',
    category: 'compliance',
    defaultSeverity: 'critical',
    fields: [],
    defaultTitle: 'Compliance threshold breach',
    defaultDescription: 'A compliance framework score has dropped below its configured threshold.',
  },
  {
    type: 'compliance.evaluation_completed',
    label: 'Compliance Evaluation Completed',
    description: 'A compliance framework evaluation has completed.',
    category: 'compliance',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Compliance evaluation completed',
    defaultDescription: 'A compliance framework evaluation has completed.',
  },
  {
    type: 'compliance.framework_enabled',
    label: 'Compliance Framework Enabled',
    description: 'A compliance framework has been enabled.',
    category: 'compliance',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Compliance framework enabled',
    defaultDescription: 'A compliance framework has been enabled.',
  },
  {
    type: 'compliance.framework_disabled',
    label: 'Compliance Framework Disabled',
    description: 'A compliance framework has been disabled.',
    category: 'compliance',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Compliance framework disabled',
    defaultDescription: 'A compliance framework has been disabled.',
  },
];

// ---------------------------------------------------------------------------
// System
// ---------------------------------------------------------------------------

const SYSTEM_EVENTS: EventTypeInfo[] = [
  {
    type: 'catalog.sync_failed',
    label: 'Catalog Sync Failed',
    description: 'Hub catalog synchronization has failed.',
    category: 'system',
    defaultSeverity: 'critical',
    fields: [
      {
        name: 'error',
        type: 'string',
        sample: 'connection refused',
        description: 'Error message from the sync failure',
      },
    ],
    defaultTitle: 'Catalog sync failed',
    defaultDescription: 'Hub catalog synchronization failed. Error: {{.error}}',
  },
  {
    type: 'catalog.synced',
    label: 'Catalog Synced',
    description: 'Hub catalog synchronized successfully.',
    category: 'system',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Catalog synced',
    defaultDescription: 'Hub catalog synchronized successfully.',
  },
  {
    type: 'license.expired',
    label: 'License Expired',
    description: 'The platform license has expired.',
    category: 'system',
    defaultSeverity: 'critical',
    fields: [],
    defaultTitle: 'License expired',
    defaultDescription: 'The platform license has expired.',
  },
  {
    type: 'license.expiring',
    label: 'License Expiring Soon',
    description: 'The platform license is expiring soon.',
    category: 'system',
    defaultSeverity: 'warning',
    fields: [],
    defaultTitle: 'License expiring soon',
    defaultDescription: 'The platform license is expiring soon.',
  },
  {
    type: 'notification.failed',
    label: 'Notification Delivery Failed',
    description: 'A notification failed to be delivered via its configured channel.',
    category: 'system',
    defaultSeverity: 'warning',
    fields: [
      {
        name: 'trigger_type',
        type: 'string',
        sample: 'deployment.completed',
        description: 'Event that triggered the notification',
      },
      {
        name: 'channel_id',
        type: 'string',
        sample: 'ch-abc123',
        description: 'Notification channel ID',
      },
      {
        name: 'status',
        type: 'string',
        sample: 'failed',
        description: 'Delivery status',
      },
    ],
    defaultTitle: 'Notification delivery failed',
    defaultDescription:
      'Failed to deliver {{.trigger_type}} notification via channel {{.channel_id}}.',
  },
  {
    type: 'notification.sent',
    label: 'Notification Sent',
    description: 'A notification has been delivered successfully.',
    category: 'system',
    defaultSeverity: 'info',
    fields: [],
    defaultTitle: 'Notification sent',
    defaultDescription: 'A notification has been delivered successfully.',
  },
];

// ---------------------------------------------------------------------------
// Exports
// ---------------------------------------------------------------------------

/** All event types grouped by category for the dropdown */
export const EVENT_CATEGORIES = [
  { id: 'deployments', label: 'Deployments' },
  { id: 'agents', label: 'Agents' },
  { id: 'cves', label: 'CVEs' },
  { id: 'compliance', label: 'Compliance' },
  { id: 'system', label: 'System' },
] as const;

/** All events sorted by category order then by label */
export const ALL_EVENTS: EventTypeInfo[] = [
  ...DEPLOYMENT_EVENTS,
  ...AGENT_EVENTS,
  ...CVE_EVENTS,
  ...COMPLIANCE_EVENTS,
  ...SYSTEM_EVENTS,
];

/** Lookup map for O(1) access by event type string */
const EVENT_MAP = new Map<string, EventTypeInfo>(ALL_EVENTS.map((e) => [e.type, e]));

/** Returns event metadata for a given event type string, or undefined if not found */
export function getEventInfo(eventType: string): EventTypeInfo | undefined {
  return EVENT_MAP.get(eventType);
}

/** Returns all events belonging to a given category */
export function getEventsByCategory(category: string): EventTypeInfo[] {
  return ALL_EVENTS.filter((e) => e.category === category);
}
