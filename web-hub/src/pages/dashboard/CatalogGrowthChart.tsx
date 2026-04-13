import { useState } from 'react';
import { useNavigate } from 'react-router';
import { SkeletonCard } from '@patchiq/ui';
import { useCatalogGrowth } from '../../api/hooks/useDashboard';
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
} from 'recharts';

export const CatalogGrowthChart = () => {
  const navigate = useNavigate();
  const { data: growth, isLoading, isError } = useCatalogGrowth(90);
  const [hovered, setHovered] = useState(false);

  if (isLoading) {
    return <SkeletonCard />;
  }

  if (isError) {
    return (
      <div
        style={{
          background: 'var(--bg-card)',
          border: '1px solid var(--signal-critical)',
          borderRadius: 8,
          boxShadow: 'var(--shadow-sm)',
        }}
      >
        <div style={{ padding: '16px 20px 0' }}>
          <h3 style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
            Catalog Growth
          </h3>
          <p style={{ fontSize: 11, color: 'var(--signal-critical)' }}>
            Failed to load catalog growth data.
          </p>
        </div>
      </div>
    );
  }

  const chartData = (growth ?? []).map((item, i) => {
    const cumulativeTotal = (growth ?? [])
      .slice(0, i + 1)
      .reduce((sum, g) => sum + g.entries_added, 0);
    return {
      day: new Date(item.day).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      entries: cumulativeTotal,
      daily: item.entries_added,
    };
  });

  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: 'var(--bg-card)',
        border: `1px solid ${hovered ? 'var(--text-faint)' : 'var(--border)'}`,
        borderRadius: 8,
        boxShadow: 'var(--shadow-sm)',
        transition: 'border-color 150ms ease',
        display: 'flex',
        flexDirection: 'column' as const,
      }}
    >
      <div
        style={{ padding: '16px 20px 0', cursor: 'pointer' }}
        onClick={() => void navigate('/catalog')}
      >
        <h3 style={{ fontSize: 13, color: 'var(--text-muted)', fontWeight: 400 }}>
          Catalog Growth
          <span style={{ fontSize: 10, color: 'var(--text-faint)', marginLeft: 6 }}>→</span>
        </h3>
        <p style={{ fontSize: 11, color: 'var(--text-faint)' }}>
          Cumulative entries over last 90 days
        </p>
      </div>
      <div style={{ padding: '12px 20px 16px', flex: 1 }}>
        <ResponsiveContainer width="100%" height={180}>
          <AreaChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
            <XAxis
              dataKey="day"
              tick={{ fontSize: 11, fill: 'var(--text-muted)' }}
              tickLine={false}
              interval={Math.floor(chartData.length / 8)}
            />
            <YAxis tick={{ fontSize: 11, fill: 'var(--text-muted)' }} tickLine={false} />
            <Tooltip
              contentStyle={{
                background: 'var(--bg-elevated)',
                border: '1px solid var(--border)',
                borderRadius: 8,
                color: 'var(--text-primary)',
                fontSize: 12,
              }}
            />
            <Area
              type="monotone"
              dataKey="entries"
              stroke="var(--accent)"
              fill="var(--accent-subtle)"
              strokeWidth={2}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
