import React from 'react';
import { timeAgo } from './time';
import type { components } from '../api/types';

type AuditEvent = components['schemas']['AuditEvent'];

export type EventCategory =
  | 'Endpoint'
  | 'Deployment'
  | 'Policy'
  | 'Compliance'
  | 'Auth'
  | 'Patch'
  | 'System';

const CATEGORY_PREFIXES: [string[], EventCategory][] = [
  [['endpoint.', 'heartbeat.', 'inventory.', 'agent.', 'scan.'], 'Endpoint'],
  [['deployment.'], 'Deployment'],
  [['policy.'], 'Policy'],
  [['compliance.'], 'Compliance'],
  [['auth.'], 'Auth'],
  [['cve.', 'patch.'], 'Patch'],
];

export function getEventCategory(eventType: string): EventCategory {
  const lower = eventType.toLowerCase();
  for (const [prefixes, category] of CATEGORY_PREFIXES) {
    if (prefixes.some((p) => lower.startsWith(p))) return category;
  }
  return 'System';
}

// Monochrome category colors (Design Rule #1: Color = Signal only)
export function getCategoryColor(_category: EventCategory): string {
  return 'var(--text-secondary)';
}

// Monochrome category badges (Design Rule #2: No colored badge backgrounds)
export function getCategoryBadgeClassName(_category: EventCategory): string {
  return 'text-[var(--text-secondary)] border-[var(--border-strong)] bg-[var(--bg-card-hover)]';
}

const ACTOR_GRADIENTS = [
  'linear-gradient(135deg, #10b981, #059669)',
  'linear-gradient(135deg, #22c55e, #16a34a)',
  'linear-gradient(135deg, #f59e0b, #d97706)',
  'linear-gradient(135deg, #ef4444, #dc2626)',
  'linear-gradient(135deg, #10b981, #047857)',
  'linear-gradient(135deg, #f59e0b, #b45309)',
];

function hashString(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = (Math.imul(31, h) + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

export function getActorGradient(actorId: string): string {
  return ACTOR_GRADIENTS[hashString(actorId) % ACTOR_GRADIENTS.length];
}

export function getActorInitials(actorId: string, eventType?: string): string {
  if (!actorId) return '??';
  if (actorId === 'system' && eventType) {
    // Show category-based abbreviation instead of generic "SY"
    const category = getEventCategory(eventType);
    switch (category) {
      case 'Endpoint':
        return 'EP';
      case 'Deployment':
        return 'DP';
      case 'Policy':
        return 'PO';
      case 'Compliance':
        return 'CO';
      case 'Auth':
        return 'AU';
      case 'Patch':
        return 'PA';
      case 'System':
        return 'SY';
    }
  }
  if (actorId === 'system') return 'SY';
  // UUID pattern — show generic
  if (/^[0-9a-f-]{36}$/i.test(actorId)) return '??';
  // email: take first two chars of local part
  const local = actorId.includes('@') ? actorId.split('@')[0] : actorId;
  return local.slice(0, 2).toUpperCase();
}

export function getEventSummary(event: AuditEvent): { text: string; details: string } {
  const actor = event.actor_id ?? 'unknown';
  const type = event.type ?? '';
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const p = (event.payload ?? {}) as Record<string, any>;

  const hostname = p.hostname ?? p.endpoint_id ?? '';
  const deploymentId = p.deployment_id ?? p.id ?? '';
  const policyName = p.name ?? p.policy_id ?? '';

  if (type.startsWith('endpoint.enrolled')) {
    return {
      text: `${actor} enrolled endpoint ${hostname}`,
      details: [p.os_family, p.os_version, p.agent_version ? `Agent ${p.agent_version}` : '']
        .filter(Boolean)
        .join(' · '),
    };
  }
  if (type.startsWith('endpoint.')) {
    const action = type.split('.').slice(1).join(' ');
    return {
      text: `${actor} ${action} on ${hostname || event.resource_id}`,
      details: '',
    };
  }
  if (type.startsWith('deployment.created')) {
    return {
      text: `${actor} created deployment ${deploymentId}`,
      details: p.target_count ? `${p.target_count} target endpoints` : '',
    };
  }
  if (type.startsWith('deployment.completed')) {
    return {
      text: `${actor} completed deployment ${deploymentId}`,
      details: [
        p.succeeded != null ? `${p.succeeded} succeeded` : '',
        p.failed ? `${p.failed} failed` : '',
      ]
        .filter(Boolean)
        .join(' · '),
    };
  }
  if (type.startsWith('deployment.')) {
    const action = type.split('.').slice(1).join(' ');
    return {
      text: `${actor} ${action} deployment ${deploymentId || event.resource_id}`,
      details: '',
    };
  }
  if (type.startsWith('policy.')) {
    const action = type.split('.').slice(1).join(' ');
    return {
      text: `${actor} ${action} policy ${policyName || event.resource_id}`,
      details: '',
    };
  }
  if (type.startsWith('compliance.evaluation')) {
    return {
      text: `${actor} compliance evaluation completed for ${p.framework ?? event.resource_id}`,
      details:
        p.score_after != null
          ? `Score: ${p.score_after}%${p.score_before != null ? ` (was ${p.score_before}%)` : ''}`
          : '',
    };
  }
  if (type.startsWith('compliance.')) {
    return {
      text: `${actor} ${type.split('.').slice(1).join(' ')}`,
      details: '',
    };
  }
  if (type.startsWith('auth.login')) {
    return {
      text: `${actor} logged in`,
      details: p.method ? `Method: ${p.method}` : '',
    };
  }
  if (type.startsWith('auth.logout')) {
    return { text: `${actor} logged out`, details: '' };
  }
  if (type.startsWith('auth.')) {
    return { text: `${actor} ${type.split('.').slice(1).join(' ')}`, details: '' };
  }
  if (type.startsWith('cve.') || type.startsWith('patch.')) {
    const cveId = p.cve_id ?? p.patch_id ?? event.resource_id;
    return {
      text: `${actor} ${type.split('.').slice(1).join(' ')} ${cveId}`,
      details: p.cvss != null ? `CVSS ${p.cvss}` : '',
    };
  }
  // Generic fallback
  const parts = type.split('.');
  const readableType = parts.join(' ');
  return {
    text: `${actor} ${readableType}`,
    details: event.resource ? `${event.resource}: ${event.resource_id ?? ''}` : '',
  };
}

export function groupEventsByDate(events: AuditEvent[]): Map<string, AuditEvent[]> {
  const map = new Map<string, AuditEvent[]>();
  for (const event of events) {
    if (!event.timestamp) continue;
    const date = new Date(event.timestamp);
    const label = date.toLocaleDateString('en-US', {
      month: 'long',
      day: 'numeric',
      year: 'numeric',
    });
    const group = map.get(label) ?? [];
    group.push(event);
    map.set(label, group);
  }
  return map;
}

export function formatEventTime(timestamp: string | null | undefined): string {
  if (!timestamp) return '—';
  const date = new Date(timestamp);
  if (isNaN(date.getTime())) return '—';

  const absoluteDate = date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
  const relativeTime = timeAgo(timestamp);

  return `${absoluteDate} · ${relativeTime}`;
}

function shortId(id: string): string {
  // Return the full ID — callers should render it verbatim.
  // Truncation was removed to avoid ambiguity; callers use CopyableId for display.
  return id;
}

function displayActor(actorId: string): string {
  if (actorId === 'system') return 'System';
  if (!actorId) return 'Unknown';
  return actorId;
}

function actor(actorId: string | null | undefined): React.ReactNode {
  return <span className="font-semibold">{displayActor(actorId ?? 'unknown')}</span>;
}

function bold(text: string | null | undefined): React.ReactNode {
  return <span className="font-semibold">{text}</span>;
}

// Humanize a dotted/snake_case event type into a lowercase verb phrase.
// e.g. "deployment_target.timed_out" -> "deployment target timed out"
function humanizeEventType(type: string): string {
  return type
    .split('.')
    .map((seg) => seg.replace(/_/g, ' ').trim())
    .filter(Boolean)
    .join(' ')
    .toLowerCase();
}

// Extract a primary "target" string from a payload for title enrichment.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function extractPayloadTarget(p: Record<string, any>): string | null {
  const keys = [
    'hostname',
    'endpoint_name',
    'endpoint',
    'name',
    'patch_name',
    'patch_id',
    'cve_id',
    'policy_name',
    'deployment_id',
    'workflow_name',
    'framework',
    'username',
    'email',
  ];
  for (const k of keys) {
    const v = p[k];
    if (v != null && v !== '') return String(v);
  }
  return null;
}

// Build a subtitle from common payload fields, joined by " · ".
// eslint-disable-next-line @typescript-eslint/no-explicit-any
function extractPayloadSubtitle(p: Record<string, any>): string | null {
  const parts: string[] = [];
  if (p.status) parts.push(String(p.status));
  if (p.os_family || p.os_version) {
    parts.push([p.os_family, p.os_version].filter(Boolean).join(' · '));
  }
  if (p.agent_version) parts.push(`Agent ${p.agent_version}`);
  const targets = p.target_count ?? p.targets;
  if (targets != null) parts.push(`${targets} targets`);
  const succeeded = p.succeeded ?? p.success;
  if (succeeded != null) parts.push(`${succeeded} succeeded`);
  if (p.failed != null) parts.push(`${p.failed} failed`);
  if (p.wave_number != null) parts.push(`Wave ${p.wave_number}`);
  if (p.cvss != null) parts.push(`CVSS ${p.cvss}`);
  if (p.epss != null) parts.push(`EPSS ${Math.round(Number(p.epss) * 100)}%`);
  if (p.severity) parts.push(String(p.severity));
  if (p.score_before != null && p.score_after != null) {
    parts.push(`${p.score_before}% → ${p.score_after}%`);
  }
  if (p.duration_minutes != null) parts.push(`${p.duration_minutes}min`);
  if (p.count != null) parts.push(`${p.count}`);
  if (p.reason) parts.push(String(p.reason).replace(/_/g, ' '));
  if (p.error) parts.push(String(p.error));
  return parts.length > 0 ? parts.join(' · ') : null;
}

export function formatEventDescription(event: AuditEvent): {
  title: React.ReactNode;
  subtitle: React.ReactNode | null;
} {
  const a = event.actor_id ?? 'unknown';
  const type = event.type ?? '';
  const resourceId = event.resource_id ?? '';
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const p = (event.payload ?? {}) as Record<string, any>;

  switch (type) {
    case 'heartbeat.received':
      return {
        title: (
          <>
            {actor(a)} heartbeat from {bold(p.hostname || resourceId)}
          </>
        ),
        subtitle: p.status ? `Status: ${p.status}` : null,
      };
    case 'deployment.target_sent':
      return {
        title: (
          <>
            {actor(a)} sent patch to {bold(p.endpoint || resourceId)}
          </>
        ),
        subtitle: p.patch ? <>Patch: {bold(p.patch)}</> : null,
      };
    case 'deployment.wave_started':
      return {
        title: (
          <>
            {actor(a)} started wave {bold(String(p.wave_number ?? ''))}
          </>
        ),
        subtitle: null,
      };
    case 'deployment.wave_completed':
      return {
        title: (
          <>
            {actor(a)} completed wave {bold(String(p.wave_number ?? ''))}
          </>
        ),
        subtitle:
          p.success != null || p.failed != null
            ? `${p.success ?? 0} succeeded · ${p.failed ?? 0} failed`
            : null,
      };
    case 'deployment.started':
      return {
        title: <>{actor(a)} started deployment</>,
        subtitle:
          p.wave != null || p.targets != null
            ? `Wave ${p.wave ?? ''} · ${p.targets ?? ''} targets`
            : null,
      };
    case 'deployment.created':
      return {
        title: (
          <>
            {actor(a)} created deployment {bold(p.deployment_id || resourceId)}
            {p.policy ? <> via policy {bold(p.policy)}</> : null}
          </>
        ),
        subtitle:
          [
            p.triggered_by
              ? `Triggered by ${p.triggered_by}${p.triggered_cvss != null ? ` (CVSS ${p.triggered_cvss})` : ''}`
              : null,
            p.targets != null ? `${p.targets} target endpoints` : null,
            p.waves != null ? `Wave 1 of ${p.waves}` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.completed':
      return {
        title: (
          <>
            {actor(a)} completed deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle:
          [
            p.success != null ? `${p.success} succeeded` : null,
            p.failed != null ? `${p.failed} failed` : null,
            p.duration_minutes != null ? `${p.duration_minutes}min` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.failed':
      return {
        title: <>{actor(a)} deployment failed</>,
        subtitle:
          [
            p.success != null ? `${p.success} succeeded` : null,
            p.failed != null ? `${p.failed} failed` : null,
            p.reason ? p.reason : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.endpoint_completed':
      return {
        title: (
          <>
            {actor(a)} endpoint {bold(p.endpoint || resourceId)} {p.status ?? ''}
          </>
        ),
        subtitle: p.status === 'failed' && p.error ? p.error : null,
      };
    case 'deployment.cancelled':
      return {
        title: (
          <>
            {actor(a)} cancelled deployment {bold(resourceId)}
          </>
        ),
        subtitle: null,
      };
    case 'endpoint.updated':
      return {
        title: (
          <>
            {actor(a)} endpoint {bold(p.hostname || resourceId)} status changed
          </>
        ),
        subtitle: p.old != null && p.new != null ? `${p.old} → ${p.new}` : null,
      };
    case 'catalog.synced':
      return {
        title: <>{actor(a)} catalog synced</>,
        subtitle:
          p.entries_received != null || p.hub_url
            ? `${p.entries_received ?? ''} entries from ${p.hub_url ?? ''}`.trim()
            : null,
      };
    case 'catalog.sync_started':
      return {
        title: <>{actor(a)} catalog sync started</>,
        subtitle: null,
      };
    case 'auth.login':
      return {
        title: (
          <>
            {actor(a)} logged in{p.method ? <> via {bold(p.method)}</> : null}
          </>
        ),
        subtitle:
          [
            p.ip ? `IP: ${p.ip}` : null,
            p.mfa ? 'MFA verified' : null,
            p.user_agent ? p.user_agent.split(')')[0].replace('Mozilla/5.0 (', '') : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'auth.logout':
      return {
        title: <>{actor(a)} logged out</>,
        subtitle: null,
      };
    case 'notification.sent':
      return {
        title: (
          <>
            {actor(a)} notification sent via {bold(p.channel ?? '')}
          </>
        ),
        subtitle: p.trigger ? <>Trigger: {bold(p.trigger)}</> : null,
      };
    case 'compliance.threshold_breach':
      return {
        title: <>{actor(a)} compliance threshold breach</>,
        subtitle: p.framework ? (
          <>
            {bold(p.framework)}: {p.score ?? ''} (threshold: {p.threshold ?? ''})
          </>
        ) : null,
      };
    case 'compliance.evaluation_completed':
      return {
        title: <>{actor(a)} compliance evaluation completed</>,
        subtitle:
          [
            p.frameworks ? `${p.frameworks} frameworks` : null,
            p.overall_score != null ? `Score: ${p.overall_score}` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'inventory.scan_completed':
      return {
        title: <>{actor(a)} inventory scan completed</>,
        subtitle: p.packages_found != null ? `${p.packages_found} packages found` : null,
      };
    case 'inventory.received':
      return {
        title: <>{actor(a)} inventory received</>,
        subtitle: p.packages != null ? `${p.packages} packages` : null,
      };
    case 'policy.evaluated':
      return {
        title: <>{actor(a)} policy evaluated</>,
        subtitle:
          [
            p.matched_endpoints != null ? `${p.matched_endpoints} endpoints` : null,
            p.patches_applicable != null ? `${p.patches_applicable} patches` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'policy.created':
      return {
        title: (
          <>
            {actor(a)} created policy {bold(p.name || resourceId)}
          </>
        ),
        subtitle: null,
      };
    case 'policy.updated':
      return {
        title: (
          <>
            {actor(a)} updated policy {bold(p.name || resourceId)}
          </>
        ),
        subtitle:
          p.field_changed && p.old_value != null && p.new_value != null
            ? `${p.field_changed}: ${p.old_value} → ${p.new_value}`
            : (p.description ?? null),
      };
    case 'audit.retention_purge_completed':
      return {
        title: <>{actor(a)} audit retention purge</>,
        subtitle:
          [
            p.partitions_pruned != null ? `${p.partitions_pruned} partitions pruned` : null,
            p.retention_days != null ? `${p.retention_days} day retention` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'repository.synced':
      return {
        title: <>{actor(a)} repository synced</>,
        subtitle: p.patches_found != null ? `${p.patches_found} patches found` : null,
      };
    case 'cve.remediation_available':
      return {
        title: (
          <>
            {actor(a)} remediation available for {bold(p.cve_id ?? resourceId)}
          </>
        ),
        subtitle: p.patch ? <>Patch: {bold(p.patch)}</> : null,
      };
    case 'endpoint.scan.completed':
      return {
        title: (
          <>
            {actor(a)} triggered scan on {bold(p.hostname || resourceId)}
          </>
        ),
        subtitle:
          [
            p.packages_scanned != null ? `${p.packages_scanned} packages scanned` : null,
            p.new_cves_found != null ? `${p.new_cves_found} new CVEs found` : null,
            p.cves && Array.isArray(p.cves) && p.cves.length > 0 ? p.cves.join(', ') : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'agent.offline':
      return {
        title: (
          <>
            {actor(a)} detected agent offline: {bold(p.hostname || resourceId)}
          </>
        ),
        subtitle:
          [
            p.last_heartbeat
              ? `Last heartbeat: ${new Date(p.last_heartbeat).toLocaleString()}`
              : null,
            p.alert_sent ? 'Alert sent' : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'compliance.report.exported':
      return {
        title: (
          <>
            {actor(a)} exported compliance report for {bold(p.framework || resourceId)}
          </>
        ),
        subtitle:
          [
            p.format ? `Format: ${p.format.toUpperCase()}` : null,
            p.score != null ? `Score: ${p.score}%` : null,
            p.size_bytes != null ? `${(p.size_bytes / 1024 / 1024).toFixed(1)} MB` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'cve.kev_added':
      return {
        title: (
          <>
            {actor(a)} added {bold(p.cve_id || resourceId)} to KEV (Known Exploited Vulnerabilities)
          </>
        ),
        subtitle:
          [
            p.cvss != null ? `CVSS ${p.cvss} Critical` : null,
            p.epss != null ? `EPSS ${Math.round(p.epss * 100)}%` : null,
            p.affected_endpoints != null ? `${p.affected_endpoints} affected endpoints` : null,
            p.auto_deploy_triggered ? 'Auto-deploy policy triggered' : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.rollback_initiated':
      return {
        title: (
          <>
            {actor(a)} initiated rollback on deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle:
          [
            p.reason ? p.reason.replace(/_/g, ' ') : null,
            p.threshold != null && p.actual != null
              ? `Failure rate: ${Math.round(p.actual * 100)}% (threshold: ${Math.round(p.threshold * 100)}%)`
              : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.multi_wave_created':
      return {
        title: (
          <>
            {actor(a)} created multi-wave deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle:
          [
            p.waves != null ? `${p.waves} waves` : null,
            p.total_endpoints != null ? `${p.total_endpoints} endpoints` : null,
            p.approval_required ? 'Approval required' : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'endpoint_group.scan.completed':
      return {
        title: (
          <>
            {actor(a)} ran on-demand scan on endpoint group {bold(p.group || resourceId)}
          </>
        ),
        subtitle:
          [
            p.endpoints_scanned != null ? `${p.endpoints_scanned} endpoints scanned` : null,
            p.critical_patches_found != null
              ? `${p.critical_patches_found} critical patches found`
              : null,
            p.duration_seconds != null
              ? `${Math.floor(p.duration_seconds / 60)}m ${p.duration_seconds % 60}s`
              : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'rbac.role_updated':
      return {
        title: (
          <>
            {actor(a)} updated RBAC role {bold(p.role || resourceId)}
            {p.permission_added ? (
              <>
                {' '}
                — added permission{' '}
                <code className="bg-muted px-1 py-0.5 rounded text-[10px]">
                  {p.permission_added}
                </code>
              </>
            ) : null}
          </>
        ),
        subtitle: p.affected_users != null ? `${p.affected_users} users affected` : null,
      };
    case 'endpoint.decommissioned':
      return {
        title: (
          <>
            {actor(a)} deleted {bold(p.hostname || resourceId)}
          </>
        ),
        subtitle:
          [
            p.agent_uninstalled ? 'Agent gracefully uninstalled' : null,
            p.data_retained ? 'Historical data retained per retention policy' : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'workflow.created':
      return {
        title: (
          <>
            {actor(a)} created workflow {bold(p.name || resourceId)}
          </>
        ),
        subtitle:
          [
            p.node_count != null ? `${p.node_count} nodes` : null,
            p.trigger ? `Trigger: ${p.trigger}` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'workflow.published':
      return { title: <>{actor(a)} published workflow</>, subtitle: null };
    case 'schedule.created':
      return { title: <>{actor(a)} created schedule</>, subtitle: null };
    case 'role.created':
      return { title: <>{actor(a)} created role</>, subtitle: null };
    case 'user_role.assigned':
      return { title: <>{actor(a)} assigned role</>, subtitle: null };
    case 'group.created':
      return { title: <>{actor(a)} created group</>, subtitle: null };
    case 'group.members_updated':
      return { title: <>{actor(a)} updated group members</>, subtitle: null };
    case 'channel.created':
      return { title: <>{actor(a)} created notification channel</>, subtitle: null };
    case 'scan.triggered':
      return {
        title: <>{actor(a)} triggered endpoint scan</>,
        subtitle: resourceId ? `Target: ${shortId(resourceId)}` : null,
      };
    case 'scan.dispatched':
      return {
        title: <>Agent picked up scan</>,
        subtitle: resourceId ? `Target: ${shortId(resourceId)}` : null,
      };
    case 'scan.completed': {
      const succeeded = (p as { succeeded?: boolean })?.succeeded ?? true;
      return {
        title: <>Scan {succeeded ? 'completed' : 'failed'}</>,
        subtitle: resourceId ? `Target: ${shortId(resourceId)}` : null,
      };
    }
    case 'endpoint.deleted':
      return {
        title: (
          <>
            {actor(a)} deleted endpoint {bold(p.hostname || shortId(resourceId))}
          </>
        ),
        subtitle: null,
      };
    case 'endpoint.created':
      return {
        title: (
          <>
            {actor(a)} created endpoint {bold(p.hostname || shortId(resourceId))}
          </>
        ),
        subtitle: p.os_family ? `OS: ${p.os_family}` : null,
      };
    case 'deployment_target.timed_out':
      return {
        title: <>{actor(a)} deployment target timed out</>,
        subtitle: resourceId ? `Target: ${shortId(resourceId)}` : null,
      };
    case 'deployment_target.sent':
      return {
        title: <>{actor(a)} sent deployment target</>,
        subtitle: resourceId ? `Target: ${shortId(resourceId)}` : null,
      };
    case 'command.timed_out':
      return {
        title: <>{actor(a)} command timed out</>,
        subtitle: resourceId ? `Command: ${shortId(resourceId)}` : null,
      };
    case 'command.dispatched':
      return {
        title: <>{actor(a)} dispatched command</>,
        subtitle: resourceId ? `Command: ${shortId(resourceId)}` : null,
      };
    case 'command.result.received':
      return {
        title: <>{actor(a)} received command result</>,
        subtitle: resourceId ? `Command: ${shortId(resourceId)}` : null,
      };
    case 'endpoint.enrolled':
      return {
        title: (
          <>
            {actor(a)} enrolled endpoint {bold(p.hostname || shortId(resourceId))}
          </>
        ),
        subtitle:
          [p.os_family, p.agent_version ? `Agent ${p.agent_version}` : '']
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'agent.disconnected':
      return {
        title: (
          <>
            {actor(a)} agent disconnected: {bold(p.hostname || shortId(resourceId))}
          </>
        ),
        subtitle: null,
      };
    case 'endpoint.scan_requested':
      return {
        title: (
          <>
            {actor(a)} requested scan on {bold(p.hostname || resourceId)}
          </>
        ),
        subtitle: extractPayloadSubtitle(p),
      };
    case 'endpoint.tagged':
      return {
        title: (
          <>
            {actor(a)} tagged endpoint {bold(p.hostname || resourceId)}
            {p.tag ? <> with {bold(p.tag)}</> : null}
          </>
        ),
        subtitle: null,
      };
    case 'endpoint.untagged':
      return {
        title: (
          <>
            {actor(a)} untagged endpoint {bold(p.hostname || resourceId)}
            {p.tag ? <> from {bold(p.tag)}</> : null}
          </>
        ),
        subtitle: null,
      };
    case 'cve.discovered':
      return {
        title: (
          <>
            {actor(a)} discovered CVE {bold(p.cve_id || resourceId)}
          </>
        ),
        subtitle:
          [
            p.cvss != null ? `CVSS ${p.cvss}` : null,
            p.severity ? String(p.severity) : null,
            p.epss != null ? `EPSS ${Math.round(Number(p.epss) * 100)}%` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'cve.linked_to_endpoint':
      return {
        title: (
          <>
            {actor(a)} linked {bold(p.cve_id || resourceId)} to{' '}
            {bold(p.hostname || p.endpoint_id || '')}
          </>
        ),
        subtitle:
          [p.cvss != null ? `CVSS ${p.cvss}` : null, p.package ? `Package: ${p.package}` : null]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'patch.discovered':
      return {
        title: (
          <>
            {actor(a)} discovered patch {bold(p.patch_id || p.name || resourceId)}
          </>
        ),
        subtitle:
          [p.severity ? String(p.severity) : null, p.source ? `Source: ${p.source}` : null]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'patch.applied':
      return {
        title: (
          <>
            {actor(a)} applied patch {bold(p.patch_id || p.patch || resourceId)}
            {p.hostname ? <> on {bold(p.hostname)}</> : null}
          </>
        ),
        subtitle:
          [
            p.status ? String(p.status) : null,
            p.duration_minutes != null ? `${p.duration_minutes}min` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.wave_failed':
      return {
        title: (
          <>
            {actor(a)} wave {bold(String(p.wave_number ?? ''))} failed
          </>
        ),
        subtitle:
          [
            p.success != null ? `${p.success} succeeded` : null,
            p.failed != null ? `${p.failed} failed` : null,
            p.reason ? String(p.reason).replace(/_/g, ' ') : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'deployment.rollback_triggered':
      return {
        title: (
          <>
            {actor(a)} triggered rollback on deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle: p.reason ? String(p.reason).replace(/_/g, ' ') : null,
      };
    case 'deployment.rolled_back':
      return {
        title: (
          <>
            {actor(a)} rolled back deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle: extractPayloadSubtitle(p),
      };
    case 'deployment.rollback_failed':
      return {
        title: (
          <>
            {actor(a)} rollback failed on deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle: p.error ? String(p.error) : (p.reason ?? null),
      };
    case 'deployment.retry_triggered':
      return {
        title: (
          <>
            {actor(a)} retried deployment {bold(p.deployment_id || resourceId)}
          </>
        ),
        subtitle: extractPayloadSubtitle(p),
      };
    case 'policy.auto_deployed':
      return {
        title: (
          <>
            {actor(a)} auto-deployed policy {bold(p.policy_name || p.name || resourceId)}
          </>
        ),
        subtitle:
          [
            p.triggered_by ? `Triggered by ${p.triggered_by}` : null,
            p.target_count != null ? `${p.target_count} targets` : null,
          ]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'policy.deleted':
      return {
        title: (
          <>
            {actor(a)} deleted policy {bold(p.name || resourceId)}
          </>
        ),
        subtitle: null,
      };
    case 'group.updated':
      return {
        title: (
          <>
            {actor(a)} updated group {bold(p.name || resourceId)}
          </>
        ),
        subtitle: null,
      };
    case 'group.deleted':
      return {
        title: (
          <>
            {actor(a)} deleted group {bold(p.name || resourceId)}
          </>
        ),
        subtitle: null,
      };
    case 'agent.connected':
      return {
        title: (
          <>
            {actor(a)} agent connected: {bold(p.hostname || resourceId)}
          </>
        ),
        subtitle: p.agent_version ? `Agent ${p.agent_version}` : null,
      };
    case 'alert.created':
      return {
        title: (
          <>
            {actor(a)} alert raised{p.title ? <>: {bold(p.title)}</> : null}
          </>
        ),
        subtitle:
          [p.severity ? String(p.severity) : null, p.rule_name ? `Rule: ${p.rule_name}` : null]
            .filter(Boolean)
            .join(' · ') || null,
      };
    case 'alert.status_updated':
      return {
        title: <>{actor(a)} updated alert status</>,
        subtitle: p.status ? `Status: ${p.status}` : null,
      };
    default: {
      const humanType = humanizeEventType(type);
      const target = extractPayloadTarget(p);
      const subtitle = extractPayloadSubtitle(p);
      return {
        title: (
          <>
            {actor(a)} {humanType || event.action || 'performed action'}
            {target ? <> {bold(target)}</> : null}
          </>
        ),
        subtitle: subtitle ?? (resourceId ? `Resource: ${shortId(resourceId)}` : null),
      };
    }
  }
}
