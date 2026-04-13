/* eslint-disable @typescript-eslint/no-explicit-any */
import { useQueries } from '@tanstack/react-query';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { api } from '../../../api/client';
import type { FrameworkScoreSummary } from '../../../api/hooks/useCompliance';

// Single accent at varying opacity levels for each framework line
const FRAMEWORK_OPACITIES = [1, 0.7, 0.5, 0.35, 0.25, 0.18];

// Dash patterns to distinguish lines for colorblind users
const FRAMEWORK_DASH_PATTERNS = ['', '8 4', '4 4', '12 4 4 4', '2 4', '8 2 2 2'];

function getFrameworkColor(_name: string, index: number): string {
  const opacity = FRAMEWORK_OPACITIES[index % FRAMEWORK_OPACITIES.length];
  return `color-mix(in srgb, var(--accent) ${Math.round(opacity * 100)}%, transparent)`;
}

function getFrameworkDashPattern(index: number): string {
  return FRAMEWORK_DASH_PATTERNS[index % FRAMEWORK_DASH_PATTERNS.length];
}

interface ComplianceTrendProps {
  frameworks: FrameworkScoreSummary[];
}

export function ComplianceTrend({ frameworks }: ComplianceTrendProps) {
  const trendQueries = useQueries({
    queries: (frameworks ?? []).map((fw) => ({
      queryKey: ['compliance', 'frameworks', fw.framework_id, 'trend'],
      queryFn: async () => {
        const { data, error } = await api.GET(
          '/api/v1/compliance/frameworks/{frameworkId}/trend',
          {
            params: { path: { frameworkId: fw.framework_id } },
          },
        );
        if (error) throw error;
        return { frameworkId: fw.framework_id, name: fw.name, data };
      },
    })),
  });

  const isLoading = trendQueries.some((q) => q.isLoading);
  const allData = trendQueries.filter((q) => q.data).map((q) => q.data!);

  // Merge all trend data into a single array keyed by date
  const dateMap = new Map<string, Record<string, number>>();
  for (const { name, data } of allData) {
    if (!data) continue;
    for (const point of data) {
      const date = point.evaluated_at.slice(0, 10);
      const existing = dateMap.get(date) ?? {};
      existing[name] = parseFloat(point.score);
      dateMap.set(date, existing);
    }
  }

  const chartData = Array.from(dateMap.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([date, scores]) => ({ date, ...scores }));

  const dateRange = (() => {
    if (chartData.length < 2) return '';
    const first = new Date(chartData[0].date);
    const last = new Date(chartData[chartData.length - 1].date);
    const days = Math.round((last.getTime() - first.getTime()) / 86_400_000);
    if (days <= 1) return 'Last 24 Hours';
    if (days <= 14) return `Last ${days} Days`;
    if (days <= 60) return `Last ${Math.round(days / 7)} Weeks`;
    return `Last ${Math.round(days / 30)} Months`;
  })();

  if (isLoading || chartData.length === 0) return null;

  return (
    <div
      style={{
        background: 'var(--bg-card)',
        border: '1px solid var(--border)',
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        padding: 20,
        marginBottom: 4,
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
        <div
          style={{
            fontFamily: 'var(--font-sans)',
            fontSize: 13,
            fontWeight: 600,
            color: 'var(--text-primary)',
          }}
        >
          {`Compliance Trend — ${dateRange}`}
        </div>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 16 }}>
          {frameworks.map((fw, i) => {
            const dash = getFrameworkDashPattern(i);
            return (
              <div
                key={fw.framework_id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 6,
                  fontFamily: 'var(--font-mono)',
                  fontSize: 10,
                  color: 'var(--text-muted)',
                }}
              >
                <svg width={16} height={8} style={{ flexShrink: 0 }}>
                  <line
                    x1={0}
                    y1={4}
                    x2={16}
                    y2={4}
                    stroke={getFrameworkColor(fw.name, i)}
                    strokeWidth={2}
                    strokeDasharray={dash || undefined}
                  />
                </svg>
                {fw.name}
              </div>
            );
          })}
        </div>
      </div>

      <div
        role="img"
        aria-label={`Compliance trend chart showing score over time for ${frameworks.length} framework${frameworks.length !== 1 ? 's' : ''}`}
      >
        <ResponsiveContainer width="100%" height={200}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis
              dataKey="date"
              tick={{ fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
              tickFormatter={(v: string) => {
                const d = new Date(v);
                return d.toLocaleDateString('en', { month: 'short', day: 'numeric' });
              }}
            />
            <YAxis
              domain={[0, 100]}
              tick={{ fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' }}
              tickFormatter={(v: number) => `${v}%`}
              label={{
                value: 'Compliance Score (%)',
                angle: -90,
                position: 'insideLeft',
                style: { fontSize: 10, fill: 'var(--text-muted)', fontFamily: 'var(--font-mono)' },
              }}
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
            {frameworks.map((fw, i) => {
              const dash = getFrameworkDashPattern(i);
              return (
                <Line
                  key={fw.framework_id}
                  type="monotone"
                  dataKey={fw.name}
                  stroke={getFrameworkColor(fw.name, i)}
                  strokeWidth={2}
                  strokeDasharray={dash || undefined}
                  dot={false}
                />
              );
            })}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
