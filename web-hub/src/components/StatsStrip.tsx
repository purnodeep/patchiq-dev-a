interface StatItem {
  label: string;
  value: string | number;
  sub?: string;
  trend?: string;
  color?: string;
  barPercent?: number;
  barColor?: string;
}

export function StatsStrip({ items }: { items: StatItem[] }) {
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {items.map((item) => (
        <div
          key={item.label}
          className="rounded-lg p-4"
          style={{
            background: 'var(--bg-card)',
            border: '1px solid var(--border)',
            borderTop: `2px solid var(--accent)`,
            boxShadow: 'var(--shadow-sm)',
          }}
        >
          <p
            className="text-xs font-medium uppercase tracking-wide"
            style={{ color: 'var(--text-muted)' }}
          >
            {item.label}
          </p>
          <p
            className="mt-1 text-2xl font-bold"
            style={{ color: 'var(--text-emphasis)', fontFamily: 'var(--font-mono)' }}
          >
            {item.value}
          </p>
          {item.sub && (
            <p className="mt-0.5 text-xs" style={{ color: 'var(--text-faint)' }}>
              {item.sub}
            </p>
          )}
          {item.trend && (
            <p className="mt-1 text-xs" style={{ color: 'var(--signal-healthy)' }}>
              {item.trend}
            </p>
          )}
          {item.barPercent !== undefined && (
            <div className="mt-2 h-1.5 w-full rounded-full" style={{ background: 'var(--border)' }}>
              <div
                className="h-1.5 rounded-full"
                style={{
                  width: `${Math.min(100, item.barPercent)}%`,
                  backgroundColor: 'var(--accent)',
                }}
              />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
