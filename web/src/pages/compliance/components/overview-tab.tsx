import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import {
  useComplianceTrend,
  type CategoryBreakdown,
  type ComplianceScore,
} from '../../../api/hooks/useCompliance';
import type { components } from '../../../api/types';

type ControlResult = components['schemas']['ControlResult'];

// TODO(#332): replace with generated type after OpenAPI spec regeneration
interface TrendPoint {
  evaluated_at: string;
  score: string;
}

// ─── Design tokens ────────────────────────────────────────────────────────────

const mono: React.CSSProperties = { fontFamily: 'var(--font-mono)' };

const sectionLabel: React.CSSProperties = {
  fontFamily: 'var(--font-mono)',
  fontSize: 10,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.06em',
  color: 'var(--text-muted)',
  marginBottom: 12,
};

function scoreColor(score: number): string {
  if (score >= 90) return 'var(--accent)';
  if (score >= 70) return 'var(--signal-warning)';
  return 'var(--signal-critical)';
}

// ─── Donut Gauge ─────────────────────────────────────────────────────────────

function DonutGauge({ value, labelColor }: { value: number; labelColor: string }) {
  const r = 44;
  const cx = 56;
  const cy = 56;
  const circumference = 2 * Math.PI * r;
  const filled = (value / 100) * circumference;

  return (
    <svg
      width={112}
      height={112}
      style={{ display: 'block' }}
      role="img"
      aria-label={`Compliance score: ${value}%`}
    >
      <circle cx={cx} cy={cy} r={r} fill="none" stroke="var(--border)" strokeWidth={10} />
      <circle
        cx={cx}
        cy={cy}
        r={r}
        fill="none"
        stroke="var(--accent)"
        strokeWidth={10}
        strokeDasharray={`${filled} ${circumference - filled}`}
        strokeDashoffset={circumference / 4}
        strokeLinecap="round"
        style={{ transition: 'stroke-dasharray 0.6s ease' }}
      />
      <text
        x={cx}
        y={cy - 4}
        textAnchor="middle"
        fill={labelColor}
        fontSize={22}
        fontWeight={700}
        fontFamily="var(--font-mono)"
      >
        {value}%
      </text>
      <text
        x={cx}
        y={cy + 13}
        textAnchor="middle"
        fill="var(--text-muted)"
        fontSize={9}
        fontFamily="var(--font-mono)"
      >
        overall
      </text>
    </svg>
  );
}

// ─── Trend Chart ──────────────────────────────────────────────────────────────

function ComplianceTrendPanel({ frameworkId }: { frameworkId: string }) {
  const { data: trend, isLoading } = useComplianceTrend(frameworkId);
  const trendArr = Array.isArray(trend) ? trend : [];

  const chartData = (trendArr as TrendPoint[]).map((point) => ({
    date: point.evaluated_at?.slice(0, 10) ?? '',
    score: Math.round(parseFloat(point.score) || 0),
  }));

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 12,
        }}
      >
        <div style={sectionLabel}>Compliance Score Trend</div>
        <span style={{ ...mono, fontSize: 10, color: 'var(--text-muted)' }}>Area chart</span>
      </div>

      {isLoading || chartData.length === 0 ? (
        <div
          style={{
            height: 220,
            background: 'var(--bg-inset)',
            borderRadius: 6,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            ...mono,
            fontSize: 11,
            color: 'var(--text-muted)',
          }}
        >
          {isLoading ? 'Loading trend data…' : 'No trend data yet — run an evaluation'}
        </div>
      ) : (
        <div
          role="img"
          aria-label={`Compliance score trend chart showing scores over time. Current score: ${chartData.length > 0 ? chartData[chartData.length - 1].score : 0}%.`}
          style={{ position: 'relative' }}
        >
          <span
            style={{
              position: 'absolute',
              width: 1,
              height: 1,
              padding: 0,
              margin: -1,
              overflow: 'hidden',
              clip: 'rect(0, 0, 0, 0)',
              whiteSpace: 'nowrap',
              borderWidth: 0,
            }}
          >
            Compliance score trend with {chartData.length} data points.
            {chartData.length > 0 && ` Latest score: ${chartData[chartData.length - 1].score}%.`}
          </span>
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={chartData} margin={{ top: 4, right: 4, left: -16, bottom: 0 }}>
              <defs>
                <linearGradient id="complianceGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="var(--accent)" stopOpacity={0.25} />
                  <stop offset="100%" stopColor="var(--accent)" stopOpacity={0.03} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
                tickFormatter={(v: string) => {
                  const d = new Date(v);
                  return d.toLocaleDateString('en', { month: 'short', day: 'numeric' });
                }}
                axisLine={false}
                tickLine={false}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
                tickFormatter={(v: number) => `${v}%`}
                axisLine={false}
                tickLine={false}
              />
              <Tooltip
                contentStyle={{
                  background: 'var(--bg-elevated)',
                  border: '1px solid var(--border)',
                  borderRadius: 6,
                  fontSize: 11,
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--text-primary)',
                }}
                formatter={(value: number | undefined) =>
                  value != null ? [`${value.toFixed(1)}%`] : []
                }
              />
              <Area
                type="monotone"
                dataKey="score"
                stroke="var(--accent)"
                strokeWidth={2}
                fill="url(#complianceGrad)"
                dot={false}
                activeDot={{ r: 4, fill: 'var(--accent)' }}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}

// ─── Category Breakdown Panel ─────────────────────────────────────────────────

function CategoryBreakdownPanel({ categories }: { categories: CategoryBreakdown[] }) {
  return (
    <div>
      <div style={sectionLabel}>Control Categories</div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {(categories ?? []).map((cat) => {
          const controls = cat.controls ?? [];
          const evaluated = controls.filter((c) => c.status !== 'na').length;
          const pass = controls.filter((c) => c.status === 'pass').length;
          const score = evaluated > 0 ? Math.round((pass / evaluated) * 100) : 0;
          const color = scoreColor(score);
          return (
            <div key={cat.category} style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span
                style={{
                  ...mono,
                  fontSize: 11,
                  color: 'var(--text-primary)',
                  flex: 1,
                  minWidth: 0,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {cat.category}
              </span>
              <div
                style={{
                  width: 120,
                  height: 5,
                  borderRadius: 3,
                  background: 'var(--bg-inset)',
                  overflow: 'hidden',
                  flexShrink: 0,
                }}
              >
                <div
                  style={{
                    height: '100%',
                    borderRadius: 3,
                    width: `${score}%`,
                    background: color,
                    transition: 'width 0.4s ease',
                  }}
                />
              </div>
              <span
                style={{
                  ...mono,
                  fontSize: 11,
                  fontWeight: 700,
                  color,
                  width: 36,
                  textAlign: 'right',
                  flexShrink: 0,
                }}
              >
                {score}%
              </span>
              <span
                style={{
                  ...mono,
                  fontSize: 10,
                  color: 'var(--text-muted)',
                  width: 40,
                  textAlign: 'right',
                  flexShrink: 0,
                }}
              >
                {pass}/{evaluated}
              </span>
            </div>
          );
        })}
        {(categories ?? []).length === 0 && (
          <div style={{ ...mono, fontSize: 12, color: 'var(--text-muted)', padding: '16px 0' }}>
            No category data — run an evaluation.
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Score Breakdown Panel ────────────────────────────────────────────────────

function ScoreBreakdownPanel({
  scoreVal,
  statusColor,
  passing,
  failing,
  total,
}: {
  scoreVal: number;
  statusColor: string;
  passing: number;
  failing: number;
  total: number;
}) {
  const na = total - passing - failing;

  return (
    <div>
      <div style={sectionLabel}>Score Breakdown</div>
      <div
        style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', marginBottom: 20 }}
      >
        <DonutGauge value={scoreVal} labelColor={statusColor} />
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {[
          { label: 'Passing', value: passing, color: 'var(--accent)' },
          {
            label: 'Failing',
            value: failing,
            color: failing > 0 ? 'var(--signal-critical)' : 'var(--text-muted)',
          },
          { label: 'Not Evaluated', value: na < 0 ? 0 : na, color: 'var(--text-muted)' },
        ].map(({ label, value, color }) => (
          <div
            key={label}
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              borderBottom: '1px solid var(--border)',
              paddingBottom: 7,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <span
                style={{
                  width: 6,
                  height: 6,
                  borderRadius: '50%',
                  background: color,
                  display: 'inline-block',
                }}
              />
              <span style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>{label}</span>
            </div>
            <span style={{ ...mono, fontSize: 13, fontWeight: 700, color }}>{value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ─── Failing Controls Panel ───────────────────────────────────────────────────

function FailingControlsPanel({ categories }: { categories: CategoryBreakdown[] }) {
  const failingControlsMap = new Map<string, ControlResult & { category: string }>();

  for (const cat of categories ?? []) {
    const controls = cat.controls ?? [];
    for (const ctrl of controls) {
      if (ctrl.status === 'fail' && !failingControlsMap.has(ctrl.control_id)) {
        failingControlsMap.set(ctrl.control_id, { ...ctrl, category: cat.category });
      }
    }
  }

  const failingControls = Array.from(failingControlsMap.values());

  failingControls.sort((a, b) => (b.total_endpoints ?? 0) - (a.total_endpoints ?? 0));
  const top = failingControls.slice(0, 6);

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 12,
        }}
      >
        <div style={sectionLabel}>Failing Controls</div>
        {failingControls.length > 0 && (
          <span style={{ ...mono, fontSize: 10, color: 'var(--text-muted)' }}>
            {failingControls.length} controls
          </span>
        )}
      </div>

      {top.length === 0 ? (
        <div
          style={{
            padding: '24px 0',
            textAlign: 'center',
            ...mono,
            fontSize: 12,
            color: 'var(--accent)',
          }}
        >
          All controls passing ✓
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
          {top.map((ctrl, idx) => {
            const failing = (ctrl.total_endpoints ?? 0) - (ctrl.passing_endpoints ?? 0);
            return (
              <div
                key={`${ctrl.control_id}-${ctrl.category}-${idx}`}
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: 10,
                  padding: '8px 10px',
                  background: 'color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  border: '1px solid color-mix(in srgb, var(--signal-critical) 1%, transparent)',
                  borderRadius: 6,
                }}
              >
                <span
                  style={{
                    ...mono,
                    fontSize: 10,
                    fontWeight: 700,
                    color: 'var(--signal-critical)',
                    flexShrink: 0,
                    paddingTop: 1,
                  }}
                >
                  {ctrl.control_id}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div
                    style={{
                      ...mono,
                      fontSize: 11,
                      color: 'var(--text-primary)',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    {ctrl.name}
                  </div>
                  {failing > 0 && (
                    <div
                      style={{ ...mono, fontSize: 10, color: 'var(--text-muted)', marginTop: 2 }}
                    >
                      {failing} endpoint{failing !== 1 ? 's' : ''} failing
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ─── Evaluation History Timeline ──────────────────────────────────────────────

function EvalHistoryTimeline({ frameworkId }: { frameworkId: string }) {
  const { data: trend, isLoading } = useComplianceTrend(frameworkId);
  const trendArr = Array.isArray(trend) ? trend : [];
  const recent = [...(trendArr as TrendPoint[])]
    .sort(
      (a, b) => new Date(b.evaluated_at).getTime() - new Date(a.evaluated_at).getTime(),
    )
    .slice(0, 6)
    .reverse();

  if (isLoading || recent.length === 0) return null;

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        padding: '16px 24px',
        boxShadow: 'var(--shadow-sm)',
      }}
    >
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          marginBottom: 16,
        }}
      >
        <div style={sectionLabel}>Evaluation History</div>
        <span style={{ ...mono, fontSize: 10, color: 'var(--text-muted)' }}>
          Last {recent.length} evaluations · Most recent →
        </span>
      </div>

      <div style={{ display: 'flex', alignItems: 'flex-start', overflowX: 'auto' }}>
        {recent.map((point, i) => {
          const score = Math.round(parseFloat(point.score) || 0);
          const prev = i > 0 ? Math.round(parseFloat(recent[i - 1].score) || 0) : null;
          const delta = prev !== null ? score - prev : null;
          const isLast = i === recent.length - 1;
          const color = isLast ? 'var(--accent)' : 'var(--text-muted)';

          return (
            <div
              key={point.evaluated_at}
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                flex: 1,
                minWidth: 80,
                position: 'relative',
              }}
            >
              {/* Connector */}
              {i < recent.length - 1 && (
                <div
                  style={{
                    position: 'absolute',
                    top: 9,
                    left: '50%',
                    width: '100%',
                    height: 1,
                    background: 'var(--border)',
                  }}
                />
              )}
              {/* Dot */}
              <div
                style={{
                  width: 18,
                  height: 18,
                  borderRadius: '50%',
                  background: isLast
                    ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
                    : 'var(--bg-inset)',
                  border: `2px solid ${color}`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  flexShrink: 0,
                  zIndex: 1,
                }}
              >
                <div
                  style={{
                    width: 6,
                    height: 6,
                    borderRadius: '50%',
                    background: color,
                  }}
                />
              </div>
              {/* Info */}
              <div style={{ marginTop: 8, textAlign: 'center' }}>
                <div style={{ ...mono, fontSize: 11, color: 'var(--text-muted)' }}>
                  {new Date(point.evaluated_at).toLocaleDateString('en', {
                    month: 'short',
                    day: 'numeric',
                  })}
                  {isLast && (
                    <span
                      style={{ color: 'var(--accent)', marginLeft: 4, textTransform: 'uppercase' }}
                    >
                      Now
                    </span>
                  )}
                </div>
                <div
                  style={{
                    ...mono,
                    fontSize: 13,
                    fontWeight: 700,
                    color: isLast ? 'var(--accent)' : 'var(--text-primary)',
                    marginTop: 2,
                  }}
                >
                  {score}%
                </div>
                {delta !== null && (
                  <div
                    style={{
                      ...mono,
                      fontSize: 9,
                      color:
                        delta > 0
                          ? 'var(--accent)'
                          : delta < 0
                            ? 'var(--signal-critical)'
                            : 'var(--text-muted)',
                    }}
                  >
                    {delta > 0 ? `↑ ${delta}%` : delta < 0 ? `↓ ${Math.abs(delta)}%` : '—'}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─── Main OverviewTab ─────────────────────────────────────────────────────────

interface OverviewTabProps {
  categories?: CategoryBreakdown[];
  frameworkId: string;
  scoreVal: number;
  statusColor: string;
  passing: number;
  failing: number;
  totalControls: number;
  score?: ComplianceScore;
}

export function OverviewTab({
  categories = [],
  frameworkId,
  scoreVal,
  statusColor,
  passing,
  failing,
  totalControls,
}: OverviewTabProps) {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <h2
        style={{
          position: 'absolute',
          width: 1,
          height: 1,
          padding: 0,
          margin: -1,
          overflow: 'hidden',
          clip: 'rect(0, 0, 0, 0)',
          whiteSpace: 'nowrap',
          borderWidth: 0,
        }}
      >
        Framework Overview
      </h2>
      {/* Row 1: Trend (60%) + Score Breakdown (40%) */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '3fr 2fr',
          gap: 1,
          background: 'var(--border)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <div style={{ background: 'var(--bg-card)', padding: '20px 24px' }}>
          <ComplianceTrendPanel frameworkId={frameworkId} />
        </div>
        <div style={{ background: 'var(--bg-card)', padding: '20px 24px' }}>
          <ScoreBreakdownPanel
            scoreVal={scoreVal}
            statusColor={statusColor}
            passing={passing}
            failing={failing}
            total={totalControls}
          />
        </div>
      </div>

      {/* Row 2: Category Breakdown (60%) + Failing Controls (40%) */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '3fr 2fr',
          gap: 1,
          background: 'var(--border)',
          border: '1px solid var(--border)',
          borderRadius: 8,
          overflow: 'hidden',
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <div style={{ background: 'var(--bg-card)', padding: '20px 24px' }}>
          <CategoryBreakdownPanel categories={categories} />
        </div>
        <div style={{ background: 'var(--bg-card)', padding: '20px 24px' }}>
          <FailingControlsPanel categories={categories} />
        </div>
      </div>

      {/* Evaluation History Timeline */}
      <EvalHistoryTimeline frameworkId={frameworkId} />
    </div>
  );
}
