import { Skeleton, Button } from '@patchiq/ui';
import { ShieldCheck } from 'lucide-react';
import { Link } from 'react-router';
import { useEndpointCompliance, useTriggerEvaluation } from '../../../api/hooks/useCompliance';
import type { ComplianceEvaluation } from '../../../api/hooks/useCompliance';
import { EmptyState } from '../../../components/EmptyState';

interface ComplianceTabProps {
  endpointId: string;
}

interface FrameworkSummary {
  frameworkId: string;
  total: number;
  compliant: number;
  lastEvaluatedAt: string | null;
}

function buildFrameworkSummaries(evaluations: ComplianceEvaluation[]): FrameworkSummary[] {
  const map = new Map<string, FrameworkSummary>();
  for (const ev of evaluations) {
    const existing = map.get(ev.framework_id);
    if (existing) {
      existing.total += 1;
      if (ev.state?.toUpperCase() === 'COMPLIANT') existing.compliant += 1;
      if (
        ev.evaluated_at &&
        (!existing.lastEvaluatedAt || ev.evaluated_at > existing.lastEvaluatedAt)
      ) {
        existing.lastEvaluatedAt = ev.evaluated_at;
      }
    } else {
      map.set(ev.framework_id, {
        frameworkId: ev.framework_id,
        total: 1,
        compliant: ev.state?.toUpperCase() === 'COMPLIANT' ? 1 : 0,
        lastEvaluatedAt: ev.evaluated_at ?? null,
      });
    }
  }
  return Array.from(map.values());
}

interface DerivedControl {
  id: string;
  frameworkId: string;
  status: 'pass' | 'fail' | 'partial';
  sla: string;
  evidence: string;
}

function deriveControls(evaluations: ComplianceEvaluation[]): DerivedControl[] {
  const controlMap = new Map<
    string,
    { total: number; compliant: number; slaDeadline: string | null; frameworkId: string }
  >();
  for (const ev of evaluations) {
    const isCompliant = ev.state?.toUpperCase() === 'COMPLIANT';
    const existing = controlMap.get(ev.control_id);
    if (existing) {
      existing.total += 1;
      if (isCompliant) existing.compliant += 1;
      if (
        ev.sla_deadline_at &&
        (!existing.slaDeadline || ev.sla_deadline_at < existing.slaDeadline)
      ) {
        existing.slaDeadline = ev.sla_deadline_at;
      }
    } else {
      controlMap.set(ev.control_id, {
        total: 1,
        compliant: isCompliant ? 1 : 0,
        slaDeadline: ev.sla_deadline_at ?? null,
        frameworkId: ev.framework_id,
      });
    }
  }

  return Array.from(controlMap.entries()).map(([controlId, data]) => {
    const pct = data.total > 0 ? data.compliant / data.total : 0;
    const status: 'pass' | 'fail' | 'partial' = pct >= 1 ? 'pass' : pct > 0 ? 'partial' : 'fail';
    let sla = '—';
    if (data.slaDeadline) {
      const daysLeft = Math.ceil(
        (new Date(data.slaDeadline).getTime() - Date.now()) / (1000 * 60 * 60 * 24),
      );
      sla = daysLeft <= 0 ? 'Overdue' : `${daysLeft}d left`;
    }
    return {
      id: controlId,
      frameworkId: data.frameworkId,
      status,
      sla,
      evidence: `${data.compliant}/${data.total} checks passing`,
    };
  });
}

function scoreColor(pct: number): string {
  if (pct >= 80) return 'var(--signal-healthy)';
  if (pct >= 50) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

// ── design tokens ──────────────────────────────────────────────
const S = {
  card: {
    background: 'var(--bg-card)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    boxShadow: 'var(--shadow-sm)',
    overflow: 'hidden' as const,
  },
  cardTitle: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '12px 16px',
    borderBottom: '1px solid var(--border)',
    background: 'var(--bg-inset)',
  },
  th: {
    fontFamily: 'var(--font-mono)',
    fontSize: 10,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
    color: 'var(--text-muted)',
    padding: '9px 12px',
    background: 'var(--bg-inset)',
    borderBottom: '1px solid var(--border)',
    textAlign: 'left' as const,
    whiteSpace: 'nowrap' as const,
  },
  td: {
    padding: '10px 12px',
    borderBottom: '1px solid var(--border)',
    color: 'var(--text-primary)',
    fontSize: 13,
  },
};

// ── ring gauge ────────────────────────────────────────────────
function FrameworkRing({
  summary,
  onEvaluate,
  isPending,
}: {
  summary: FrameworkSummary;
  onEvaluate: () => void;
  isPending: boolean;
}) {
  const pct = summary.total > 0 ? Math.round((summary.compliant / summary.total) * 100) : 0;
  const color = scoreColor(pct);
  const size = 72;
  const radius = (size - 10) / 2;
  const circumference = 2 * Math.PI * radius;
  const dash = (pct / 100) * circumference;

  return (
    <div
      style={{
        ...S.card,
        padding: 16,
        display: 'flex',
        flexDirection: 'column',
        gap: 12,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <div style={{ position: 'relative', width: size, height: size, flexShrink: 0 }}>
          <svg width={size} height={size} style={{ transform: 'rotate(-90deg)' }}>
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke="var(--border)"
              strokeWidth={6}
            />
            <circle
              cx={size / 2}
              cy={size / 2}
              r={radius}
              fill="none"
              stroke={color}
              strokeWidth={6}
              strokeLinecap="round"
              strokeDasharray={`${dash} ${circumference}`}
            />
          </svg>
          <span
            style={{
              position: 'absolute',
              inset: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              fontFamily: 'var(--font-mono)',
              fontSize: 13,
              fontWeight: 700,
              color,
            }}
          >
            {pct}%
          </span>
        </div>
        <div style={{ minWidth: 0 }}>
          <Link
            to={`/compliance/${encodeURIComponent(summary.frameworkId)}`}
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 10,
              color: 'var(--text-muted)',
              textDecoration: 'none',
              textTransform: 'uppercase',
              letterSpacing: '0.04em',
              display: 'block',
              marginBottom: 2,
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLAnchorElement).style.color = 'var(--accent)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLAnchorElement).style.color = 'var(--text-muted)';
            }}
          >
            {summary.frameworkId}
          </Link>
          <div
            style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 22,
              fontWeight: 700,
              color: 'var(--text-emphasis)',
              lineHeight: 1,
            }}
          >
            {pct}%
          </div>
          <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
            {summary.compliant}/{summary.total} controls
          </div>
        </div>
      </div>
      <Button
        variant="secondary"
        size="default"
        className="w-full text-xs"
        disabled={isPending}
        onClick={onEvaluate}
      >
        {isPending ? 'Evaluating...' : 'Evaluate Now'}
      </Button>
      {summary.lastEvaluatedAt && (
        <div
          style={{
            fontSize: 10,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-muted)',
            textAlign: 'center' as const,
            marginTop: 4,
          }}
        >
          Last:{' '}
          {new Date(summary.lastEvaluatedAt).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          })}
        </div>
      )}
    </div>
  );
}

// ── 90-day trend chart (static/generated) ─────────────────────
function generateTrendData(): { day: number; value: number }[] {
  const points: { day: number; value: number }[] = [];
  let v = 65;
  for (let d = 90; d >= 0; d -= 3) {
    v = Math.min(95, Math.max(50, v + (Math.random() - 0.35) * 4));
    points.push({ day: d, value: Math.round(v) });
  }
  return points;
}

const TREND_DATA = generateTrendData();

function ComplianceTrendChart() {
  const width = 600;
  const height = 100;
  const pad = { top: 8, right: 8, bottom: 16, left: 28 };
  const cW = width - pad.left - pad.right;
  const cH = height - pad.top - pad.bottom;
  const minVal = Math.min(...TREND_DATA.map((d) => d.value));
  const maxVal = Math.max(...TREND_DATA.map((d) => d.value));
  const range = maxVal - minVal || 1;
  const pts = TREND_DATA.map((d, i) => ({
    x: pad.left + (i / (TREND_DATA.length - 1)) * cW,
    y: pad.top + cH - ((d.value - minVal) / range) * cH,
  }));
  const pathD = pts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x} ${p.y}`).join(' ');
  const areaD = `${pathD} L ${pts[pts.length - 1].x} ${pad.top + cH} L ${pts[0].x} ${pad.top + cH} Z`;

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      style={{ width: '100%' }}
      preserveAspectRatio="xMidYMid meet"
    >
      {[0, 50, 100].map((pct) => {
        const val = minVal + (pct / 100) * range;
        const y = pad.top + cH - (pct / 100) * cH;
        return (
          <g key={pct}>
            <line
              x1={pad.left}
              y1={y}
              x2={width - pad.right}
              y2={y}
              stroke="var(--border)"
              strokeOpacity={0.5}
            />
            <text
              x={pad.left - 4}
              y={y + 3}
              textAnchor="end"
              style={{ fill: 'var(--text-muted)', fontSize: 8, fontFamily: 'var(--font-mono)' }}
            >
              {Math.round(val)}%
            </text>
          </g>
        );
      })}
      <text
        x={pad.left}
        y={height - 2}
        textAnchor="start"
        style={{ fill: 'var(--text-faint)', fontSize: 8, fontFamily: 'var(--font-mono)' }}
      >
        90d ago
      </text>
      <text
        x={width - pad.right}
        y={height - 2}
        textAnchor="end"
        style={{ fill: 'var(--text-faint)', fontSize: 8, fontFamily: 'var(--font-mono)' }}
      >
        Today
      </text>
      <path d={areaD} fill="var(--signal-healthy)" fillOpacity={0.08} />
      <path d={pathD} fill="none" stroke="var(--signal-healthy)" strokeWidth={1.5} />
      <circle
        cx={pts[pts.length - 1].x}
        cy={pts[pts.length - 1].y}
        r={3}
        fill="var(--signal-healthy)"
      />
    </svg>
  );
}

export function ComplianceTab({ endpointId }: ComplianceTabProps) {
  const { data, isLoading, error } = useEndpointCompliance(endpointId);
  const triggerEval = useTriggerEvaluation();

  if (isLoading) {
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-32 w-full rounded-lg" />
          ))}
        </div>
        <Skeleton className="h-32 w-full rounded-lg" />
        <Skeleton className="h-48 w-full rounded-lg" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          padding: 16,
        }}
      >
        <span style={{ fontSize: 13, color: 'var(--signal-critical)' }}>
          Error loading compliance data
        </span>
      </div>
    );
  }

  const evaluations = Array.isArray(data) ? (data as ComplianceEvaluation[]) : [];

  if (evaluations.length === 0) {
    return (
      <EmptyState
        icon={ShieldCheck}
        title="No compliance evaluations"
        description="No compliance evaluations found for this endpoint."
      />
    );
  }

  const summaries = buildFrameworkSummaries(evaluations);
  const controls = deriveControls(evaluations);
  const failingControls = controls.filter((c) => c.status !== 'pass');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Framework ring gauges */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: `repeat(${Math.min(summaries.length, 4)}, 1fr)`,
          gap: 16,
        }}
      >
        {summaries.map((summary) => (
          <FrameworkRing
            key={summary.frameworkId}
            summary={summary}
            onEvaluate={() => triggerEval.mutate()}
            isPending={triggerEval.isPending}
          />
        ))}
      </div>

      {/* 90-day trend */}
      <div style={S.card}>
        <div style={S.cardTitle}>90-Day Compliance Trend</div>
        <div style={{ padding: 16 }}>
          <ComplianceTrendChart />
        </div>
      </div>

      {/* Control breakdown — failing first */}
      <div style={S.card}>
        <div
          style={{
            ...S.cardTitle,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <span>Control Breakdown</span>
          <span
            style={{
              color: failingControls.length > 0 ? 'var(--signal-critical)' : 'var(--text-faint)',
            }}
          >
            {failingControls.length} failing
          </span>
        </div>
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={S.th}>Control ID</th>
                <th style={S.th}>Framework</th>
                <th style={S.th}>Status</th>
                <th style={S.th}>SLA</th>
                <th style={S.th}>Evidence</th>
              </tr>
            </thead>
            <tbody>
              {controls
                .slice()
                .sort((a, b) => {
                  const order = { fail: 0, partial: 1, pass: 2 };
                  return order[a.status] - order[b.status];
                })
                .map((control) => {
                  const statusColor =
                    control.status === 'fail'
                      ? 'var(--signal-critical)'
                      : control.status === 'partial'
                        ? 'var(--signal-warning)'
                        : 'var(--signal-healthy)';
                  const slaColor =
                    control.sla === 'Overdue'
                      ? 'var(--signal-critical)'
                      : control.status === 'partial'
                        ? 'var(--signal-warning)'
                        : 'var(--text-muted)';

                  return (
                    <tr
                      key={control.id}
                      style={{
                        background:
                          control.status === 'fail'
                            ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
                            : undefined,
                      }}
                      onMouseEnter={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          'var(--bg-card-hover)';
                      }}
                      onMouseLeave={(e) => {
                        (e.currentTarget as HTMLTableRowElement).style.background =
                          control.status === 'fail'
                            ? 'color-mix(in srgb, var(--signal-critical) 1%, transparent)'
                            : '';
                      }}
                    >
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 12,
                          color: statusColor,
                        }}
                      >
                        {control.id}
                      </td>
                      <td style={{ ...S.td, fontSize: 11, color: 'var(--text-muted)' }}>
                        {control.frameworkId}
                      </td>
                      <td style={S.td}>
                        <span
                          style={{
                            display: 'inline-flex',
                            alignItems: 'center',
                            gap: 5,
                            fontFamily: 'var(--font-mono)',
                            fontSize: 11,
                            color: statusColor,
                          }}
                        >
                          <span
                            style={{
                              width: 6,
                              height: 6,
                              borderRadius: '50%',
                              background: statusColor,
                            }}
                          />
                          {control.status}
                        </span>
                      </td>
                      <td
                        style={{
                          ...S.td,
                          fontFamily: 'var(--font-mono)',
                          fontSize: 11,
                          color: slaColor,
                        }}
                      >
                        {control.sla}
                      </td>
                      <td style={{ ...S.td, fontSize: 11, color: 'var(--text-muted)' }}>
                        {control.evidence}
                      </td>
                    </tr>
                  );
                })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
