/* eslint-disable @typescript-eslint/no-explicit-any */
import { useParams, Link, useLocation, useSearchParams } from 'react-router';
import { toast } from 'sonner';
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from '@patchiq/ui';
import { RefreshCw, Download, Info, ArrowLeft } from 'lucide-react';
import { useCan } from '../../app/auth/AuthContext';
import { useComplianceFramework, useTriggerEvaluation } from '../../api/hooks/useCompliance';
import { timeAgo } from '../../lib/time';
import { OverviewTab } from './components/overview-tab';
import { ControlsTab } from './components/controls-tab';
import { EndpointsTab, type ControlDef } from './components/endpoints-tab';
import { SlaTab } from './components/sla-tab';
// EvidenceTab removed — feature not yet implemented

// ─── Framework display names ──────────────────────────────────────────────────

const FRAMEWORK_NAMES: Record<string, string> = {
  cis: 'CIS Controls v8',
  hipaa: 'HIPAA Security Rule',
  nist_800_53: 'NIST 800-53',
  pci_dss: 'PCI DSS v4.0',
  iso_27001: 'ISO 27001',
  soc_2: 'SOC 2 Type II',
};

function getFrameworkDisplayName(id: string): string {
  const lower = id.toLowerCase();
  if (FRAMEWORK_NAMES[lower]) return FRAMEWORK_NAMES[lower];
  // Fallback: capitalize each word, replace underscores/hyphens with spaces
  return id
    .replace(/[-_]/g, ' ')
    .split(' ')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function getComplianceStatus(score: number): { label: string; color: string } {
  if (score >= 95) return { label: 'Compliant', color: 'var(--accent)' };
  if (score >= 80) return { label: 'Needs Improvement', color: 'var(--signal-warning)' };
  return { label: 'Non-Compliant', color: 'var(--signal-critical)' };
}

// ─── Design tokens ───────────────────────────────────────────────────────────

const mono: React.CSSProperties = { fontFamily: 'var(--font-mono)' };

const chipStyle: React.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: 4,
  padding: '2px 8px',
  border: '1px solid var(--border)',
  borderRadius: 4,
  fontFamily: 'var(--font-mono)',
  fontSize: 11,
  fontWeight: 500,
  color: 'var(--text-muted)',
  background: 'var(--bg-inset)',
};

// ─── Tabs ─────────────────────────────────────────────────────────────────────

const TABS = [
  { value: 'overview', label: 'Overview' },
  { value: 'controls', label: 'Controls' },
  { value: 'endpoints', label: 'Endpoints' },
  { value: 'sla', label: 'SLA Tracking' },
] as const;

type TabValue = (typeof TABS)[number]['value'];

// ─── Health Strip ─────────────────────────────────────────────────────────────

function HealthStrip({
  passing,
  total,
  endpoints,
  failing,
  lastEval,
  scoreVal,
  statusColor,
}: {
  passing: number;
  total: number;
  endpoints: number;
  failing: number;
  lastEval: string;
  scoreVal: number;
  statusColor: string;
}) {
  const passPct = total > 0 ? (passing / total) * 100 : 0;

  const metrics = [
    {
      label: 'Score',
      value: `${scoreVal}%`,
      valueColor: statusColor,
      bar: passPct,
      barColor: statusColor,
    },
    {
      label: 'Controls',
      value: `${passing} / ${total} passing`,
      valueColor: failing > 0 ? 'var(--text-emphasis)' : 'var(--accent)',
      bar: null,
      barColor: null,
    },
    {
      label: 'Endpoints',
      value: String(endpoints),
      valueColor: 'var(--text-emphasis)',
      bar: null,
      barColor: null,
    },
    {
      label: 'Last Eval',
      value: lastEval,
      valueColor: 'var(--text-primary)',
      bar: null,
      barColor: null,
    },
  ];

  return (
    <div
      style={{
        display: 'flex',
        height: 56,
        border: '1px solid var(--border)',
        borderRadius: 8,
        overflow: 'hidden',
        background: 'var(--bg-card)',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      {metrics.map((m, i) => (
        <div
          key={m.label}
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            gap: 12,
            padding: '0 20px',
            borderLeft: i > 0 ? '1px solid var(--border)' : 'none',
          }}
        >
          <span
            style={{
              ...mono,
              fontSize: 10,
              fontWeight: 600,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              color: 'var(--text-muted)',
              flexShrink: 0,
            }}
          >
            {m.label}
          </span>
          <span
            style={{
              ...mono,
              fontSize: 13,
              fontWeight: 700,
              color: m.valueColor,
              flexShrink: 0,
            }}
          >
            {m.value}
          </span>
          {m.bar !== null && m.barColor && (
            <div
              style={{
                flex: 1,
                height: 4,
                borderRadius: 2,
                background: 'var(--bg-inset)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  height: '100%',
                  borderRadius: 2,
                  width: `${m.bar}%`,
                  background: m.barColor,
                  transition: 'width 0.4s ease',
                }}
              />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// ─── Graceful error state ─────────────────────────────────────────────────────

function FrameworkErrorState({
  id,
  onEvaluate,
  isEvaluating,
}: {
  id: string;
  onEvaluate: () => void;
  isEvaluating: boolean;
}) {
  const can = useCan();
  const displayName = getFrameworkDisplayName(id);

  return (
    <div
      style={{
        padding: '20px 24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Header stub */}
      <div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            marginBottom: 8,
          }}
        >
          <h1
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 28,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              margin: 0,
              letterSpacing: '-0.02em',
            }}
          >
            {displayName}
          </h1>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              type="button"
              onClick={onEvaluate}
              disabled={isEvaluating || !can('compliance', 'create')}
              title={!can('compliance', 'create') ? "You don't have permission" : undefined}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                border: 'none',
                borderRadius: 6,
                ...mono,
                fontSize: 12,
                fontWeight: 600,
                cursor: isEvaluating || !can('compliance', 'create') ? 'not-allowed' : 'pointer',
                opacity: isEvaluating || !can('compliance', 'create') ? 0.6 : 1,
                transition: 'opacity 0.15s',
              }}
            >
              <RefreshCw
                size={12}
                style={{ animation: isEvaluating ? 'spin 1s linear infinite' : undefined }}
              />
              {isEvaluating ? 'Running…' : 'Run Evaluation'}
            </button>
          </div>
        </div>
        <div style={{ ...mono, fontSize: 12, color: 'var(--text-muted)' }}>
          Framework ID: <span style={{ color: 'var(--text-secondary)' }}>{id}</span>
        </div>
      </div>

      {/* Error state */}
      <div
        style={{
          background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
          border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
          borderRadius: 8,
          padding: '24px',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 12,
          textAlign: 'center',
        }}
      >
        <div
          style={{
            width: 40,
            height: 40,
            borderRadius: '50%',
            background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
            border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          <svg
            width="18"
            height="18"
            viewBox="0 0 24 24"
            fill="none"
            stroke="var(--signal-critical)"
            strokeWidth="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
        </div>
        <div>
          <div
            style={{
              ...mono,
              fontSize: 13,
              fontWeight: 600,
              color: 'var(--text-primary)',
              marginBottom: 4,
            }}
          >
            Framework data unavailable
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            Run an evaluation to generate compliance data for this framework.
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            type="button"
            onClick={onEvaluate}
            disabled={isEvaluating || !can('compliance', 'create')}
            title={!can('compliance', 'create') ? "You don't have permission" : undefined}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              background: 'var(--accent)',
              color: 'var(--btn-accent-text, #000)',
              border: 'none',
              borderRadius: 6,
              ...mono,
              fontSize: 11,
              fontWeight: 600,
              cursor: isEvaluating || !can('compliance', 'create') ? 'not-allowed' : 'pointer',
              opacity: isEvaluating || !can('compliance', 'create') ? 0.6 : 1,
              transition: 'opacity 0.15s',
            }}
            onMouseEnter={(e) => {
              if (!isEvaluating) e.currentTarget.style.opacity = '0.85';
            }}
            onMouseLeave={(e) => {
              if (!isEvaluating) e.currentTarget.style.opacity = '1';
            }}
          >
            <RefreshCw
              size={11}
              style={{ animation: isEvaluating ? 'spin 1s linear infinite' : undefined }}
            />
            {isEvaluating ? 'Running…' : 'Run Evaluation'}
          </button>
          <Link
            to="/compliance"
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 6,
              padding: '6px 12px',
              background: 'transparent',
              border: '1px solid var(--border)',
              borderRadius: 6,
              ...mono,
              fontSize: 11,
              color: 'var(--text-secondary)',
              textDecoration: 'none',
              transition: 'all 0.15s',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-hover)';
              e.currentTarget.style.color = 'var(--text-primary)';
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border)';
              e.currentTarget.style.color = 'var(--text-secondary)';
            }}
          >
            Back to Compliance
          </Link>
        </div>
      </div>
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export const FrameworkDetailPage = () => {
  const can = useCan();
  const { id } = useParams<{ id: string }>();
  const location = useLocation();
  const { data: detail, isLoading, isError } = useComplianceFramework(id ?? '');
  const evaluate = useTriggerEvaluation();

  const [searchParams, setSearchParams] = useSearchParams();

  // If URL has a hash fragment (e.g. #CIS-6.1 from overdue table click),
  // auto-switch to the controls tab so the user lands on the right view.
  const hashControl = location.hash ? location.hash.slice(1) : '';
  const tabFromUrl = searchParams.get('tab');
  const validTabs: TabValue[] = ['overview', 'controls', 'endpoints', 'sla'];
  const activeTab: TabValue = hashControl
    ? 'controls'
    : tabFromUrl && validTabs.includes(tabFromUrl as TabValue)
      ? (tabFromUrl as TabValue)
      : 'overview';

  const setActiveTab = (tab: TabValue) => {
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.set('tab', tab);
        return next;
      },
      { replace: true },
    );
  };

  if (isLoading) {
    return (
      <div
        style={{
          padding: '20px 24px',
          display: 'flex',
          flexDirection: 'column',
          gap: 16,
          background: 'var(--bg-page)',
          minHeight: '100%',
        }}
      >
        {[48, 56, 400].map((h, i) => (
          <div
            key={i}
            style={{
              height: h,
              borderRadius: 8,
              background: 'var(--bg-card)',
              border: '1px solid var(--border)',
              animation: 'pulse 1.5s ease-in-out infinite',
            }}
          />
        ))}
      </div>
    );
  }

  if (isError || !detail) {
    return (
      <FrameworkErrorState
        id={id ?? 'unknown'}
        onEvaluate={() => evaluate.mutate()}
        isEvaluating={evaluate.isPending}
      />
    );
  }

  const categories = detail.categories ?? [];
  const allControls = categories.flatMap((c) => c.controls ?? []);
  const totalControls = allControls.length;
  const passing = allControls.filter((c) => c.status === 'pass').length;
  const failing = allControls.filter((c) => c.status === 'fail').length;
  const naControls = allControls.filter((c) => c.status === 'na').length;
  const evaluatedControls = totalControls - naControls;
  // For SLA-based frameworks, endpoint_scores has per-endpoint rows.
  // For non-SLA frameworks (custom check types), derive from control results.
  const endpointScoreCount = detail.endpoint_scores?.length ?? 0;
  const endpointsFromControls = Math.max(0, ...allControls.map((c) => c.total_endpoints ?? 0));
  const endpointsEvaluated = endpointScoreCount > 0 ? endpointScoreCount : endpointsFromControls;

  const scoreVal = detail.score ? Math.round(parseFloat(detail.score.score)) : 0;
  const { label: statusLabel, color: statusColor } = getComplianceStatus(scoreVal);
  const lastEval = timeAgo(detail.score?.evaluated_at);

  return (
    <div
      style={{
        padding: '20px 24px',
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        background: 'var(--bg-page)',
      }}
    >
      {/* Breadcrumb */}
      <nav
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          ...mono,
          fontSize: 11,
          color: 'var(--text-muted)',
        }}
      >
        <Link
          to="/compliance"
          style={{
            color: 'var(--text-muted)',
            textDecoration: 'none',
            display: 'inline-flex',
            alignItems: 'center',
          }}
          onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
          onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-muted)')}
          title="Back to Compliance"
        >
          <ArrowLeft size={12} style={{ marginRight: 4 }} />
        </Link>
        <Link
          to="/"
          style={{ color: 'var(--text-muted)', textDecoration: 'none' }}
          onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
          onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-muted)')}
        >
          Home
        </Link>
        <svg
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M9 18l6-6-6-6" />
        </svg>
        <Link
          to="/compliance"
          style={{ color: 'var(--text-muted)', textDecoration: 'none' }}
          onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--text-secondary)')}
          onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-muted)')}
        >
          Compliance
        </Link>
        <svg
          width="12"
          height="12"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <path d="M9 18l6-6-6-6" />
        </svg>
        <span style={{ color: 'var(--text-secondary)', fontWeight: 500 }}>
          {detail.framework?.name}
        </span>
      </nav>

      {/* ── Header (2-row, no card) ── */}
      <div>
        {/* Row 1: Framework name + actions */}
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: 16,
            marginBottom: 8,
          }}
        >
          <h1
            style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 28,
              fontWeight: 600,
              color: 'var(--text-emphasis)',
              margin: 0,
              letterSpacing: '-0.02em',
            }}
          >
            {detail.framework?.name}
          </h1>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
            <button
              type="button"
              onClick={() =>
                evaluate.mutate(undefined, {
                  onSuccess: (data) => {
                    const d = data as
                      | { frameworks_evaluated?: number; total_evaluations?: number }
                      | undefined;
                    toast.success(
                      `Evaluation complete — ${d?.total_evaluations ?? 0} controls evaluated`,
                    );
                  },
                  onError: (err) =>
                    toast.error(err instanceof Error ? err.message : 'Evaluation failed'),
                })
              }
              disabled={evaluate.isPending || !can('compliance', 'create')}
              title={!can('compliance', 'create') ? "You don't have permission" : undefined}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '7px 14px',
                background: 'var(--accent)',
                color: 'var(--btn-accent-text, #000)',
                border: 'none',
                borderRadius: 6,
                ...mono,
                fontSize: 12,
                fontWeight: 600,
                cursor: evaluate.isPending ? 'not-allowed' : 'pointer',
                opacity: evaluate.isPending ? 0.6 : 1,
                transition: 'opacity 0.15s',
              }}
              onMouseEnter={(e) => {
                if (!evaluate.isPending) e.currentTarget.style.opacity = '0.85';
              }}
              onMouseLeave={(e) => {
                if (!evaluate.isPending) e.currentTarget.style.opacity = '1';
              }}
            >
              <RefreshCw
                size={12}
                style={{ animation: evaluate.isPending ? 'spin 1s linear infinite' : undefined }}
              />
              {evaluate.isPending ? 'Evaluating…' : 'Evaluate Now'}
            </button>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    disabled
                    style={{
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: 6,
                      padding: '7px 14px',
                      background: 'transparent',
                      color: 'var(--text-muted)',
                      border: '1px solid var(--border)',
                      borderRadius: 6,
                      ...mono,
                      fontSize: 12,
                      fontWeight: 500,
                      cursor: 'not-allowed',
                    }}
                  >
                    <Download size={12} />
                    Export Report
                    <span
                      style={{
                        fontSize: 9,
                        color: 'var(--text-muted)',
                        textTransform: 'uppercase',
                        letterSpacing: '0.06em',
                      }}
                    >
                      Soon
                    </span>
                  </button>
                </TooltipTrigger>
                <TooltipContent>Report export will be available in a future release</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>

        {/* Row 2: Status dot + score + metadata chips */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          {/* Status dot */}
          <span style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
            <span
              style={{
                width: 7,
                height: 7,
                borderRadius: '50%',
                background: statusColor,
                display: 'inline-block',
              }}
            />
            <span
              style={{
                ...mono,
                fontSize: 12,
                fontWeight: 700,
                color: statusColor,
              }}
            >
              {statusLabel}
            </span>
          </span>

          {/* Score chip */}
          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
            <span
              style={{
                ...mono,
                fontSize: 20,
                fontWeight: 600,
                color: statusColor,
              }}
            >
              {scoreVal}%
            </span>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span style={{ display: 'inline-flex', cursor: 'help' }}>
                    <Info size={14} style={{ color: 'var(--text-muted)' }} />
                  </span>
                </TooltipTrigger>
                <TooltipContent side="bottom" style={{ maxWidth: 280, fontSize: 12 }}>
                  Score reflects CVE remediation rate within SLA deadlines. Control pass rate shows
                  simulated control statuses.
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </span>

          {/* Metadata chips */}
          <span style={chipStyle}>
            {evaluatedControls} of {totalControls} controls evaluated
          </span>
          <span style={chipStyle}>Last eval: {lastEval}</span>
          {detail.framework?.version && <span style={chipStyle}>v{detail.framework.version}</span>}
        </div>
      </div>

      {/* ── Health Strip ── */}
      <HealthStrip
        passing={passing}
        total={evaluatedControls}
        endpoints={endpointsEvaluated}
        failing={failing}
        lastEval={lastEval}
        scoreVal={scoreVal}
        statusColor={statusColor}
      />

      {/* ── Tabs ── */}
      <div>
        <div
          style={{
            display: 'flex',
            borderBottom: '1px solid var(--border)',
            gap: 0,
            marginBottom: 16,
          }}
        >
          {TABS.map((tab) => {
            const isActive = activeTab === tab.value;
            return (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                style={{
                  fontSize: 13,
                  fontWeight: isActive ? 600 : 400,
                  color: isActive ? 'var(--text-emphasis)' : 'var(--text-muted)',
                  padding: '8px 16px',
                  background: 'none',
                  border: 'none',
                  borderBottom: isActive ? '2px solid var(--accent)' : '2px solid transparent',
                  cursor: 'pointer',
                  transition: 'color 150ms ease, border-color 150ms ease',
                  marginBottom: -1,
                }}
                onMouseEnter={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-secondary)';
                }}
                onMouseLeave={(e) => {
                  if (!isActive)
                    (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)';
                }}
              >
                {tab.label}
              </button>
            );
          })}
        </div>

        {activeTab === 'overview' && (
          <OverviewTab
            categories={categories}
            frameworkId={id!}
            scoreVal={scoreVal}
            statusColor={statusColor}
            passing={passing}
            failing={failing}
            totalControls={totalControls}
            score={detail.score}
          />
        )}
        {activeTab === 'controls' && <ControlsTab frameworkId={id!} />}
        {activeTab === 'endpoints' && (
          <EndpointsTab
            endpoints={detail.non_compliant_endpoints}
            frameworkId={id!}
            controls={allControls.map(
              (c) =>
                ({
                  control_id: c.control_id,
                  name: c.name,
                  description: c.description,
                  status: c.status,
                  passing_endpoints: c.passing_endpoints,
                  total_endpoints: c.total_endpoints,
                  remediation_hint: c.remediation_hint,
                }) as ControlDef,
            )}
          />
        )}
        {activeTab === 'sla' && <SlaTab frameworkId={id!} />}
        {/* EvidenceTab removed — not yet implemented */}
      </div>
    </div>
  );
};
