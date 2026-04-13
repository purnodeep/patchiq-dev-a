import { useState, useMemo } from 'react';
import { useParams, Link } from 'react-router';
import {
  XCircle,
  RotateCcw,
  Download,
  MoreHorizontal,
  ChevronDown,
  ChevronRight,
} from 'lucide-react';
import { toast } from 'sonner';
import {
  Button,
  Skeleton,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  SeverityText,
} from '@patchiq/ui';
import {
  useDeployment,
  useCancelDeployment,
  useRetryDeployment,
  useRollbackDeployment,
  useDeploymentWaves,
  useDeploymentPatches,
} from '../../api/hooks/useDeployments';
import { usePolicy } from '../../api/hooks/usePolicies';
import { formatDeploymentId } from '../../lib/format';
import { MonospaceOutput } from '../../components/MonospaceOutput';
import { WavePipelineLanes } from './components/WavePipelineLanes';
import { EndpointDotMap } from './components/EndpointDotMap';
import { DeploymentTimeline } from './components/DeploymentTimeline';
import { timeAgo } from '../../lib/time';
import { useCan } from '../../app/auth/AuthContext';
import type { components } from '../../api/types';

type DeploymentTarget = components['schemas']['DeploymentTarget'];

// ── Helpers ──────────────────────────────────────────────────────────────────

function displayUser(id: string | null | undefined): string {
  if (!id) return 'System';
  if (id.startsWith('cc000000')) return 'Admin';
  return id.slice(0, 8) + '\u2026';
}

const ACTIVE_STATUSES = new Set(['created', 'running', 'rolling_back', 'scheduled']);

function isActive(status: string): boolean {
  return ACTIVE_STATUSES.has(status);
}

// ── Style tokens ─────────────────────────────────────────────────────────────

const CARD: React.CSSProperties = {
  background: 'var(--bg-card)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  boxShadow: 'var(--shadow-sm)',
};

const TH: React.CSSProperties = {
  padding: '9px 12px',
  textAlign: 'left',
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  whiteSpace: 'nowrap',
  background: 'var(--bg-inset)',
  borderBottom: '1px solid var(--border)',
};

const TD: React.CSSProperties = {
  padding: '9px 12px',
  verticalAlign: 'middle',
  borderBottom: '1px solid var(--border)',
  fontSize: 12,
};

// ── Status colors ─────────────────────────────────────────────────────────────

const statusColorMap: Record<string, string> = {
  running: 'var(--signal-healthy)',
  completed: 'var(--signal-healthy)',
  failed: 'var(--signal-critical)',
  rollback_failed: 'var(--signal-critical)',
  rolling_back: 'var(--signal-warning)',
  scheduled: 'var(--text-muted)',
  created: 'var(--text-muted)',
  cancelled: 'var(--text-muted)',
};

const waveStatusConfig: Record<string, { barColor: string; textColor: string; label: string }> = {
  completed: {
    barColor: 'var(--signal-healthy)',
    textColor: 'var(--signal-healthy)',
    label: '\u2713 Completed',
  },
  running: { barColor: 'var(--accent)', textColor: 'var(--accent)', label: '\u25b6 In Progress' },
  failed: {
    barColor: 'var(--signal-critical)',
    textColor: 'var(--signal-critical)',
    label: '\u2717 Failed',
  },
};

const defaultWaveConfig = {
  barColor: 'var(--bg-inset)',
  textColor: 'var(--text-muted)',
  label: '\u25f7 Waiting',
};

function getWaveConfig(status: string) {
  return waveStatusConfig[status] ?? defaultWaveConfig;
}

const targetStatusCfg: Record<string, { color: string; label: string; pulse: boolean }> = {
  pending: { color: 'var(--signal-warning)', label: '\u25f7 Pending', pulse: false },
  sent: { color: 'var(--accent)', label: '\u27f3 Sent', pulse: false },
  executing: { color: 'var(--accent)', label: '\u27f3 Executing', pulse: true },
  succeeded: { color: 'var(--signal-healthy)', label: '\u2713 Succeeded', pulse: false },
  failed: { color: 'var(--signal-critical)', label: '\u2717 Failed', pulse: false },
};

// ── Donut gauge ───────────────────────────────────────────────────────────────

function DonutGauge({ pct, color, size = 80 }: { pct: number; color: string; size?: number }) {
  const r = (size - 10) / 2;
  const circ = 2 * Math.PI * r;
  const dash = (pct / 100) * circ;
  const cx = size / 2;
  const cy = size / 2;
  return (
    <svg width={size} height={size} style={{ transform: 'rotate(-90deg)' }}>
      <circle cx={cx} cy={cy} r={r} fill="none" stroke="var(--bg-inset)" strokeWidth={7} />
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke={color}
        strokeWidth={7}
        strokeDasharray={`${dash} ${circ - dash}`}
        strokeLinecap="round"
        style={{ transition: 'stroke-dasharray 0.6s ease' }}
      />
    </svg>
  );
}

// ── Target row ─────────────────────────────────────────────────────────────────

function TargetRow({
  target,
  expanded,
  onToggle,
}: {
  target: DeploymentTarget;
  expanded: boolean;
  onToggle: () => void;
}) {
  const cfg = targetStatusCfg[target.status] ?? {
    color: 'var(--text-muted)',
    label: target.status,
    pulse: false,
  };

  const duration =
    target.started_at && target.completed_at
      ? (() => {
          const ms =
            new Date(target.completed_at).getTime() - new Date(target.started_at).getTime();
          const mins = Math.floor(ms / 60000);
          const secs = Math.floor((ms % 60000) / 1000);
          return `${mins}m ${secs.toString().padStart(2, '0')}s`;
        })()
      : '\u2014';

  return (
    <>
      <tr
        onClick={onToggle}
        style={{
          cursor: 'pointer',
          background:
            target.status === 'failed'
              ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
              : 'transparent',
          transition: 'background 0.1s',
        }}
        onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--bg-card-hover)')}
        onMouseLeave={(e) =>
          (e.currentTarget.style.background =
            target.status === 'failed'
              ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
              : 'transparent')
        }
      >
        <td style={TD}>
          {expanded ? (
            <ChevronDown style={{ width: 13, height: 13, color: 'var(--text-muted)' }} />
          ) : (
            <ChevronRight style={{ width: 13, height: 13, color: 'var(--text-muted)' }} />
          )}
        </td>
        <td style={TD}>
          {target.endpoint_id ? (
            <Link
              to={`/endpoints/${target.endpoint_id}`}
              style={{
                fontFamily: 'var(--font-mono)',
                fontSize: 11,
                color: 'var(--accent)',
                textDecoration: 'none',
              }}
              onClick={(e) => e.stopPropagation()}
            >
              {target.hostname}
            </Link>
          ) : (
            <span
              style={{ fontFamily: 'var(--font-mono)', fontSize: 11, color: 'var(--text-primary)' }}
            >
              {target.hostname}
            </span>
          )}
        </td>
        <td style={TD}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              fontWeight: 500,
              color: cfg.color,
              animation: cfg.pulse ? 'pulse-dot 1.5s ease-in-out infinite' : undefined,
            }}
          >
            {cfg.label}
          </span>
        </td>
        <td style={TD}>
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color:
                target.exit_code === 0
                  ? 'var(--signal-healthy)'
                  : target.exit_code != null
                    ? 'var(--signal-critical)'
                    : 'var(--text-muted)',
            }}
          >
            {target.exit_code != null ? target.exit_code : '\u2014'}
          </span>
        </td>
        <td
          style={{
            ...TD,
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
          }}
        >
          {target.started_at ? timeAgo(target.started_at) : '\u2014'}
        </td>
        <td
          style={{
            ...TD,
            color: 'var(--text-muted)',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
          }}
        >
          {duration}
        </td>
        <td
          style={{
            ...TD,
            maxWidth: 200,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            color: 'var(--signal-critical)',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
          }}
        >
          {target.error ?? ''}
        </td>
      </tr>
      {expanded && (
        <tr>
          <td
            colSpan={7}
            style={{
              padding: '12px 20px',
              background: 'var(--bg-inset)',
              borderBottom: '1px solid var(--border)',
            }}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {target.output && (
                <div>
                  <p
                    style={{
                      fontSize: 10,
                      fontWeight: 600,
                      color: 'var(--text-muted)',
                      marginBottom: 6,
                      textTransform: 'uppercase',
                      letterSpacing: '0.06em',
                    }}
                  >
                    stdout
                  </p>
                  <MonospaceOutput content={target.output} />
                </div>
              )}
              {target.error && (
                <div>
                  <p
                    style={{
                      fontSize: 10,
                      fontWeight: 600,
                      color: 'var(--signal-critical)',
                      marginBottom: 6,
                      textTransform: 'uppercase',
                      letterSpacing: '0.06em',
                    }}
                  >
                    stderr
                  </p>
                  <MonospaceOutput
                    content={target.error}
                    className="border-red-500/30 text-red-500"
                  />
                </div>
              )}
              {!target.output && !target.error && (
                <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>No output available</p>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export function DeploymentDetailPage() {
  const can = useCan();
  const { id } = useParams<{ id: string }>();
  const { data, isLoading, isError, refetch } = useDeployment(id ?? '');
  const cancelMutation = useCancelDeployment();
  const retryMutation = useRetryDeployment();
  const rollbackMutation = useRollbackDeployment();
  const { data: waves } = useDeploymentWaves(id ?? '');
  const { data: patches } = useDeploymentPatches(id ?? '');
  const { data: policy } = usePolicy(data?.policy_id ?? '');

  const [expandedTargets, setExpandedTargets] = useState<Set<string>>(new Set());
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false);
  const [retryDialogOpen, setRetryDialogOpen] = useState(false);
  const [rollbackDialogOpen, setRollbackDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');
  const [moreOpen, setMoreOpen] = useState(false);

  const toggleTarget = (targetId: string) => {
    setExpandedTargets((prev) => {
      const next = new Set(prev);
      if (next.has(targetId)) next.delete(targetId);
      else next.add(targetId);
      return next;
    });
  };

  const handleCancel = async () => {
    if (!id) return;
    try {
      await cancelMutation.mutateAsync(id);
      setCancelDialogOpen(false);
    } catch {
      toast.error('Failed to cancel deployment');
    }
  };

  const handleRetry = async () => {
    if (!id) return;
    try {
      await retryMutation.mutateAsync(id);
      setRetryDialogOpen(false);
    } catch {
      toast.error('Failed to retry deployment');
    }
  };

  const handleRollback = async () => {
    if (!id) return;
    try {
      await rollbackMutation.mutateAsync(id);
      setRollbackDialogOpen(false);
    } catch {
      toast.error('Failed to rollback deployment');
    }
  };

  // De-duplicate targets by endpoint_id (B-15/B-16): one row per endpoint, keeping worst status
  // NOTE: useMemo must be called unconditionally before any early returns (Rules of Hooks).
  const STATUS_PRIORITY: Record<string, number> = {
    failed: 4,
    executing: 3,
    sent: 2,
    pending: 1,
    succeeded: 0,
  };
  const allTargets = data?.targets ?? [];
  const uniqueTargets = useMemo(() => {
    const map = new Map<string, DeploymentTarget>();
    for (const t of allTargets) {
      const key = t.endpoint_id ?? t.id;
      const existing = map.get(key);
      if (!existing || (STATUS_PRIORITY[t.status] ?? 0) > (STATUS_PRIORITY[existing.status] ?? 0)) {
        map.set(key, t);
      }
    }
    return Array.from(map.values());
  }, [allTargets]);

  // ── Loading ────────────────────────────────────────────────────────────────

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-20 rounded-lg" />
        <Skeleton className="h-14 rounded-lg" />
        <Skeleton className="h-64 rounded-lg" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div style={{ padding: 24 }}>
        <div
          style={{
            borderRadius: 8,
            border: '1px solid var(--signal-critical)',
            background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
            color: 'var(--signal-critical)',
            padding: '12px 16px',
            fontSize: 13,
          }}
        >
          Failed to load deployment details.{' '}
          <button
            onClick={() => refetch()}
            style={{
              background: 'none',
              border: 'none',
              color: 'var(--signal-critical)',
              cursor: 'pointer',
              textDecoration: 'underline',
              padding: 0,
              fontSize: 13,
            }}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  const targets = data.targets ?? [];

  const pending = Math.max(0, data.target_count - data.completed_count);
  const overallPct =
    data.target_count > 0 ? Math.round((data.completed_count / data.target_count) * 100) : 0;
  const succeededPct =
    data.target_count > 0 ? Math.round((data.success_count / data.target_count) * 100) : 0;
  const sortedWaves = waves ? [...waves].sort((a, b) => a.wave_number - b.wave_number) : [];
  const currentWave = sortedWaves.find((w) => w.status === 'running');
  const completedWaveCount = sortedWaves.filter((w) => w.status === 'completed').length;
  const policyName = policy?.name;
  const shortId = formatDeploymentId(data.id);
  const statusColor = statusColorMap[data.status] ?? 'var(--text-muted)';
  const statusLabel = data.status.charAt(0).toUpperCase() + data.status.slice(1).replace(/_/g, ' ');

  // Duration
  const durationMs =
    data.started_at && data.completed_at
      ? new Date(data.completed_at).getTime() - new Date(data.started_at).getTime()
      : data.started_at
        ? Date.now() - new Date(data.started_at).getTime()
        : null;
  const durationLabel = durationMs
    ? (() => {
        const mins = Math.floor(durationMs / 60000);
        const hrs = Math.floor(mins / 60);
        const remMins = mins % 60;
        return hrs > 0 ? `${hrs}h ${remMins}m` : `${mins}m`;
      })()
    : '\u2014';

  const tabs = [
    { id: 'overview', label: 'Overview' },
    { id: 'waves', label: `Waves${sortedWaves.length > 0 ? ` (${sortedWaves.length})` : ''}` },
    {
      id: 'endpoints',
      label: `Endpoints${uniqueTargets.length > 0 ? ` (${uniqueTargets.length})` : ''}`,
    },
    { id: 'patches', label: 'Patches Deployed' },
    { id: 'timeline', label: 'Timeline' },
  ];

  return (
    <div
      style={{
        padding: '20px 24px',
        background: 'var(--bg-page)',
        minHeight: '100%',
        display: 'flex',
        flexDirection: 'column',
        gap: 0,
      }}
    >
      <style>{`
        @keyframes pulse-dot { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
      `}</style>

      {/* ── Header row 1: Title + Actions ─────────────────────────────────── */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-start',
          justifyContent: 'space-between',
          gap: 16,
          marginBottom: 6,
        }}
      >
        <div>
          <h1
            style={{
              fontSize: 22,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              margin: 0,
              letterSpacing: '-0.01em',
              lineHeight: 1.3,
            }}
          >
            {policyName ? (
              <>
                <Link
                  to={`/policies/${data.policy_id}`}
                  style={{ color: 'var(--text-emphasis)', textDecoration: 'none' }}
                  onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--accent)')}
                  onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-emphasis)')}
                >
                  {policyName}
                </Link>
                <span style={{ color: 'var(--text-muted)', fontWeight: 400 }}> — </span>
                <span
                  style={{
                    fontFamily: 'var(--font-mono)',
                    fontSize: 14,
                    fontWeight: 500,
                    color: 'var(--text-muted)',
                  }}
                >
                  {shortId}
                </span>
              </>
            ) : (
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 16 }}>{shortId}</span>
            )}
          </h1>
        </div>

        {/* Actions */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexShrink: 0 }}>
          {isActive(data.status) && (
            <button
              type="button"
              onClick={() => setCancelDialogOpen(true)}
              disabled={!can('deployments', 'cancel')}
              title={!can('deployments', 'cancel') ? "You don't have permission" : undefined}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 5,
                padding: '6px 12px',
                borderRadius: 6,
                fontSize: 12,
                fontWeight: 500,
                cursor: !can('deployments', 'cancel') ? 'not-allowed' : 'pointer',
                background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                color: 'var(--signal-critical)',
                border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                opacity: !can('deployments', 'cancel') ? 0.5 : 1,
              }}
            >
              <XCircle style={{ width: 12, height: 12 }} />
              Cancel
            </button>
          )}
          {data.status === 'failed' && (
            <button
              type="button"
              onClick={() => setRetryDialogOpen(true)}
              disabled={!can('deployments', 'update')}
              title={!can('deployments', 'update') ? "You don't have permission" : undefined}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 5,
                padding: '6px 12px',
                borderRadius: 6,
                fontSize: 12,
                fontWeight: 500,
                cursor: !can('deployments', 'update') ? 'not-allowed' : 'pointer',
                background: 'color-mix(in srgb, var(--accent) 8%, transparent)',
                color: 'var(--accent)',
                border: '1px solid color-mix(in srgb, var(--accent) 20%, transparent)',
                opacity: !can('deployments', 'update') ? 0.5 : 1,
              }}
            >
              <RotateCcw style={{ width: 12, height: 12 }} />
              Retry Failed ({data.failed_count})
            </button>
          )}
          <div style={{ position: 'relative' }}>
            <button
              type="button"
              onClick={() => setMoreOpen((v) => !v)}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                width: 32,
                height: 32,
                borderRadius: 6,
                cursor: 'pointer',
                background: 'transparent',
                border: '1px solid var(--border)',
                color: 'var(--text-muted)',
              }}
            >
              <MoreHorizontal style={{ width: 14, height: 14 }} />
            </button>
            {moreOpen && (
              <div
                style={{
                  position: 'absolute',
                  right: 0,
                  top: 36,
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  boxShadow: 'var(--shadow-sm)',
                  minWidth: 160,
                  zIndex: 50,
                  overflow: 'hidden',
                }}
                onMouseLeave={() => setMoreOpen(false)}
              >
                {isActive(data.status) && (
                  <button
                    type="button"
                    onClick={() => {
                      setMoreOpen(false);
                      setCancelDialogOpen(true);
                    }}
                    disabled={!can('deployments', 'cancel')}
                    title={!can('deployments', 'cancel') ? "You don't have permission" : undefined}
                    style={{
                      display: 'flex',
                      width: '100%',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 12px',
                      fontSize: 12,
                      color: 'var(--signal-critical)',
                      background: 'transparent',
                      border: 'none',
                      cursor: !can('deployments', 'cancel') ? 'not-allowed' : 'pointer',
                      textAlign: 'left',
                      opacity: !can('deployments', 'cancel') ? 0.5 : 1,
                    }}
                  >
                    <XCircle style={{ width: 12, height: 12 }} />
                    Cancel
                  </button>
                )}
                {data.status === 'failed' && (
                  <button
                    type="button"
                    onClick={() => {
                      setMoreOpen(false);
                      setRetryDialogOpen(true);
                    }}
                    disabled={!can('deployments', 'update')}
                    title={!can('deployments', 'update') ? "You don't have permission" : undefined}
                    style={{
                      display: 'flex',
                      width: '100%',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 12px',
                      fontSize: 12,
                      color: 'var(--accent)',
                      background: 'transparent',
                      border: 'none',
                      cursor: !can('deployments', 'update') ? 'not-allowed' : 'pointer',
                      textAlign: 'left',
                      opacity: !can('deployments', 'update') ? 0.5 : 1,
                    }}
                  >
                    <RotateCcw style={{ width: 12, height: 12 }} />
                    Retry Failed
                  </button>
                )}
                {(data.status === 'completed' || data.status === 'failed') && (
                  <button
                    type="button"
                    onClick={() => {
                      setMoreOpen(false);
                      setRollbackDialogOpen(true);
                    }}
                    disabled={!can('deployments', 'update')}
                    title={!can('deployments', 'update') ? "You don't have permission" : undefined}
                    style={{
                      display: 'flex',
                      width: '100%',
                      alignItems: 'center',
                      gap: 8,
                      padding: '8px 12px',
                      fontSize: 12,
                      color: 'var(--signal-warning)',
                      background: 'transparent',
                      border: 'none',
                      cursor: !can('deployments', 'update') ? 'not-allowed' : 'pointer',
                      textAlign: 'left',
                      opacity: !can('deployments', 'update') ? 0.5 : 1,
                    }}
                  >
                    Rollback
                  </button>
                )}
                <button
                  type="button"
                  disabled
                  style={{
                    display: 'flex',
                    width: '100%',
                    alignItems: 'center',
                    gap: 8,
                    padding: '8px 12px',
                    fontSize: 12,
                    color: 'var(--text-muted)',
                    background: 'transparent',
                    border: 'none',
                    cursor: 'not-allowed',
                    textAlign: 'left',
                  }}
                >
                  <Download style={{ width: 12, height: 12 }} />
                  Export Report
                </button>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* ── Header row 2: Status + metadata chips ─────────────────────────── */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: '4px 12px',
          marginBottom: 20,
        }}
      >
        {/* Status dot + text */}
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <span
            style={{
              width: 7,
              height: 7,
              borderRadius: '50%',
              background: statusColor,
              flexShrink: 0,
              animation: isActive(data.status) ? 'pulse-dot 1.5s ease-in-out infinite' : undefined,
            }}
          />
          <span style={{ fontSize: 13, fontWeight: 500, color: statusColor }}>{statusLabel}</span>
        </span>

        {sortedWaves.length > 0 && (
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-muted)',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 4,
              padding: '2px 8px',
            }}
          >
            Wave {currentWave?.wave_number ?? completedWaveCount}/{sortedWaves.length}
          </span>
        )}

        {data.started_at && (
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-muted)',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 4,
              padding: '2px 8px',
            }}
          >
            Started {timeAgo(data.started_at)}
          </span>
        )}

        {data.created_by && (
          <span
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 11,
              color: 'var(--text-muted)',
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              borderRadius: 4,
              padding: '2px 8px',
            }}
          >
            By {displayUser(data.created_by)}
          </span>
        )}

        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: 'var(--text-muted)',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 4,
            padding: '2px 8px',
          }}
        >
          {data.target_count} endpoints
        </span>
      </div>

      {/* ── Health Strip ──────────────────────────────────────────────────── */}
      <div
        style={{
          display: 'flex',
          height: 56,
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
          marginBottom: 20,
        }}
      >
        {[
          {
            label: 'Progress',
            value: `${overallPct}%`,
            fill: overallPct,
            color: 'var(--accent)',
          },
          {
            label: 'Succeeded',
            value: `${data.success_count}/${data.target_count}`,
            fill: succeededPct,
            color: 'var(--signal-healthy)',
          },
          {
            label: 'Failed',
            value: `${data.failed_count}/${data.target_count}`,
            fill:
              data.target_count > 0 ? Math.round((data.failed_count / data.target_count) * 100) : 0,
            color: 'var(--signal-critical)',
          },
          {
            label: 'Duration',
            value: durationLabel,
            fill: null,
            color: 'var(--text-primary)',
          },
        ].map((item, i, arr) => (
          <div
            key={item.label}
            style={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'center',
              padding: '0 18px',
              borderRight: i < arr.length - 1 ? '1px solid var(--border)' : 'none',
              gap: 5,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span
                style={{
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                }}
              >
                {item.label}
              </span>
              <span
                style={{
                  fontFamily: 'var(--font-mono)',
                  fontSize: 13,
                  fontWeight: 700,
                  color: item.color,
                }}
              >
                {item.value}
              </span>
            </div>
            {item.fill !== null && (
              <div
                style={{
                  height: 3,
                  borderRadius: 2,
                  background: 'var(--bg-inset)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    width: `${item.fill}%`,
                    height: '100%',
                    background: item.color,
                    borderRadius: 2,
                    transition: 'width 0.5s ease',
                  }}
                />
              </div>
            )}
          </div>
        ))}
      </div>

      {/* ── Tab bar ───────────────────────────────────────────────────────── */}
      <div
        style={{
          borderBottom: '1px solid var(--border)',
          display: 'flex',
          gap: 0,
          marginBottom: 20,
        }}
      >
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id)}
            style={{
              padding: '8px 16px',
              fontSize: 13,
              fontWeight: activeTab === tab.id ? 600 : 400,
              background: 'transparent',
              border: 'none',
              borderBottom:
                activeTab === tab.id ? '2px solid var(--accent)' : '2px solid transparent',
              color: activeTab === tab.id ? 'var(--text-emphasis)' : 'var(--text-muted)',
              cursor: 'pointer',
              transition: 'color 150ms ease, border-color 150ms ease',
              marginBottom: -1,
            }}
            onMouseEnter={(e) => {
              if (activeTab !== tab.id) e.currentTarget.style.color = 'var(--text-primary)';
            }}
            onMouseLeave={(e) => {
              if (activeTab !== tab.id) e.currentTarget.style.color = 'var(--text-muted)';
            }}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* ── Overview Tab ─────────────────────────────────────────────────── */}
      {activeTab === 'overview' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {/* Hero row: Wave Pipeline + Stats */}
          <div style={{ display: 'grid', gridTemplateColumns: '60fr 40fr', gap: 12 }}>
            {/* Wave pipeline hero */}
            <div style={{ ...CARD, padding: '20px 24px' }}>
              <div
                style={{
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                  marginBottom: 16,
                }}
              >
                Wave Pipeline
              </div>

              {sortedWaves.length > 0 ? (
                <>
                  <WavePipelineLanes waves={sortedWaves} />

                  {/* Gate markers between waves */}
                  <div style={{ marginTop: 16, display: 'flex', flexDirection: 'column', gap: 4 }}>
                    {sortedWaves.map((wave, i) => {
                      if (i === sortedWaves.length - 1) return null;
                      const nextWave = sortedWaves[i + 1];
                      const gateStatus =
                        wave.status === 'completed'
                          ? nextWave.status !== 'pending'
                            ? 'passed'
                            : 'waiting'
                          : 'locked';
                      const gateColor =
                        gateStatus === 'passed'
                          ? 'var(--signal-healthy)'
                          : gateStatus === 'waiting'
                            ? 'var(--accent)'
                            : 'var(--bg-inset)';
                      return (
                        <div
                          key={`gate-${i}`}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            paddingLeft: 70,
                          }}
                        >
                          <div style={{ flex: 1, height: 1, background: 'var(--border)' }} />
                          <span
                            style={{
                              fontFamily: 'var(--font-mono)',
                              fontSize: 9,
                              color: gateColor,
                              background: `color-mix(in srgb, ${gateColor} 9%, transparent)`,
                              border: `1px solid color-mix(in srgb, ${gateColor} 20%, transparent)`,
                              borderRadius: 10,
                              padding: '2px 8px',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            Gate:{' '}
                            {gateStatus === 'passed'
                              ? '✓ Passed'
                              : gateStatus === 'waiting'
                                ? '… Waiting'
                                : '⏸ Locked'}
                          </span>
                          <div style={{ flex: 1, height: 1, background: 'var(--border)' }} />
                        </div>
                      );
                    })}
                  </div>
                </>
              ) : (
                /* No waves: show overall progress */
                <>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
                    <div
                      style={{
                        flex: 1,
                        height: 24,
                        borderRadius: 4,
                        background: 'color-mix(in srgb, white 3%, transparent)',
                        overflow: 'hidden',
                        display: 'flex',
                        position: 'relative',
                      }}
                    >
                      {data.success_count > 0 && (
                        <div
                          style={{
                            flex: data.success_count,
                            background: 'var(--signal-healthy)',
                          }}
                        />
                      )}
                      {data.failed_count > 0 && (
                        <div
                          style={{ flex: data.failed_count, background: 'var(--signal-critical)' }}
                        />
                      )}
                      {pending > 0 && (
                        <div style={{ flex: pending, background: 'var(--bg-inset)' }} />
                      )}
                    </div>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 18,
                        fontWeight: 700,
                        color: 'var(--text-emphasis)',
                        minWidth: 48,
                        textAlign: 'right',
                      }}
                    >
                      {overallPct}%
                    </span>
                  </div>
                  <div
                    style={{
                      fontSize: 11,
                      color: 'var(--text-muted)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >
                    {data.completed_count} / {data.target_count} endpoints processed
                  </div>
                </>
              )}

              {data.started_at && !data.completed_at && (
                <div
                  style={{
                    marginTop: 14,
                    fontSize: 11,
                    color: 'var(--text-muted)',
                    fontFamily: 'var(--font-mono)',
                  }}
                >
                  Running for {durationLabel}
                </div>
              )}
            </div>

            {/* Deployment stats */}
            <div
              style={{ ...CARD, padding: '20px 24px', display: 'flex', flexDirection: 'column' }}
            >
              <div
                style={{
                  fontSize: 10,
                  fontFamily: 'var(--font-mono)',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.06em',
                  color: 'var(--text-muted)',
                  marginBottom: 16,
                }}
              >
                Deployment Stats
              </div>

              {/* Donut gauge for success rate */}
              <div style={{ display: 'flex', alignItems: 'center', gap: 16, marginBottom: 20 }}>
                <div style={{ position: 'relative', width: 80, height: 80, flexShrink: 0 }}>
                  <DonutGauge
                    pct={succeededPct}
                    color={
                      data.failed_count > 0 ? 'var(--signal-critical)' : 'var(--signal-healthy)'
                    }
                    size={80}
                  />
                  <div
                    style={{
                      position: 'absolute',
                      inset: 0,
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      justifyContent: 'center',
                    }}
                  >
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 15,
                        fontWeight: 700,
                        color: 'var(--text-emphasis)',
                        lineHeight: 1,
                      }}
                    >
                      {succeededPct}%
                    </span>
                  </div>
                </div>
                <div>
                  <div style={{ fontSize: 11, color: 'var(--text-muted)', marginBottom: 2 }}>
                    Success Rate
                  </div>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 20,
                      fontWeight: 700,
                      color:
                        data.failed_count > 0 ? 'var(--signal-critical)' : 'var(--signal-healthy)',
                    }}
                  >
                    {succeededPct}%
                  </div>
                </div>
              </div>

              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {[
                  {
                    label: 'Patches Applied',
                    value: data.success_count,
                    color: 'var(--signal-healthy)',
                  },
                  {
                    label: 'Failed Targets',
                    value: data.failed_count,
                    color: data.failed_count > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
                  },
                  { label: 'Duration', value: durationLabel, color: 'var(--text-primary)' },
                  { label: 'Rollbacks', value: 0, color: 'var(--text-muted)' },
                ].map((stat) => (
                  <div
                    key={stat.label}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      paddingBottom: 8,
                      borderBottom: '1px solid var(--border)',
                    }}
                  >
                    <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>{stat.label}</span>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 13,
                        fontWeight: 600,
                        color: stat.color,
                      }}
                    >
                      {stat.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Second row: Endpoint dot map + Timeline */}
          {targets.length > 0 && (
            <div style={{ display: 'grid', gridTemplateColumns: '60fr 40fr', gap: 12 }}>
              {/* Endpoint dot map */}
              <div style={{ ...CARD, padding: '20px 24px' }}>
                <div
                  style={{
                    fontSize: 10,
                    fontFamily: 'var(--font-mono)',
                    fontWeight: 500,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: 'var(--text-muted)',
                    marginBottom: 12,
                  }}
                >
                  Endpoint Status Map
                </div>
                <EndpointDotMap targets={targets} />
                <div style={{ display: 'flex', gap: 12, marginTop: 12, flexWrap: 'wrap' }}>
                  {[
                    { color: 'var(--signal-healthy)', label: 'Succeeded' },
                    { color: 'var(--signal-critical)', label: 'Failed' },
                    { color: 'var(--accent)', label: 'Running' },
                    { color: 'var(--bg-inset)', label: 'Pending' },
                  ].map(({ color, label }) => (
                    <span
                      key={label}
                      style={{
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: 5,
                        fontSize: 10,
                        color: 'var(--text-muted)',
                      }}
                    >
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          background: color,
                          flexShrink: 0,
                        }}
                      />
                      {label}
                    </span>
                  ))}
                </div>
              </div>

              {/* Timeline */}
              <div style={{ ...CARD, padding: '20px 24px', overflowY: 'auto', maxHeight: 280 }}>
                <div
                  style={{
                    fontSize: 10,
                    fontFamily: 'var(--font-mono)',
                    fontWeight: 500,
                    textTransform: 'uppercase',
                    letterSpacing: '0.06em',
                    color: 'var(--text-muted)',
                    marginBottom: 12,
                  }}
                >
                  Timeline
                </div>
                <DeploymentTimeline deployment={data} waves={sortedWaves} targets={targets} />
              </div>
            </div>
          )}
        </div>
      )}

      {/* ── Waves Tab ─────────────────────────────────────────────────────── */}
      {activeTab === 'waves' &&
        (sortedWaves.length === 0 ? (
          <div
            style={{
              display: 'flex',
              height: 120,
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 8,
              border: '1px solid var(--border)',
              color: 'var(--text-muted)',
              fontSize: 13,
            }}
          >
            No wave data available.
          </div>
        ) : (
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))',
              gap: 12,
            }}
          >
            {sortedWaves.map((wave) => {
              const wavePct =
                wave.target_count > 0
                  ? Math.round(((wave.success_count + wave.failed_count) / wave.target_count) * 100)
                  : 0;
              const wc = getWaveConfig(wave.status);
              const isRunning = wave.status === 'running';

              return (
                <div
                  key={wave.id}
                  style={{
                    ...CARD,
                    padding: '16px 18px',
                    borderColor: isRunning
                      ? 'color-mix(in srgb, var(--accent) 35%, transparent)'
                      : 'var(--border)',
                  }}
                >
                  <div
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'flex-start',
                      marginBottom: 12,
                    }}
                  >
                    <div>
                      <div
                        style={{
                          fontSize: 14,
                          fontWeight: 600,
                          color: 'var(--text-emphasis)',
                          marginBottom: 3,
                        }}
                      >
                        Wave {wave.wave_number}
                      </div>
                      <div
                        style={{
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: 'var(--text-muted)',
                        }}
                      >
                        {wave.target_count} endpoints · {wave.percentage}% of fleet
                      </div>
                    </div>
                    <span
                      style={{
                        fontFamily: 'var(--font-mono)',
                        fontSize: 13,
                        fontWeight: 700,
                        color: wc.textColor,
                      }}
                    >
                      {wavePct}%
                    </span>
                  </div>

                  <div
                    style={{
                      height: 5,
                      borderRadius: 3,
                      background: 'var(--bg-inset)',
                      overflow: 'hidden',
                      marginBottom: 10,
                    }}
                  >
                    <div
                      style={{
                        width: `${wavePct}%`,
                        height: '100%',
                        background: wc.barColor,
                        borderRadius: 3,
                        transition: 'width 0.4s ease',
                      }}
                    />
                  </div>

                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 11,
                      color: wc.textColor,
                      marginBottom: 4,
                    }}
                  >
                    {isRunning
                      ? `${wc.label} \u2014 ${wave.success_count + wave.failed_count}/${wave.target_count}`
                      : wc.label}
                  </div>
                  <div
                    style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 10,
                      color: 'var(--text-muted)',
                    }}
                  >
                    {wave.success_count} succeeded · {wave.failed_count} failed
                  </div>
                </div>
              );
            })}
          </div>
        ))}

      {/* ── Endpoints Tab ─────────────────────────────────────────────────── */}
      {activeTab === 'endpoints' &&
        (uniqueTargets.length === 0 ? (
          <div
            style={{
              display: 'flex',
              height: 120,
              alignItems: 'center',
              justifyContent: 'center',
              borderRadius: 8,
              border: '1px solid var(--border)',
              color: 'var(--text-muted)',
              fontSize: 13,
            }}
          >
            No endpoint targets yet.
          </div>
        ) : (
          <div style={{ ...CARD, overflow: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', minWidth: 700 }}>
              <thead>
                <tr>
                  <th style={{ ...TH, width: 32 }} />
                  <th style={TH}>Hostname</th>
                  <th style={TH}>Status</th>
                  <th style={TH}>Exit Code</th>
                  <th style={TH}>Start Time</th>
                  <th style={TH}>Duration</th>
                  <th style={TH}>Error</th>
                </tr>
              </thead>
              <tbody>
                {uniqueTargets.map((target) => (
                  <TargetRow
                    key={target.id}
                    target={target}
                    expanded={expandedTargets.has(target.id)}
                    onToggle={() => toggleTarget(target.id)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        ))}

      {/* ── Patches Tab ───────────────────────────────────────────────────── */}
      {activeTab === 'patches' && (
        <div style={CARD}>
          {!patches ||
          (patches as components['schemas']['DeploymentPatchSummary'][]).length === 0 ? (
            <div
              style={{
                display: 'flex',
                height: 120,
                alignItems: 'center',
                justifyContent: 'center',
                color: 'var(--text-muted)',
                fontSize: 13,
              }}
            >
              No patch data available.
            </div>
          ) : (
            <div style={{ overflow: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                  <tr>
                    <th style={TH}>Patch Name</th>
                    <th style={TH}>Version</th>
                    <th style={TH}>Severity</th>
                    <th style={TH}>Success Rate</th>
                  </tr>
                </thead>
                <tbody>
                  {(patches as components['schemas']['DeploymentPatchSummary'][]).map((p) => {
                    const rate =
                      p.total_targets > 0
                        ? Math.round((p.success_count / p.total_targets) * 100)
                        : 0;
                    return (
                      <tr
                        key={p.patch_id}
                        style={{ transition: 'background 0.1s' }}
                        onMouseEnter={(e) =>
                          (e.currentTarget.style.background = 'var(--bg-card-hover)')
                        }
                        onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                      >
                        <td style={{ ...TD, color: 'var(--text-primary)' }}>{p.patch_title}</td>
                        <td
                          style={{
                            ...TD,
                            color: 'var(--text-muted)',
                            fontFamily: 'var(--font-mono)',
                            fontSize: 11,
                          }}
                        >
                          {p.patch_version ?? '\u2014'}
                        </td>
                        <td style={TD}>
                          {p.patch_severity ? (
                            <SeverityText severity={p.patch_severity} />
                          ) : (
                            <span style={{ color: 'var(--text-muted)', fontSize: 10 }}>\u2014</span>
                          )}
                        </td>
                        <td style={TD}>
                          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                            <div
                              style={{
                                width: 120,
                                height: 5,
                                borderRadius: 3,
                                background: 'var(--bg-inset)',
                                overflow: 'hidden',
                              }}
                            >
                              <div
                                style={{
                                  width: `${rate}%`,
                                  height: '100%',
                                  background: 'var(--signal-healthy)',
                                  borderRadius: 3,
                                }}
                              />
                            </div>
                            <span
                              style={{
                                fontFamily: 'var(--font-mono)',
                                fontSize: 11,
                                color: 'var(--text-muted)',
                              }}
                            >
                              {p.success_count}/{p.total_targets} ({rate}%)
                            </span>
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ── Timeline Tab ─────────────────────────────────────────────────── */}
      {activeTab === 'timeline' && (
        <div style={{ ...CARD, padding: '20px 24px' }}>
          <div
            style={{
              fontSize: 11,
              fontFamily: 'var(--font-mono)',
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              color: 'var(--text-muted)',
              marginBottom: 16,
            }}
          >
            Deployment Event Timeline
          </div>
          <DeploymentTimeline deployment={data} waves={sortedWaves} targets={targets} />
        </div>
      )}

      {/* ── Dialogs ──────────────────────────────────────────────────────── */}

      <Dialog open={cancelDialogOpen} onOpenChange={setCancelDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Cancel Deployment</DialogTitle>
          </DialogHeader>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: '8px 0 16px' }}>
            Are you sure you want to cancel this deployment? Endpoints that have already completed
            will not be rolled back.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button variant="outline" onClick={() => setCancelDialogOpen(false)}>
              Keep Running
            </Button>
            <Button
              variant="destructive"
              onClick={handleCancel}
              disabled={cancelMutation.isPending}
            >
              {cancelMutation.isPending ? 'Cancelling...' : 'Confirm Cancel'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={retryDialogOpen} onOpenChange={setRetryDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Retry Failed Endpoints</DialogTitle>
          </DialogHeader>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: '8px 0 16px' }}>
            This will retry the deployment on {data.failed_count} failed endpoint(s).
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button variant="outline" onClick={() => setRetryDialogOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleRetry} disabled={retryMutation.isPending}>
              {retryMutation.isPending ? 'Retrying...' : 'Confirm Retry'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={rollbackDialogOpen} onOpenChange={setRollbackDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Rollback Deployment</DialogTitle>
          </DialogHeader>
          <p style={{ fontSize: 13, color: 'var(--text-secondary)', margin: '8px 0 16px' }}>
            This will rollback all applied patches from this deployment. This action cannot be
            undone.
          </p>
          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button variant="outline" onClick={() => setRollbackDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleRollback}
              disabled={rollbackMutation.isPending}
            >
              {rollbackMutation.isPending ? 'Rolling back...' : 'Confirm Rollback'}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}
