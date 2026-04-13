import { useState } from 'react';
import { useParams, useNavigate } from 'react-router';
import { Pencil, Trash2, Copy, RefreshCw, ArrowRight, MoreHorizontal } from 'lucide-react';
import { Skeleton } from '@patchiq/ui';
import { toast } from 'sonner';
import {
  usePolicy,
  useEvaluatePolicy,
  useDeletePolicy,
  useTogglePolicy,
} from '../../api/hooks/usePolicies';
import { DeploymentWizard } from '../../components/DeploymentWizard';
import type { DeploymentWizardInitialState } from '../../types/deployment-wizard';
import { OverviewTab } from './tabs/OverviewTab';
import { PatchScopeTab } from './tabs/PatchScopeTab';
import { GroupsEndpointsTab } from './tabs/GroupsEndpointsTab';
import { EvalHistoryTab } from './tabs/EvalHistoryTab';
import { DeploymentHistoryTab } from './tabs/DeploymentHistoryTab';
import { ScheduleTab } from './tabs/ScheduleTab';
import { useCan } from '../../app/auth/AuthContext';
import { timeAgo } from '../../lib/time';

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type PolicyDetail = any;

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'patch-scope', label: 'Patch Scope' },
  { id: 'groups-endpoints', label: 'Groups & Endpoints' },
  { id: 'eval-history', label: 'Evaluation History' },
  { id: 'deploy-history', label: 'Deployment History' },
  { id: 'schedule', label: 'Schedule' },
] as const;

type TabId = (typeof tabs)[number]['id'];

// ── Mode colors ────────────────────────────────────────────────────────────────

const modeColorMap: Record<string, string> = {
  automatic: 'var(--signal-healthy)',
  manual: 'var(--accent)',
  advisory: 'var(--text-muted)',
};

const modeLabelMap: Record<string, string> = {
  automatic: 'Automatic',
  manual: 'Manual',
  advisory: 'Advisory',
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

// ── Component ──────────────────────────────────────────────────────────────────

export const PolicyDetailPage = () => {
  const can = useCan();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: rawPolicy, isLoading, isError, refetch } = usePolicy(id!);
  const evaluate = useEvaluatePolicy(id!);
  const deletePolicy = useDeletePolicy(id!);
  const toggle = useTogglePolicy(id!);
  const [activeTab, setActiveTab] = useState<TabId>('overview');
  const [wizardOpen, setWizardOpen] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);
  const wizardInitial: DeploymentWizardInitialState = {
    sourceType: 'policy',
    policyId: id!,
    startStep: 'targets',
  };

  const policy = rawPolicy as PolicyDetail | undefined;

  if (!id)
    return (
      <div style={{ padding: 24, fontSize: 13, color: 'var(--signal-critical)' }}>
        Policy not found
      </div>
    );

  if (isLoading) {
    return (
      <div style={{ padding: 24, display: 'flex', flexDirection: 'column', gap: 12 }}>
        <Skeleton className="h-7 w-52" />
        <Skeleton className="h-14 rounded-xl" />
        <Skeleton className="h-[200px] rounded-xl" />
      </div>
    );
  }

  if (isError || !policy) {
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
          Failed to load policy.{' '}
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

  const handleDelete = async () => {
    if (!window.confirm(`Delete policy "${policy.name}"? This cannot be undone.`)) return;
    await deletePolicy.mutateAsync();
    navigate('/policies');
  };

  const modeColor = modeColorMap[policy.mode] ?? 'var(--text-muted)';
  const modeLabel = modeLabelMap[policy.mode] ?? policy.mode;

  const lastEval = policy.recent_evaluations?.[0];
  const endpointCount = policy.target_endpoints_count ?? (policy.matched_endpoints ?? []).length;
  const patchCount = lastEval?.matched_patches ?? null;

  const passRate =
    lastEval && lastEval.in_scope_endpoints > 0
      ? Math.round((lastEval.compliant_count / lastEval.in_scope_endpoints) * 100)
      : null;

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
          {policy.name}
        </h1>

        {/* Actions */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexShrink: 0 }}>
          <button
            type="button"
            onClick={() =>
              evaluate.mutate(undefined, {
                onSuccess: () => toast.success('Policy evaluation started'),
                onError: (err) =>
                  toast.error(
                    `Evaluation failed: ${(err as { message?: string })?.message || 'Unknown error'}`,
                  ),
              })
            }
            disabled={evaluate.isPending || !can('policies', 'read')}
            title={!can('policies', 'read') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              padding: '6px 12px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              cursor: evaluate.isPending || !can('policies', 'read') ? 'not-allowed' : 'pointer',
              background: 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              border: 'none',
              opacity: evaluate.isPending || !can('policies', 'read') ? 0.7 : 1,
            }}
          >
            <RefreshCw style={{ width: 12, height: 12 }} />
            {evaluate.isPending ? 'Evaluating...' : 'Evaluate Now'}
          </button>
          <button
            type="button"
            onClick={() => setWizardOpen(true)}
            disabled={!can('deployments', 'create')}
            title={!can('deployments', 'create') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              padding: '6px 12px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              cursor: !can('deployments', 'create') ? 'not-allowed' : 'pointer',
              background: 'color-mix(in srgb, var(--accent) 8%, transparent)',
              color: 'var(--accent)',
              border: '1px solid color-mix(in srgb, var(--accent) 25%, transparent)',
              opacity: !can('deployments', 'create') ? 0.5 : 1,
            }}
          >
            <ArrowRight style={{ width: 12, height: 12 }} />
            Deploy Now
          </button>
          <button
            type="button"
            onClick={() => navigate(`/policies/${id}/edit`)}
            disabled={!can('policies', 'update')}
            title={!can('policies', 'update') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 5,
              padding: '6px 12px',
              borderRadius: 6,
              fontSize: 12,
              fontWeight: 500,
              cursor: !can('policies', 'update') ? 'not-allowed' : 'pointer',
              background: 'transparent',
              color: 'var(--text-primary)',
              border: '1px solid var(--border)',
              textDecoration: 'none',
              opacity: !can('policies', 'update') ? 0.5 : 1,
            }}
          >
            <Pencil style={{ width: 12, height: 12 }} />
            Edit
          </button>
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
                <button
                  type="button"
                  onClick={() => {
                    setMoreOpen(false);
                  }}
                  style={{
                    display: 'flex',
                    width: '100%',
                    alignItems: 'center',
                    gap: 8,
                    padding: '8px 12px',
                    fontSize: 12,
                    color: 'var(--text-primary)',
                    background: 'transparent',
                    border: 'none',
                    cursor: 'pointer',
                    textAlign: 'left',
                  }}
                >
                  <Copy style={{ width: 12, height: 12 }} />
                  Clone Policy
                </button>
                <button
                  type="button"
                  onClick={() => {
                    setMoreOpen(false);
                    handleDelete();
                  }}
                  disabled={deletePolicy.isPending || !can('policies', 'delete')}
                  title={!can('policies', 'delete') ? "You don't have permission" : undefined}
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
                    cursor:
                      deletePolicy.isPending || !can('policies', 'delete')
                        ? 'not-allowed'
                        : 'pointer',
                    textAlign: 'left',
                    opacity: !can('policies', 'delete') ? 0.5 : 1,
                  }}
                >
                  <Trash2 style={{ width: 12, height: 12 }} />
                  {deletePolicy.isPending ? 'Deleting...' : 'Delete Policy'}
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
        {/* Mode */}
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <span
            style={{
              width: 7,
              height: 7,
              borderRadius: '50%',
              background: modeColor,
              flexShrink: 0,
            }}
          />
          <span style={{ fontSize: 13, fontWeight: 500, color: modeColor }}>{modeLabel}</span>
        </span>

        {/* Enabled state */}
        <span
          style={{
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            color: policy.enabled ? 'var(--signal-healthy)' : 'var(--text-muted)',
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderRadius: 4,
            padding: '2px 8px',
            cursor: can('policies', 'update') ? 'pointer' : 'not-allowed',
            opacity: !can('policies', 'update') ? 0.5 : 1,
          }}
          onClick={() => can('policies', 'update') && toggle.mutate(!policy.enabled)}
          title={!can('policies', 'update') ? "You don't have permission" : 'Toggle policy'}
        >
          {policy.enabled ? 'Enabled ●' : 'Disabled ○'}
        </span>

        {/* Schedule */}
        {policy.schedule_type === 'recurring' && policy.schedule_cron && (
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
            {policy.schedule_cron}
          </span>
        )}

        {/* Created */}
        {policy.created_at && (
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
            Created {timeAgo(policy.created_at)}
          </span>
        )}

        {/* Endpoint count */}
        {endpointCount > 0 && (
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
            {endpointCount} endpoints
          </span>
        )}
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
            label: 'Pass Rate',
            value: passRate != null ? `${passRate}%` : '\u2014',
            fill: passRate,
            color: 'var(--signal-healthy)',
          },
          {
            label: 'Endpoints',
            value: String(endpointCount),
            fill: null,
            color: 'var(--text-primary)',
          },
          {
            label: 'Patches',
            value: patchCount != null ? `${patchCount} matched` : '\u2014',
            fill: null,
            color: 'var(--text-primary)',
          },
          {
            label: 'Next Run',
            value: policy.schedule_type === 'recurring' ? 'Scheduled' : 'On demand',
            fill: null,
            color: policy.schedule_type === 'recurring' ? 'var(--accent)' : 'var(--text-muted)',
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
                  fontWeight: 600,
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

      {/* ── Tab content ──────────────────────────────────────────────────── */}
      {activeTab === 'overview' && <OverviewTab policy={policy} DonutGauge={DonutGauge} />}
      {activeTab === 'patch-scope' && <PatchScopeTab policy={policy} />}
      {activeTab === 'groups-endpoints' && <GroupsEndpointsTab policy={policy} />}
      {activeTab === 'eval-history' && <EvalHistoryTab policy={policy} />}
      {activeTab === 'deploy-history' && <DeploymentHistoryTab policy={policy} />}
      {activeTab === 'schedule' && <ScheduleTab policy={policy} />}

      <DeploymentWizard
        open={wizardOpen}
        onOpenChange={setWizardOpen}
        initialState={wizardInitial}
      />
    </div>
  );
};
